package snapapi

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

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
	if contentType == protos.ContentType_EXTERNAL_MEDIA && len(media) == 1 {
		decoded := c.envelopeContentsForDecode(ctx, chatID, id, envelope)
		key, iv, err := ExternalMediaEncryptionKeys(decoded)
		if err != nil {
			log.Printf("snapapi media: external encryption metadata unavailable message_id=%s: %v", id, err)
		} else if len(key) > 0 {
			media[0].Key, media[0].IV = key, iv
		}
	}
	if contentType == protos.ContentType_NOTE && len(media) == 1 {
		// Voice notes carry the AES key/IV (base64) inside the note metadata
		// and reference the audio content object with an unassigned media
		// type; present them as audio so the bridge renders m.audio.
		decoded := c.envelopeContentsForDecode(ctx, chatID, id, envelope)
		key, iv, _, err := NoteAudioEncryptionKeys(decoded)
		if err != nil {
			log.Printf("snapapi media: note encryption metadata unavailable message_id=%s: %v", id, err)
		} else if len(key) > 0 {
			media[0].Key, media[0].IV = key, iv
		}
		if media[0].Kind != MediaKindVideo {
			media[0].Kind = MediaKindAudio
			if media[0].MimeType == "" || media[0].MimeType == "application/octet-stream" {
				media[0].MimeType = "audio/mp4"
			}
		}
	}
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
		ID:              id,
		AuthorID:        authorID,
		Author:          authorName,
		Text:            body,
		Timestamp:       createdAt,
		Outgoing:        authorID != "" && authorID == c.SelfUserID(),
		IsSnap:          isSnap,
		ContentType:     contentType.String(),
		Media:           media,
		Version:         msg.GetMetaData().GetConversationVersion(),
		Saved:           saved,
		DisappearAfter:  disappearAfter,
		QuotedMessageID: msg.GetMetaData().GetQuotedMetadata().GetQuotedMessageId(),
		Tombstone:       msg.GetMetaData().GetTombstone(),
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
		if text := decodedChatBody(encodedContents); contentType == protos.ContentType_CHAT && text != "" {
			return text, false
		}
		if err := protos.DecodeProtoMessage(encodedContents, &contents); err == nil {
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
	case protos.ContentType_NOTE:
		// Voice notes are ordinary persistent media, not snaps.
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

func plaintextChatBody(raw []byte) string {
	if len(raw) == 0 || !utf8.Valid(raw) {
		return ""
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	for _, r := range text {
		if r == 0 || (r < 0x20 && r != '\n' && r != '\r' && r != '\t') {
			return ""
		}
	}
	return text
}

func decodedChatBody(raw []byte) string {
	var contents protos.Contents
	if err := protos.DecodeProtoMessage(raw, &contents); err == nil {
		if text := contents.GetText().GetText(); strings.TrimSpace(text) != "" {
			return text
		}
	}
	if text := plaintextChatBody(raw); text != "" {
		return text
	}
	if text := protoStringAtPath(raw, []int{4, 4, 2, 1}); text != "" {
		return text
	}
	return ""
}

func protoStringAtPath(raw []byte, path []int) string {
	if len(raw) == 0 || len(path) == 0 {
		return ""
	}
	field := path[0]
	for offset := 0; offset < len(raw); {
		tag, next, ok := readProtoVarint(raw, offset)
		if !ok {
			return ""
		}
		offset = next
		currentField := int(tag >> 3)
		wireType := int(tag & 0x7)
		switch wireType {
		case 0:
			_, next, ok = readProtoVarint(raw, offset)
			if !ok {
				return ""
			}
			offset = next
		case 1:
			if offset+8 > len(raw) {
				return ""
			}
			offset += 8
		case 2:
			length, next, ok := readProtoVarint(raw, offset)
			if !ok || length > uint64(len(raw)-next) {
				return ""
			}
			start := next
			end := start + int(length)
			offset = end
			if currentField != field {
				continue
			}
			value := raw[start:end]
			if len(path) == 1 {
				return plaintextChatBody(value)
			}
			if text := protoStringAtPath(value, path[1:]); text != "" {
				return text
			}
		case 5:
			if offset+4 > len(raw) {
				return ""
			}
			offset += 4
		default:
			return ""
		}
	}
	return ""
}

func readProtoVarint(raw []byte, offset int) (uint64, int, bool) {
	var value uint64
	for shift := uint(0); offset < len(raw) && shift < 64; shift += 7 {
		b := raw[offset]
		offset++
		value |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return value, offset, true
		}
	}
	return 0, offset, false
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
		if decrypted, ok := decryptEELGCM(raw, eel.GetCek(), eel.GetNonce()); ok {
			return decrypted
		}
		if decrypted, ok := decryptEELGCM(raw, eel.GetCek(), eel.GetCekIv()); ok {
			return decrypted
		}
		if c != nil && c.eelDecrypter != nil {
			cacheKey := chatID + "|" + messageID
			c.mu.Lock()
			if cached := c.eelPlaintext[cacheKey]; len(cached) > 0 {
				out := append([]byte(nil), cached...)
				c.mu.Unlock()
				return out
			}
			lastFailure, alreadyFailed := c.failedEEL[cacheKey]
			c.mu.Unlock()
			// Full EEL decrypt depends on Snapchat's Fidelius/WASM session. If
			// the helper is not ready for a historical message yet, retrying it
			// every poll burns connector time and can make live sync look flaky.
			// New message IDs still get one immediate attempt; known misses are
			// retried periodically in case the helper becomes available later.
			if alreadyFailed && time.Since(lastFailure) < 2*time.Minute {
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
				log.Printf("snapapi decode: EEL helper failed message_id=%s conversation_id=%s content=%d cek=%d cek_iv=%d nonce=%d sender_pub=%d sender_version=%d failure=%T",
					messageID, chatID, len(raw), len(eel.GetCek()), len(eel.GetCekIv()), len(eel.GetNonce()), len(eel.GetSenderPublicKey()), eel.GetSenderVersion(), err)
				c.mu.Lock()
				if c.failedEEL == nil {
					c.failedEEL = make(map[string]time.Time)
				}
				c.failedEEL[cacheKey] = time.Now()
				c.mu.Unlock()
				return nil
			}
			c.mu.Lock()
			if c.eelPlaintext == nil {
				c.eelPlaintext = make(map[string][]byte)
			}
			c.eelPlaintext[cacheKey] = append([]byte(nil), decrypted...)
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
		if result.GetConversationDestinationResult().GetCreatedMessageId() != 0 {
			return fmt.Sprintf("%d", result.GetConversationDestinationResult().GetCreatedMessageId())
		}
	}
	return ""
}
