package snapapi

import (
	"context"
	"fmt"
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
	if err := c.doGRPC(ctx, paths.SYNC_CONVERSATIONS, req, &resp); err != nil {
		return nil, err
	}

	entries := append([]*protos.ConversationEntry{}, resp.GetConversations()...)
	nextToken := resp.GetSyncToken()
	if len(nextToken) == 0 {
		nextToken = syncToken
	}
	if len(state.SyncToken) == 0 || resp.GetIncomplete() {
		paged, pagedToken, err := c.queryAllConversations(ctx, nextToken, fetchLimit)
		if err == nil && len(paged) > 0 {
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
		if err := c.hydrateConversations(ctx, needsConversation); err != nil {
			return nil, err
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
	encoded, err := encodeUUIDString(chatID)
	if err != nil {
		return nil, err
	}
	req := &protos.DeltaSyncRequest{
		SelfUserId:     c.selfUUID(),
		ConversationId: encoded,
	}
	var resp protos.DeltaSyncResponse
	if err = c.doGRPC(ctx, paths.DELTA_SYNC_CONVERSATIONS, req, &resp); err != nil {
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
	req := &protos.BatchDeltaSyncRequest{DeltaSyncRequests: requests}
	var resp protos.BatchDeltaSyncResponse
	if err := c.doGRPC(ctx, paths.BATCH_DELTA_SYNC_CONVERSATIONS, req, &resp); err != nil {
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
		if err := c.doGRPC(ctx, paths.QUERY_CONVERSATIONS, req, &resp); err != nil {
			return entries, token, err
		}
		entries = append(entries, resp.GetConversations()...)
		if len(resp.GetSyncToken()) > 0 {
			token = resp.GetSyncToken()
		}
		last := resp.GetLastConversation()
		if resp.GetNoMore() || last == nil {
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
	}
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
