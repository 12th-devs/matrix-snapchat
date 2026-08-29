package connector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/snapapi"
	"github.com/colej/mautrix-snapchat/internal/store"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/networkid"
)

func (sa *SnapchatAPI) isSeen(chatName, messageID string) bool {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	set, ok := sa.seenByChat[chatName]
	if !ok {
		return false
	}
	_, seen := set[messageID]
	return seen
}

func (sa *SnapchatAPI) markSeen(chatName, messageID string) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	set, ok := sa.seenByChat[chatName]
	if !ok {
		set = make(map[string]struct{})
		sa.seenByChat[chatName] = set
	}
	set[messageID] = struct{}{}
}

func (sa *SnapchatAPI) rememberChat(chatID, chatURL, chatName, otherUserID string) {
	sa.rememberChatDetails(sidecar.Chat{
		ID:          chatID,
		URL:         chatURL,
		Name:        chatName,
		OtherUserID: otherUserID,
	})
}

func (sa *SnapchatAPI) rememberChatDetails(chat sidecar.Chat) {
	chatID := strings.TrimSpace(chat.ID)
	if chatID == "" {
		return
	}
	otherUserID := strings.TrimSpace(chat.OtherUserID)
	sa.mu.Lock()
	defer sa.mu.Unlock()
	ref := sa.chatsByID[chatID]
	ref.ID = chatID
	if strings.TrimSpace(chat.URL) != "" {
		ref.URL = chat.URL
	}
	if strings.TrimSpace(chat.Name) != "" {
		ref.Name = chat.Name
	}
	if otherUserID != "" {
		if ref.OtherUserID != otherUserID {
			delete(sa.queuedPortalResyncs, chatID)
		}
		ref.OtherUserID = otherUserID
	}
	if len(chat.ParticipantIDs) > 0 {
		ref.ParticipantIDs = append([]string{}, chat.ParticipantIDs...)
		ref.IsGroup = len(chat.ParticipantIDs) > 1
		if ref.IsGroup {
			ref.OtherUserID = ""
		}
	}
	if len(ref.ParticipantIDs) == 0 && otherUserID != "" {
		ref.ParticipantIDs = []string{otherUserID}
	}
	if chat.IsGroup {
		ref.IsGroup = true
		ref.OtherUserID = ""
	}
	if strings.TrimSpace(chat.Username) != "" {
		ref.Username = strings.TrimSpace(chat.Username)
		if ref.OtherUserID != "" {
			sa.ghostUsernames[string(makeUserID(ref.OtherUserID))] = ref.Username
		}
	}
	if chat.DisappearAfterSeconds > 0 {
		ref.DisappearAfterSeconds = chat.DisappearAfterSeconds
	}
	sa.chatsByID[chatID] = ref
	log.Printf("bridgev2 map: remembered chat chat_id=%s other_user_id=%s participants=%d is_group=%t username=%q name=%q url=%q", chatID, ref.OtherUserID, len(ref.ParticipantIDs), ref.IsGroup, ref.Username, ref.Name, ref.URL)
	if ref.Name != "" && ref.OtherUserID != "" {
		sa.rememberGhostNameLocked(makeUserID(ref.OtherUserID), ref.Name)
		if ref.Username != "" {
			sa.ghostUsernames[string(makeUserID(ref.OtherUserID))] = ref.Username
		}
	} else if ref.Name != "" {
		sa.rememberGhostNameLocked(makeUserID(ref.ID), ref.Name)
	}
}

func (sa *SnapchatAPI) rememberChatDisappear(chatID string, seconds int64) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	if sa.chatDisappearAfter == nil {
		sa.chatDisappearAfter = make(map[string]int64)
	}
	ref := sa.chatsByID[chatID]
	ref.ID = chatID
	if seconds <= 0 {
		delete(sa.chatDisappearAfter, chatID)
		ref.DisappearAfterSeconds = 0
		sa.chatsByID[chatID] = ref
		return
	}
	sa.chatDisappearAfter[chatID] = seconds
	ref.DisappearAfterSeconds = seconds
	sa.chatsByID[chatID] = ref
}

func (sa *SnapchatAPI) rememberGhostName(userID networkid.UserID, name string) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.rememberGhostNameLocked(userID, name)
}

func (sa *SnapchatAPI) rememberGhostNameLocked(userID networkid.UserID, name string) {
	id := strings.TrimSpace(string(userID))
	name = strings.TrimSpace(name)
	if id == "" || name == "" || strings.EqualFold(name, "unknown") {
		return
	}
	sa.ghostNames[id] = name
}

func (sa *SnapchatAPI) lookupGhostName(userID networkid.UserID) string {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	return sa.ghostNames[string(userID)]
}

func (sa *SnapchatAPI) rememberGhostUsername(userID networkid.UserID, username string) {
	id := strings.TrimSpace(string(userID))
	username = strings.TrimSpace(username)
	if id == "" || username == "" {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.ghostUsernames[id] = username
}

func (sa *SnapchatAPI) lookupGhostUsername(userID networkid.UserID) string {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	return sa.ghostUsernames[string(userID)]
}

func (sa *SnapchatAPI) recordRecentOutgoing(chatID, body, remoteID string) {
	chatID = strings.TrimSpace(chatID)
	body = normalizeOutgoingBody(body)
	remoteID = baseSnapchatMessageID(remoteID)
	if chatID == "" {
		return
	}
	entry := pendingOutgoing{
		RemoteID: remoteID,
		Body:     body,
		SentAt:   time.Now(),
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	items := append(sa.recentOutgoing[chatID], entry)
	cutoff := time.Now().Add(-15 * time.Minute)
	filtered := items[:0]
	for _, item := range items {
		if item.SentAt.After(cutoff) {
			filtered = append(filtered, item)
		}
	}
	sa.recentOutgoing[chatID] = filtered
}

func (sa *SnapchatAPI) shouldSuppressOutgoingEcho(chatID string, message sidecar.Message) bool {
	if !message.Outgoing {
		return false
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return false
	}
	body := normalizeOutgoingBody(message.Text)
	remoteID := baseSnapchatMessageID(message.ID)
	now := time.Now()
	cutoff := now.Add(-15 * time.Minute)
	sa.mu.Lock()
	defer sa.mu.Unlock()
	items := sa.recentOutgoing[chatID]
	if len(items) == 0 {
		return false
	}
	filtered := items[:0]
	suppressed := false
	for _, item := range items {
		if item.SentAt.Before(cutoff) {
			continue
		}
		if !suppressed {
			if item.RemoteID != "" && remoteID != "" && item.RemoteID == remoteID {
				suppressed = true
			} else if item.Body != "" && body != "" && item.Body == body && now.Sub(item.SentAt) <= 90*time.Second {
				suppressed = true
			}
		}
		filtered = append(filtered, item)
	}
	if len(filtered) == 0 {
		delete(sa.recentOutgoing, chatID)
	} else {
		sa.recentOutgoing[chatID] = filtered
	}
	return suppressed
}

func (sa *SnapchatAPI) rememberChatState(chatID string, unread bool, preview, lastMessage string) {
	if chatID == "" {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.chatState[chatID] = chatFingerprint(unread, preview, lastMessage)
}

func (sa *SnapchatAPI) recordMessageFetchFailure(chatID string) {
	if chatID == "" {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.messageFailures[chatID]++
	backoff := time.Duration(sa.messageFailures[chatID]) * 2 * time.Minute
	if backoff > 15*time.Minute {
		backoff = 15 * time.Minute
	}
	sa.messageRetryAfter[chatID] = time.Now().Add(backoff)
}

func (sa *SnapchatAPI) recordMessageFetchSuccess(chatID string) {
	if chatID == "" {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	delete(sa.messageFailures, chatID)
	delete(sa.messageRetryAfter, chatID)
}

func (sa *SnapchatAPI) lookupMessageRetryAfter(chatID string) (time.Time, bool) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	retryAfter, ok := sa.messageRetryAfter[chatID]
	return retryAfter, ok
}

func (sa *SnapchatAPI) rememberAPIMessage(chatID string, message snapapi.Message) {
	messageID, ok := parseSnapchatMessageID(message.ID)
	if !ok || chatID == "" {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	versions := sa.messageVersions[chatID]
	if versions == nil {
		versions = make(map[int64]int64)
		sa.messageVersions[chatID] = versions
	}
	if message.Version > 0 {
		versions[messageID] = message.Version
	}
	if !message.Outgoing && messageID >= sa.lastMessageID[chatID] {
		sa.lastMessageID[chatID] = messageID
		if message.Version > 0 {
			sa.lastMessageVersion[chatID] = message.Version
		}
	}
}

func (sa *SnapchatAPI) readReceiptTarget(chatID string, receipt *bridgev2.MatrixReadReceipt) (int64, int64) {
	var messageID int64
	if receipt != nil && receipt.ExactMessage != nil {
		messageID, _ = parseSnapchatMessageID(string(receipt.ExactMessage.ID))
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	if messageID <= 0 {
		messageID = sa.lastMessageID[chatID]
	}
	version := sa.lastMessageVersion[chatID]
	if messageID > 0 {
		if versions := sa.messageVersions[chatID]; versions != nil && versions[messageID] > 0 {
			version = versions[messageID]
		}
	}
	return messageID, version
}

func (sa *SnapchatAPI) lookupChatName(chatID string) string {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	ref, ok := sa.chatsByID[chatID]
	if !ok {
		return ""
	}
	return ref.Name
}

func (sa *SnapchatAPI) lookupChatURL(chatID string) string {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	ref, ok := sa.chatsByID[chatID]
	if !ok {
		return ""
	}
	return ref.URL
}

func (sa *SnapchatAPI) lookupChat(chatID string) sidecar.Chat {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	ref, ok := sa.chatsByID[chatID]
	if !ok {
		return sidecar.Chat{}
	}
	return sidecar.Chat{
		ID:             ref.ID,
		OtherUserID:    ref.OtherUserID,
		ParticipantIDs: append([]string{}, ref.ParticipantIDs...),
		IsGroup:        ref.IsGroup,
		URL:            ref.URL,
		Name:           ref.Name,
		Username:       ref.Username,
	}
}

func (sa *SnapchatAPI) lookupChatRef(chatID string) chatRef {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return chatRef{}
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	ref := sa.chatsByID[chatID]
	ref.ParticipantIDs = append([]string{}, ref.ParticipantIDs...)
	return ref
}

func (sa *SnapchatAPI) shouldSyncFromReadReceipt(chatID string) bool {
	const minReadReceiptSyncInterval = 15 * time.Second
	now := time.Now()
	sa.mu.Lock()
	defer sa.mu.Unlock()
	if last, ok := sa.lastReadReceiptSync[chatID]; ok && now.Sub(last) < minReadReceiptSyncInterval {
		return false
	}
	sa.lastReadReceiptSync[chatID] = now
	return true
}

func (sa *SnapchatAPI) isSidebarBaselineReady() bool {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	return sa.sidebarBaselineReady
}

func (sa *SnapchatAPI) markSidebarBaselineReady() {
	sa.mu.Lock()
	sa.sidebarBaselineReady = true
	sa.mu.Unlock()
}

// claimSidebarBaseline atomically reserves the one-time baseline reconciliation.
// Polls can overlap while a large Matrix portal sync is still being processed, so
// reading the flag and setting it after the loop is not sufficient.
func (sa *SnapchatAPI) claimSidebarBaseline() bool {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	if sa.sidebarBaselineReady {
		return false
	}
	sa.sidebarBaselineReady = true
	return true
}

func (sa *SnapchatAPI) completeSidebarBaseline() {
	sa.mu.Lock()
	sa.sidebarBaselineAt = time.Now()
	sa.mu.Unlock()
}

func (sa *SnapchatAPI) claimHybridBackfill(delay time.Duration) bool {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	if sa.hybridBackfillStarted || sa.sidebarBaselineAt.IsZero() || time.Since(sa.sidebarBaselineAt) < delay {
		return false
	}
	sa.hybridBackfillStarted = true
	return true
}

func (sa *SnapchatAPI) requestTimeout() time.Duration {
	seconds := 20
	if sa != nil && sa.Connector != nil && sa.Connector.Config.RequestTimeoutSeconds > 0 {
		seconds = sa.Connector.Config.RequestTimeoutSeconds
	}
	return time.Duration(seconds+15) * time.Second
}

func (sa *SnapchatAPI) messageFetchLimit() int {
	if sa != nil && sa.Connector != nil && sa.Connector.Config.MessageFetchLimit > 0 {
		return sa.Connector.Config.MessageFetchLimit
	}
	return 20
}

func (sa *SnapchatAPI) conversationFetchLimit() int {
	limit := sa.messageFetchLimit()
	if limit < 100 {
		return 100
	}
	return limit
}

func (sa *SnapchatAPI) useAPI() bool {
	return sa != nil && sa.Connector != nil && sa.Connector.apiMode() != "dom_only"
}

func (sa *SnapchatAPI) canFallbackToDOM() bool {
	return sa != nil && sa.Connector != nil && sa.Connector.domFallbackEnabled()
}

func (sa *SnapchatAPI) autoFetchMessagesEnabled() bool {
	return sa != nil && sa.Connector != nil && sa.Connector.Config.AutoFetchMessages
}

func (sa *SnapchatAPI) readReceiptsEnabled() bool {
	return sa != nil && sa.Connector != nil && sa.Connector.Config.ReadReceiptsEnabled
}

func (sa *SnapchatAPI) typingEnabled() bool {
	return sa != nil && sa.Connector != nil && sa.Connector.Config.TypingEnabled
}

func (sa *SnapchatAPI) snapMediaEnabled() bool {
	return sa != nil && sa.Connector != nil && sa.Connector.Config.SnapMediaEnabled
}

func (sa *SnapchatAPI) snapMediaOnReadEnabled() bool {
	return sa != nil && sa.Connector != nil && sa.Connector.Config.SnapMediaOnRead
}

func (sa *SnapchatAPI) sendMediaEnabled() bool {
	return sa != nil && sa.Connector != nil && sa.Connector.Config.SendMediaEnabled
}

func (sa *SnapchatAPI) ensureSnapClient(ctx context.Context) (*snapapi.Client, error) {
	if sa == nil || sa.Client == nil {
		return nil, fmt.Errorf("connector client is not initialized")
	}
	sa.apiMu.Lock()
	if sa.apiClient != nil && time.Since(sa.apiAuthCheckedAt) < 10*time.Minute {
		client := sa.apiClient
		sa.apiMu.Unlock()
		return client, nil
	}
	sa.apiMu.Unlock()

	auth, err := sa.Client.APIAuth(ctx)
	if err != nil {
		return nil, fmt.Errorf("get Snapchat API auth: %w", err)
	}
	sa.updateLoginState(&sidecar.SessionStatus{
		State:         auth.State,
		Authenticated: auth.Authenticated,
		URL:           auth.URL,
	})
	if !auth.Authenticated {
		return nil, fmt.Errorf("connector session is not authenticated (state=%s)", auth.State)
	}
	cookieString := strings.TrimSpace(auth.CookieString)
	if cookieString == "" {
		return nil, fmt.Errorf("connector did not return Snapchat cookies")
	}
	savedState := sa.loadAPISyncState()
	selfUserID := strings.TrimSpace(auth.SelfUserID)
	if selfUserID != "" {
		savedSelfUserID := strings.TrimSpace(savedState.SelfUserID)
		if savedSelfUserID != "" && !strings.EqualFold(savedSelfUserID, selfUserID) {
			log.Printf("bridgev2 sync: Snapchat account changed, resetting API sync state old_self=%s new_self=%s label=%s", savedSelfUserID, selfUserID, sa.Label)
			sa.resetSyncState()
		}
	}

	sa.apiMu.Lock()
	reused := sa.apiClient != nil && sa.apiCookieString == cookieString
	client := sa.apiClient
	if reused {
		sa.apiAuthCheckedAt = time.Now()
	}
	sa.apiMu.Unlock()
	if reused {
		return client, nil
	}

	client, err = snapapi.New(snapapi.Config{
		CookieString:        cookieString,
		SSOToken:            auth.SSOToken,
		SelfUserID:          selfUserID,
		UserAgent:           auth.BrowserUserAgent,
		SnapClientUserAgent: auth.SnapClientUserAgent,
		SecChUA:             auth.SecChUA,
		SecChUAPlatform:     auth.SecChUAPlatform,
		GRPCWebUserAgent:    auth.GRPCWebUserAgent,
		MCSCOFIDsBin:        auth.MCSCOFIDsBin,
		Timeout:             sa.requestTimeout(),
		FetchLimit:          sa.messageFetchLimit(),
		EELDecrypter:        connectorEELDecrypter{client: sa.Client, api: sa},
	})
	if err != nil {
		return nil, err
	}
	if err = client.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authenticate Snapchat API: %w", err)
	}

	sa.apiMu.Lock()
	sa.apiClient = client
	sa.apiCookieString = cookieString
	sa.apiAuthCheckedAt = time.Now()
	sa.apiMu.Unlock()
	return client, nil
}

func (sa *SnapchatAPI) invalidateSnapClient() {
	if sa == nil {
		return
	}
	sa.apiMu.Lock()
	sa.apiClient = nil
	sa.apiCookieString = ""
	sa.apiAuthCheckedAt = time.Time{}
	sa.apiMu.Unlock()
}

func (sa *SnapchatAPI) sendTextAPI(ctx context.Context, chatID, body string) (string, error) {
	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		return "", err
	}
	messageID, err := client.SendText(ctx, chatID, body)
	if err != nil {
		sa.invalidateSnapClient()
	} else if seconds := int64(client.RetentionDuration(chatID) / time.Second); seconds > 0 {
		sa.rememberChatDisappear(chatID, seconds)
	}
	return messageID, err
}

func (sa *SnapchatAPI) sendMediaAPI(ctx context.Context, chatID string, media sidecar.MediaAttachment, caption string) (string, error) {
	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		return "", err
	}
	messageID, err := client.SendMedia(ctx, chatID, snapapi.MediaAttachment{
		ID:       media.ID,
		FileName: media.FileName,
		MimeType: media.MimeType,
		Data:     media.Data,
	}, caption)
	if err != nil {
		sa.invalidateSnapClient()
	} else if seconds := int64(client.RetentionDuration(chatID) / time.Second); seconds > 0 {
		sa.rememberChatDisappear(chatID, seconds)
	}
	return messageID, err
}

func (sa *SnapchatAPI) loadAPISyncState() snapapi.State {
	state := snapapi.State{ConversationVersions: make(map[string]int64)}
	if sa == nil || sa.Connector == nil || sa.Connector.store == nil {
		return state
	}
	saved, err := sa.Connector.store.GetAPISyncState(string(makeUserLoginID(sa.Label)))
	if err != nil || saved == nil {
		if err != nil {
			log.Printf("bridgev2 sync: failed to load API sync state: %v", err)
		}
		return state
	}
	state.SyncToken = append([]byte{}, saved.SyncToken...)
	state.ConversationVersions = saved.ConversationState
	if state.ConversationVersions == nil {
		state.ConversationVersions = make(map[string]int64)
	}
	state.SelfUserID = saved.SelfUserID
	return state
}

func (sa *SnapchatAPI) saveAPISyncState(state snapapi.State) {
	if sa == nil || sa.Connector == nil || sa.Connector.store == nil {
		return
	}
	if state.ConversationVersions == nil {
		state.ConversationVersions = make(map[string]int64)
	}
	if err := sa.Connector.store.UpsertAPISyncState(store.APISyncState{
		UserID:            string(makeUserLoginID(sa.Label)),
		SyncToken:         append([]byte{}, state.SyncToken...),
		ConversationState: state.ConversationVersions,
		SelfUserID:        state.SelfUserID,
		UpdatedAt:         time.Now(),
	}); err != nil {
		log.Printf("bridgev2 sync: failed to save API sync state: %v", err)
	}
}

func (sa *SnapchatAPI) resetSyncState() {
	sa.apiMu.Lock()
	sa.apiClient = nil
	sa.apiCookieString = ""
	sa.apiMu.Unlock()
	sa.mu.Lock()
	sa.seenByChat = make(map[string]map[string]struct{})
	sa.chatsByID = make(map[string]chatRef)
	sa.ghostNames = make(map[string]string)
	sa.ghostUsernames = make(map[string]string)
	sa.chatState = make(map[string]string)
	sa.lastMessageID = make(map[string]int64)
	sa.lastMessageVersion = make(map[string]int64)
	sa.messageVersions = make(map[string]map[int64]int64)
	sa.messageFailures = make(map[string]int)
	sa.messageRetryAfter = make(map[string]time.Time)
	sa.lastReadReceiptSync = make(map[string]time.Time)
	sa.ghostAvatarCheckedAt = make(map[string]time.Time)
	sa.queuedPortalResyncs = make(map[string]struct{})
	sa.sidebarBaselineReady = false
	sa.sidebarBaselineAt = time.Time{}
	sa.hybridBackfillStarted = false
	sa.mu.Unlock()
	if sa.Connector != nil {
		_ = sa.Connector.resetSyncState()
	}
}

func (sa *SnapchatAPI) persistChat(chat sidecar.Chat) {
	if sa.Connector == nil || sa.Connector.store == nil {
		return
	}
	if ref := sa.lookupChatRef(chat.ID); ref.ID != "" {
		if strings.TrimSpace(chat.OtherUserID) == "" {
			chat.OtherUserID = ref.OtherUserID
		}
		if len(chat.ParticipantIDs) == 0 && len(ref.ParticipantIDs) > 0 {
			chat.ParticipantIDs = append([]string{}, ref.ParticipantIDs...)
		}
		if strings.TrimSpace(chat.Username) == "" {
			chat.Username = ref.Username
		}
		if !chat.IsGroup {
			chat.IsGroup = ref.IsGroup
		}
	}
	participantIDs := append([]string{}, chat.ParticipantIDs...)
	otherUserID := strings.TrimSpace(chat.OtherUserID)
	if len(participantIDs) == 0 && otherUserID != "" {
		participantIDs = []string{otherUserID}
	}
	roomType := "dm"
	if chat.IsGroup || len(participantIDs) > 1 {
		roomType = "group_dm"
	}
	if err := sa.Connector.store.UpsertPortal(store.PortalState{
		PortalKey:      chat.ID,
		RemoteID:       chat.ID,
		RemoteName:     chat.Name,
		OtherUserID:    otherUserID,
		ParticipantIDs: participantIDs,
		RoomType:       roomType,
		Username:       strings.TrimSpace(chat.Username),
		Preview:        chat.Preview,
		LastMessage:    chat.LastMessage,
		Unread:         chat.Unread,
		LastSyncedAt:   time.Now(),
	}); err != nil {
		log.Printf("bridgev2 sync: failed to persist chat metadata chat_id=%s name=%q: %v", chat.ID, chat.Name, err)
	}
}

func (sa *SnapchatAPI) updateLoginState(status *sidecar.SessionStatus) {
	if status == nil {
		return
	}
	if meta, ok := sa.UserLoginMetadata(); ok {
		meta.Authenticated = status.Authenticated
		meta.LastURL = status.URL
	}
	if sa.Connector != nil && sa.Connector.store != nil {
		_ = sa.Connector.store.UpsertLogin(store.LoginState{
			UserID:        string(makeUserLoginID(sa.Label)),
			RemoteID:      status.URL,
			RemoteName:    "Snapchat Web",
			SessionJSON:   store.MarshalJSON(status),
			LastSeenState: status.State,
		})
	}
}

func (sa *SnapchatAPI) currentSelfUserID() string {
	if sa == nil {
		return ""
	}
	sa.apiMu.Lock()
	client := sa.apiClient
	sa.apiMu.Unlock()
	if client != nil {
		if self := strings.TrimSpace(client.SelfUserID()); self != "" {
			return self
		}
	}
	return strings.TrimSpace(sa.loadAPISyncState().SelfUserID)
}

func (sa *SnapchatAPI) isSelfAuthor(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || sa == nil {
		return false
	}
	if strings.EqualFold(value, "you") || strings.EqualFold(value, "me") {
		return true
	}
	if strings.EqualFold(value, sa.Label) ||
		strings.EqualFold(normalizeIDPart(value), string(makeUserLoginID(sa.Label))) {
		return true
	}
	if selfUserID := sa.currentSelfUserID(); selfUserID != "" {
		return strings.EqualFold(value, selfUserID) ||
			strings.EqualFold(normalizeIDPart(value), normalizeIDPart(selfUserID))
	}
	return false
}

func (sa *SnapchatAPI) normalizeRemoteMessageDirection(chatID string, message sidecar.Message) sidecar.Message {
	if !message.Outgoing && (sa.isSelfAuthor(message.AuthorID) || sa.isSelfAuthor(message.Author)) {
		log.Printf("bridgev2 sender_resolve: normalizing self-authored message as outgoing chat_id=%s message_id=%s author_id=%s author=%q",
			chatID, message.ID, message.AuthorID, message.Author)
		message.Outgoing = true
		if strings.TrimSpace(message.Author) == "" || sa.isSelfAuthor(message.Author) {
			message.Author = "You"
		}
	}
	return message
}

func (sa *SnapchatAPI) normalizeAPIMessageDirection(chatID string, message snapapi.Message) snapapi.Message {
	if !message.Outgoing && (sa.isSelfAuthor(message.AuthorID) || sa.isSelfAuthor(message.Author)) {
		log.Printf("bridgev2 sender_resolve: normalizing self-authored API message as outgoing chat_id=%s message_id=%s author_id=%s author=%q content_type=%s",
			chatID, message.ID, message.AuthorID, message.Author, message.ContentType)
		message.Outgoing = true
		if strings.TrimSpace(message.Author) == "" || sa.isSelfAuthor(message.Author) {
			message.Author = "You"
		}
	}
	return message
}

func (sa *SnapchatAPI) messageSenderID(chat sidecar.Chat, message sidecar.Message) networkid.UserID {
	message = sa.normalizeRemoteMessageDirection(chat.ID, message)
	if message.Outgoing {
		return makeUserID(sa.Label)
	}
	if otherUserID := strings.TrimSpace(chat.OtherUserID); otherUserID != "" {
		if strings.TrimSpace(message.AuthorID) == "" || strings.EqualFold(message.AuthorID, otherUserID) {
			return makeUserID(otherUserID)
		}
	}
	if id := strings.TrimSpace(message.AuthorID); id != "" {
		return makeUserID(id)
	}
	if senderID := sa.remoteUserIDForChat(chat.ID, chat.Name); strings.TrimSpace(string(senderID)) != "" {
		return senderID
	}
	return makeUserID(message.Author)
}
