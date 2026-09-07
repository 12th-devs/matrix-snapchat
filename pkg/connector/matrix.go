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
	"maunium.net/go/mautrix/event"
)

func (sa *SnapchatAPI) HandleMatrixMessage(ctx context.Context, msg *bridgev2.MatrixMessage) (*bridgev2.MatrixMessageResponse, error) {
	started := time.Now()
	if msg == nil || msg.Portal == nil || msg.Content == nil {
		return nil, fmt.Errorf("missing Matrix message data")
	}
	chatID := string(msg.Portal.ID)
	sa.noteTypingPresenceChat(chatID)
	eventID := ""
	if msg.Event != nil {
		eventID = string(msg.Event.ID)
	}
	zerolog.Ctx(ctx).Info().
		Str("diag", "send_latency").
		Str("stage", "handle_matrix_message_start").
		Str("chat_id", chatID).
		Str("event_id", eventID).
		Msg("send-latency: HandleMatrixMessage start")
	if media, caption, ok, err := sa.matrixMediaToSend(ctx, msg); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if !sa.useAPI() {
			return nil, fmt.Errorf("send snapchat media: api_mode=%s is unsafe in no-open mode; API sending is required", sa.Connector.apiMode())
		}
		messageID, err := sa.sendMediaAPI(ctx, chatID, media, caption)
		if err != nil {
			return nil, fmt.Errorf("send snapchat media via API (DOM fallback disabled): %w", err)
		}
		if messageID == "" {
			messageID = chatID + "-media-" + media.ID
		}
		sa.markSeen(chatID, messageID)
		sa.recordRecentOutgoing(chatID, caption, messageID)
		return &bridgev2.MatrixMessageResponse{
			DB: &database.Message{
				ID:       makeMessageID(scopedSnapchatMessageID(chatID, messageID)),
				SenderID: makeUserID(sa.Label),
				// The send itself proves the single attachment was delivered:
				// Snapchat accepted the message after a successful upload.
				// Persisting delivery proof keeps later syncs of the outgoing
				// echo from re-downloading the media or editing the event.
				Metadata: &MediaDeliveryMetadata{MediaDelivered: true, AttachmentCount: 1, PresentationVersion: 1},
			},
		}, nil
	}
	body, ok := sa.matrixTextToSend(msg)
	textDone := time.Now()
	zerolog.Ctx(ctx).Info().
		Str("diag", "send_latency").
		Str("stage", "matrix_text_to_send_done").
		Str("chat_id", chatID).
		Str("event_id", eventID).
		Bool("accepted", ok).
		Dur("duration", textDone.Sub(started)).
		Msg("send-latency: matrixTextToSend done")
	if !ok {
		log.Printf("bridgev2 send: ignored non-chat Matrix message chat_id=%s msgtype=%s", chatID, msg.Content.MsgType)
		return sa.ignoredMatrixMessageResponse(msg), nil
	}
	if !sa.useAPI() {
		return nil, fmt.Errorf("send snapchat message: api_mode=%s is unsafe in no-open mode; API sending is required", sa.Connector.apiMode())
	}
	messageID, err := sa.sendTextAPI(ctx, chatID, body, sa.matrixReplyTarget(chatID, msg))
	sendDone := time.Now()
	zerolog.Ctx(ctx).Info().
		Str("diag", "send_latency").
		Str("stage", "handle_matrix_message_send_done").
		Str("chat_id", chatID).
		Str("event_id", eventID).
		Str("message_id", messageID).
		Dur("send_text_api_duration", sendDone.Sub(textDone)).
		Dur("total_handle_matrix_message_duration", sendDone.Sub(started)).
		Err(err).
		Msg("send-latency: HandleMatrixMessage send done")
	if err != nil {
		return nil, fmt.Errorf("send snapchat message via API (DOM fallback disabled): %w", err)
	}
	if messageID == "" {
		messageID = chatID + "-" + body
	}
	sa.markSeen(chatID, messageID)
	sa.recordRecentOutgoing(chatID, body, messageID)
	return &bridgev2.MatrixMessageResponse{
		DB: &database.Message{
			ID:       makeMessageID(scopedSnapchatMessageID(chatID, messageID)),
			SenderID: makeUserID(sa.Label),
		},
	}, nil
}

func (sa *SnapchatAPI) HandleMatrixReadReceipt(ctx context.Context, receipt *bridgev2.MatrixReadReceipt) error {
	if receipt == nil {
		return nil
	}
	chatID := ""
	if receipt.Portal != nil {
		chatID = string(receipt.Portal.ID)
	}
	if chatID == "" {
		return nil
	}

	_, hasExactTarget := readReceiptRemoteID(receipt)
	allowedSync := hasExactTarget || sa.shouldSyncFromReadReceipt(chatID)
	if allowedSync {
		sa.hydrateSnapMediaFromReadReceipt(ctx, chatID, receipt)
	}

	diagBase := func() *zerolog.Event {
		evt := zerolog.Ctx(ctx).Debug().Str("diag", "beeper_to_snapchat_receipt").Str("chat_id", chatID).Bool("implicit", receipt.Implicit).Bool("enabled", sa.readReceiptsEnabled()).Bool("has_exact_target", hasExactTarget).Str("receipt_event_id", string(receipt.EventID))
		if receipt.ExactMessage != nil {
			evt = evt.Str("exact_target_row", string(receipt.ExactMessage.ID))
		}
		return evt
	}
	diagBase().Bool("allowed_sync", allowedSync).Msg("receipt-diag: handler entry")

	if receipt.Implicit || !sa.readReceiptsEnabled() {
		log.Printf("bridgev2 event_class: kind=read_receipt source=matrix chat_id=%s enabled=%t implicit=%t action=skip", chatID, sa.readReceiptsEnabled(), receipt.Implicit)
		diagBase().Str("stop", "implicit_or_disabled").Msg("receipt-diag: skipped")
		return nil
	}
	if !allowedSync {
		log.Printf("bridgev2 event_class: kind=read_receipt source=matrix chat_id=%s action=skip reason=unsafe_sync", chatID)
		diagBase().Str("stop", "unsafe_sync").Msg("receipt-diag: skipped")
		return nil
	}
	if !sa.useAPI() {
		log.Printf("bridgev2 receipts: skipping DOM read receipt side effects chat_id=%s read_up_to=%s", chatID, receipt.ReadUpTo.Format(time.RFC3339))
		diagBase().Str("stop", "non_api").Msg("receipt-diag: skipped")
		return nil
	}
	messageID, version := sa.readReceiptTarget(chatID, receipt)
	if messageID <= 0 {
		log.Printf("bridgev2 receipts: no API message id available chat_id=%s read_up_to=%s", chatID, receipt.ReadUpTo.Format(time.RFC3339))
		diagBase().Int64("resolved_message_id", messageID).Str("stop", "no_message_id").Msg("receipt-diag: skipped")
		return nil
	}
	state := sa.lookupStoredMessage(chatID, fmt.Sprintf("%d", messageID))
	stateKind := ""
	if state != nil {
		stateKind = state.Kind
	}
	if !sa.shouldSendReadReceiptToSnapchat(chatID, messageID, hasExactTarget) {
		log.Printf("bridgev2 receipts: skipping unsafe/non-chat read receipt chat_id=%s message_id=%d exact=%t", chatID, messageID, hasExactTarget)
		diagBase().Int64("resolved_message_id", messageID).Str("state_kind", stateKind).Str("stop", "guard_rejected").Msg("receipt-diag: skipped")
		return nil
	}
	diagBase().Int64("resolved_message_id", messageID).Int64("stored_version", version).Str("state_kind", stateKind).Msg("receipt-diag: target resolved, calling MarkRead")
	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		log.Printf("bridgev2 receipts: API client unavailable chat_id=%s message_id=%d: %v", chatID, messageID, err)
		zerolog.Ctx(ctx).Error().Err(err).Str("diag", "beeper_to_snapchat_receipt").Int64("resolved_message_id", messageID).Msg("receipt-diag: client unavailable")
		return nil
	}
	if err = client.MarkRead(ctx, chatID, messageID, version); err != nil {
		sa.invalidateSnapClient()
		log.Printf("bridgev2 receipts: API read update failed chat_id=%s message_id=%d version=%d: %v", chatID, messageID, version, err)
		zerolog.Ctx(ctx).Error().Err(err).Str("diag", "beeper_to_snapchat_receipt").Int64("resolved_message_id", messageID).Int64("version", version).Msg("receipt-diag: MarkRead returned error")
		return nil
	}
	zerolog.Ctx(ctx).Info().Str("diag", "beeper_to_snapchat_receipt").Int64("resolved_message_id", messageID).Int64("version", version).Msg("receipt-diag: MarkRead returned success")
	log.Printf("bridgev2 event_class: kind=read_receipt source=matrix chat_id=%s message_id=%d version=%d action=sent", chatID, messageID, version)
	log.Printf("bridgev2 receipts: marked Snapchat read via API chat_id=%s message_id=%d version=%d", chatID, messageID, version)
	return nil
}

func (sa *SnapchatAPI) HandleMatrixTyping(ctx context.Context, msg *bridgev2.MatrixTyping) error {
	if msg == nil || msg.Portal == nil || !sa.typingEnabled() {
		return nil
	}
	if msg.Type != bridgev2.TypingTypeText {
		return nil
	}
	chatID := string(msg.Portal.ID)
	if chatID == "" {
		return nil
	}
	sa.noteTypingPresenceChat(chatID)
	if !sa.useAPI() {
		log.Printf("bridgev2 event_class: kind=typing source=matrix chat_id=%s typing=%t action=skip reason=non_api", chatID, msg.IsTyping)
		log.Printf("bridgev2 typing: skipping non-API typing side effects chat_id=%s typing=%t", chatID, msg.IsTyping)
		return nil
	}
	const typingPulseMs = 1500
	if !sa.claimTypingUpdate(chatID, msg.IsTyping, 1200*time.Millisecond) {
		log.Printf("bridgev2 event_class: kind=typing source=matrix chat_id=%s typing=%t action=skip reason=debounce", chatID, msg.IsTyping)
		return nil
	}
	if err := sa.Client.SetTyping(ctx, chatID, msg.IsTyping, typingPulseMs); err != nil {
		log.Printf("bridgev2 event_class: kind=typing source=matrix chat_id=%s typing=%t action=failed", chatID, msg.IsTyping)
		log.Printf("bridgev2 typing: Snapchat typing API unavailable chat_id=%s typing=%t: %v", chatID, msg.IsTyping, err)
		return nil
	}
	log.Printf("bridgev2 event_class: kind=typing source=matrix chat_id=%s typing=%t duration_ms=%d action=sent", chatID, msg.IsTyping, typingPulseMs)
	log.Printf("bridgev2 typing: sent Snapchat typing update chat_id=%s typing=%t duration_ms=%d", chatID, msg.IsTyping, typingPulseMs)
	return nil
}

func (sa *SnapchatAPI) shouldSendReadReceiptToSnapchat(chatID string, messageID int64, exactTarget bool) bool {
	if messageID <= 0 {
		return false
	}
	state := sa.lookupStoredMessage(chatID, fmt.Sprintf("%d", messageID))
	if state == nil && exactTarget {
		return false
	}
	if state == nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(state.Kind)) {
	case "snap", "media":
		return false
	}
	if state.HasMedia {
		return false
	}
	return true
}

func (sa *SnapchatAPI) hydrateSnapMediaFromReadReceipt(ctx context.Context, chatID string, receipt *bridgev2.MatrixReadReceipt) {
	if !sa.snapMediaEnabled() || !sa.snapMediaOnReadEnabled() || !sa.useAPI() {
		return
	}
	targetID, exactTarget := readReceiptRemoteID(receipt)
	messageID, ok := parseSnapchatMessageID(targetID)
	version := int64(0)
	if !ok || messageID <= 0 {
		messageID, version = sa.readReceiptTarget(chatID, receipt)
		targetID = fmt.Sprintf("%d", messageID)
	}
	if messageID <= 0 || targetID == "" {
		return
	}
	if state := sa.lookupStoredMessage(chatID, targetID); state != nil {
		if state.HydratedAt != nil {
			return
		}
		if !state.HasMedia && state.Kind != "snap" {
			return
		}
	}

	chat := sa.lookupChat(chatID)
	if chat.ID == "" {
		chat = sidecar.Chat{
			ID:   chatID,
			URL:  snapchatConversationURL(chatID),
			Name: sa.lookupChatName(chatID),
		}
		if chat.Name == "" {
			chat.Name = chatID
		}
	}
	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		log.Printf("bridgev2 media: API client unavailable for read hydrate chat=%q id=%s message_id=%s: %v", chat.Name, chat.ID, targetID, err)
		return
	}
	limit := sa.messageFetchLimit()
	if limit < 40 {
		limit = 40
	}
	messages, err := client.QueryMessages(ctx, chatID, 0, limit)
	if err != nil {
		sa.invalidateSnapClient()
		log.Printf("bridgev2 media: API query failed for read hydrate chat=%q id=%s message_id=%s version=%d: %v", chat.Name, chat.ID, targetID, version, err)
		return
	}

	var apiMessage snapapi.Message
	for _, candidate := range messages {
		if baseSnapchatMessageID(candidate.ID) == targetID {
			apiMessage = candidate
			break
		}
	}
	if apiMessage.ID == "" {
		log.Printf("bridgev2 media: target snap not found in API read hydrate chat=%q id=%s message_id=%s", chat.Name, chat.ID, targetID)
		return
	}
	sa.rememberAPIMessage(chat.ID, apiMessage)
	if !apiMessage.IsSnap && len(apiMessage.Media) == 0 {
		return
	}

	media := sa.downloadAPIMessageMedia(ctx, client, apiMessage)
	message := connectorMessageFromAPI(apiMessage)
	message.ID = targetID
	if len(media) == 0 {
		message.Text = "Snap unavailable"
		if exactTarget {
			sa.queueRemoteMessageEdit(chat, targetID, message)
		} else {
			message.ID = targetID + "-snap-unavailable"
			sa.queueRemoteMessage(chat, message)
		}
		return
	}
	message.Media = media
	if exactTarget {
		sa.queueRemoteMessageEdit(chat, targetID, message)
	} else {
		message.ID = mediaMessageID(targetID, media)
		sa.queueRemoteMessage(chat, message)
	}
	// Hydration is committed by the event completion hook after Matrix delivery.
}

func readReceiptRemoteID(receipt *bridgev2.MatrixReadReceipt) (string, bool) {
	if receipt == nil || receipt.ExactMessage == nil {
		return "", false
	}
	id := baseSnapchatMessageID(string(receipt.ExactMessage.ID))
	return id, id != ""
}

func (sa *SnapchatAPI) lookupStoredMessage(chatID, messageID string) *store.MessageState {
	if sa == nil || sa.Connector == nil || sa.Connector.store == nil || chatID == "" || messageID == "" {
		return nil
	}
	state, err := sa.Connector.store.GetMessage(chatID, messageID)
	if err != nil {
		log.Printf("bridgev2 store: failed to lookup message state chat_id=%s message_id=%s: %v", chatID, messageID, err)
		return nil
	}
	return state
}

func (sa *SnapchatAPI) markStoredMessageHydrated(chatID, messageID string, apiMessage snapapi.Message, message sidecar.Message) {
	if sa == nil || sa.Connector == nil || sa.Connector.store == nil || chatID == "" || messageID == "" {
		return
	}
	now := time.Now()
	_ = sa.Connector.store.UpsertMessages([]store.MessageState{{
		PortalKey:    chatID,
		RemoteID:     messageID,
		Author:       message.Author,
		Text:         message.Text,
		Outgoing:     message.Outgoing,
		TimestampRaw: message.Timestamp,
		Kind:         messageKindFromAPI(apiMessage),
		HasMedia:     len(apiMessage.Media) > 0,
		HydratedAt:   &now,
		LastSeenAt:   now,
	}})
	if err := sa.Connector.store.MarkMessageHydrated(chatID, messageID); err != nil {
		log.Printf("bridgev2 store: failed to mark message hydrated chat_id=%s message_id=%s: %v", chatID, messageID, err)
	}
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
	if isBridgeGeneratedSnapchatMarker(body) {
		return "", false
	}
	return body, true
}

// matrixReplyTarget resolves the Beeper reply relation to the numeric Snapchat
// message ID that CreateContentMessage.FeatureAttachment expects. A missing or
// unresolvable target sends the message without a reply relation (logged).
func (sa *SnapchatAPI) matrixReplyTarget(chatID string, msg *bridgev2.MatrixMessage) int64 {
	if msg == nil || msg.ReplyTo == nil {
		return 0
	}
	targetID := string(msg.ReplyTo.ID)
	messageID, ok := parseSnapchatMessageID(targetID)
	if !ok {
		log.Printf("bridgev2 send: reply target not resolvable to a Snapchat message chat_id=%s target=%s - sending without reply", chatID, targetID)
		return 0
	}
	return messageID
}

// chatIDFromScopedMessageID extracts the chat ID from a scoped message ID of
// the form "<chatID>.msg.<messageID>". Returns empty for unscoped IDs.
func chatIDFromScopedMessageID(raw string) string {
	raw = strings.TrimSpace(raw)
	if idx := strings.LastIndex(raw, ".msg."); idx > 0 {
		return raw[:idx]
	}
	return ""
}

// HandleMatrixEdit handles edits of previously bridged messages. Snapchat has
// no edit operation in its web/API protocol (no content-editing update action),
// so edits are rejected explicitly rather than faked with a follow-up message.
// The capability matrix advertises Edit: CapLevelRejected, so this is a safety
// net for clients that send an edit anyway.
func (sa *SnapchatAPI) HandleMatrixEdit(ctx context.Context, msg *bridgev2.MatrixEdit) error {
	if msg == nil || msg.EditTarget == nil {
		return nil
	}
	return fmt.Errorf("Snapchat does not support editing sent messages; the edit was not forwarded")
}

// HandleMatrixMessageRemove redacts/deletes a previously bridged message and
// unsends the corresponding Snapchat message (UpdateAction_Erase).
func (sa *SnapchatAPI) HandleMatrixMessageRemove(ctx context.Context, msg *bridgev2.MatrixMessageRemove) error {
	removeLog := zerolog.Nop()
	if sa.UserLogin != nil {
		removeLog = sa.UserLogin.Log
	}
	if msg == nil || msg.TargetMessage == nil {
		removeLog.Debug().Str("action", "handle matrix remove").Msg("remove handler entered with no mapped target message - nothing to erase")
		return nil
	}
	targetID := string(msg.TargetMessage.ID)
	chatID := chatIDFromScopedMessageID(targetID)
	messageID, _ := parseSnapchatMessageID(targetID)
	removeLog.Debug().Str("action", "handle matrix remove").Str("target_message_id", targetID).Str("chat_id", chatID).Int64("snap_message_id", messageID).Msg("remove handler entered")
	if messageID <= 0 {
		log.Printf("bridgev2 remove: target is not a Snapchat-mapped message id=%s - nothing to erase", targetID)
		return nil
	}
	if chatID == "" {
		return fmt.Errorf("delete snapchat message: target %s is missing the chat scope", targetID)
	}
	if !sa.useAPI() {
		return fmt.Errorf("delete snapchat message: api_mode=%s is unsafe in no-open mode; API sending is required", sa.Connector.apiMode())
	}
	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		return fmt.Errorf("snapchat client unavailable for message delete: %w", err)
	}
	eraseErr := client.EraseMessage(ctx, chatID, messageID)
	if eraseErr != nil {
		sa.invalidateSnapClient()
		removeLog.Warn().Str("action", "handle matrix remove").Str("chat_id", chatID).Int64("snap_message_id", messageID).Err(eraseErr).Msg("EraseMessage returned error")
		return fmt.Errorf("unsend Snapchat message failed chat_id=%s message_id=%d: %w", chatID, messageID, eraseErr)
	}
	removeLog.Info().Str("action", "handle matrix remove").Str("chat_id", chatID).Int64("snap_message_id", messageID).Msg("EraseMessage returned success")
	log.Printf("bridgev2 event_class: kind=message_delete source=matrix chat_id=%s message_id=%d action=sent", chatID, messageID)
	return nil
}

func (sa *SnapchatAPI) matrixMediaToSend(ctx context.Context, msg *bridgev2.MatrixMessage) (sidecar.MediaAttachment, string, bool, error) {
	if msg == nil || msg.Content == nil {
		return sidecar.MediaAttachment{}, "", false, nil
	}
	// Sticker-shaped events (m.sticker events and msgtype-less media, both
	// mapped to CapMsgSticker by the caps gate) carry a plain image payload
	// and ride the ordinary image path.
	if msg.Content.MsgType == event.CapMsgSticker ||
		(msg.Content.MsgType == "" && (msg.Content.URL != "" || msg.Content.File != nil)) {
		msg.Content.MsgType = event.MsgImage
	}
	switch msg.Content.MsgType {
	case event.MsgImage, event.MsgVideo:
	case event.MsgAudio:
		// Beeper voice notes arrive as m.audio (MSC3245 voice marker is not
		// required); they are forwarded as Snapchat voice notes.
	case event.MsgFile:
		return sidecar.MediaAttachment{}, "", true, fmt.Errorf("outbound Snapchat media supports only images, MP4 video, and voice notes in this version (got %s)", msg.Content.MsgType)
	default:
		return sidecar.MediaAttachment{}, "", false, nil
	}
	if !sa.sendMediaEnabled() {
		log.Printf("bridgev2 send: media message ignored because send_media_enabled=false msgtype=%s", msg.Content.MsgType)
		return sidecar.MediaAttachment{}, "", true, fmt.Errorf("Snapchat snap/media sending is disabled in bridge config")
	}
	if msg.Portal == nil || msg.Portal.Bridge == nil || msg.Portal.Bridge.Bot == nil {
		return sidecar.MediaAttachment{}, "", true, fmt.Errorf("missing Matrix media downloader")
	}
	uri := msg.Content.URL
	if uri == "" && msg.Content.File != nil {
		uri = msg.Content.File.URL
	}
	if uri == "" {
		return sidecar.MediaAttachment{}, "", true, fmt.Errorf("Matrix media message did not include a downloadable URL")
	}
	data, err := msg.Portal.Bridge.Bot.DownloadMedia(ctx, uri, msg.Content.File)
	if err != nil {
		return sidecar.MediaAttachment{}, "", true, fmt.Errorf("download Matrix media: %w", err)
	}
	if len(data) == 0 {
		return sidecar.MediaAttachment{}, "", true, fmt.Errorf("Matrix media download was empty")
	}
	mimeType := ""
	if msg.Content.Info != nil {
		mimeType = strings.TrimSpace(msg.Content.Info.MimeType)
	}
	mimeType = normalizeMediaMIME(data, mimeType)
	if msg.Content.MsgType == event.MsgAudio {
		// Go's sniffer classifies any ftyp container (audio-only included)
		// as video/mp4; a voice note must reach the audio classifier.
		if mimeType == "video/mp4" {
			mimeType = "audio/mp4"
		}
	}
	if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/webp" && mimeType != "video/mp4" && mimeType != "audio/mp4" {
		return sidecar.MediaAttachment{}, "", true, fmt.Errorf("outbound Snapchat media supports only image/jpeg, image/png, image/webp, video/mp4, and audio/mp4 voice notes (got %q)", mimeType)
	}
	fileName := strings.TrimSpace(msg.Content.FileName)
	if fileName == "" {
		fileName = strings.TrimSpace(msg.Content.Body)
	}
	fileName = mediaFileNameForMIME(fileName, msg.Content.MsgType, mimeType)
	caption := strings.TrimSpace(msg.Content.Body)
	if caption == fileName {
		caption = ""
	}
	return sidecar.MediaAttachment{
		ID:       stableTextID(fileName + ":" + fmt.Sprint(len(data))),
		FileName: sanitizeOutgoingFileName(fileName),
		MimeType: mimeType,
		Data:     data,
	}, caption, true, nil
}
