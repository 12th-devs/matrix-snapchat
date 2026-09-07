package connector

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/snapapi"
)

// Dead-session handling: after this many consecutive 401/403 responses the
// messaging session is treated as dead and we stop hammering Snapchat, surfacing
// a clear re-login-required state instead. A recovery check still runs at
// reLoginRecoveryInterval so a successful re-login resumes sync automatically.
const (
	reLoginUnauthorizedThreshold = 6
	reLoginRecoveryInterval      = 30 * time.Second
	reLoginLogInterval           = 60 * time.Second
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
	if !sa.resyncMu.TryLock() {
		return
	}
	defer sa.resyncMu.Unlock()
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil {
		log.Printf("bridgev2 sync: skipping poll because user login is not fully initialized label=%s", sa.Label)
		return
	}
	// Remote typing state comes from the browser session's presence store and
	// is independent of the messaging API poll outcome.
	sa.scheduleTypingStateSync(ctx)
	if sa.Connector.apiMode() == "hybrid" {
		sa.pollOnceDOM(ctx)
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
	chats = sa.enrichSidebarChatsFromAPI(ctx, chats)
	log.Printf("bridgev2 sync: fetched %d chats for label=%s", len(chats), sa.Label)
	sidebarUpdates := 0
	baselineReady := !sa.claimSidebarBaseline()
	for _, chat := range chats {
		sa.rememberChatDetails(chat)
		sa.persistChat(chat)
		if !baselineReady {
			sa.queueChatResync(ctx, chat.ID, chat.Name)
		}
		if fingerprint, ok := sa.shouldQueueSidebarUpdate(chat, baselineReady); ok {
			sa.queueSidebarUpdate(chat, fingerprint)
			sidebarUpdates++
		}
	}
	if !baselineReady {
		sa.completeSidebarBaseline()
	} else if sa.autoFetchMessagesEnabled() && sa.claimHybridBackfill(30*time.Second) {
		log.Printf("bridgev2 sync: starting delayed hybrid message backfill label=%s", sa.Label)
		maxHybridBackfillChats := sa.messageSyncLimitPerPoll(false)
		synced := 0
		for _, chat := range chats {
			if synced >= maxHybridBackfillChats || ctx.Err() != nil {
				break
			}
			if err := sa.syncChatMessagesAPI(ctx, chat, 0, "delayed hybrid sidebar backfill", false, false); err != nil {
				log.Printf("bridgev2 sync: delayed hybrid message backfill failed chat=%q id=%s: %v", chat.Name, chat.ID, err)
				continue
			}
			synced++
		}
		log.Printf("bridgev2 sync: delayed hybrid message backfilled %d current chats for label=%s", synced, sa.Label)
	}
	log.Printf("bridgev2 sync: queued %d sidebar update notices for label=%s", sidebarUpdates, sa.Label)
}

func (sa *SnapchatAPI) enrichSidebarChatsFromAPI(ctx context.Context, chats []sidecar.Chat) []sidecar.Chat {
	if len(chats) == 0 || !sa.useAPI() {
		return chats
	}
	needsIdentity := false
	for _, chat := range chats {
		if strings.TrimSpace(chat.ID) != "" && strings.TrimSpace(chat.OtherUserID) == "" {
			needsIdentity = true
			break
		}
	}
	if !needsIdentity {
		return chats
	}

	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		log.Printf("bridgev2 sync: sidebar identity enrichment skipped, API client unavailable: %v", err)
		return chats
	}
	result, err := client.Sync(ctx, snapapi.State{}, sa.conversationFetchLimit())
	if err != nil {
		sa.invalidateSnapClient()
		log.Printf("bridgev2 sync: sidebar identity enrichment failed: %v", err)
		return chats
	}

	byID := make(map[string]sidecar.Chat, len(result.Chats))
	for _, apiChat := range result.Chats {
		chat := connectorChatFromAPI(apiChat)
		if strings.TrimSpace(chat.ID) == "" {
			continue
		}
		byID[chat.ID] = chat
	}

	enriched := 0
	for i := range chats {
		if strings.TrimSpace(chats[i].ID) == "" {
			continue
		}
		apiChat, ok := byID[chats[i].ID]
		if !ok {
			continue
		}
		if strings.TrimSpace(chats[i].OtherUserID) == "" && strings.TrimSpace(apiChat.OtherUserID) != "" {
			chats[i].OtherUserID = apiChat.OtherUserID
			enriched++
		}
		if len(apiChat.ParticipantIDs) > 0 {
			chats[i].ParticipantIDs = append([]string{}, apiChat.ParticipantIDs...)
			chats[i].IsGroup = apiChat.IsGroup || len(apiChat.ParticipantIDs) > 1
			if chats[i].IsGroup {
				chats[i].OtherUserID = ""
			}
		}
		if strings.TrimSpace(chats[i].Username) == "" && strings.TrimSpace(apiChat.Username) != "" {
			chats[i].Username = apiChat.Username
		}
		if chats[i].DisappearAfterSeconds <= 0 && apiChat.DisappearAfterSeconds > 0 {
			chats[i].DisappearAfterSeconds = apiChat.DisappearAfterSeconds
		}
	}
	if enriched > 0 {
		log.Printf("bridgev2 sync: enriched %d sidebar chats with Snapchat API identity metadata", enriched)
	}
	return chats
}

// recordSyncFailure counts consecutive unauthorized (401/403) sync failures and,
// once the bounded threshold is reached, marks the messaging session as dead and
// surfaces a clear re-login-required state instead of letting 403 polling continue.
func (sa *SnapchatAPI) recordSyncFailure(err error, now time.Time) {
	if !errors.Is(err, snapapi.ErrUnauthorized) {
		// A non-auth failure is transient; don't count it toward the dead-session
		// threshold, but don't clear an already-confirmed dead state either (re-login
		// recovery clears it once Sync succeeds again).
		return
	}
	sa.apiMu.Lock()
	sa.consecutiveUnauthorized++
	consec := sa.consecutiveUnauthorized
	alreadyDead := sa.reLoginRequired
	// Always push the next recovery check out so a confirmed-dead session is
	// retried at most once per recovery interval, never hot-looped at poll rate.
	sa.nextRecoveryCheckAt = now.Add(reLoginRecoveryInterval)
	transitioned := !alreadyDead && consec >= reLoginUnauthorizedThreshold
	if transitioned {
		sa.reLoginRequired = true
	}
	sa.apiMu.Unlock()

	if transitioned {
		status := sidecar.SessionStatus{
			State:           "re_login_required",
			Authenticated:   false,
			ReLoginRequired: true,
			Message:         "Snapchat messaging session is no longer valid. Re-login in the open Snapchat browser window/profile to resume syncing.",
		}
		sa.updateLoginState(&status)
		log.Printf("bridgev2 sync: SNAPCHAT RE-LOGIN REQUIRED label=%s consecutive_unauthorized=%d - messaging session is dead; use the open Snapchat window to log back in", sa.Label, consec)
	} else if !alreadyDead {
		log.Printf("bridgev2 sync: API unauthorized label=%s consecutive=%d/%d (not yet dead)", sa.Label, consec, reLoginUnauthorizedThreshold)
	}
}

// recordSyncSuccess clears the dead-session state so a recovered session resumes
// normal syncing automatically.
func (sa *SnapchatAPI) recordSyncSuccess(now time.Time) {
	sa.apiMu.Lock()
	wasDead := sa.reLoginRequired
	sa.consecutiveUnauthorized = 0
	sa.reLoginRequired = false
	sa.nextRecoveryCheckAt = time.Time{}
	sa.apiMu.Unlock()
	if wasDead {
		log.Printf("bridgev2 sync: SNAPCHAT SESSION RECOVERED label=%s - messaging session restored, resuming sync", sa.Label)
		sa.updateLoginState(&sidecar.SessionStatus{
			State:           "ready",
			Authenticated:   true,
			ReLoginRequired: false,
		})
	}
}

// syncSuppressed reports whether the dead-session gate should skip this poll:
// a confirmed-dead session waits for the recovery interval before one bounded
// retry, so repeated 403s never hammer Snapchat at poll rate.
func (sa *SnapchatAPI) syncSuppressed(now time.Time) bool {
	sa.apiMu.Lock()
	defer sa.apiMu.Unlock()
	return sa.reLoginRequired && now.Before(sa.nextRecoveryCheckAt)
}

func (sa *SnapchatAPI) pollOnceAPI(ctx context.Context) error {
	now := time.Now()
	if sa.syncSuppressed(now) {
		// Session is confirmed dead; wait for the recovery interval before the
		// next attempt so we stop the repeated 403 polling.
		return nil
	}

	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		// Auth-layer failures (surface here before any Sync call) feed the same
		// dead-session state machine; unrelated errors stay neutral.
		sa.recordSyncFailure(err, now)
		return err
	}
	prevState := sa.loadAPISyncState()
	result, err := client.Sync(ctx, prevState, sa.conversationFetchLimit())
	if err != nil {
		sa.invalidateSnapClient()
		sa.recordSyncFailure(err, now)
		return err
	}
	sa.recordSyncSuccess(now)

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
	maxMessageSyncsPerPoll := sa.messageSyncLimitPerPoll(firstSync)
	limitedMessageSyncs := 0
	for idx, apiChat := range result.Chats {
		chat := connectorChatFromAPI(apiChat)
		sa.rememberChatDetails(chat)
		retentionChanged := sa.rememberChatDisappear(chat.ID, chat.DisappearAfterSeconds)
		sa.persistChat(chat)
		sa.syncReadWatermarks(ctx, chat)
		_, changed := result.ChangedChatIDs[chat.ID]
		if (!baselineReady || changed || retentionChanged) && portalResyncs < maxPortalResyncsPerPoll {
			sa.queueChatResync(ctx, chat.ID, chat.Name)
			portalResyncs++
		}
		if fingerprint, ok := sa.shouldQueueSidebarUpdate(chat, baselineReady); ok {
			sa.queueSidebarUpdate(chat, fingerprint)
			sidebarUpdates++
		}

		if sa.autoFetchMessagesEnabled() {
			shouldFetch := changed && !firstSync
			if firstSync && (idx < maxMessageSyncsPerPoll || chat.Unread) {
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
			} else if shouldFetch {
				limitedMessageSyncs++
			}
		}
	}
	sa.saveAPISyncState(result.State)
	sa.markSidebarBaselineReady()
	log.Printf("bridgev2 sync: API fetched %d chats, synced %d chats, limited %d changed chats, resynced %d portals, queued %d sidebar notices, first_sync=%t initial_fetches=%d auto_fetch_messages=%t label=%s",
		len(result.Chats), messageSyncs, limitedMessageSyncs, portalResyncs, sidebarUpdates, firstSync, initialFetches, sa.autoFetchMessagesEnabled(), sa.Label)
	return nil
}

func (sa *SnapchatAPI) messageSyncLimitPerPoll(firstSync bool) int {
	limit := sa.messageFetchLimit()
	if limit < 8 {
		limit = 8
	}
	if firstSync && limit < 20 {
		limit = 20
	}
	if limit > 40 {
		limit = 40
	}
	return limit
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
	if body == "New message" || body == "New Snap" {
		// Unrecognized preview: log it so the next client-rendered system
		// line pattern can be pinned from real wire evidence.
		log.Printf("bridgev2 sidebar: generic notice synthesized chat_id=%s unread=%t preview=%q", chat.ID, chat.Unread, chat.Preview)
	}
	hash := sha1.Sum([]byte(chat.ID + "|" + fingerprint + "|" + body))
	message := sidecar.Message{
		ID:        "sidebar-" + chat.ID + "-" + hex.EncodeToString(hash[:8]),
		Author:    chat.Name,
		Text:      body,
		Timestamp: time.Now().Format(time.RFC3339),
		Outgoing:  false,
		// Synthesized sidebar placeholders are conversation events, not chat
		// text: mark them STATUS so convertMessage renders them as gray
		// m.notice lines instead of chatty messages.
		ContentType: "STATUS",
	}
	sa.queueRemoteMessage(chat, message)
}
