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
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/bridgev2/simplevent"
	"maunium.net/go/mautrix/event"
)

type messageSyncAction string

const (
	messageSyncNew       messageSyncAction = "new_message"
	messageSyncEdit      messageSyncAction = "message_edit"
	messageSyncUnchanged messageSyncAction = "unchanged_message"
)

func (sa *SnapchatAPI) syncChatMessages(ctx context.Context, chat sidecar.Chat, reason string) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" {
		return
	}
	if !sa.autoFetchMessagesEnabled() {
		log.Printf("bridgev2 sync: skipping message fetch in no-open mode chat=%q id=%s reason=%s", chat.Name, chat.ID, reason)
		return
	}
	if sa.useAPI() {
		includeMedia := sa.snapMediaEnabled() && !sa.snapMediaOnReadEnabled()
		if err := sa.syncChatMessagesAPI(ctx, chat, 0, reason, includeMedia, true); err == nil {
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

func (sa *SnapchatAPI) syncChatMessagesDOM(ctx context.Context, chat sidecar.Chat, reason string) {
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

func (sa *SnapchatAPI) syncChatMessagesAPI(ctx context.Context, chat sidecar.Chat, version int64, reason string, includeMedia bool, requireAutoFetch bool, resync ...*ResyncStats) error {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" {
		return nil
	}
	if requireAutoFetch && !sa.autoFetchMessagesEnabled() {
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
	if retention, ok := client.RetentionDurationKnown(chat.ID); ok {
		seconds := int64(retention / time.Second)
		chat.DisappearAfterSeconds = seconds
		changed := false
		if seconds > 0 {
			changed = sa.rememberChatDisappear(chat.ID, seconds)
		} else {
			changed = sa.forgetChatDisappear(chat.ID)
		}
		if changed {
			sa.queueChatResync(ctx, chat.ID, chat.Name)
		}
	}
	log.Printf("bridgev2 sync: API fetched %d messages for chat=%q id=%s reason=%s", len(messages), chat.Name, chat.ID, reason)
	var stats *ResyncStats
	var wait []context.Context
	if len(resync) > 0 {
		stats = resync[0]
		stats.MessagesFetched += len(messages)
		wait = []context.Context{ctx}
	}
	states := make([]store.MessageState, 0, len(messages))
	for _, apiMessage := range messages {
		apiMessage = sa.normalizeAPIMessageDirection(chat.ID, apiMessage)
		message := connectorMessageFromAPI(apiMessage)
		if message.DisappearAfterSeconds <= 0 && chat.DisappearAfterSeconds > 0 {
			message.DisappearAfterSeconds = chat.DisappearAfterSeconds
		}
		message = sa.normalizeRemoteMessageDirection(chat.ID, message)
		if message.ID == "" {
			continue
		}
		baseRemoteID := baseSnapchatMessageID(message.ID)
		if baseRemoteID == "" {
			baseRemoteID = message.ID
		}
		// Outgoing echoes of media this login just sent are suppressed before
		// any media descriptor resolution, CDN download, or decryption. The
		// state row is still recorded so later syncs/restarts stay skipped via
		// the persisted Matrix delivery proof.
		if message.Outgoing && sa.hasExactRecentOutgoingRemoteID(chat.ID, baseRemoteID) {
			sa.markSeen(chat.ID, baseRemoteID)
			log.Printf("bridgev2 echo: suppressed outgoing API echo before media render chat_id=%s message_id=%s", chat.ID, baseRemoteID)
			if stats != nil {
				stats.MessagesAlreadyMapped++
			}
			now := time.Now()
			states = append(states, store.MessageState{
				PortalKey:  chat.ID,
				RemoteID:   baseRemoteID,
				Author:     message.Author,
				Text:       message.Text,
				Outgoing:   true,
				Kind:       messageKindFromAPI(apiMessage),
				HasMedia:   len(apiMessage.Media) > 0,
				HydratedAt: &now,
				LastSeenAt: now,
			})
			continue
		}
		if shouldSuppressAPIMessage(apiMessage, message) {
			log.Printf("bridgev2 sync: suppressing passive API message chat_id=%s message_id=%s content_type=%s", chat.ID, baseRemoteID, apiMessage.ContentType)
			continue
		}
		existing := sa.lookupStoredMessage(chat.ID, baseRemoteID)
		if stats != nil {
			parts, err := sa.currentMessageParts(ctx, chat.ID, baseRemoteID)
			if err != nil {
				return err
			}
			if len(parts) == 0 {
				// Keep stored content for decryption fallback, but not as send proof.
				sa.mu.Lock()
				delete(sa.seenByChat[chat.ID], message.ID)
				sa.mu.Unlock()
			} else {
				stats.MessagesAlreadyMapped++
			}
		}
		if includeMedia && sa.snapMediaEnabled() && len(apiMessage.Media) > 0 {
			if stats != nil {
				stats.MediaCandidates++
			}
			// HydratedAt alone does not prove successful media delivery for
			// historical rows; bridge-side MediaDelivered metadata does. The
			// pre-download media identities are deterministic and stable, so
			// the delivered media-scoped mapping can be proven without
			// downloading anything.
			proofMessage := message
			for _, media := range apiMessage.Media {
				proofMessage.Media = append(proofMessage.Media, sidecar.MediaAttachment{ID: media.ID, MimeType: media.MimeType})
			}
			confirmed := sa.mediaDeliveryConfirmed(ctx, chat.ID, baseRemoteID, proofMessage, len(apiMessage.Media))
			decision := decideMediaHydration(
				existing,
				confirmed,
				isSnapRow(apiMessage),
			)
			switch decision {
			case mediaHydrationSkip:
				if confirmed {
					zerolog.Ctx(ctx).Info().Str("chat_id", chat.ID).Str("message_id", baseRemoteID).Msg("media: SKIP current-room-delivery")
					if stats != nil {
						if err := sa.repairMediaPresentation(ctx, chat, baseRemoteID, proofMessage); err != nil {
							return err
						}
						stats.MediaSkipped++
					}
					continue
				}
				if existing != nil && existing.HydratedAt != nil {
					zerolog.Ctx(ctx).Debug().
						Str("diag", "snapchat_media").
						Str("chat_id", chat.ID).
						Str("message_id", apiMessage.ID).
						Str("content_type", apiMessage.ContentType).
						Bool("is_snap", apiMessage.IsSnap).
						Msg("media: skipping already hydrated attachment")
				} else {
					logMediaEvidence(ctx, "media: snap placeholder retained (view-once not auto-opened)", chat.ID, apiMessage)
				}
			case mediaHydrationRecover:
				if !sa.beginMediaRehydration(chat.ID, baseRemoteID) {
					zerolog.Ctx(ctx).Info().
						Str("diag", "snapchat_media").
						Str("chat_id", chat.ID).
						Str("message_id", apiMessage.ID).
						Msg("media: rehydration transition lost; leaving unchanged")
					break
				}
				zerolog.Ctx(ctx).Info().
					Str("diag", "snapchat_media").
					Str("chat_id", chat.ID).
					Str("message_id", apiMessage.ID).
					Str("content_type", apiMessage.ContentType).
					Bool("is_snap", apiMessage.IsSnap).
					Msg("media: stale hydrated state without delivery proof; rehydrating once")
				existing.HydratedAt = nil
				if sa.renderAPIMessageMedia(ctx, chat, client, apiMessage, &message, baseRemoteID, &existing) {
					continue
				}
			case mediaHydrationRender:
				if sa.renderAPIMessageMedia(ctx, chat, client, apiMessage, &message, baseRemoteID, &existing) {
					continue
				}
			}
		} else if len(apiMessage.Media) > 0 {
			zerolog.Ctx(ctx).Info().
				Str("diag", "snapchat_media").
				Str("chat_id", chat.ID).
				Str("message_id", apiMessage.ID).
				Str("content_type", apiMessage.ContentType).
				Bool("is_snap", apiMessage.IsSnap).
				Bool("include_media", includeMedia).
				Bool("snap_media_enabled", sa.snapMediaEnabled()).
				Bool("snap_media_on_read", sa.snapMediaOnReadEnabled()).
				Int("attachment_count", len(apiMessage.Media)).
				Msg("media: API media metadata present but render disabled")
		}
		sa.rememberAPIMessage(chat.ID, apiMessage)
		if apiMessage.Tombstone {
			// Erased/unsent on Snapchat: remove the bridged message instead of
			// displaying it. A tombstone for an unknown message is skipped.
			if existing != nil {
				log.Printf("bridgev2 event_class: kind=message_delete source=api chat_id=%s message_id=%s action=queue", chat.ID, baseRemoteID)
				sa.queueRemoteMessageRemove(chat, baseRemoteID, message)
			} else {
				log.Printf("bridgev2 sync: skipping tombstone for unbridged message chat_id=%s message_id=%s", chat.ID, baseRemoteID)
			}
			continue
		}
		if restored, ok := restoreStoredDecryptedText(existing, message, apiMessage); ok {
			log.Printf("bridgev2 decrypt-cache: restored stored text for transient undecrypted API message chat_id=%s message_id=%s len=%d", chat.ID, baseRemoteID, len(restored))
			message.Text = restored
			apiMessage.Text = restored
		}
		if suppressUndecryptedBackfill(reason, existing, message, apiMessage) {
			log.Printf("bridgev2 sync: suppressing undecrypted historical placeholder chat_id=%s message_id=%s reason=%s", chat.ID, baseRemoteID, reason)
			continue
		}
		if stats != nil {
			parts, err := sa.currentMessageParts(ctx, chat.ID, baseRemoteID)
			if err != nil {
				return err
			}
			if len(parts) == 0 {
				existing = nil
			}
		}
		mappingID := baseRemoteID
		queued := false
		switch action := classifyMessageSync(existing, message); action {
		case messageSyncEdit:
			log.Printf("bridgev2 event_class: kind=%s source=api chat_id=%s message_id=%s api_kind=%s old_len=%d new_len=%d", action, chat.ID, baseRemoteID, messageKindFromAPI(apiMessage), len(existing.Text), len(message.Text))
			if err := sa.queueRemoteMessageEdit(chat, baseRemoteID, message, wait...); err != nil {
				return err
			}
			queued = true
		case messageSyncNew:
			if len(message.Media) > 0 && sa.UserLogin.Bridge.DB != nil {
				parts, lookupErr := sa.currentMessageParts(ctx, chat.ID, message.ID)
				if lookupErr != nil {
					return lookupErr
				}
				if lookupErr == nil && len(parts) > 0 {
					if err := sa.queueRemoteMessageEdit(chat, message.ID, message, wait...); err != nil {
						return err
					}
					mappingID, queued = message.ID, true
					break
				}
			}
			log.Printf("bridgev2 event_class: kind=%s source=api chat_id=%s message_id=%s api_kind=%s sender_id=%s outgoing=%t", action, chat.ID, baseRemoteID, messageKindFromAPI(apiMessage), message.AuthorID, message.Outgoing)
			if err := sa.queueRemoteMessage(chat, message, wait...); err != nil {
				return err
			}
			mappingID, queued = message.ID, true
		case messageSyncUnchanged:
			log.Printf("bridgev2 event_class: kind=%s source=api chat_id=%s message_id=%s api_kind=%s", action, chat.ID, baseRemoteID, messageKindFromAPI(apiMessage))
		}
		if stats != nil && queued {
			parts, err := sa.UserLogin.Bridge.DB.Message.GetAllPartsByID(ctx, sa.UserLogin.ID, makeMessageID(scopedSnapchatMessageID(chat.ID, mappingID)))
			if err != nil || len(parts) == 0 {
				return fmt.Errorf("Matrix send did not persist mapping for %s: %v", mappingID, err)
			}
			stats.MessagesBridged++
			if deliveredMediaParts(parts, len(message.Media)) {
				stats.MediaHydrated++
			}
		}
		states = append(states, store.MessageState{
			PortalKey:    chat.ID,
			RemoteID:     baseRemoteID,
			Author:       message.Author,
			Text:         message.Text,
			Outgoing:     message.Outgoing,
			TimestampRaw: message.Timestamp,
			Kind:         messageKindFromAPI(apiMessage),
			HasMedia:     len(apiMessage.Media) > 0,
			LastSeenAt:   time.Now(),
		})
	}
	if len(states) > 0 && sa.Connector != nil && sa.Connector.store != nil {
		_ = sa.Connector.store.UpsertMessages(states)
	}
	return nil
}

func isSnapRow(message snapapi.Message) bool {
	if message.IsSnap {
		return true
	}
	switch strings.ToUpper(strings.TrimSpace(message.ContentType)) {
	case "SNAP", "SNAP_NOT_VIEWABLE":
		return true
	}
	return false
}

// isSnapSidecarMessage mirrors isSnapRow for sidecar messages that reach
// convertMessage: disappearing Snaps must stay distinguishable from saved chat
// media in the Matrix representation.
func isSnapSidecarMessage(message sidecar.Message) bool {
	if message.IsSnap {
		return true
	}
	switch strings.ToUpper(strings.TrimSpace(message.ContentType)) {
	case "SNAP", "SNAP_NOT_VIEWABLE":
		return true
	}
	return false
}

func logMediaEvidence(ctx context.Context, decision, chatID string, apiMessage snapapi.Message) {
	var media snapapi.MediaAttachment
	if len(apiMessage.Media) > 0 {
		media = apiMessage.Media[0]
	}
	zerolog.Ctx(ctx).Info().
		Str("diag", "snapchat_media").
		Str("chat_id", chatID).
		Str("message_id", apiMessage.ID).
		Str("content_type", apiMessage.ContentType).
		Bool("is_snap", apiMessage.IsSnap).
		Str("kind", string(media.Kind)).
		Bool("url_present", media.URL != "").
		Int("inline_bytes", len(media.Data)).
		Int("key_bytes", len(media.Key)).
		Int("iv_bytes", len(media.IV)).
		Str("mime_hint", media.MimeType).
		Int("attachment_count", len(apiMessage.Media)).
		Msg(decision)
}

func shouldSuppressAPIMessage(apiMessage snapapi.Message, message sidecar.Message) bool {
	contentType := strings.ToUpper(strings.TrimSpace(apiMessage.ContentType))
	switch contentType {
	case "STATUS", "STATUS_SAVE_TO_CAMERA_ROLL", "STATUS_CONVERSATION_CAPTURE_SCREENSHOT",
		"STATUS_CONVERSATION_CAPTURE_RECORD", "STATUS_CALL_MISSED_VIDEO", "STATUS_CALL_MISSED_AUDIO":
		return true
	}
	return strings.HasPrefix(strings.TrimSpace(message.Text), "[Snapchat status]")
}

func restoreStoredDecryptedText(existing *store.MessageState, message sidecar.Message, apiMessage snapapi.Message) (string, bool) {
	if existing == nil {
		return "", false
	}
	if apiMessage.IsSnap || message.IsSnap || len(apiMessage.Media) > 0 || len(message.Media) > 0 {
		return "", false
	}
	if existing.HasMedia || existing.Kind == "media" || existing.Kind == "snap" {
		return "", false
	}
	if strings.TrimSpace(message.Text) != "[Snapchat message unavailable]" {
		return "", false
	}
	text := strings.TrimSpace(existing.Text)
	if text == "" || isGeneratedSnapchatNotice(text) {
		return "", false
	}
	return text, true
}

func suppressUndecryptedBackfill(reason string, existing *store.MessageState, message sidecar.Message, apiMessage snapapi.Message) bool {
	if reason != "resync-all" && (reason != "startup stored portal backfill" || existing != nil) {
		return false
	}
	if apiMessage.IsSnap || message.IsSnap || len(apiMessage.Media) > 0 || len(message.Media) > 0 {
		return false
	}
	return strings.TrimSpace(message.Text) == "[Snapchat message unavailable]"
}

func classifyMessageSync(existing *store.MessageState, message sidecar.Message) messageSyncAction {
	if existing == nil {
		return messageSyncNew
	}
	if shouldQueueMessageEdit(existing, message) {
		return messageSyncEdit
	}
	return messageSyncUnchanged
}

func shouldQueueMessageEdit(existing *store.MessageState, message sidecar.Message) bool {
	if existing == nil {
		return false
	}
	if (existing.HasMedia || existing.Kind == "media" || existing.Kind == "snap") && len(message.Media) > 0 {
		return existing.HydratedAt == nil
	}
	oldText := strings.TrimSpace(existing.Text)
	newText := strings.TrimSpace(message.Text)
	if oldText == "" || newText == "" || oldText == newText {
		return false
	}
	// Media hydration edits are queued explicitly by hydrateSnapMediaFromReadReceipt.
	// During ordinary API sync, keep snap placeholders stable unless the update is
	// plain text replacing a previous undecoded/plain placeholder.
	if existing.HasMedia || existing.Kind == "media" || existing.Kind == "snap" {
		return false
	}
	return true
}

func (sa *SnapchatAPI) queueRemoteMessage(chat sidecar.Chat, message sidecar.Message, wait ...context.Context) error {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" || message.ID == "" {
		return nil
	}
	message = sa.normalizeRemoteMessageDirection(chat.ID, message)
	if chat.DisappearAfterSeconds > 0 {
		if message.DisappearAfterSeconds <= 0 && !message.Saved {
			message.DisappearAfterSeconds = chat.DisappearAfterSeconds
		}
		sa.rememberChatDisappear(chat.ID, chat.DisappearAfterSeconds)
	} else if message.DisappearAfterSeconds <= 0 && !message.Saved {
		if chatSetting := sa.disappearingSettingForChat(chat.ID); chatSetting != nil {
			message.DisappearAfterSeconds = int64(chatSetting.Timer / time.Second)
		}
	}
	if sa.shouldSuppressOutgoingEcho(chat.ID, message) {
		sa.markSeen(chat.ID, message.ID)
		log.Printf("bridgev2 echo: suppressed outgoing API echo chat_id=%s message_id=%s content_type=%s disappear_after=%d", chat.ID, message.ID, message.ContentType, message.DisappearAfterSeconds)
		return nil
	}
	if len(message.Media) == 0 && sa.isSeen(chat.ID, message.ID) {
		log.Printf("bridgev2 dedupe: already saw message chat_id=%s message_id=%s", chat.ID, message.ID)
		return nil
	}
	if len(message.Media) == 0 {
		sa.markSeen(chat.ID, message.ID)
	}
	copyMsg := message
	sa.rememberChatDetails(chat)
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
	if message.Outgoing {
		// Keep the remote sender tied to the login while marking it as from-me.
		// SenderLogin makes bridgev2 send as the logged-in Matrix user, while
		// Sender gives the database a stable remote sender for dedupe/storage.
		sender.Sender = makeUserID(sa.Label)
		sender.SenderLogin = makeUserLoginID(sa.Label)
	}
	if strings.HasPrefix(message.ID, "sidebar-") {
		// Sidebar notices are bridge-generated indicators, not real remote chat
		// messages. Send them through the bridge bot so they don't race ghost joins
		// while freshly recreated portal rooms are still settling.
		sender = bridgev2.EventSender{}
	}
	eventID := scopedSnapchatMessageID(chat.ID, message.ID)
	log.Printf("bridgev2 event: queue message chat_id=%s message_id=%s event_id=%s sender_id=%s outgoing=%t text=%q media=%d is_snap=%t content_type=%s", chat.ID, message.ID, eventID, senderID, message.Outgoing, strings.TrimSpace(message.Text), len(message.Media), message.IsSnap, message.ContentType)
	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(chat.ID),
		Receiver: sa.UserLogin.ID,
	}
	done := make(chan struct{})
	post := sa.mediaDeliveryPostHandle(chat, message.ID, message)
	result := sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.Message[sidecar.Message]{
		EventMeta: simplevent.EventMeta{
			Type: bridgev2.RemoteEventMessage,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("message_id", message.ID).Str("event_id", eventID)
			},
			PortalKey:    portalKey,
			CreatePortal: true,
			Sender:       sender,
			Timestamp:    parseMessageTimestamp(message.Timestamp),
			PostHandleFunc: func(ctx context.Context, portal *bridgev2.Portal) {
				if post != nil {
					post(ctx, portal)
				}
				close(done)
			},
		},
		ID:                 makeMessageID(eventID),
		Data:               copyMsg,
		ConvertMessageFunc: sa.convertMessage,
	})
	return waitRemoteMessage(result, done, wait)
}

func waitRemoteMessage(result bridgev2.EventHandlingResult, done <-chan struct{}, wait []context.Context) error {
	if !result.Success {
		return fmt.Errorf("queue remote message: %v", result.Error)
	}
	if len(wait) > 0 && result.Queued {
		select {
		case <-done:
		case <-wait[0].Done():
			return wait[0].Err()
		}
	}
	return nil
}

func (sa *SnapchatAPI) queueRemoteMessageRemove(chat sidecar.Chat, targetMessageID string, message sidecar.Message) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" || targetMessageID == "" {
		return
	}
	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(chat.ID),
		Receiver: sa.UserLogin.ID,
	}
	sender := bridgev2.EventSender{
		IsFromMe: message.Outgoing,
		Sender:   sa.messageSenderID(chat, message),
	}
	if message.Outgoing {
		sender.Sender = makeUserID(sa.Label)
		sender.SenderLogin = makeUserLoginID(sa.Label)
	} else if strings.TrimSpace(message.AuthorID) != "" {
		sa.rememberGhostName(makeUserID(message.AuthorID), message.Author)
	}
	sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.MessageRemove{
		EventMeta: simplevent.EventMeta{
			Type: bridgev2.RemoteEventMessageRemove,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("target_message_id", targetMessageID)
			},
			PortalKey: portalKey,
			Sender:    sender,
			Timestamp: time.Now(),
		},
		TargetMessage: makeMessageID(scopedSnapchatMessageID(chat.ID, targetMessageID)),
	})
}

func (sa *SnapchatAPI) queueRemoteMessageEdit(chat sidecar.Chat, targetMessageID string, message sidecar.Message, wait ...context.Context) error {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" || targetMessageID == "" || message.ID == "" {
		return nil
	}
	message = sa.normalizeRemoteMessageDirection(chat.ID, message)
	copyMsg := message
	sa.rememberChatDetails(chat)
	sa.rememberChatDisappear(chat.ID, chat.DisappearAfterSeconds)
	if !message.Outgoing {
		if strings.TrimSpace(message.AuthorID) != "" {
			sa.rememberGhostName(makeUserID(message.AuthorID), message.Author)
		}
		if strings.TrimSpace(chat.OtherUserID) != "" {
			sa.rememberGhostName(makeUserID(chat.OtherUserID), chat.Name)
		}
	}
	sender := bridgev2.EventSender{
		IsFromMe: message.Outgoing,
		Sender:   sa.messageSenderID(chat, message),
	}
	if message.Outgoing {
		sender.Sender = makeUserID(sa.Label)
		sender.SenderLogin = makeUserLoginID(sa.Label)
	}
	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(chat.ID),
		Receiver: sa.UserLogin.ID,
	}
	targetEventID := scopedSnapchatMessageID(chat.ID, targetMessageID)
	editID := mediaMessageID(targetEventID, message.Media)
	log.Printf("bridgev2 edit: queue edit chat_id=%s target_message_id=%s target_event_id=%s edit_id=%s media=%d", chat.ID, targetMessageID, targetEventID, editID, len(message.Media))
	if len(message.Media) == 0 {
		editID = targetEventID + "-edit-" + stableTextID(message.Text)
	}
	done := make(chan struct{})
	post := sa.mediaDeliveryPostHandle(chat, targetMessageID, message)
	result := sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.Message[sidecar.Message]{
		EventMeta: simplevent.EventMeta{
			Type: bridgev2.RemoteEventEdit,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("message_id", message.ID).Str("target_message_id", targetMessageID).Str("target_event_id", targetEventID)
			},
			PortalKey:    portalKey,
			CreatePortal: true,
			Sender:       sender,
			Timestamp:    parseMessageTimestamp(message.Timestamp),
			PostHandleFunc: func(ctx context.Context, portal *bridgev2.Portal) {
				if post != nil {
					post(ctx, portal)
				}
				close(done)
			},
		},
		ID:                 makeMessageID(editID),
		TargetMessage:      makeMessageID(targetEventID),
		Data:               copyMsg,
		ConvertMessageFunc: sa.convertMessage,
		ConvertEditFunc:    sa.convertMessageEdit,
	})
	return waitRemoteMessage(result, done, wait)
}

func (sa *SnapchatAPI) convertMessage(ctx context.Context, portal *bridgev2.Portal, intent bridgev2.MatrixAPI, message sidecar.Message) (*bridgev2.ConvertedMessage, error) {
	body := strings.TrimSpace(message.Text)
	if body == "" {
		body = "[Unsupported Snapchat event]"
	}
	portalID := ""
	if portal != nil {
		portalID = string(portal.ID)
	}
	if len(message.Media) > 0 && portal != nil {
		if state := sa.lookupStoredMessage(portalID, baseSnapchatMessageID(message.ID)); state != nil && state.HydratedAt != nil {
			return nil, bridgev2.ErrIgnoringRemoteEvent
		}
		parts := make([]*bridgev2.ConvertedMessagePart, 0, len(message.Media))
		uploadIntent := mediaUploadIntent(portal, intent)
		for _, media := range message.Media {
			if len(media.Data) == 0 {
				continue
			}
			if err := snapapi.ValidateRenderableMediaPayload(media.Data, media.MimeType, snapapi.MediaKindFile); err != nil {
				zerolog.Ctx(ctx).Warn().
					Err(err).
					Str("diag", "snapchat_media").
					Str("message_id", message.ID).
					Str("media_id", media.ID).
					Int("size", len(media.Data)).
					Str("mime", media.MimeType).
					Msg("media: rejected unsafe Snapchat media before Matrix upload")
				continue
			}
			mimeType := normalizeMediaMIME(media.Data, media.MimeType)
			if strings.HasPrefix(media.MimeType, "audio/") && mimeType == "video/mp4" {
				// Go's sniffer reports video/mp4 for voice-note ftyp bytes.
				mimeType = media.MimeType
			}
			msgType := matrixMsgTypeForMedia(mimeType)
			fileName := mediaFileNameForMIME(media.FileName, msgType, mimeType)
			mxc, file, err := uploadIntent.UploadMedia(ctx, portal.MXID, media.Data, fileName, mimeType)
			if err != nil {
				zerolog.Ctx(ctx).Warn().
					Err(err).
					Str("diag", "snapchat_media").
					Str("message_id", message.ID).
					Str("media_id", media.ID).
					Msg("media: failed to upload Snapchat media")
				return nil, fmt.Errorf("upload Snapchat attachment %s: %w", media.ID, err)
			}
			zerolog.Ctx(ctx).Info().
				Str("diag", "snapchat_media").
				Str("message_id", message.ID).
				Str("media_id", media.ID).
				Str("matrix_msgtype", string(msgType)).
				Str("mime", mimeType).
				Int("size", len(media.Data)).
				Msg("media: uploaded Snapchat media")
			partBody := body
			snapMessage := isSnapSidecarMessage(message)
			if snapMessage && (partBody == "" || isGeneratedSnapchatNotice(partBody)) {
				// Disappearing Snaps get a visible emoji caption so they read
				// differently from saved chat media in every client; ordinary
				// chat media keeps the filename body.
				partBody = snapMediaCaption(msgType)
			} else if partBody == "" || partBody == "Media" || isGeneratedSnapchatNotice(partBody) {
				// Some Matrix clients use body as the displayed/downloaded filename
				// for encrypted media, so don't use generated placeholders as the body here.
				partBody = fileName
			}
			part := &bridgev2.ConvertedMessagePart{
				ID:         mediaPartID(len(parts)),
				DBMetadata: &MediaDeliveryMetadata{MediaDelivered: true, AttachmentCount: len(message.Media), PresentationVersion: 1},
				Type:       event.EventMessage,
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
			}
			extra := map[string]any{
				"net.colej.snapchat.type": snapTypeMarker(snapMessage, msgType),
			}
			if msgType == event.MsgAudio {
				// MSC3245 voice marker so clients render the voice-note UI.
				extra["org.matrix.msc3245.voice"] = map[string]any{}
			}
			part.Extra = extra
			parts = append(parts, part)
		}
		if len(parts) > 0 {
			if len(parts) != len(message.Media) {
				return nil, fmt.Errorf("incomplete Snapchat media conversion: %d/%d attachments", len(parts), len(message.Media))
			}
			return &bridgev2.ConvertedMessage{
				ReplyTo:   sa.resolveReplyTarget(portalID, message.QuotedMessageID),
				Parts:     parts,
				Disappear: sa.disappearingSettingForMessage(portalID, message),
			}, nil
		}
	}
	msgType := event.MsgText
	if isSystemEventContentType(message.ContentType) || strings.HasPrefix(body, "[Unsupported Snapchat") || strings.HasPrefix(body, "[Snapchat") || isGeneratedSnapchatNotice(body) {
		msgType = event.MsgNotice
	}
	return &bridgev2.ConvertedMessage{
		ReplyTo: sa.resolveReplyTarget(portalID, message.QuotedMessageID),
		Parts: []*bridgev2.ConvertedMessagePart{{
			Type: event.EventMessage,
			Content: &event.MessageEventContent{
				MsgType: msgType,
				Body:    body,
			},
		}},
		Disappear: sa.disappearingSettingForMessage(portalID, message),
	}, nil
}

// resolveReplyTarget maps an incoming Snapchat reply's quoted message ID to the
// bridged Matrix event. The mapping is deterministic ("<chatID>.msg.<id>") and
// the target must already be bridged locally (persisted in the connector store,
// so it survives restarts). A miss logs clearly and returns nil so the message
// is sent as a normal message instead of a dangling reply.
func (sa *SnapchatAPI) resolveReplyTarget(portalID, quotedRawID string) *networkid.MessageOptionalPartID {
	quotedRawID = strings.TrimSpace(quotedRawID)
	if quotedRawID == "" || quotedRawID == "0" {
		return nil
	}
	baseID := baseSnapchatMessageID(quotedRawID)
	if sa.lookupStoredMessage(portalID, baseID) == nil {
		log.Printf("bridgev2 reply: quoted Snapchat message not in local mapping chat_id=%s quoted=%s - sending without reply relation", portalID, quotedRawID)
		return nil
	}
	id := makeMessageID(scopedSnapchatMessageID(portalID, quotedRawID))
	return &networkid.MessageOptionalPartID{MessageID: id}
}

func (sa *SnapchatAPI) convertMessageEdit(ctx context.Context, portal *bridgev2.Portal, intent bridgev2.MatrixAPI, existing []*database.Message, message sidecar.Message) (*bridgev2.ConvertedEdit, error) {
	converted, err := sa.convertMessage(ctx, portal, intent, message)
	if err != nil {
		return nil, err
	}
	edit := &bridgev2.ConvertedEdit{}
	var added []*bridgev2.ConvertedMessagePart
	for idx, part := range converted.Parts {
		var target *database.Message
		if len(message.Media) > 0 {
			for _, candidate := range existing {
				if candidate.PartID == part.ID {
					target = candidate
					break
				}
			}
		} else if idx < len(existing) {
			target = existing[idx]
		}
		if target != nil {
			if meta, ok := target.Metadata.(*MediaDeliveryMetadata); len(message.Media) > 0 && ok && meta.MediaDelivered && meta.AttachmentCount == len(message.Media) && target.MXID != "" && !target.HasFakeMXID() {
				continue
			}
			edit.ModifiedParts = append(edit.ModifiedParts, part.ToEditPart(target))
		} else {
			added = append(added, part)
		}
	}
	if len(added) > 0 {
		edit.AddedParts = &bridgev2.ConvertedMessage{Parts: added}
	}
	return edit, nil
}

func truncateDescriptorID(id string) string {
	if len(id) > 16 {
		return id[:16] + "..."
	}
	return id
}

// renderAPIMessageMedia downloads and attaches clear media bytes for an
// ordinary API message. It reports whether the message was reconciled as
// already delivered, in which case the caller must skip the rest of the sync
// iteration for this message.
func (sa *SnapchatAPI) renderAPIMessageMedia(ctx context.Context, chat sidecar.Chat, client *snapapi.Client, apiMessage snapapi.Message, message *sidecar.Message, baseRemoteID string, existing **store.MessageState) bool {
	zerolog.Ctx(ctx).Info().
		Str("diag", "snapchat_media").
		Str("chat_id", chat.ID).
		Str("message_id", apiMessage.ID).
		Str("content_type", apiMessage.ContentType).
		Bool("is_snap", apiMessage.IsSnap).
		Int("attachment_count", len(apiMessage.Media)).
		Msg("media: rendering API media")
	message.Media = sa.downloadAPIMessageMedia(ctx, client, apiMessage)
	if len(message.Media) == 0 {
		return false
	}
	message.ID = mediaMessageID(message.ID, message.Media)
	if sa.mediaDeliveryConfirmed(ctx, chat.ID, baseRemoteID, *message, len(message.Media)) {
		return true
	}
	// The state row may precede a failed first Matrix send. An edit
	// needs an actual placeholder mapping, not just that state row.
	if *existing != nil && sa.UserLogin != nil && sa.UserLogin.Bridge != nil && sa.UserLogin.Bridge.DB != nil {
		parts, lookupErr := sa.currentMessageParts(ctx, chat.ID, baseRemoteID)
		if lookupErr == nil && len(parts) == 0 {
			*existing = nil
		}
	}
	return false
}

func (sa *SnapchatAPI) downloadAPIMessageMedia(ctx context.Context, client *snapapi.Client, message snapapi.Message) []sidecar.MediaAttachment {
	attachments := make([]sidecar.MediaAttachment, 0, len(message.Media))
	for _, media := range message.Media {
		zerolog.Ctx(ctx).Info().
			Str("diag", "snapchat_media").
			Str("message_id", message.ID).
			Str("media_id", media.ID).
			Str("content_type", message.ContentType).
			Bool("is_snap", message.IsSnap).
			Str("kind", string(media.Kind)).
			Bool("url_present", media.URL != "").
			Int("inline_bytes", len(media.Data)).
			Int("key_bytes", len(media.Key)).
			Int("iv_bytes", len(media.IV)).
			Str("mime_hint", media.MimeType).
			Msg("media: classify incoming attachment")
		data, mimeType, info, err := client.DownloadMediaWithInfo(ctx, media)
		if err != nil {
			zerolog.Ctx(ctx).Warn().
				Err(err).
				Str("diag", "snapchat_media").
				Str("message_id", message.ID).
				Str("media_id", media.ID).
				Str("content_type", message.ContentType).
				Bool("is_snap", message.IsSnap).
				Str("source", info.Source).
				Str("descriptor_shape", info.DescriptorShape).
				Str("cdn_fallback_reason", info.CDNFallbackReason).
				Bool("url_present", media.URL != "").
				Int("inline_bytes", len(media.Data)).
				Int("key_bytes", len(media.Key)).
				Int("iv_bytes", len(media.IV)).
				Msg("media: incoming attachment unavailable")
			continue
		}
		fileName := strings.TrimSpace(media.FileName)
		mimeType = normalizeMediaMIME(data, mimeType)
		if strings.HasPrefix(media.MimeType, "audio/") && mimeType == "video/mp4" {
			// Go's sniffer classifies any ftyp container (voice-note audio
			// included) as video/mp4; trust the declared audio type.
			mimeType = media.MimeType
		}
		fileName = mediaFileNameForMIME(fileName, matrixMsgTypeForMedia(mimeType), mimeType)
		zerolog.Ctx(ctx).Info().
			Str("diag", "snapchat_media").
			Str("message_id", message.ID).
			Str("media_id", media.ID).
			Str("source", info.Source).
			Str("decrypt_path", info.DecryptPath).
			Str("mime", mimeType).
			Int("size", len(data)).
			Str("matrix_msgtype", string(matrixMsgTypeForMedia(mimeType))).
			Msg("media: downloaded incoming attachment")
		attachments = append(attachments, sidecar.MediaAttachment{
			ID:       media.ID,
			FileName: fileName,
			MimeType: mimeType,
			Data:     data,
		})
	}
	if len(attachments) != len(message.Media) {
		return nil
	}
	return attachments
}
