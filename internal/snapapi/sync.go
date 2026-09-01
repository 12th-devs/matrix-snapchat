package snapapi

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
)

func (c *Client) Sync(ctx context.Context, state State, fetchLimit int) (*SyncResult, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}
	if fetchLimit <= 0 {
		fetchLimit = 40
	}
	if state.ConversationVersions == nil {
		state.ConversationVersions = make(map[string]int64)
	}

	versionInfos := conversationVersionInfos(state.ConversationVersions)
	syncToken := state.SyncToken
	if len(syncToken) == 0 {
		syncToken = []byte("useV3")
	}

	req := &protos.SyncConversationsRequest{
		SelfUserId:   c.selfUUID(),
		SyncToken:    syncToken,
		VersionInfos: versionInfos,
	}
	var resp protos.SyncConversationsResponse
	if err := c.doGRPC(ctx, syncConversationsURL, req, &resp); err != nil {
		return nil, err
	}

	entries := append([]*protos.ConversationEntry{}, resp.GetConversations()...)
	nextToken := resp.GetSyncToken()
	if len(nextToken) == 0 {
		nextToken = syncToken
	}
	if len(state.SyncToken) == 0 || resp.GetIncomplete() {
		// The full conversation listing must NOT carry the incremental sync
		// token: the server treats that as "you already know everything" and
		// returns nothing, which truncated first syncs to the SyncConversations
		// page size (40 of 74 chats). A fresh listing starts with no token.
		paged, pagedToken, err := c.queryAllConversations(ctx, nil, fetchLimit)
		if err != nil {
			log.Printf("snapapi sync: conversation listing incomplete: %v (keeping %d sync entries)", err, len(entries))
		}
		if len(paged) > 0 {
			entries = mergeEntries(entries, paged)
			if len(pagedToken) > 0 {
				nextToken = pagedToken
			}
		}
	}

	newVersions := cloneVersions(state.ConversationVersions)
	changed := make(map[string]struct{})
	needsConversation := make([]*protos.ConversationEntry, 0, len(entries))
	for _, entry := range entries {
		id := uuidToString(entry.GetVersionInfo().GetConversationId())
		if id == "" || entry.GetIsDeleted() || entry.GetHideFromSendtoAndSearch() {
			continue
		}
		version := entry.GetVersionInfo().GetConversationVersion()
		if prev, ok := newVersions[id]; !ok || prev != version || entry.GetNeedsSync() {
			changed[id] = struct{}{}
			needsConversation = append(needsConversation, entry)
		}
		newVersions[id] = version
	}
	if len(needsConversation) > 0 {
		// Hydration failure must not fail the whole sync: the entries already
		// carry everything needed for portal creation, and conversations are
		// hydrated lazily (one DeltaSync) whenever their messages are needed.
		if err := c.hydrateConversations(ctx, needsConversation); err != nil {
			log.Printf("snapapi sync: conversation hydration incomplete for %d chats: %v", len(needsConversation), err)
		}
	}
	if err := c.resolveEntryNames(ctx, entries); err != nil {
		// Names are important but not important enough to block sync. The bridge
		// can still create stable portals using conversation IDs.
	}

	chats := make([]Chat, 0, len(entries))
	for _, entry := range entries {
		id := uuidToString(entry.GetVersionInfo().GetConversationId())
		if id == "" || entry.GetIsDeleted() || entry.GetHideFromSendtoAndSearch() {
			continue
		}
		chat := c.chatFromEntry(entry)
		if chat.ID == "" {
			continue
		}
		chats = append(chats, chat)
	}
	sort.SliceStable(chats, func(i, j int) bool {
		return chats[i].LastActivityAt.After(chats[j].LastActivityAt)
	})

	return &SyncResult{
		Chats:          chats,
		ChangedChatIDs: changed,
		State: State{
			SyncToken:            nextToken,
			ConversationVersions: newVersions,
			SelfUserID:           c.SelfUserID(),
		},
	}, nil
}

func (c *Client) conversation(ctx context.Context, chatID string) (*protos.Conversation, error) {
	c.mu.Lock()
	if conv := c.conversations[chatID]; conv != nil {
		c.mu.Unlock()
		return conv, nil
	}
	c.mu.Unlock()
	return c.conversationFresh(ctx, chatID)
}

// conversationFresh always performs a DeltaSync for the conversation (bypassing
// the cache) and refreshes the cache. Message mutations (read receipts, erases)
// require the conversation's CURRENT version, which a long-lived cached
// conversation cannot provide.
func (c *Client) conversationFresh(ctx context.Context, chatID string) (*protos.Conversation, error) {
	encoded, err := encodeUUIDString(chatID)
	if err != nil {
		return nil, err
	}
	req := &protos.DeltaSyncRequest{
		SelfUserId:     c.selfUUID(),
		ConversationId: encoded,
	}
	var resp protos.DeltaSyncResponse
	if err = c.doGRPC(ctx, deltaSyncURL, req, &resp); err != nil {
		return nil, err
	}
	conv := resp.GetConversation()
	if conv == nil {
		return nil, fmt.Errorf("conversation %s not found", chatID)
	}
	c.mu.Lock()
	c.conversations[chatID] = conv
	c.conversationIDs[chatID] = conv.GetConversationId()
	c.mu.Unlock()
	return conv, nil
}

func (c *Client) forgetConversation(chatID string) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.conversations, chatID)
	delete(c.conversationIDs, chatID)
}

func (c *Client) hydrateConversations(ctx context.Context, entries []*protos.ConversationEntry) error {
	requests := make([]*protos.DeltaSyncRequest, 0, len(entries))
	for _, entry := range entries {
		convID := entry.GetVersionInfo().GetConversationId()
		id := uuidToString(convID)
		if id == "" {
			continue
		}
		requests = append(requests, &protos.DeltaSyncRequest{
			SelfUserId:     c.selfUUID(),
			ConversationId: convID,
		})
	}
	if len(requests) == 0 {
		return nil
	}
	// Snapchat rejects oversized BatchDeltaSync requests (grpc-status:3
	// INVALID_ARGUMENT was observed with 63 conversations), so the batch is
	// chunked, and a rejected chunk falls back to per-conversation DeltaSync so
	// one bad conversation cannot block the rest.
	const batchChunkSize = 20
	for start := 0; start < len(requests); start += batchChunkSize {
		end := start + batchChunkSize
		if end > len(requests) {
			end = len(requests)
		}
		chunk := requests[start:end]
		if err := c.hydrateConversationsBatch(ctx, chunk); err != nil {
			log.Printf("snapapi sync: batch hydration chunk [%d,%d) failed (%v); falling back to per-conversation delta sync", start, end, err)
			c.hydrateConversationsIndividually(ctx, chunk)
		}
	}
	return nil
}

// hydrateConversationsBatch sends one BatchDeltaSync request and caches the
// returned conversations.
func (c *Client) hydrateConversationsBatch(ctx context.Context, requests []*protos.DeltaSyncRequest) error {
	req := &protos.BatchDeltaSyncRequest{DeltaSyncRequests: requests}
	var resp protos.BatchDeltaSyncResponse
	if err := c.doGRPC(ctx, batchDeltaSyncURL, req, &resp); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, item := range resp.GetDeltaSyncResponses() {
		conv := item.GetSuccessResponse().GetConversation()
		if conv == nil {
			continue
		}
		id := uuidToString(conv.GetConversationId())
		if id == "" {
			continue
		}
		c.conversations[id] = conv
		c.conversationIDs[id] = conv.GetConversationId()
	}
	return nil
}

// hydrateConversationsIndividually hydrates conversations one DeltaSync at a
// time; failures are logged and skipped so a single bad conversation cannot
// stall the whole sync.
func (c *Client) hydrateConversationsIndividually(ctx context.Context, requests []*protos.DeltaSyncRequest) {
	for _, dr := range requests {
		id := uuidToString(dr.GetConversationId())
		if id == "" {
			continue
		}
		req := &protos.DeltaSyncRequest{SelfUserId: dr.GetSelfUserId(), ConversationId: dr.GetConversationId()}
		var resp protos.DeltaSyncResponse
		if err := c.doGRPC(ctx, deltaSyncURL, req, &resp); err != nil {
			log.Printf("snapapi sync: per-conversation delta sync failed chat_id=%s: %v", id, err)
			continue
		}
		conv := resp.GetConversation()
		if conv == nil {
			continue
		}
		c.mu.Lock()
		c.conversations[id] = conv
		c.conversationIDs[id] = conv.GetConversationId()
		c.mu.Unlock()
	}
}

// syncConversationsURL, queryConversationsURL, batchDeltaSyncURL, deltaSyncURL,
// updateContentMessageURL and createContentMessageURL are package-level seams
// so tests can point the endpoints at a local server; production uses the real
// constants.
var (
	syncConversationsURL    = paths.SYNC_CONVERSATIONS
	queryConversationsURL   = paths.QUERY_CONVERSATIONS
	batchDeltaSyncURL       = paths.BATCH_DELTA_SYNC_CONVERSATIONS
	deltaSyncURL            = paths.DELTA_SYNC_CONVERSATIONS
	updateContentMessageURL = paths.UPDATE_CONTENT_MESSAGE
	createContentMessageURL = paths.CREATE_CONTENT_MESSAGE
)

func (c *Client) queryAllConversations(ctx context.Context, syncToken []byte, pageSize int) ([]*protos.ConversationEntry, []byte, error) {
	if pageSize <= 0 {
		pageSize = 100
	}
	var entries []*protos.ConversationEntry
	var pagination *protos.QueryConversationsRequest_PaginationInfo
	token := syncToken
	for page := 0; page < 5; page++ {
		req := &protos.QueryConversationsRequest{
			SelfUserId:        c.selfUUID(),
			RequestedPageSize: int32(pageSize),
			PaginationInfo:    pagination,
			SyncToken:         token,
		}
		var resp protos.QueryConversationsResponse
		if err := c.doGRPC(ctx, queryConversationsURL, req, &resp); err != nil {
			return entries, token, fmt.Errorf("query conversations page %d: %w", page, err)
		}
		pageEntries := resp.GetConversations()
		entries = append(entries, pageEntries...)
		log.Printf("snapapi sync: query conversations page %d: received %d entries (total %d) no_more=%v last_indicator=%v token_len=%d",
			page, len(pageEntries), len(entries), resp.GetNoMore(), resp.GetLastConversation() != nil, len(resp.GetSyncToken()))
		if len(resp.GetSyncToken()) > 0 {
			token = resp.GetSyncToken()
		}
		if resp.GetNoMore() {
			break
		}
		if len(pageEntries) == 0 {
			// Empty page without no_more: treat as the end instead of looping.
			log.Printf("snapapi sync: query conversations page %d: empty page, stopping", page)
			break
		}
		last := resp.GetLastConversation()
		if last == nil {
			log.Printf("snapapi sync: query conversations page %d: missing last-conversation indicator, cannot page further", page)
			break
		}
		pagination = &protos.QueryConversationsRequest_PaginationInfo{
			OldestConversationOrderTimestamp: last.GetOldestConversationOrderTimestamp(),
			OldestConversationId:             last.GetOldestConversationId(),
		}
	}
	return entries, token, nil
}

func (c *Client) resolveEntryNames(ctx context.Context, entries []*protos.ConversationEntry) error {
	unknown := make([]string, 0)
	seen := make(map[string]struct{})
	for _, entry := range entries {
		needsName := strings.TrimSpace(entry.GetTitle()) == ""
		for _, participant := range entry.GetParticipants() {
			id := uuidToString(participant)
			if id == "" || id == c.SelfUserID() {
				continue
			}
			if !c.needsPublicProfile(id, needsName) {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			unknown = append(unknown, id)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	profiles, err := c.fetchPublicProfiles(ctx, unknown)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, profile := range profiles {
		if profile.Username != "" {
			c.usernamesByUserID[id] = profile.Username
		}
		if profile.Name != "" {
			c.namesByUserID[id] = profile.Name
		}
		if profile.AvatarURL != "" {
			c.avatarsByUserID[id] = profile.AvatarURL
		}
	}
	return nil
}

func (c *Client) chatFromEntry(entry *protos.ConversationEntry) Chat {
	id := uuidToString(entry.GetVersionInfo().GetConversationId())
	name := strings.TrimSpace(entry.GetTitle())
	otherUserID := ""
	username := ""
	participants := make([]string, 0, len(entry.GetParticipants()))
	nonSelfParticipants := 0
	for _, participant := range entry.GetParticipants() {
		userID := uuidToString(participant)
		if userID == "" || userID == c.SelfUserID() {
			continue
		}
		participants = append(participants, userID)
		nonSelfParticipants++
		if otherUserID == "" {
			otherUserID = userID
			username = c.lookupUsername(userID)
		}
		if name == "" {
			name = c.lookupName(userID)
			if name == "" {
				name = c.lookupUsername(userID)
			}
			if name == "" {
				name = shortID(userID)
			}
		}
	}
	if nonSelfParticipants != 1 {
		otherUserID = ""
		username = ""
	}
	if name == "" {
		name = shortID(id)
	}
	info := entry.GetLastFeedUpdateInfo()
	preview, unread := previewFromDisplayInfo(info)
	lastAt := timeFromMillis(entry.GetLastEventTimestamp())
	if info.GetDisplayTimestamp() > 0 {
		lastAt = timeFromMillis(info.GetDisplayTimestamp())
	}
	return Chat{
		ID:             id,
		OtherUserID:    otherUserID,
		ParticipantIDs: participants,
		IsGroup:        nonSelfParticipants > 1,
		Name:           name,
		Username:       username,
		AvatarURL:      c.lookupAvatarURL(otherUserID),
		Preview:        preview,
		LastMessage:    preview,
		Unread:         unread,
		Version:        entry.GetVersionInfo().GetConversationVersion(),
		LastActivityAt: lastAt,
		DisappearAfter: c.retentionDurationForChat(id),
		ReadWatermarks: c.participantReadWatermarks(id),
	}
}

// participantReadWatermarks collects each participant's ReadHighWatermark
// (last-read message ID) from the hydrated conversation object, if available.
// These watermarks are what Snapchat updates when a participant reads messages.
func (c *Client) participantReadWatermarks(chatID string) map[string]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	conv := c.conversations[chatID]
	if conv == nil {
		return nil
	}
	watermarks := make(map[string]int64)
	for _, participant := range conv.GetParticipants() {
		if participant == nil {
			continue
		}
		if watermark := participant.GetReadHighWatermark(); watermark > 0 {
			watermarks[uuidToString(participant.GetUserId())] = watermark
		}
	}
	if len(watermarks) == 0 {
		return nil
	}
	return watermarks
}

func conversationVersionInfos(versions map[string]int64) []*protos.ConversationVersionInfo {
	infos := make([]*protos.ConversationVersionInfo, 0, len(versions))
	for id, version := range versions {
		encoded, err := encodeUUIDString(id)
		if err != nil {
			continue
		}
		infos = append(infos, &protos.ConversationVersionInfo{
			ConversationId:      encoded,
			ConversationVersion: version,
		})
	}
	return infos
}

func mergeEntries(primary, secondary []*protos.ConversationEntry) []*protos.ConversationEntry {
	byID := make(map[string]*protos.ConversationEntry, len(primary)+len(secondary))
	for _, entry := range secondary {
		if id := uuidToString(entry.GetVersionInfo().GetConversationId()); id != "" {
			byID[id] = entry
		}
	}
	for _, entry := range primary {
		if id := uuidToString(entry.GetVersionInfo().GetConversationId()); id != "" {
			byID[id] = entry
		}
	}
	merged := make([]*protos.ConversationEntry, 0, len(byID))
	for _, entry := range byID {
		merged = append(merged, entry)
	}
	return merged
}

func cloneVersions(in map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
