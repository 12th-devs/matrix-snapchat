package bridgev2

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/bridgev2/simplevent"
	"maunium.net/go/mautrix/event"

	"github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/snapapi"
	"github.com/colej/mautrix-snapchat/internal/store"
)

type chatRef struct {
	ID          string
	URL         string
	Name        string
	OtherUserID string
}

type SnapchatAPI struct {
	Connector *SnapchatConnector
	UserLogin *bridgev2.UserLogin
	Label     string
	Client    *connector.Client

	apiMu                sync.Mutex
	apiClient            *snapapi.Client
	apiCookieString      string
	apiAuthCheckedAt     time.Time
	mu                   sync.Mutex
	seenByChat           map[string]map[string]struct{}
	chatsByID            map[string]chatRef
	ghostNames           map[string]string
	chatState            map[string]string
	lastMessageID        map[string]int64
	lastMessageVersion   map[string]int64
	messageVersions      map[string]map[int64]int64
	messageFailures      map[string]int
	messageRetryAfter    map[string]time.Time
	lastReadReceiptSync  map[string]time.Time
	queuedPortalResyncs  map[string]struct{}
	sidebarBaselineReady bool
}

var _ bridgev2.NetworkAPI = (*SnapchatAPI)(nil)
var _ bridgev2.ReadReceiptHandlingNetworkAPI = (*SnapchatAPI)(nil)

var relativeTimestampPattern = regexp.MustCompile(`(?i)^(\d+)\s*([mhdwy])$`)
var sidebarRelativeAgePattern = regexp.MustCompile(`(?i)(^|[\s\-·|:])\d+\s*[mhdwy]\b`)

func NewSnapchatAPI(sc *SnapchatConnector, login *bridgev2.UserLogin, label string) *SnapchatAPI {
	return &SnapchatAPI{
		Connector:           sc,
		UserLogin:           login,
		Label:               label,
		Client:              sc.newClient(),
		seenByChat:          make(map[string]map[string]struct{}),
		chatsByID:           make(map[string]chatRef),
		ghostNames:          make(map[string]string),
		chatState:           make(map[string]string),
		lastMessageID:       make(map[string]int64),
		lastMessageVersion:  make(map[string]int64),
		messageVersions:     make(map[string]map[int64]int64),
		messageFailures:     make(map[string]int),
		messageRetryAfter:   make(map[string]time.Time),
		lastReadReceiptSync: make(map[string]time.Time),
		queuedPortalResyncs: make(map[string]struct{}),
	}
}

func (sa *SnapchatAPI) Connect(ctx context.Context) {
	sa.restoreChatMappings()
	sa.queueStoredPortalResyncs()
	go sa.pollLoop(ctx)
}

func (sa *SnapchatAPI) Disconnect() {}

func (sa *SnapchatAPI) IsLoggedIn() bool {
	status, err := sa.Client.Status(context.Background())
	if err == nil {
		sa.updateLoginState(status)
		return status.Authenticated
	}

	meta, _ := sa.UserLoginMetadata()
	if meta != nil && meta.Authenticated {
		return true
	}
	if sa.Connector != nil && sa.Connector.store != nil {
		state, stateErr := sa.Connector.store.GetLogin(string(makeUserLoginID(sa.Label)))
		if stateErr == nil && state != nil && strings.EqualFold(state.LastSeenState, "ready") {
			return true
		}
	}
	return false
}

func (sa *SnapchatAPI) LogoutRemote(ctx context.Context) {}

func (sa *SnapchatAPI) GetCapabilities(ctx context.Context, portal *bridgev2.Portal) *event.RoomFeatures {
	return sa.Connector.chatCapabilities()
}

func (sa *SnapchatAPI) IsThisUser(ctx context.Context, userID networkid.UserID) bool {
	return userID == makeUserID(sa.Label)
}

func (sa *SnapchatAPI) GetChatInfo(ctx context.Context, portal *bridgev2.Portal) (*bridgev2.ChatInfo, error) {
	chatID := string(portal.ID)
	name := sa.lookupChatName(chatID)
	if name == "" {
		name = chatID
	}
	return sa.chatInfoFor(chatID, name), nil
}

func (sa *SnapchatAPI) GetUserInfo(ctx context.Context, ghost *bridgev2.Ghost) (*bridgev2.UserInfo, error) {
	name := sa.lookupGhostName(ghost.ID)
	if name == "" {
		name = string(ghost.ID)
	}
	return &bridgev2.UserInfo{
		Name:        &name,
		Identifiers: []string{fmt.Sprintf("snapchat:%s", ghost.ID)},
	}, nil
}

func (sa *SnapchatAPI) HandleMatrixMessage(ctx context.Context, msg *bridgev2.MatrixMessage) (*bridgev2.MatrixMessageResponse, error) {
	if msg == nil || msg.Portal == nil || msg.Content == nil {
		return nil, fmt.Errorf("missing Matrix message data")
	}
	chatID := string(msg.Portal.ID)
	if media, caption, ok, err := sa.matrixMediaToSend(ctx, msg); ok || err != nil {
		if err != nil {
			return nil, err
		}
		chatName := sa.lookupChatName(chatID)
		chatURL := sa.lookupChatURL(chatID)
		if err = sa.Client.SendMedia(ctx, chatID, chatName, chatURL, media, caption); err != nil {
			return nil, fmt.Errorf("send snapchat media: %w", err)
		}
		return &bridgev2.MatrixMessageResponse{
			DB: &database.Message{
				ID:       makeMessageID(chatID + "-media-" + media.ID),
				SenderID: makeUserID(sa.Label),
			},
		}, nil
	}
	body, ok := sa.matrixTextToSend(msg)
	if !ok {
		log.Printf("bridgev2 send: ignored non-chat Matrix message chat_id=%s msgtype=%s", chatID, msg.Content.MsgType)
		return sa.ignoredMatrixMessageResponse(msg), nil
	}
	if !sa.useAPI() {
		return nil, fmt.Errorf("send snapchat message: api_mode=%s is unsafe in no-open mode; API sending is required", sa.Connector.apiMode())
	}
	messageID, err := sa.sendTextAPI(ctx, chatID, body)
	if err != nil {
		return nil, fmt.Errorf("send snapchat message via API (DOM fallback disabled): %w", err)
	}
	if messageID == "" {
		messageID = chatID + "-" + body
	}
	return &bridgev2.MatrixMessageResponse{
		DB: &database.Message{
			ID:       makeMessageID(messageID),
			SenderID: makeUserID(sa.Label),
		},
	}, nil
}

func (sa *SnapchatAPI) HandleMatrixReadReceipt(ctx context.Context, receipt *bridgev2.MatrixReadReceipt) error {
	if receipt == nil || receipt.Implicit {
		return nil
	}
	if !sa.readReceiptsEnabled() {
		return nil
	}
	chatID := ""
	if receipt.Portal != nil {
		chatID = string(receipt.Portal.ID)
	}
	if chatID == "" || !sa.shouldSyncFromReadReceipt(chatID) {
		return nil
	}
	if !sa.useAPI() {
		log.Printf("bridgev2 receipts: skipping DOM read receipt side effects chat_id=%s read_up_to=%s", chatID, receipt.ReadUpTo.Format(time.RFC3339))
		return nil
	}
	messageID, version := sa.readReceiptTarget(chatID, receipt)
	if messageID <= 0 {
		log.Printf("bridgev2 receipts: no API message id available chat_id=%s read_up_to=%s", chatID, receipt.ReadUpTo.Format(time.RFC3339))
		return nil
	}
	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		log.Printf("bridgev2 receipts: API client unavailable chat_id=%s message_id=%d: %v", chatID, messageID, err)
		return nil
	}
	if err = client.MarkRead(ctx, chatID, messageID, version); err != nil {
		sa.invalidateSnapClient()
		log.Printf("bridgev2 receipts: API read update failed chat_id=%s message_id=%d version=%d: %v", chatID, messageID, version, err)
		return nil
	}
	log.Printf("bridgev2 receipts: marked Snapchat read via API chat_id=%s message_id=%d version=%d", chatID, messageID, version)
	return nil
}

func (sa *SnapchatAPI) UserLoginMetadata() (*UserLoginMetadata, bool) {
	if sa == nil || sa.UserLogin == nil {
		return nil, false
	}
	meta, ok := sa.UserLogin.Metadata.(*UserLoginMetadata)
	return meta, ok
}

func (sa *SnapchatAPI) pollLoop(ctx context.Context) {
	sa.pollOnce(ctx)

	ticker := time.NewTicker(sa.Connector.pollInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sa.pollOnce(ctx)
		}
	}
}

func (sa *SnapchatAPI) pollOnce(ctx context.Context) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil {
		log.Printf("bridgev2 sync: skipping poll because user login is not fully initialized label=%s", sa.Label)
		return
	}
	if !sa.useAPI() {
		log.Printf("bridgev2 sync: skipping DOM poll in no-open mode label=%s api_mode=%s", sa.Label, sa.Connector.apiMode())
		return
	}
	if err := sa.pollOnceAPI(ctx); err == nil {
		return
	} else {
		log.Printf("bridgev2 sync: API poll failed label=%s: %v", sa.Label, err)
		return
	}
}

func (sa *SnapchatAPI) pollOnceDOM(ctx context.Context) {
	chats, err := sa.Client.ListChats(ctx)
	if err != nil {
		log.Printf("bridgev2 sync: list chats failed label=%s: %v", sa.Label, err)
		return
	}
	chats = dedupeChatsByName(chats)
	log.Printf("bridgev2 sync: fetched %d chats for label=%s", len(chats), sa.Label)
	sidebarUpdates := 0
	baselineReady := sa.isSidebarBaselineReady()
	for _, chat := range chats {
		sa.rememberChat(chat.ID, chat.URL, chat.Name, chat.OtherUserID)
		sa.persistChat(chat.ID, chat.Name, chat.Preview, chat.LastMessage, chat.Unread)
		sa.queueChatResync(chat.ID, chat.Name)
		if fingerprint, ok := sa.shouldQueueSidebarUpdate(chat, baselineReady); ok {
			sa.queueSidebarUpdate(chat, fingerprint)
			sidebarUpdates++
		}
	}
	sa.markSidebarBaselineReady()
	log.Printf("bridgev2 sync: queued %d sidebar update notices for label=%s", sidebarUpdates, sa.Label)
}

func (sa *SnapchatAPI) pollOnceAPI(ctx context.Context) error {
	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		return err
	}
	prevState := sa.loadAPISyncState()
	result, err := client.Sync(ctx, prevState, sa.messageFetchLimit())
	if err != nil {
		sa.invalidateSnapClient()
		return err
	}

	firstSync := len(prevState.SyncToken) == 0 && len(prevState.ConversationVersions) == 0
	baselineReady := sa.isSidebarBaselineReady()
	sidebarUpdates := 0
	messageSyncs := 0
	initialFetches := 0
	for idx, apiChat := range result.Chats {
		chat := connectorChatFromAPI(apiChat)
		sa.rememberChat(chat.ID, chat.URL, chat.Name, chat.OtherUserID)
		sa.persistChat(chat.ID, chat.Name, chat.Preview, chat.LastMessage, chat.Unread)
		sa.queueChatResync(chat.ID, chat.Name)
		if fingerprint, ok := sa.shouldQueueSidebarUpdate(chat, baselineReady); ok {
			sa.queueSidebarUpdate(chat, fingerprint)
			sidebarUpdates++
		}

		if sa.autoFetchMessagesEnabled() {
			_, changed := result.ChangedChatIDs[chat.ID]
			shouldFetch := changed && !firstSync
			if firstSync && (idx < 8 || chat.Unread) {
				shouldFetch = true
				initialFetches++
			}
			if shouldFetch {
				if err := sa.syncChatMessagesAPI(ctx, chat, apiChat.Version, "api poll"); err != nil {
					log.Printf("bridgev2 sync: API message fetch failed chat=%q id=%s: %v", chat.Name, chat.ID, err)
				} else {
					messageSyncs++
				}
			}
		}
	}
	sa.saveAPISyncState(result.State)
	sa.markSidebarBaselineReady()
	log.Printf("bridgev2 sync: API fetched %d chats, synced %d chats, queued %d sidebar notices, first_sync=%t initial_fetches=%d auto_fetch_messages=%t label=%s",
		len(result.Chats), messageSyncs, sidebarUpdates, firstSync, initialFetches, sa.autoFetchMessagesEnabled(), sa.Label)
	return nil
}

func (sa *SnapchatAPI) selectPollTargets(chats []connector.Chat) []connector.Chat {
	const maxPriorityChats = 8
	const maxTopChats = 3

	priority := make([]connector.Chat, 0, maxPriorityChats)
	selected := make(map[string]string, maxPriorityChats)
	now := time.Now()

	sa.mu.Lock()
	defer sa.mu.Unlock()

	addTarget := func(chat connector.Chat, fingerprint string) {
		if len(priority) >= maxPriorityChats || chat.ID == "" {
			return
		}
		if _, ok := selected[chat.ID]; ok {
			return
		}
		priority = append(priority, chat)
		selected[chat.ID] = fingerprint
	}

	for idx, chat := range chats {
		if retryAfter, ok := sa.messageRetryAfter[chat.ID]; ok && now.Before(retryAfter) {
			continue
		}

		fingerprint := chatFingerprint(chat.Unread, chat.Preview, chat.LastMessage)
		prev, ok := sa.chatState[chat.ID]

		if idx < maxTopChats {
			addTarget(chat, fingerprint)
		}
		if chat.Unread || !ok || prev != fingerprint {
			addTarget(chat, fingerprint)
			// Leave unselected changed chats with their old fingerprint so they stay
			// eligible on the next poll instead of becoming permanently empty rooms.
			continue
		}
	}

	for chatID, fingerprint := range selected {
		sa.chatState[chatID] = fingerprint
	}
	return priority
}

func (sa *SnapchatAPI) shouldQueueSidebarUpdate(chat connector.Chat, baselineReady bool) (string, bool) {
	if chat.ID == "" {
		return "", false
	}
	fingerprint := chatFingerprint(chat.Unread, chat.Preview, chat.LastMessage)

	sa.mu.Lock()
	prev, ok := sa.chatState[chat.ID]
	sa.chatState[chat.ID] = fingerprint
	sa.mu.Unlock()

	if !baselineReady {
		return fingerprint, false
	}
	if !ok || prev == fingerprint {
		return fingerprint, false
	}
	if chat.Unread {
		return fingerprint, true
	}
	_, shouldQueue := sidebarUpdateText(chat)
	return fingerprint, shouldQueue
}

func (sa *SnapchatAPI) queueSidebarUpdate(chat connector.Chat, fingerprint string) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" {
		return
	}
	body, ok := sidebarUpdateText(chat)
	if !ok {
		return
	}
	hash := sha1.Sum([]byte(chat.ID + "|" + fingerprint + "|" + body))
	message := connector.Message{
		ID:        "sidebar-" + chat.ID + "-" + hex.EncodeToString(hash[:8]),
		Author:    chat.Name,
		Text:      body,
		Timestamp: time.Now().Format(time.RFC3339),
		Outgoing:  false,
	}
	sa.queueRemoteMessage(chat, message)
}

func sidebarUpdateText(chat connector.Chat) (string, bool) {
	detail := strings.TrimSpace(chat.Preview)
	if detail == "" {
		detail = strings.TrimSpace(chat.LastMessage)
	}
	lower := strings.ToLower(detail)
	if isPassiveSidebarStatus(lower) {
		return "", false
	}
	if chat.Unread || looksLikeSnapOrMediaStatus(lower) {
		return "New Snap", true
	}
	if detail == "" {
		return "", false
	}
	return "New Chat", true
}

func isPassiveSidebarStatus(lower string) bool {
	lower = strings.TrimSpace(lower)
	if lower == "" {
		return false
	}
	status := strings.FieldsFunc(lower, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == 183 || r == ':' || r == '-' || r == '|'
	})
	if len(status) == 0 {
		return false
	}
	switch status[0] {
	case "opened", "delivered", "sent", "read", "seen":
		return true
	default:
		return false
	}
}

func looksLikeSnapOrMediaStatus(lower string) bool {
	lower = strings.TrimSpace(lower)
	return strings.Contains(lower, "snap") ||
		strings.Contains(lower, "received") ||
		strings.Contains(lower, "photo") ||
		strings.Contains(lower, "video")
}

func dedupeChatsByName(chats []connector.Chat) []connector.Chat {
	if len(chats) < 2 {
		return chats
	}
	deduped := make([]connector.Chat, 0, len(chats))
	seen := make(map[string]struct{}, len(chats))
	for _, chat := range chats {
		key := strings.ToLower(strings.TrimSpace(chat.Name))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(chat.ID))
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, chat)
	}
	return deduped
}

func (sa *SnapchatAPI) syncChatMessages(ctx context.Context, chat connector.Chat, reason string) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" {
		return
	}
	if !sa.autoFetchMessagesEnabled() {
		log.Printf("bridgev2 sync: skipping message fetch in no-open mode chat=%q id=%s reason=%s", chat.Name, chat.ID, reason)
		return
	}
	if sa.useAPI() {
		if err := sa.syncChatMessagesAPI(ctx, chat, 0, reason); err == nil {
			return
		} else if !sa.canFallbackToDOM() {
			log.Printf("bridgev2 sync: API message fetch failed chat=%q id=%s reason=%s: %v", chat.Name, chat.ID, reason, err)
			return
		} else {
			log.Printf("bridgev2 sync: API message fetch failed, falling back to DOM chat=%q id=%s reason=%s: %v", chat.Name, chat.ID, reason, err)
		}
	}
	sa.syncChatMessagesDOM(ctx, chat, reason)
}

func (sa *SnapchatAPI) syncChatMessagesDOM(ctx context.Context, chat connector.Chat, reason string) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" {
		return
	}
	if !sa.autoFetchMessagesEnabled() {
		log.Printf("bridgev2 sync: refusing DOM message fetch in no-open mode chat=%q id=%s reason=%s", chat.Name, chat.ID, reason)
		return
	}
	if retryAfter, ok := sa.lookupMessageRetryAfter(chat.ID); ok && time.Now().Before(retryAfter) {
		log.Printf("bridgev2 sync: skipping message fetch due backoff chat=%q id=%s reason=%s", chat.Name, chat.ID, reason)
		return
	}
	messages, err := sa.Client.Messages(ctx, chat.ID, chat.Name, chat.URL)
	if err != nil {
		sa.recordMessageFetchFailure(chat.ID)
		log.Printf("bridgev2 sync: fetch messages failed chat=%q id=%s reason=%s: %v", chat.Name, chat.ID, reason, err)
		return
	}
	sa.recordMessageFetchSuccess(chat.ID)
	log.Printf("bridgev2 sync: fetched %d messages for chat=%q id=%s reason=%s", len(messages), chat.Name, chat.ID, reason)
	for _, message := range messages {
		sa.queueRemoteMessage(chat, message)
	}
}

func (sa *SnapchatAPI) syncChatMessagesAPI(ctx context.Context, chat connector.Chat, version int64, reason string) error {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" {
		return nil
	}
	if !sa.autoFetchMessagesEnabled() {
		log.Printf("bridgev2 sync: refusing API message fetch in no-open mode chat=%q id=%s reason=%s", chat.Name, chat.ID, reason)
		return nil
	}
	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		return err
	}
	messages, err := client.QueryMessages(ctx, chat.ID, version, sa.messageFetchLimit())
	if err != nil {
		return err
	}
	log.Printf("bridgev2 sync: API fetched %d messages for chat=%q id=%s reason=%s", len(messages), chat.Name, chat.ID, reason)
	states := make([]store.MessageState, 0, len(messages))
	for _, apiMessage := range messages {
		message := connectorMessageFromAPI(apiMessage)
		if message.ID == "" {
			continue
		}
		if sa.snapMediaEnabled() && len(apiMessage.Media) > 0 {
			message.Media = sa.downloadAPIMessageMedia(ctx, client, apiMessage)
		}
		sa.rememberAPIMessage(chat.ID, apiMessage)
		states = append(states, store.MessageState{
			PortalKey:    chat.ID,
			RemoteID:     message.ID,
			Author:       message.Author,
			Text:         message.Text,
			Outgoing:     message.Outgoing,
			TimestampRaw: message.Timestamp,
			LastSeenAt:   time.Now(),
		})
		sa.queueRemoteMessage(chat, message)
	}
	if len(states) > 0 && sa.Connector != nil && sa.Connector.store != nil {
		_ = sa.Connector.store.UpsertMessages(states)
	}
	return nil
}

func (sa *SnapchatAPI) queueRemoteMessage(chat connector.Chat, message connector.Message) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" || message.ID == "" {
		return
	}
	if sa.isSeen(chat.ID, message.ID) {
		return
	}
	sa.markSeen(chat.ID, message.ID)
	if message.Outgoing {
		log.Printf("bridgev2 sync: skipping outgoing Snapchat echo chat=%q id=%s message_id=%s", chat.Name, chat.ID, message.ID)
		return
	}

	copyMsg := message
	sa.rememberChat(chat.ID, chat.URL, chat.Name, chat.OtherUserID)
	if !message.Outgoing {
		if strings.TrimSpace(message.AuthorID) != "" {
			sa.rememberGhostName(makeUserID(message.AuthorID), message.Author)
		}
		if strings.TrimSpace(chat.OtherUserID) != "" {
			sa.rememberGhostName(makeUserID(chat.OtherUserID), chat.Name)
		}
	}
	senderID := sa.messageSenderID(chat, message)
	sender := bridgev2.EventSender{
		IsFromMe: message.Outgoing,
		Sender:   senderID,
	}
	if strings.HasPrefix(message.ID, "sidebar-") {
		// Sidebar notices are bridge-generated indicators, not real remote chat
		// messages. Send them through the bridge bot so they don't race ghost joins
		// while freshly recreated portal rooms are still settling.
		sender = bridgev2.EventSender{}
	}
	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(chat.ID),
		Receiver: sa.UserLogin.ID,
	}
	sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.Message[connector.Message]{
		EventMeta: simplevent.EventMeta{
			Type: bridgev2.RemoteEventMessage,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("message_id", message.ID)
			},
			PortalKey:    portalKey,
			CreatePortal: true,
			Sender:       sender,
			Timestamp:    parseMessageTimestamp(message.Timestamp),
		},
		ID:                 makeMessageID(message.ID),
		Data:               copyMsg,
		ConvertMessageFunc: sa.convertMessage,
	})
}

func chatFingerprint(unread bool, preview, lastMessage string) string {
	return fmt.Sprintf("%t|%s|%s", unread, normalizeSidebarFingerprintPart(preview), normalizeSidebarFingerprintPart(lastMessage))
}

func normalizeSidebarFingerprintPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = sidebarRelativeAgePattern.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(value), " ")
}

func parseMessageTimestamp(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts
	}

	now := time.Now()
	location := now.Location()
	lower := strings.ToLower(raw)
	noon := func(ts time.Time) time.Time {
		return time.Date(ts.Year(), ts.Month(), ts.Day(), 12, 0, 0, 0, location)
	}
	switch lower {
	case "today":
		return noon(now)
	case "yesterday":
		return noon(now.AddDate(0, 0, -1))
	}
	if ts, ok := parseRelativeDateWithClock(lower, raw, now, location); ok {
		return ts
	}
	if match := relativeTimestampPattern.FindStringSubmatch(raw); match != nil {
		amount, err := strconv.Atoi(match[1])
		if err == nil {
			switch strings.ToLower(match[2]) {
			case "m":
				return now.Add(-time.Duration(amount) * time.Minute)
			case "h":
				return now.Add(-time.Duration(amount) * time.Hour)
			case "d":
				return now.AddDate(0, 0, -amount)
			case "w":
				return now.AddDate(0, 0, -7*amount)
			case "y":
				return now.AddDate(-amount, 0, 0)
			}
		}
	}
	if weekday, ok := parseWeekday(lower); ok {
		daysBack := (int(now.Weekday()) - int(weekday) + 7) % 7
		return noon(now.AddDate(0, 0, -daysBack))
	}

	layouts := []string{
		"January 2, 2006 3:04 PM",
		"January 2, 2006 at 3:04 PM",
		"January 2 2006 3:04 PM",
		"Jan 2, 2006 3:04 PM",
		"Jan 2, 2006 at 3:04 PM",
		"January 2 2006",
		"January 2, 2006",
		"January 2",
		"January 2 3:04 PM",
		"January 2 at 3:04 PM",
		"Jan 2 2006",
		"Jan 2, 2006",
		"Jan 2",
		"Jan 2 3:04 PM",
		"Jan 2 at 3:04 PM",
		"3:04 PM",
	}
	for _, layout := range layouts {
		if ts, err := time.ParseInLocation(layout, raw, location); err == nil {
			hasYear := strings.Contains(layout, "2006")
			hasMonth := strings.Contains(layout, "Jan") || strings.Contains(layout, "January")
			if !hasYear && hasMonth {
				ts = time.Date(now.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond(), location)
				if ts.After(now.Add(24 * time.Hour)) {
					ts = ts.AddDate(-1, 0, 0)
				}
				if !strings.Contains(layout, "3:04") {
					ts = noon(ts)
				}
			} else if !hasYear && !hasMonth {
				ts = time.Date(now.Year(), now.Month(), now.Day(), ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond(), location)
				if ts.After(now.Add(2 * time.Hour)) {
					ts = ts.AddDate(0, 0, -1)
				}
			}
			return ts
		}
	}
	return time.Time{}
}

func parseRelativeDateWithClock(lower, raw string, now time.Time, location *time.Location) (time.Time, bool) {
	parseClock := func(value string) (time.Time, bool) {
		value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "at "))
		value = strings.ReplaceAll(value, ".", "")
		layouts := []string{"3:04 PM", "3 PM", "15:04", "15"}
		for _, layout := range layouts {
			if ts, err := time.ParseInLocation(layout, value, location); err == nil {
				return ts, true
			}
		}
		return time.Time{}, false
	}
	withClock := func(day time.Time, rest string) (time.Time, bool) {
		clock, ok := parseClock(rest)
		if !ok {
			return time.Time{}, false
		}
		return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, location), true
	}
	for _, prefix := range []string{"today", "yesterday"} {
		if strings.HasPrefix(lower, prefix+" ") {
			day := now
			if prefix == "yesterday" {
				day = now.AddDate(0, 0, -1)
			}
			return withClock(day, strings.TrimSpace(raw[len(prefix):]))
		}
	}
	for _, field := range strings.Fields(lower) {
		weekday, ok := parseWeekday(strings.TrimSuffix(field, ","))
		if !ok {
			continue
		}
		prefixLen := len(field)
		if len(raw) < prefixLen {
			return time.Time{}, false
		}
		daysBack := (int(now.Weekday()) - int(weekday) + 7) % 7
		return withClock(now.AddDate(0, 0, -daysBack), strings.TrimSpace(raw[prefixLen:]))
	}
	return time.Time{}, false
}

func parseWeekday(raw string) (time.Weekday, bool) {
	switch raw {
	case "sunday", "sun":
		return time.Sunday, true
	case "monday", "mon":
		return time.Monday, true
	case "tuesday", "tue", "tues":
		return time.Tuesday, true
	case "wednesday", "wed":
		return time.Wednesday, true
	case "thursday", "thu", "thur", "thurs":
		return time.Thursday, true
	case "friday", "fri":
		return time.Friday, true
	case "saturday", "sat":
		return time.Saturday, true
	default:
		return time.Sunday, false
	}
}

func (sa *SnapchatAPI) convertMessage(ctx context.Context, portal *bridgev2.Portal, intent bridgev2.MatrixAPI, message connector.Message) (*bridgev2.ConvertedMessage, error) {
	body := strings.TrimSpace(message.Text)
	if body == "" {
		body = "[Unsupported Snapchat event]"
	}
	if len(message.Media) > 0 && portal != nil {
		parts := make([]*bridgev2.ConvertedMessagePart, 0, len(message.Media))
		for _, media := range message.Media {
			if len(media.Data) == 0 {
				continue
			}
			mimeType := normalizeMediaMIME(media.Data, media.MimeType)
			msgType := matrixMsgTypeForMedia(mimeType)
			fileName := mediaFileNameForMIME(media.FileName, msgType, mimeType)
			mxc, file, err := intent.UploadMedia(ctx, portal.MXID, media.Data, fileName, mimeType)
			if err != nil {
				log.Printf("bridgev2 sync: failed to upload Snapchat media message_id=%s media_id=%s: %v", message.ID, media.ID, err)
				continue
			}
			partBody := body
			if partBody == "" || isGeneratedSnapchatNotice(partBody) {
				// Some Matrix clients use body as the displayed/downloaded filename
				// for encrypted media, so don't use "New Snap" as the body here.
				partBody = fileName
			}
			parts = append(parts, &bridgev2.ConvertedMessagePart{
				Type: event.EventMessage,
				Content: &event.MessageEventContent{
					MsgType:  msgType,
					Body:     partBody,
					FileName: fileName,
					URL:      mxc,
					File:     file,
					Info: &event.FileInfo{
						MimeType: mimeType,
						Size:     len(media.Data),
					},
				},
			})
		}
		if len(parts) > 0 {
			return &bridgev2.ConvertedMessage{Parts: parts}, nil
		}
	}
	msgType := event.MsgText
	if strings.HasPrefix(body, "[Unsupported Snapchat") || strings.HasPrefix(body, "[Snapchat") || isGeneratedSnapchatNotice(body) {
		msgType = event.MsgNotice
	}
	return &bridgev2.ConvertedMessage{
		Parts: []*bridgev2.ConvertedMessagePart{{
			Type: event.EventMessage,
			Content: &event.MessageEventContent{
				MsgType: msgType,
				Body:    body,
			},
		}},
	}, nil
}

func (sa *SnapchatAPI) matrixTextToSend(msg *bridgev2.MatrixMessage) (string, bool) {
	body := strings.TrimSpace(msg.Content.Body)
	if body == "" {
		return "", false
	}
	switch msg.Content.MsgType {
	case event.MsgText:
	case event.MsgEmote:
	default:
		return "", false
	}
	if isGeneratedSnapchatNotice(body) {
		return "", false
	}
	return body, true
}

func (sa *SnapchatAPI) matrixMediaToSend(ctx context.Context, msg *bridgev2.MatrixMessage) (connector.MediaAttachment, string, bool, error) {
	if msg == nil || msg.Content == nil {
		return connector.MediaAttachment{}, "", false, nil
	}
	switch msg.Content.MsgType {
	case event.MsgImage, event.MsgVideo, event.MsgFile:
	default:
		return connector.MediaAttachment{}, "", false, nil
	}
	if !sa.snapMediaEnabled() {
		log.Printf("bridgev2 send: media message ignored because snap_media_enabled=false msgtype=%s", msg.Content.MsgType)
		return connector.MediaAttachment{}, "", true, fmt.Errorf("Snapchat snap/media sending is disabled in bridge config")
	}
	if msg.Portal == nil || msg.Portal.Bridge == nil || msg.Portal.Bridge.Bot == nil {
		return connector.MediaAttachment{}, "", true, fmt.Errorf("missing Matrix media downloader")
	}
	uri := msg.Content.URL
	if uri == "" && msg.Content.File != nil {
		uri = msg.Content.File.URL
	}
	if uri == "" {
		return connector.MediaAttachment{}, "", true, fmt.Errorf("Matrix media message did not include a downloadable URL")
	}
	data, err := msg.Portal.Bridge.Bot.DownloadMedia(ctx, uri, msg.Content.File)
	if err != nil {
		return connector.MediaAttachment{}, "", true, fmt.Errorf("download Matrix media: %w", err)
	}
	if len(data) == 0 {
		return connector.MediaAttachment{}, "", true, fmt.Errorf("Matrix media download was empty")
	}
	mimeType := ""
	if msg.Content.Info != nil {
		mimeType = strings.TrimSpace(msg.Content.Info.MimeType)
	}
	mimeType = normalizeMediaMIME(data, mimeType)
	fileName := strings.TrimSpace(msg.Content.FileName)
	if fileName == "" {
		fileName = strings.TrimSpace(msg.Content.Body)
	}
	fileName = mediaFileNameForMIME(fileName, msg.Content.MsgType, mimeType)
	caption := strings.TrimSpace(msg.Content.Body)
	if caption == fileName {
		caption = ""
	}
	return connector.MediaAttachment{
		ID:       stableTextID(fileName + ":" + fmt.Sprint(len(data))),
		FileName: sanitizeOutgoingFileName(fileName),
		MimeType: mimeType,
		Data:     data,
	}, caption, true, nil
}

func isGeneratedSnapchatNotice(body string) bool {
	trimmed := strings.TrimSpace(body)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(trimmed, "[Snapchat") || strings.HasPrefix(trimmed, "[Unsupported Snapchat") {
		return true
	}
	if isPassiveSidebarStatus(lower) {
		return true
	}
	switch lower {
	case "received", "read", "seen", "opened", "delivered", "sent", "new snap", "new chat":
		return true
	default:
		return false
	}
}

func (sa *SnapchatAPI) downloadAPIMessageMedia(ctx context.Context, client *snapapi.Client, message snapapi.Message) []connector.MediaAttachment {
	attachments := make([]connector.MediaAttachment, 0, len(message.Media))
	for _, media := range message.Media {
		data, mimeType, err := client.DownloadMedia(ctx, media)
		if err != nil {
			log.Printf("bridgev2 sync: Snapchat media unavailable message_id=%s media_id=%s url=%t data=%d: %v", message.ID, media.ID, media.URL != "", len(media.Data), err)
			continue
		}
		fileName := strings.TrimSpace(media.FileName)
		mimeType = normalizeMediaMIME(data, mimeType)
		fileName = mediaFileNameForMIME(fileName, matrixMsgTypeForMedia(mimeType), mimeType)
		attachments = append(attachments, connector.MediaAttachment{
			ID:       media.ID,
			FileName: fileName,
			MimeType: mimeType,
			Data:     data,
		})
	}
	return attachments
}

func matrixMsgTypeForMedia(mimeType string) event.MessageType {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return event.MsgImage
	case strings.HasPrefix(mimeType, "video/"):
		return event.MsgVideo
	default:
		return event.MsgFile
	}
}

func normalizeMediaMIME(data []byte, hinted string) string {
	hinted = strings.ToLower(strings.TrimSpace(strings.Split(hinted, ";")[0]))
	switch hinted {
	case "image/jpg":
		hinted = "image/jpeg"
	}
	detected := ""
	if len(data) > 0 {
		detected = strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0]))
		if detected == "image/jpg" {
			detected = "image/jpeg"
		}
	}
	if detected != "" && detected != "application/octet-stream" {
		return detected
	}
	if hinted != "" {
		return hinted
	}
	return "application/octet-stream"
}

func mediaFileNameForMIME(fileName string, msgType event.MessageType, mimeType string) string {
	fileName = sanitizeOutgoingFileName(fileName)
	lower := strings.ToLower(fileName)
	if fileName == "" || fileName == "snap-media.bin" || strings.HasSuffix(lower, ".bin") || !strings.Contains(fileName, ".") {
		return defaultMediaFileName(msgType, mimeType)
	}
	return fileName
}

func defaultMediaFileName(msgType event.MessageType, mimeType string) string {
	lowerMime := strings.ToLower(mimeType)
	switch {
	case msgType == event.MsgImage && strings.Contains(lowerMime, "gif"):
		return "snap.gif"
	case msgType == event.MsgImage && strings.Contains(lowerMime, "png"):
		return "snap.png"
	case msgType == event.MsgImage && strings.Contains(lowerMime, "webp"):
		return "snap.webp"
	case msgType == event.MsgImage:
		return "snap.jpg"
	case msgType == event.MsgVideo && strings.Contains(lowerMime, "quicktime"):
		return "snap.mov"
	case msgType == event.MsgVideo:
		return "snap.mp4"
	case strings.HasPrefix(lowerMime, "audio/"):
		return "snap-audio.mp3"
	default:
		return "snap-media.bin"
	}
}

func sanitizeOutgoingFileName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
		} else if r == ' ' {
			builder.WriteByte('-')
		}
	}
	if builder.Len() == 0 {
		return ""
	}
	return builder.String()
}

func stableTextID(value string) string {
	hash := sha1.Sum([]byte(value))
	return hex.EncodeToString(hash[:])[:16]
}

func (sa *SnapchatAPI) ignoredMatrixMessageResponse(msg *bridgev2.MatrixMessage) *bridgev2.MatrixMessageResponse {
	eventID := ""
	if msg != nil && msg.Event != nil {
		eventID = string(msg.Event.ID)
	}
	if eventID == "" {
		eventID = fmt.Sprintf("ignored-%d", time.Now().UnixNano())
	}
	return &bridgev2.MatrixMessageResponse{
		DB: &database.Message{
			ID:       makeMessageID("ignored-" + eventID),
			SenderID: makeUserID(sa.Label),
		},
	}
}

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
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	ref := sa.chatsByID[chatID]
	ref.ID = chatID
	if strings.TrimSpace(chatURL) != "" {
		ref.URL = chatURL
	}
	if strings.TrimSpace(chatName) != "" {
		ref.Name = chatName
	}
	if strings.TrimSpace(otherUserID) != "" {
		if ref.OtherUserID != otherUserID {
			delete(sa.queuedPortalResyncs, chatID)
		}
		ref.OtherUserID = otherUserID
	}
	sa.chatsByID[chatID] = ref
	if ref.Name != "" && ref.OtherUserID != "" {
		sa.rememberGhostNameLocked(makeUserID(ref.OtherUserID), ref.Name)
	} else if ref.Name != "" {
		sa.rememberGhostNameLocked(makeUserID(ref.ID), ref.Name)
	}
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

func (sa *SnapchatAPI) lookupChat(chatID string) connector.Chat {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	ref, ok := sa.chatsByID[chatID]
	if !ok {
		return connector.Chat{}
	}
	return connector.Chat{
		ID:          ref.ID,
		OtherUserID: ref.OtherUserID,
		URL:         ref.URL,
		Name:        ref.Name,
	}
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

func (sa *SnapchatAPI) snapMediaEnabled() bool {
	return sa != nil && sa.Connector != nil && sa.Connector.Config.SnapMediaEnabled
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
	sa.updateLoginState(&connector.SessionStatus{
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
	selfUserID := strings.TrimSpace(auth.SelfUserID)
	if selfUserID == "" {
		selfUserID = strings.TrimSpace(sa.loadAPISyncState().SelfUserID)
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
		SelfUserID:          selfUserID,
		UserAgent:           auth.BrowserUserAgent,
		SnapClientUserAgent: auth.SnapClientUserAgent,
		SecChUA:             auth.SecChUA,
		SecChUAPlatform:     auth.SecChUAPlatform,
		GRPCWebUserAgent:    auth.GRPCWebUserAgent,
		MCSCOFIDsBin:        auth.MCSCOFIDsBin,
		Timeout:             sa.requestTimeout(),
		FetchLimit:          sa.messageFetchLimit(),
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

func (sa *SnapchatAPI) remoteUserIDForChat(chatID, chatName string) networkid.UserID {
	chatID = strings.TrimSpace(chatID)
	chatName = strings.TrimSpace(chatName)
	otherUserID := ""
	sa.mu.Lock()
	if ref, ok := sa.chatsByID[chatID]; ok {
		otherUserID = strings.TrimSpace(ref.OtherUserID)
		if chatName == "" {
			chatName = strings.TrimSpace(ref.Name)
		}
	}
	sa.mu.Unlock()
	if chatName != "" && !strings.EqualFold(chatName, chatID) {
		return makeUserID(chatName)
	}
	if otherUserID != "" {
		return makeUserID(otherUserID)
	}
	if chatID != "" {
		return makeUserID(chatID)
	}
	return makeUserID(chatName)
}

func (sa *SnapchatAPI) chatInfoFor(chatID, chatName string) *bridgev2.ChatInfo {
	name := strings.TrimSpace(chatName)
	if name == "" {
		name = "Snapchat"
	}
	remoteUserID := sa.remoteUserIDForChat(chatID, name)
	sa.rememberGhostName(remoteUserID, name)
	ownUserID := makeUserID(sa.Label)
	roomType := database.RoomTypeDM
	return &bridgev2.ChatInfo{
		Name: &name,
		Type: &roomType,
		Members: &bridgev2.ChatMemberList{
			IsFull:           true,
			TotalMemberCount: 2,
			OtherUserID:      remoteUserID,
			MemberMap: bridgev2.ChatMemberMap{
				ownUserID: {
					EventSender: bridgev2.EventSender{
						IsFromMe:    true,
						Sender:      ownUserID,
						SenderLogin: makeUserLoginID(sa.Label),
					},
				},
				remoteUserID: {
					EventSender: bridgev2.EventSender{
						Sender: remoteUserID,
					},
					UserInfo: &bridgev2.UserInfo{
						Name: &name,
						Identifiers: []string{
							fmt.Sprintf("snapchat:%s", remoteUserID),
						},
					},
				},
			},
		},
	}
}

func (sa *SnapchatAPI) queueChatResync(chatID, chatName string) bool {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chatID == "" {
		return false
	}
	name := strings.TrimSpace(chatName)
	if name == "" {
		name = chatID
	}
	sa.mu.Lock()
	if _, queued := sa.queuedPortalResyncs[chatID]; queued {
		sa.mu.Unlock()
		return false
	}
	sa.queuedPortalResyncs[chatID] = struct{}{}
	sa.mu.Unlock()

	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(chatID),
		Receiver: sa.UserLogin.ID,
	}
	log.Printf("bridgev2 sync: queue portal create chat=%q id=%s", name, chatID)
	sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.ChatResync{
		EventMeta: simplevent.EventMeta{
			Type: bridgev2.RemoteEventChatResync,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("chat", name).Str("chat_id", chatID)
			},
			PortalKey:    portalKey,
			CreatePortal: true,
		},
		ChatInfo: sa.chatInfoFor(chatID, name),
	})
	return true
}

func (sa *SnapchatAPI) queueStoredPortalResyncs() {
	if sa.Connector == nil || sa.Connector.store == nil {
		return
	}
	portals, err := sa.Connector.store.ListPortals()
	if err != nil {
		log.Printf("bridgev2 sync: failed to load stored portals for resync: %v", err)
		return
	}
	queued := 0
	for _, portal := range portals {
		chatID := portal.RemoteID
		if chatID == "" {
			chatID = portal.PortalKey
		}
		chatName := portal.RemoteName
		if chatName == "" {
			chatName = chatID
		}
		sa.rememberChat(chatID, "", chatName, "")
		if sa.queueChatResync(chatID, chatName) {
			queued++
		}
	}
	log.Printf("bridgev2 sync: queued %d stored portal creates for label=%s", queued, sa.Label)
}

func (sa *SnapchatAPI) restoreChatMappings() {
	if sa.Connector == nil || sa.Connector.store == nil {
		return
	}

	portals, err := sa.Connector.store.ListPortals()
	if err != nil {
		return
	}
	for _, portal := range portals {
		if portal.RemoteID == "" || portal.RemoteName == "" {
			continue
		}
		sa.rememberChat(portal.RemoteID, "", portal.RemoteName, "")
		sa.rememberChatState(portal.RemoteID, portal.Unread, portal.Preview, portal.LastMessage)
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
	sa.chatState = make(map[string]string)
	sa.lastMessageID = make(map[string]int64)
	sa.lastMessageVersion = make(map[string]int64)
	sa.messageVersions = make(map[string]map[int64]int64)
	sa.messageFailures = make(map[string]int)
	sa.messageRetryAfter = make(map[string]time.Time)
	sa.lastReadReceiptSync = make(map[string]time.Time)
	sa.queuedPortalResyncs = make(map[string]struct{})
	sa.sidebarBaselineReady = false
	sa.mu.Unlock()
	if sa.Connector != nil {
		_ = sa.Connector.resetSyncState()
	}
}

func (sa *SnapchatAPI) persistChat(chatID, chatName, preview, lastMessage string, unread bool) {
	if sa.Connector == nil || sa.Connector.store == nil {
		return
	}
	_ = sa.Connector.store.UpsertPortal(store.PortalState{
		PortalKey:    chatID,
		RemoteID:     chatID,
		RemoteName:   chatName,
		Preview:      preview,
		LastMessage:  lastMessage,
		Unread:       unread,
		LastSyncedAt: time.Now(),
	})
}

func (sa *SnapchatAPI) updateLoginState(status *connector.SessionStatus) {
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

func connectorChatFromAPI(chat snapapi.Chat) connector.Chat {
	name := strings.TrimSpace(chat.Name)
	if name == "" {
		name = chat.ID
	}
	preview := strings.TrimSpace(chat.Preview)
	if preview == "" {
		preview = strings.TrimSpace(chat.LastMessage)
	}
	return connector.Chat{
		ID:          chat.ID,
		OtherUserID: chat.OtherUserID,
		URL:         snapchatConversationURL(chat.ID),
		Name:        name,
		Preview:     preview,
		Unread:      chat.Unread,
		LastMessage: preview,
	}
}

func connectorMessageFromAPI(message snapapi.Message) connector.Message {
	body := strings.TrimSpace(message.Text)
	if body == "" && message.IsSnap {
		body = "New Snap"
	}
	if body == "" {
		body = "[Unsupported Snapchat event]"
	}
	author := strings.TrimSpace(message.Author)
	if author == "" {
		author = message.AuthorID
	}
	ts := ""
	if !message.Timestamp.IsZero() {
		ts = message.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return connector.Message{
		ID:        message.ID,
		AuthorID:  message.AuthorID,
		Author:    author,
		Text:      body,
		Timestamp: ts,
		Outgoing:  message.Outgoing,
	}
}

func (sa *SnapchatAPI) messageSenderID(chat connector.Chat, message connector.Message) networkid.UserID {
	if message.Outgoing {
		return makeUserID(sa.Label)
	}
	if senderID := sa.remoteUserIDForChat(chat.ID, chat.Name); strings.TrimSpace(string(senderID)) != "" {
		return senderID
	}
	if otherUserID := strings.TrimSpace(chat.OtherUserID); otherUserID != "" {
		if strings.TrimSpace(message.AuthorID) == "" || strings.EqualFold(message.AuthorID, otherUserID) {
			return makeUserID(otherUserID)
		}
	}
	if id := strings.TrimSpace(message.AuthorID); id != "" {
		return makeUserID(id)
	}
	return makeUserID(message.Author)
}

func parseSnapchatMessageID(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "client-")
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func snapchatConversationURL(chatID string) string {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return ""
	}
	return "https://www.snapchat.com/web/" + chatID
}
