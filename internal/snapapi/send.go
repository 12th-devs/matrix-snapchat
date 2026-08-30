package snapapi

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	snapcrypto "github.com/0xzer/snapper/crypto"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
	"github.com/google/uuid"
)

func (c *Client) SendText(ctx context.Context, chatID, text string, replyToMessageID int64) (string, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return "", err
	}
	conv, err := c.conversationFresh(ctx, chatID)
	if err != nil {
		return "", err
	}
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
	if err = c.doGRPC(ctx, createContentMessageURL, req, &resp); err != nil {
		return "", err
	}
	for _, result := range resp.GetResult() {
		if !result.GetSuccess() {
			return "", fmt.Errorf("snapchat api send failed")
		}
	}
	c.forgetConversation(chatID)
	if createdID := createdMessageIDFromResponse(&resp); createdID != "" {
		return createdID, nil
	}
	return fmt.Sprintf("%d", resp.GetClientResolutionId()), nil
}

func (c *Client) SendMedia(ctx context.Context, chatID string, media MediaAttachment, caption string) (string, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return "", err
	}
	if len(media.Data) == 0 {
		return "", fmt.Errorf("missing media data")
	}
	if len(media.Data) > 16*1024*1024 {
		return "", fmt.Errorf("media is too large for safe inline Snapchat API send")
	}
	remoteType, hasAudio, err := outgoingRemoteMediaType(media.MimeType, media.Data)
	if err != nil {
		return "", err
	}
	conv, err := c.conversationFresh(ctx, chatID)
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
			ContentType: protos.ContentType_SNAP,
			RemoteMediaInfos: []*protos.ContentEnvelope_RemoteMediaInfo{{
				MediaInfo: &protos.ContentEnvelope_RemoteMediaInfo_ContentObject{
					ContentObject: append([]byte(nil), media.Data...),
				},
				MediaType: int32(remoteType),
				HasAudio:  hasAudio,
			}},
			SavePolicy: protos.ContentEnvelope_SavePolicy_PROHIBITED,
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_None{None: &protos.Empty{}},
			},
			FeedDisplayInfo: &protos.ContentEnvelope_FeedDisplayInfo{
				FeedDisplayInfo: &protos.ContentEnvelope_FeedDisplayInfo_SnapDisplayInfo{
					SnapDisplayInfo: &protos.SnapDisplayInfo{HasAudio: hasAudio},
				},
			},
		},
		FeatureAttachment: []*protos.FeatureAttachment{{
			Attachment: &protos.FeatureAttachment_SnapViewability{
				SnapViewability: &protos.SnapViewability{
					SnapPostOpenViewingPolicy: protos.SnapPostOpenViewingPolicy_POLICY_MEDIA,
				},
			},
		}},
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
	c.forgetConversation(chatID)
	createdID := createdMessageIDFromResponse(&resp)
	if createdID != "" {
		return createdID, nil
	}
	return fmt.Sprintf("%d", resp.GetClientResolutionId()), nil
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
	// UpdateContentMessage requires the conversation's CURRENT version. The
	// version carried by receipt metadata was captured when the message was
	// received and is stale by read time, which made Snapchat reject the read
	// update (success=false). Prefer the version from the freshly synced
	// conversation; the receipt version is only a fallback.
	currentVersion := conv.GetVersion()
	if currentVersion <= 0 {
		currentVersion = version
	}
	req := &protos.UpdateContentMessageRequest{
		ClientResolutionId: randomUint64(),
		CurrentVersion:     currentVersion,
		Update: &protos.UpdateAction{
			MessageId:       messageID,
			SenderId:        c.selfUUID(),
			ConversationId:  conv.GetConversationId(),
			UpdateTimestamp: time.Now().UnixMilli(),
			Update:          &protos.UpdateAction_Read{Read: &protos.Read{}},
		},
	}
	var resp protos.UpdateContentMessageResponse
	if err = c.doGRPC(ctx, updateContentMessageURL, req, &resp); err != nil {
		return err
	}
	if !resp.GetSuccess() && !resp.GetRetryable() {
		return fmt.Errorf("snapchat api read receipt failed")
	}
	return nil
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

func outgoingRemoteMediaType(mimeType string, data []byte) (protos.ContentEnvelope_RemoteMediaInfo_MediaType, bool, error) {
	mimeType = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0]))
	}
	switch {
	case mimeType == "image/gif":
		return protos.ContentEnvelope_RemoteMediaInfo_MediaType_GIF, false, nil
	case strings.HasPrefix(mimeType, "image/"):
		return protos.ContentEnvelope_RemoteMediaInfo_MediaType_IMAGE, false, nil
	case strings.HasPrefix(mimeType, "video/"):
		return protos.ContentEnvelope_RemoteMediaInfo_MediaType_VIDEO, true, nil
	default:
		return protos.ContentEnvelope_RemoteMediaInfo_MediaType_MEDIATYPE_UNKNOWN, false, fmt.Errorf("unsupported Snapchat media type %q", mimeType)
	}
}
