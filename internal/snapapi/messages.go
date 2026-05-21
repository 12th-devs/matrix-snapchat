package snapapi

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
)

func (c *Client) QueryMessages(ctx context.Context, chatID string, version int64, limit int) ([]Message, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	var convID *protos.UUID
	if version <= 0 {
		conv, err := c.conversation(ctx, chatID)
		if err != nil {
			return nil, err
		}
		convID = conv.GetConversationId()
		version = conv.GetVersion()
	}
	if convID == nil {
		var err error
		convID, err = encodeUUIDString(chatID)
		if err != nil {
			return nil, err
		}
	}
	messages, err := c.queryMessages(ctx, convID, version, limit)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 && version != 0 {
		messages, err = c.queryMessages(ctx, convID, 0, limit)
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
	return messages, nil
}

func (c *Client) queryMessages(ctx context.Context, convID *protos.UUID, version int64, limit int) ([]Message, error) {
	req := &protos.QueryMessagesRequest{
		SelfUserId:         c.selfUUID(),
		ConversationId:     convID,
		RequestedCountSize: int32(limit),
		CurrentVersion:     version,
	}
	var resp protos.QueryMessagesResponse
	if err := c.doGRPC(ctx, paths.QUERY_MESSAGES, req, &resp); err != nil {
		return nil, err
	}
	result := make([]Message, 0, len(resp.GetMessages()))
	chatID := uuidToString(convID)
	for _, msg := range resp.GetMessages() {
		converted := c.messageFromProto(ctx, chatID, msg)
		if converted.ID != "" {
			result = append(result, converted)
		}
	}
	return result, nil
}

func (c *Client) messageFromProto(ctx context.Context, chatID string, msg *protos.ContentMessage) Message {
	id := fmt.Sprintf("%d", msg.GetMessageId())
	if msg.GetMessageId() == 0 {
		id = fmt.Sprintf("client-%d", msg.GetClientResolutionId())
	}
	authorID := uuidToString(msg.GetSenderId())
	envelope := msg.GetContents()
	contentType := envelope.GetContentType()
	body, isSnap := c.messageBody(ctx, chatID, msg)
	media := mediaAttachmentsFromEnvelope(envelope, id)
	if body == "" && len(media) > 0 {
		if isSnap {
			body = "New Snap"
		} else {
			body = "Media"
		}
	}
	if body == "" {
		return Message{}
	}
	createdAt := timeFromMillis(msg.GetMetaData().GetServerCreatedAt())
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	authorName := c.lookupName(authorID)
	if authorName == "" && authorID == c.SelfUserID() {
		authorName = "You"
	}
	if authorName == "" {
		authorName = shortID(authorID)
	}
	saved := messageIsSaved(msg)
	disappearAfter := time.Duration(0)
	if !saved {
		disappearAfter = c.retentionDurationForChat(chatID)
	}
	return Message{
		ID:             id,
		AuthorID:       authorID,
		Author:         authorName,
		Text:           body,
		Timestamp:      createdAt,
		Outgoing:       authorID != "" && authorID == c.SelfUserID(),
		IsSnap:         isSnap,
		ContentType:    contentType.String(),
		Media:          media,
		Version:        msg.GetMetaData().GetConversationVersion(),
		Saved:          saved,
		DisappearAfter: disappearAfter,
	}
}

func messageIsSaved(msg *protos.ContentMessage) bool {
	if msg == nil {
		return false
	}
	if len(msg.GetMetaData().GetSavedBy()) > 0 {
		return true
	}
	return msg.GetContents().GetSavePolicy() == protos.ContentEnvelope_SavePolicy_LIFETIME
}

func (c *Client) messageBody(ctx context.Context, chatID string, msg *protos.ContentMessage) (string, bool) {
	envelope := msg.GetContents()
	if envelope == nil {
		return "New Snap", true
	}
	messageID := fmt.Sprintf("%d", msg.GetMessageId())
	if msg.GetMessageId() == 0 {
		messageID = fmt.Sprintf("client-%d", msg.GetClientResolutionId())
	}
	contentType := envelope.GetContentType()
	contentLen := len(envelope.GetContents())
	var contents protos.Contents
	if encodedContents := c.envelopeContentsForDecode(ctx, chatID, messageID, envelope); len(encodedContents) > 0 {
		if err := protos.DecodeProtoMessage(encodedContents, &contents); err == nil {
			if text := contents.GetText().GetText(); strings.TrimSpace(text) != "" {
				return text, false
			}
			if contentType == protos.ContentType_CHAT {
				log.Printf("snapapi decode: CHAT payload had no text message_id=%s content_len=%d content_variant=%T encryption=%T",
					messageID, contentLen, contents.GetContent(), envelope.GetEnvelopeEncryption().GetMethod())
			}
		} else if contentType == protos.ContentType_CHAT {
			log.Printf("snapapi decode: CHAT payload decode failed message_id=%s content_len=%d encryption=%T error=%v",
				messageID, contentLen, envelope.GetEnvelopeEncryption().GetMethod(), err)
		}
	}
	switch contentType {
	case protos.ContentType_CHAT:
		log.Printf("snapapi decode: CHAT fallback message_id=%s chat_id=%s content_len=%d encryption=%T",
			messageID, chatID, contentLen, envelope.GetEnvelopeEncryption().GetMethod())
		return "[Snapchat message unavailable]", false
	case protos.ContentType_SNAP, protos.ContentType_SNAP_NOT_VIEWABLE:
		return "New Snap", true
	case protos.ContentType_EXTERNAL_MEDIA:
		return "", false
	case protos.ContentType_STATUS, protos.ContentType_STATUS_SAVE_TO_CAMERA_ROLL,
		protos.ContentType_STATUS_CONVERSATION_CAPTURE_SCREENSHOT,
		protos.ContentType_STATUS_CONVERSATION_CAPTURE_RECORD,
		protos.ContentType_STATUS_CALL_MISSED_VIDEO,
		protos.ContentType_STATUS_CALL_MISSED_AUDIO:
		return fmt.Sprintf("[Snapchat status] %s", envelope.GetContentType().String()), false
	default:
		return "New Snap", true
	}
}

func (c *Client) envelopeContentsForDecode(ctx context.Context, chatID, messageID string, envelope *protos.ContentEnvelope) []byte {
	if envelope == nil {
		return nil
	}
	raw := envelope.GetContents()
	if len(raw) == 0 {
		return nil
	}
	if clear := envelope.GetEnvelopeEncryption().GetClearTextEelKeyEncryption(); clear != nil {
		if decrypted, ok := decryptEELGCM(raw, clear.GetCek(), clear.GetCekIv()); ok {
			return decrypted
		}
	}
	if eel := envelope.GetEnvelopeEncryption().GetEelEncryption(); eel != nil {
		if decrypted, ok := decryptEELGCM(raw, eel.GetCek(), eel.GetCekIv()); ok {
			return decrypted
		}
		if c != nil && c.eelDecrypter != nil {
			cacheKey := chatID + "|" + messageID
			c.mu.Lock()
			lastFailure, alreadyFailed := c.failedEEL[cacheKey]
			c.mu.Unlock()
			if alreadyFailed && time.Since(lastFailure) < 15*time.Second {
				return nil
			}
			decrypted, err := c.eelDecrypter.DecryptEEL(ctx, EELDecryptRequest{
				ConversationID:  chatID,
				MessageID:       messageID,
				Content:         raw,
				CEK:             eel.GetCek(),
				CEKIV:           eel.GetCekIv(),
				Nonce:           eel.GetNonce(),
				SenderPublicKey: eel.GetSenderPublicKey(),
				SenderVersion:   eel.GetSenderVersion(),
			})
			if err != nil {
				log.Printf("snapapi decode: EEL helper failed message_id=%s conversation_id=%s failure=%T", messageID, chatID, err)
				c.mu.Lock()
				if c.failedEEL == nil {
					c.failedEEL = make(map[string]time.Time)
				}
				c.failedEEL[cacheKey] = time.Now()
				c.mu.Unlock()
				return nil
			}
			c.mu.Lock()
			delete(c.failedEEL, cacheKey)
			c.mu.Unlock()
			return decrypted
		}
		log.Printf("snapapi decode: EEL helper unavailable message_id=%s conversation_id=%s cek=%d cek_iv=%d nonce=%d sender_pub=%d sender_version=%d",
			messageID, chatID, len(eel.GetCek()), len(eel.GetCekIv()), len(eel.GetNonce()), len(eel.GetSenderPublicKey()), eel.GetSenderVersion())
		return nil
	}
	return raw
}

func createdMessageIDFromResponse(resp *protos.CreateContentMessageResponse) string {
	if resp == nil {
		return ""
	}
	for _, result := range resp.GetResult() {
		if result.GetCreatedMessageId() != 0 {
			return fmt.Sprintf("%d", result.GetCreatedMessageId())
		}
	}
	return ""
}
