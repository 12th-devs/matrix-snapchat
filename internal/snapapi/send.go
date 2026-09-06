package snapapi

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	snapcrypto "github.com/0xzer/snapper/crypto"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func (c *Client) SendText(ctx context.Context, chatID, text string, replyToMessageID int64) (string, error) {
	started := time.Now()
	if err := c.ensureAuthenticated(ctx); err != nil {
		return "", err
	}
	authDone := time.Now()
	conv, err := c.conversationFresh(ctx, chatID)
	if err != nil {
		return "", err
	}
	conversationDone := time.Now()
	content := &protos.Contents{
		Content: &protos.Contents_Text{
			Text: &protos.Text{Text: text},
		},
	}
	contentBytes, err := protos.EncodeProtoMessage(content)
	if err != nil {
		return "", err
	}
	attemptID, _ := snapcrypto.EncodeUUID(uuid.NewString())
	req := &protos.CreateContentMessageRequest{
		SenderId:           c.selfUUID(),
		ClientResolutionId: randomUint64(),
		Destinations: []*protos.DeliveryDestination{{
			EncryptionInfo: &protos.EncryptionInfo{
				Method: &protos.EncryptionInfo_Fidelius{Fidelius: &protos.Empty{}},
			},
			Destination: &protos.DeliveryDestination_ConversationDestination{
				ConversationDestination: &protos.ConversationDestination{
					ConversationId: conv.GetConversationId(),
					CurrentVersion: conv.GetVersion(),
				},
			},
		}},
		Content: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    contentBytes,
			// Do not force-save bridge-sent texts. Leaving the envelope unset
			// lets Snapchat apply the conversation's normal disappearing policy.
			SavePolicy: protos.ContentEnvelope_SavePolicy_ENVELOPE_UNSET,
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_None{None: &protos.Empty{}},
			},
		},
		CreateContentMessageBlizzardData: &protos.CreateContentMessageBlizzardData{
			SendMessageAttemptId: &protos.UUID{EncodedId: attemptID},
		},
	}
	if replyToMessageID > 0 {
		// Snapchat replies are sent as a feature attachment referencing the
		// quoted message ID (the same shape the web client sends).
		req.FeatureAttachment = []*protos.FeatureAttachment{{
			Attachment: &protos.FeatureAttachment_ReplyMessageInfo{
				ReplyMessageInfo: &protos.ReplyMessageInfo{QuotedMessageId: replyToMessageID},
			},
		}}
	}
	var resp protos.CreateContentMessageResponse
	requestStarted := time.Now()
	zerolog.Ctx(ctx).Info().
		Str("diag", "send_latency").
		Str("stage", "create_content_request").
		Str("chat_id", chatID).
		Int64("conversation_version", conv.GetVersion()).
		Int64("reply_to_message_id", replyToMessageID).
		Dur("since_send_text_start", time.Since(started)).
		Dur("ensure_auth_duration", authDone.Sub(started)).
		Dur("conversation_fresh_duration", conversationDone.Sub(authDone)).
		Msg("send-latency: CreateContentMessage request")
	if err = c.doGRPC(ctx, createContentMessageURL, req, &resp); err != nil {
		return "", err
	}
	responseDone := time.Now()
	for _, result := range resp.GetResult() {
		if !result.GetSuccess() {
			return "", fmt.Errorf("snapchat api send failed")
		}
	}
	c.forgetConversation(chatID)
	if createdID := createdMessageIDFromResponse(&resp); createdID != "" {
		zerolog.Ctx(ctx).Info().
			Str("diag", "send_latency").
			Str("stage", "snapchat_response").
			Str("chat_id", chatID).
			Str("created_message_id", createdID).
			Dur("create_content_duration", responseDone.Sub(requestStarted)).
			Dur("total_send_text_duration", responseDone.Sub(started)).
			Msg("send-latency: Snapchat response")
		return createdID, nil
	}
	zerolog.Ctx(ctx).Info().
		Str("diag", "send_latency").
		Str("stage", "snapchat_response").
		Str("chat_id", chatID).
		Uint64("client_resolution_id", resp.GetClientResolutionId()).
		Dur("create_content_duration", responseDone.Sub(requestStarted)).
		Dur("total_send_text_duration", responseDone.Sub(started)).
		Msg("send-latency: Snapchat response")
	return fmt.Sprintf("%d", resp.GetClientResolutionId()), nil
}

// SendMedia sends one ordinary chat image (EXTERNAL_MEDIA), not a snap. It
// mirrors the proven Snapchat Web upload pipeline: getUploadLocations ->
// AES-256-CBC encrypt -> PUT ciphertext -> CreateContentMessage with the
// content object reference and the clear key/IV embedded in the message
// contents. Only JPEG is supported in this first version.
func (c *Client) SendMedia(ctx context.Context, chatID string, media MediaAttachment, caption string) (string, error) {
	if len(media.Data) == 0 {
		return "", fmt.Errorf("missing media data")
	}
	if len(media.Data) > maxOutgoingMediaSize {
		return "", fmt.Errorf("media is too large for Snapchat send (max %d bytes)", maxOutgoingMediaSize)
	}
	mimeType := strings.ToLower(strings.TrimSpace(strings.Split(media.MimeType, ";")[0]))
	if !isJPEGData(media.Data) || (mimeType != "" && mimeType != "image/jpeg" && mimeType != "image/jpg") {
		return "", fmt.Errorf("outbound Snapchat media supports only image/jpeg in this version (got %q)", media.MimeType)
	}
	width, height, err := imageDimensions(media.Data)
	if err != nil {
		return "", err
	}
	if caption != "" {
		log.Printf("snapapi send: media caption ignored in outbound image v1 chat_id=%s", chatID)
	}
	if err := c.ensureAuthenticated(ctx); err != nil {
		return "", err
	}
	encrypted, err := EncryptOutgoingMedia(media.Data)
	if err != nil {
		return "", fmt.Errorf("encrypt outgoing media: %w", err)
	}
	location, err := c.GetUploadLocations(ctx)
	if err != nil {
		return "", err
	}
	zerolog.Ctx(ctx).Info().
		Str("diag", "outgoing_media").
		Str("chat_id", chatID).
		Int("jpeg_bytes", len(media.Data)).
		Int("cipher_bytes", len(encrypted.Ciphertext)).
		Int("width", width).
		Int("height", height).
		Str("upload_host", uploadHostOf(location.PutURL)).
		Msg("outgoing-media: encrypted payload ready for PUT")
	descriptor, err := c.UploadEncryptedMedia(ctx, location, encrypted)
	if err != nil {
		return "", err
	}
	convID, err := encodeUUIDString(chatID)
	if err != nil {
		return "", err
	}
	attemptID, _ := snapcrypto.EncodeUUID(uuid.NewString())
	req := &protos.CreateContentMessageRequest{
		SenderId:           c.selfUUID(),
		ClientResolutionId: randomUint64(),
		// The proven web capture sends the plain conversation destination
		// (no destination EncryptionInfo) with the placeholder version 111.
		Destinations: []*protos.DeliveryDestination{{
			Destination: &protos.DeliveryDestination_ConversationDestination{
				ConversationDestination: &protos.ConversationDestination{
					ConversationId: convID,
					CurrentVersion: 111,
				},
			},
		}},
		Content: &protos.ContentEnvelope{
			ContentType: protos.ContentType_EXTERNAL_MEDIA,
			Contents:    encodeExternalMediaContents(mediaUploadClock(), width, height, encrypted.Key, encrypted.IV),
			MediaReferenceLists: []*protos.ContentEnvelope_MediaReferenceList{{
				Reference: []*protos.MediaReference{{
					ContentObject: descriptor,
					MediaType:     protos.MediaType_MEDIA_TYPE_IMAGE,
				}},
			}},
			DisplayInfo: &protos.ContentEnvelope_DisplayInfo{},
			// Persistent ordinary chat image: the web capture uses LIFETIME.
			SavePolicy: protos.ContentEnvelope_SavePolicy_LIFETIME,
		},
		CreateContentMessageBlizzardData: &protos.CreateContentMessageBlizzardData{
			SendMessageAttemptId: &protos.UUID{EncodedId: attemptID},
		},
	}
	var resp protos.CreateContentMessageResponse
	if err = c.doGRPC(ctx, paths.CREATE_CONTENT_MESSAGE, req, &resp); err != nil {
		return "", err
	}
	for _, result := range resp.GetResult() {
		if !result.GetSuccess() {
			return "", fmt.Errorf("snapchat api media send failed: %v", result.GetFailureReason())
		}
	}
	createdID := createdMessageIDFromResponse(&resp)
	if createdID != "" {
		zerolog.Ctx(ctx).Info().
			Str("diag", "outgoing_media").
			Str("chat_id", chatID).
			Str("created_message_id", createdID).
			Msg("outgoing-media: CreateContentMessage accepted")
		return createdID, nil
	}
	return fmt.Sprintf("%d", resp.GetClientResolutionId()), nil
}

func uploadHostOf(raw string) string {
	host := ""
	if parsed, err := url.Parse(raw); err == nil {
		host = parsed.Hostname()
	}
	return host
}

func (c *Client) MarkRead(ctx context.Context, chatID string, messageID int64, version int64) error {
	if messageID <= 0 {
		return nil
	}
	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}
	conv, err := c.conversationFresh(ctx, chatID)
	if err != nil {
		return err
	}
	// Reading a chat is a conversation-level watermark update: Snapchat Web
	// uses UpdateConversation with the UpdateConversationRead action (read up
	// to lastMessageId), which is what maintains the participant
	// ReadHighWatermark. UpdateContentMessage with UpdateAction_Read is
	// rejected by the server with UPDATE_NOT_APPLICABLE (verified live), so
	// this must stay a conversation update.
	currentVersion := conv.GetVersion()
	if currentVersion <= 0 {
		currentVersion = version
	}
	if selfReadHighWatermark(conv, c.SelfUserID()) >= messageID {
		zerolog.Ctx(ctx).Debug().Str("diag", "markread_already_current").Str("chat_id", chatID).Int64("last_message_id", messageID).Int64("fresh_conversation_version", conv.GetVersion()).Msg("receipt-diag: Snapchat read watermark already covers target")
		return nil
	}
	zerolog.Ctx(ctx).Debug().Str("diag", "markread_request").Str("chat_id", chatID).Int64("last_message_id", messageID).Int64("fresh_conversation_version", conv.GetVersion()).Int64("fallback_version", version).Int64("used_version", currentVersion).Msg("receipt-diag: UpdateConversation(read) request prepared")
	req := &protos.UpdateConversationRequest{
		ConversationId:     conv.GetConversationId(),
		ClientResolutionId: randomUint64(),
		CurrentVersion:     currentVersion,
		UpdateConversationAction: &protos.UpdateConversationRequest_Read{
			Read: &protos.UpdateConversationRead{
				SelfUserId: c.selfUUID(),
				ReadConversationMessageData: &protos.ReadConversationMessageData{
					LastMessageId: messageID,
				},
			},
		},
	}
	var resp protos.UpdateConversationResponse
	if err = c.doGRPC(ctx, updateConversationURL, req, &resp); err != nil {
		return err
	}
	zerolog.Ctx(ctx).Debug().Str("diag", "markread_response").Str("chat_id", chatID).Int64("last_message_id", messageID).Bool("success", resp.GetUpdateData().GetSuccess()).Int64("response_conversation_version", resp.GetUpdateData().GetCurrentVersion()).Str("failure_reason", resp.GetResult().String()).Msg("receipt-diag: UpdateConversation(read) response")
	if !resp.GetUpdateData().GetSuccess() {
		return fmt.Errorf("snapchat api read receipt failed: %s", resp.GetResult().String())
	}
	return nil
}

func selfReadHighWatermark(conv *protos.Conversation, selfUserID string) int64 {
	if conv == nil || selfUserID == "" {
		return 0
	}
	for _, participant := range conv.GetParticipants() {
		if strings.EqualFold(uuidToString(participant.GetUserId()), selfUserID) {
			return participant.GetReadHighWatermark()
		}
	}
	return 0
}

// EraseMessage unsends/deletes a Snapchat message (UpdateAction_Erase, the
// operation Snapchat Web uses for "unsend"). Like MarkRead it uses the
// conversation's CURRENT version, which Snapchat requires.
//
// A delete is only treated as successful when the server explicitly confirms
// Success=true. Retryable failures are reported as errors (never treated as
// success) so a destructive unsend is never claimed without server commit.
func (c *Client) EraseMessage(ctx context.Context, chatID string, messageID int64) error {
	if messageID <= 0 {
		return nil
	}
	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}
	conv, err := c.conversationFresh(ctx, chatID)
	if err != nil {
		return err
	}
	req := &protos.UpdateContentMessageRequest{
		ClientResolutionId: randomUint64(),
		CurrentVersion:     conv.GetVersion(),
		Update: &protos.UpdateAction{
			MessageId:       messageID,
			SenderId:        c.selfUUID(),
			ConversationId:  conv.GetConversationId(),
			UpdateTimestamp: time.Now().UnixMilli(),
			Update:          &protos.UpdateAction_Erase{Erase: &protos.Erase{}},
		},
	}
	var resp protos.UpdateContentMessageResponse
	if err = c.doGRPC(ctx, updateContentMessageURL, req, &resp); err != nil {
		return err
	}
	eraseResult := resp.GetErase()
	log.Printf("snapapi erase: response chat_id=%s message_id=%d requested_version=%d response_version=%d success=%t retryable=%t erase_result=%t erase_updated_message=%t status_message=%t failure_present=%t failure_type=%s failure_desc=%q",
		chatID, messageID, conv.GetVersion(), resp.GetCurrentVersion(),
		resp.GetSuccess(), resp.GetRetryable(),
		eraseResult != nil, eraseResult != nil && eraseResult.GetUpdatedMessage() != nil,
		resp.GetStatusMessage() != nil,
		resp.GetResult() != nil, resp.GetResult().GetFailureType(), resp.GetResult().GetFailureDescription())
	if !resp.GetSuccess() {
		return fmt.Errorf("snapchat api erase rejected chat_id=%s message_id=%d success=%t retryable=%t failure_type=%s failure_desc=%q",
			chatID, messageID, resp.GetSuccess(), resp.GetRetryable(),
			resp.GetResult().GetFailureType(), resp.GetResult().GetFailureDescription())
	}
	return nil
}
