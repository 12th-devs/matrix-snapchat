package connector

import (
	"context"
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

func (sa *SnapchatAPI) syncChatMessagesAPI(ctx context.Context, chat sidecar.Chat, version int64, reason string, includeMedia bool, requireAutoFetch bool) error {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" {
		return nil
	}
	sa.rememberChatDisappear(chat.ID, chat.DisappearAfterSeconds)
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
	log.Printf("bridgev2 sync: API fetched %d messages for chat=%q id=%s reason=%s", len(messages), chat.Name, chat.ID, reason)
	states := make([]store.MessageState, 0, len(messages))
	for _, apiMessage := range messages {
		apiMessage = sa.normalizeAPIMessageDirection(chat.ID, apiMessage)
		message := connectorMessageFromAPI(apiMessage)
		message = sa.normalizeRemoteMessageDirection(chat.ID, message)
		if message.ID == "" {
			continue
		}
		baseRemoteID := baseSnapchatMessageID(message.ID)
		if baseRemoteID == "" {
			baseRemoteID = message.ID
		}
		if includeMedia && sa.snapMediaEnabled() && len(apiMessage.Media) > 0 {
			log.Printf("bridgev2 media: rendering API media chat_id=%s message_id=%s content_type=%s is_snap=%t attachment_count=%d", chat.ID, apiMessage.ID, apiMessage.ContentType, apiMessage.IsSnap, len(apiMessage.Media))
			message.Media = sa.downloadAPIMessageMedia(ctx, client, apiMessage)
			if len(message.Media) > 0 {
				message.ID = mediaMessageID(message.ID, message.Media)
			}
		}
		sa.rememberAPIMessage(chat.ID, apiMessage)
		if existing := sa.lookupStoredMessage(chat.ID, baseRemoteID); existing != nil && existing.Text != "" && existing.Text != message.Text {
			log.Printf("bridgev2 edit: API message text changed, queueing Matrix edit chat_id=%s message_id=%s kind=%s old_len=%d new_len=%d", chat.ID, baseRemoteID, messageKindFromAPI(apiMessage), len(existing.Text), len(message.Text))
			sa.queueRemoteMessageEdit(chat, baseRemoteID, message)
		} else {
			sa.queueRemoteMessage(chat, message)
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

func (sa *SnapchatAPI) queueRemoteMessage(chat sidecar.Chat, message sidecar.Message) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" || message.ID == "" {
		return
	}
	message = sa.normalizeRemoteMessageDirection(chat.ID, message)
	sa.rememberChatDisappear(chat.ID, chat.DisappearAfterSeconds)
	if sa.shouldSuppressOutgoingEcho(chat.ID, message) {
		sa.markSeen(chat.ID, message.ID)
		log.Printf("bridgev2 echo: suppressed outgoing API echo chat_id=%s message_id=%s content_type=%s", chat.ID, message.ID, message.ContentType)
		return
	}
	if sa.isSeen(chat.ID, message.ID) {
		log.Printf("bridgev2 dedupe: already saw message chat_id=%s message_id=%s", chat.ID, message.ID)
		return
	}
	sa.markSeen(chat.ID, message.ID)
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
	if message.Outgoing {
		// Leave Sender empty for from-me events. If Sender is set to
		// browser-session, bridgev2 creates/uses a ghost before checking
		// IsFromMe, which makes Snapchat UI sends appear as received messages
		// from @sh-snapchat_browser-session in Beeper.
		sender.Sender = ""
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
	sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.Message[sidecar.Message]{
		EventMeta: simplevent.EventMeta{
			Type: bridgev2.RemoteEventMessage,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("message_id", message.ID).Str("event_id", eventID)
			},
			PortalKey:    portalKey,
			CreatePortal: true,
			Sender:       sender,
			Timestamp:    parseMessageTimestamp(message.Timestamp),
		},
		ID:                 makeMessageID(eventID),
		Data:               copyMsg,
		ConvertMessageFunc: sa.convertMessage,
	})
}

func (sa *SnapchatAPI) queueRemoteMessageEdit(chat sidecar.Chat, targetMessageID string, message sidecar.Message) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" || targetMessageID == "" || message.ID == "" {
		return
	}
	message = sa.normalizeRemoteMessageDirection(chat.ID, message)
	copyMsg := message
	sa.rememberChat(chat.ID, chat.URL, chat.Name, chat.OtherUserID)
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
		sender.Sender = ""
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
	sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.Message[sidecar.Message]{
		EventMeta: simplevent.EventMeta{
			Type: bridgev2.RemoteEventEdit,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("message_id", message.ID).Str("target_message_id", targetMessageID).Str("target_event_id", targetEventID)
			},
			PortalKey:    portalKey,
			CreatePortal: true,
			Sender:       sender,
			Timestamp:    parseMessageTimestamp(message.Timestamp),
		},
		ID:                 makeMessageID(editID),
		TargetMessage:      makeMessageID(targetEventID),
		Data:               copyMsg,
		ConvertMessageFunc: sa.convertMessage,
		ConvertEditFunc:    sa.convertMessageEdit,
	})
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
		parts := make([]*bridgev2.ConvertedMessagePart, 0, len(message.Media))
		uploadIntent := mediaUploadIntent(portal, intent)
		for _, media := range message.Media {
			if len(media.Data) == 0 {
				continue
			}
			mimeType := normalizeMediaMIME(media.Data, media.MimeType)
			msgType := matrixMsgTypeForMedia(mimeType)
			fileName := mediaFileNameForMIME(media.FileName, msgType, mimeType)
			mxc, file, err := uploadIntent.UploadMedia(ctx, portal.MXID, media.Data, fileName, mimeType)
			if err != nil {
				log.Printf("bridgev2 sync: failed to upload Snapchat media message_id=%s media_id=%s: %v", message.ID, media.ID, err)
				continue
			}
			log.Printf("bridgev2 sync: uploaded Snapchat media message_id=%s media_id=%s msgtype=%s mime=%s size=%d", message.ID, media.ID, msgType, mimeType, len(media.Data))
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
			return &bridgev2.ConvertedMessage{
				Parts:     parts,
				Disappear: sa.disappearingSettingForMessage(portalID, message),
			}, nil
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
		Disappear: sa.disappearingSettingForMessage(portalID, message),
	}, nil
}

func (sa *SnapchatAPI) convertMessageEdit(ctx context.Context, portal *bridgev2.Portal, intent bridgev2.MatrixAPI, existing []*database.Message, message sidecar.Message) (*bridgev2.ConvertedEdit, error) {
	converted, err := sa.convertMessage(ctx, portal, intent, message)
	if err != nil {
		return nil, err
	}
	edit := &bridgev2.ConvertedEdit{}
	var added []*bridgev2.ConvertedMessagePart
	for idx, part := range converted.Parts {
		if idx < len(existing) {
			edit.ModifiedParts = append(edit.ModifiedParts, part.ToEditPart(existing[idx]))
		} else {
			added = append(added, part)
		}
	}
	if len(added) > 0 {
		edit.AddedParts = &bridgev2.ConvertedMessage{Parts: added}
	}
	return edit, nil
}

func (sa *SnapchatAPI) downloadAPIMessageMedia(ctx context.Context, client *snapapi.Client, message snapapi.Message) []sidecar.MediaAttachment {
	attachments := make([]sidecar.MediaAttachment, 0, len(message.Media))
	for _, media := range message.Media {
		data, mimeType, err := client.DownloadMedia(ctx, media)
		if err != nil {
			log.Printf("bridgev2 sync: Snapchat media unavailable message_id=%s media_id=%s url=%t data=%d: %v", message.ID, media.ID, media.URL != "", len(media.Data), err)
			continue
		}
		fileName := strings.TrimSpace(media.FileName)
		mimeType = normalizeMediaMIME(data, mimeType)
		fileName = mediaFileNameForMIME(fileName, matrixMsgTypeForMedia(mimeType), mimeType)
		attachments = append(attachments, sidecar.MediaAttachment{
			ID:       media.ID,
			FileName: fileName,
			MimeType: mimeType,
			Data:     data,
		})
	}
	return attachments
}
