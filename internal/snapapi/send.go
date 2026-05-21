package snapapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	snapcrypto "github.com/0xzer/snapper/crypto"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
	"github.com/google/uuid"
)

func (c *Client) SendText(ctx context.Context, chatID, text string) (string, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return "", err
	}
	conv, err := c.conversation(ctx, chatID)
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
	var resp protos.CreateContentMessageResponse
	if err = c.doGRPC(ctx, paths.CREATE_CONTENT_MESSAGE, req, &resp); err != nil {
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
	conv, err := c.conversation(ctx, chatID)
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
	conv, err := c.conversation(ctx, chatID)
	if err != nil {
		return err
	}
	if version <= 0 {
		version = conv.GetVersion()
	}
	req := &protos.UpdateContentMessageRequest{
		ClientResolutionId: randomUint64(),
		CurrentVersion:     version,
		Update: &protos.UpdateAction{
			MessageId:       messageID,
			SenderId:        c.selfUUID(),
			ConversationId:  conv.GetConversationId(),
			UpdateTimestamp: time.Now().UnixMilli(),
			Update:          &protos.UpdateAction_Read{Read: &protos.Read{}},
		},
	}
	var resp protos.UpdateContentMessageResponse
	if err = c.doGRPC(ctx, paths.UPDATE_CONTENT_MESSAGE, req, &resp); err != nil {
		return err
	}
	if !resp.GetSuccess() && !resp.GetRetryable() {
		return fmt.Errorf("snapchat api read receipt failed")
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
