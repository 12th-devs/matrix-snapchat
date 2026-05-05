package bridgev2

import (
	"context"
	"fmt"
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
	"github.com/colej/mautrix-snapchat/internal/store"
)

type chatRef struct {
	ID   string
	URL  string
	Name string
}

type SnapchatAPI struct {
	Connector *SnapchatConnector
	UserLogin *bridgev2.UserLogin
	Label     string
	Client    *connector.Client

	mu         sync.Mutex
	seenByChat map[string]map[string]struct{}
	chatsByID  map[string]chatRef
	chatState  map[string]string
}

var _ bridgev2.NetworkAPI = (*SnapchatAPI)(nil)

func NewSnapchatAPI(sc *SnapchatConnector, login *bridgev2.UserLogin, label string) *SnapchatAPI {
	return &SnapchatAPI{
		Connector:  sc,
		UserLogin:  login,
		Label:      label,
		Client:     sc.newClient(),
		seenByChat: make(map[string]map[string]struct{}),
		chatsByID:  make(map[string]chatRef),
		chatState:  make(map[string]string),
	}
}

func (sa *SnapchatAPI) Connect(ctx context.Context) {
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
	name := sa.lookupChatName(string(portal.ID))
	if name == "" {
		name = string(portal.ID)
	}
	return &bridgev2.ChatInfo{
		Name: &[]string{name}[0],
	}, nil
}

func (sa *SnapchatAPI) GetUserInfo(ctx context.Context, ghost *bridgev2.Ghost) (*bridgev2.UserInfo, error) {
	return &bridgev2.UserInfo{
		Name:        &[]string{string(ghost.ID)}[0],
		Identifiers: []string{fmt.Sprintf("snapchat:%s", ghost.ID)},
	}, nil
}

func (sa *SnapchatAPI) HandleMatrixMessage(ctx context.Context, msg *bridgev2.MatrixMessage) (*bridgev2.MatrixMessageResponse, error) {
	chatID := string(msg.Portal.ID)
	chatName := sa.lookupChatName(chatID)
	chatURL := sa.lookupChatURL(chatID)
	if err := sa.Client.SendMessage(ctx, chatID, chatName, chatURL, msg.Content.Body); err != nil {
		return nil, fmt.Errorf("send snapchat message: %w", err)
	}
	return &bridgev2.MatrixMessageResponse{
		DB: &database.Message{
			ID:       makeMessageID(chatID + "-" + msg.Content.Body),
			SenderID: makeUserID(sa.Label),
		},
	}, nil
}

func (sa *SnapchatAPI) UserLoginMetadata() (*UserLoginMetadata, bool) {
	if sa == nil || sa.UserLogin == nil {
		return nil, false
	}
	meta, ok := sa.UserLogin.Metadata.(*UserLoginMetadata)
	return meta, ok
}

func (sa *SnapchatAPI) pollLoop(ctx context.Context) {
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
		return
	}

	chats, err := sa.Client.ListChats(ctx)
	if err != nil {
		return
	}
	pollTargets := sa.selectPollTargets(chats)
	for _, chat := range chats {
		sa.rememberChat(chat.ID, chat.URL, chat.Name)
		sa.persistChat(chat.ID, chat.Name, chat.Preview, chat.LastMessage, chat.Unread)
	}
	for _, chat := range pollTargets {
		portalKey := networkid.PortalKey{
			ID:       networkid.PortalID(chat.ID),
			Receiver: sa.UserLogin.ID,
		}
		messages, err := sa.Client.Messages(ctx, chat.ID, chat.Name, chat.URL)
		if err != nil {
			continue
		}
		for _, message := range messages {
			if sa.isSeen(chat.Name, message.ID) {
				continue
			}
			sa.markSeen(chat.Name, message.ID)

			copyMsg := message
			senderID := makeUserID(message.Author)
			if message.Outgoing {
				senderID = makeUserID(sa.Label)
			}
			sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.Message[connector.Message]{
				EventMeta: simplevent.EventMeta{
					Type: bridgev2.RemoteEventMessage,
					LogContext: func(c zerolog.Context) zerolog.Context {
						return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("message_id", message.ID)
					},
					PortalKey:    portalKey,
					CreatePortal: true,
					Sender: bridgev2.EventSender{
						IsFromMe: message.Outgoing,
						Sender:   senderID,
					},
					Timestamp: parseMessageTimestamp(message.Timestamp),
				},
				ID:                 makeMessageID(message.ID),
				Data:               copyMsg,
				ConvertMessageFunc: sa.convertMessage,
			})
		}
	}
}

func (sa *SnapchatAPI) selectPollTargets(chats []connector.Chat) []connector.Chat {
	const maxPriorityChats = 6
	const maxFallbackChats = 2

	priority := make([]connector.Chat, 0, maxPriorityChats)
	fallback := make([]connector.Chat, 0, maxFallbackChats)

	sa.mu.Lock()
	defer sa.mu.Unlock()

	for idx, chat := range chats {
		fingerprint := fmt.Sprintf("%t|%s|%s", chat.Unread, chat.Preview, chat.LastMessage)
		prev, ok := sa.chatState[chat.ID]
		sa.chatState[chat.ID] = fingerprint

		if chat.Unread || !ok || prev != fingerprint {
			if len(priority) < maxPriorityChats {
				priority = append(priority, chat)
			}
			continue
		}

		if idx < maxFallbackChats && len(fallback) < maxFallbackChats {
			fallback = append(fallback, chat)
		}
	}

	if len(priority) > 0 {
		return priority
	}
	return fallback
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
	layouts := []string{
		"January 2 2006",
		"January 2",
		"Jan 2 2006",
		"Jan 2",
	}
	for _, layout := range layouts {
		if ts, err := time.ParseInLocation(layout, raw, now.Location()); err == nil {
			if !strings.Contains(layout, "2006") {
				ts = ts.AddDate(now.Year(), 0, 0)
			}
			return ts
		}
	}
	return time.Time{}
}

func (sa *SnapchatAPI) convertMessage(ctx context.Context, portal *bridgev2.Portal, intent bridgev2.MatrixAPI, message connector.Message) (*bridgev2.ConvertedMessage, error) {
	return &bridgev2.ConvertedMessage{
		Parts: []*bridgev2.ConvertedMessagePart{{
			Type: event.EventMessage,
			Content: &event.MessageEventContent{
				MsgType: event.MsgText,
				Body:    message.Text,
			},
		}},
	}, nil
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

func (sa *SnapchatAPI) rememberChat(chatID, chatURL, chatName string) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.chatsByID[chatID] = chatRef{ID: chatID, URL: chatURL, Name: chatName}
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
		sa.rememberChat(portal.RemoteID, "", portal.RemoteName)
	}
}

func (sa *SnapchatAPI) resetSyncState() {
	sa.mu.Lock()
	sa.seenByChat = make(map[string]map[string]struct{})
	sa.chatsByID = make(map[string]chatRef)
	sa.chatState = make(map[string]string)
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
