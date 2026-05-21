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
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/event"
)

func (sa *SnapchatAPI) HandleMatrixMessage(ctx context.Context, msg *bridgev2.MatrixMessage) (*bridgev2.MatrixMessageResponse, error) {
	if msg == nil || msg.Portal == nil || msg.Content == nil {
		return nil, fmt.Errorf("missing Matrix message data")
	}
	chatID := string(msg.Portal.ID)
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

	if receipt.Implicit || !sa.readReceiptsEnabled() {
		return nil
	}
	if !allowedSync {
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
	sa.markStoredMessageHydrated(chat.ID, targetID, apiMessage, message)
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
	if isGeneratedSnapchatNotice(body) {
		return "", false
	}
	return body, true
}

func (sa *SnapchatAPI) matrixMediaToSend(ctx context.Context, msg *bridgev2.MatrixMessage) (sidecar.MediaAttachment, string, bool, error) {
	if msg == nil || msg.Content == nil {
		return sidecar.MediaAttachment{}, "", false, nil
	}
	switch msg.Content.MsgType {
	case event.MsgImage, event.MsgVideo, event.MsgFile:
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
