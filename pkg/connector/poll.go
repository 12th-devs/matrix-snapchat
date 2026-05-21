package connector

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"log"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
)

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
		sa.persistChat(chat.ID, chat.Name, chat.OtherUserID, chat.Preview, chat.LastMessage, chat.Unread)
		sa.queueChatResync(ctx, chat.ID, chat.Name)
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
	result, err := client.Sync(ctx, prevState, sa.conversationFetchLimit())
	if err != nil {
		sa.invalidateSnapClient()
		return err
	}

	firstSync := len(prevState.SyncToken) == 0 && len(prevState.ConversationVersions) == 0
	baselineReady := sa.isSidebarBaselineReady()
	sidebarUpdates := 0
	messageSyncs := 0
	initialFetches := 0
	portalResyncs := 0
	maxPortalResyncsPerPoll := 8
	if firstSync {
		maxPortalResyncsPerPoll = 100
	}
	const maxMessageSyncsPerPoll = 3
	for idx, apiChat := range result.Chats {
		chat := connectorChatFromAPI(apiChat)
		sa.rememberChat(chat.ID, chat.URL, chat.Name, chat.OtherUserID)
		sa.rememberChatDisappear(chat.ID, chat.DisappearAfterSeconds)
		sa.persistChat(chat.ID, chat.Name, chat.OtherUserID, chat.Preview, chat.LastMessage, chat.Unread)
		_, changed := result.ChangedChatIDs[chat.ID]
		if (!baselineReady || changed) && portalResyncs < maxPortalResyncsPerPoll {
			sa.queueChatResync(ctx, chat.ID, chat.Name)
			portalResyncs++
		}
		if fingerprint, ok := sa.shouldQueueSidebarUpdate(chat, baselineReady); ok {
			sa.queueSidebarUpdate(chat, fingerprint)
			sidebarUpdates++
		}

		if sa.autoFetchMessagesEnabled() {
			shouldFetch := changed && !firstSync
			if firstSync && (idx < 3 || chat.Unread) {
				shouldFetch = true
				initialFetches++
			}
			if shouldFetch && messageSyncs < maxMessageSyncsPerPoll {
				includeMedia := sa.snapMediaEnabled() && !sa.snapMediaOnReadEnabled()
				if err := sa.syncChatMessagesAPI(ctx, chat, apiChat.Version, "api poll", includeMedia, true); err != nil {
					log.Printf("bridgev2 sync: API message fetch failed chat=%q id=%s: %v", chat.Name, chat.ID, err)
				} else {
					messageSyncs++
				}
			}
		}
	}
	sa.saveAPISyncState(result.State)
	sa.markSidebarBaselineReady()
	log.Printf("bridgev2 sync: API fetched %d chats, synced %d chats, resynced %d portals, queued %d sidebar notices, first_sync=%t initial_fetches=%d auto_fetch_messages=%t label=%s",
		len(result.Chats), messageSyncs, portalResyncs, sidebarUpdates, firstSync, initialFetches, sa.autoFetchMessagesEnabled(), sa.Label)
	return nil
}

func (sa *SnapchatAPI) selectPollTargets(chats []sidecar.Chat) []sidecar.Chat {
	const maxPriorityChats = 8
	const maxTopChats = 3

	priority := make([]sidecar.Chat, 0, maxPriorityChats)
	selected := make(map[string]string, maxPriorityChats)
	now := time.Now()

	sa.mu.Lock()
	defer sa.mu.Unlock()

	addTarget := func(chat sidecar.Chat, fingerprint string) {
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

func (sa *SnapchatAPI) shouldQueueSidebarUpdate(chat sidecar.Chat, baselineReady bool) (string, bool) {
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

func (sa *SnapchatAPI) queueSidebarUpdate(chat sidecar.Chat, fingerprint string) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" {
		return
	}
	body, ok := sidebarUpdateText(chat)
	if !ok {
		return
	}
	hash := sha1.Sum([]byte(chat.ID + "|" + fingerprint + "|" + body))
	message := sidecar.Message{
		ID:        "sidebar-" + chat.ID + "-" + hex.EncodeToString(hash[:8]),
		Author:    chat.Name,
		Text:      body,
		Timestamp: time.Now().Format(time.RFC3339),
		Outgoing:  false,
	}
	sa.queueRemoteMessage(chat, message)
}
