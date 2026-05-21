package snapapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/0xzer/snapper/protos"
)

func (c *Client) DownloadMedia(ctx context.Context, media MediaAttachment) ([]byte, string, error) {
	var data []byte
	contentType := strings.TrimSpace(media.MimeType)
	if len(media.Data) > 0 {
		data = append([]byte(nil), media.Data...)
	} else if strings.TrimSpace(media.URL) != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, media.URL, nil)
		if err != nil {
			return nil, "", err
		}
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, "", fmt.Errorf("download Snapchat media returned %s", resp.Status)
		}
		if contentType == "" {
			contentType = strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, 64*1024*1024))
		if err != nil {
			return nil, "", err
		}
	} else {
		return nil, "", fmt.Errorf("Snapchat media has no URL or inline data")
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("Snapchat media was empty")
	}
	if decrypted, ok := decryptMediaCBC(data, media.Key, media.IV); ok {
		data = decrypted
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectMediaMime(data, media.Kind)
	}
	return data, contentType, nil
}

func mediaAttachmentsFromEnvelope(envelope *protos.ContentEnvelope, messageID string) []MediaAttachment {
	if envelope == nil {
		return nil
	}
	key, iv := mediaKeyFromEnvelope(envelope)
	media := make([]MediaAttachment, 0)
	for index, info := range envelope.GetRemoteMediaInfos() {
		kind, mimeType := mediaKindFromRemote(info.GetMediaType(), info.GetHasAudio())
		attachment := MediaAttachment{
			ID:       stableMediaID(messageID, "remote", fmt.Sprint(index), info.GetContentUrl(), info.GetLegacyMediaId()),
			URL:      strings.TrimSpace(info.GetContentUrl()),
			FileName: mediaFileName(messageID, index, kind, mimeType),
			MimeType: mimeType,
			Kind:     kind,
			Data:     append([]byte(nil), info.GetContentObject()...),
			Key:      key,
			IV:       iv,
		}
		if attachment.URL != "" || len(attachment.Data) > 0 {
			media = append(media, attachment)
		}
	}
	for listIndex, list := range envelope.GetMediaReferenceLists() {
		for refIndex, ref := range list.GetReference() {
			kind, mimeType := mediaKindFromReference(ref.GetMediaType())
			attachment := MediaAttachment{
				ID:       stableMediaID(messageID, "ref", fmt.Sprint(listIndex), fmt.Sprint(refIndex), ref.GetUrl(), fmt.Sprint(ref.GetMediaListId())),
				URL:      strings.TrimSpace(ref.GetUrl()),
				FileName: mediaFileName(messageID, len(media), kind, mimeType),
				MimeType: mimeType,
				Kind:     kind,
				Data:     append([]byte(nil), ref.GetContentObject()...),
				Key:      key,
				IV:       iv,
			}
			if attachment.URL != "" || len(attachment.Data) > 0 {
				media = append(media, attachment)
			}
		}
	}
	return media
}

func mediaKeyFromEnvelope(envelope *protos.ContentEnvelope) ([]byte, []byte) {
	if envelope == nil {
		return nil, nil
	}
	clear := envelope.GetEnvelopeEncryption().GetClearTextMediaKey()
	if clear == nil {
		return nil, nil
	}
	return append([]byte(nil), clear.GetMediaKey()...), append([]byte(nil), clear.GetMediaIv()...)
}

func mediaKindFromRemote(raw int32, hasAudio bool) (MediaKind, string) {
	switch protos.ContentEnvelope_RemoteMediaInfo_MediaType(raw) {
	case protos.ContentEnvelope_RemoteMediaInfo_MediaType_IMAGE:
		return MediaKindImage, "image/jpeg"
	case protos.ContentEnvelope_RemoteMediaInfo_MediaType_VIDEO:
		return MediaKindVideo, "video/mp4"
	case protos.ContentEnvelope_RemoteMediaInfo_MediaType_GIF:
		return MediaKindGIF, "image/gif"
	default:
		if hasAudio {
			return MediaKindVideo, "video/mp4"
		}
		return MediaKindFile, "application/octet-stream"
	}
}

func mediaKindFromReference(raw protos.MediaType) (MediaKind, string) {
	switch raw {
	case protos.MediaType_MEDIA_TYPE_IMAGE:
		return MediaKindImage, "image/jpeg"
	case protos.MediaType_MEDIA_TYPE_VIDEO, protos.MediaType_MEDIA_TYPE_VIDEONOAUDIO:
		return MediaKindVideo, "video/mp4"
	case protos.MediaType_MEDIA_TYPE_ANIMATEDIMAGE:
		return MediaKindGIF, "image/gif"
	case protos.MediaType_MEDIA_TYPE_AUDIO:
		return MediaKindFile, "audio/mpeg"
	default:
		return MediaKindFile, "application/octet-stream"
	}
}

func mediaFileName(messageID string, index int, kind MediaKind, mimeType string) string {
	ext := ".bin"
	switch {
	case strings.Contains(mimeType, "jpeg"):
		ext = ".jpg"
	case strings.Contains(mimeType, "png"):
		ext = ".png"
	case strings.Contains(mimeType, "gif"):
		ext = ".gif"
	case strings.Contains(mimeType, "mp4"):
		ext = ".mp4"
	case strings.Contains(mimeType, "mpeg"):
		ext = ".mp3"
	case kind == MediaKindImage:
		ext = ".jpg"
	case kind == MediaKindVideo:
		ext = ".mp4"
	}
	return fmt.Sprintf("snap-%s-%d%s", sanitizeFileNamePart(messageID), index+1, ext)
}

func sanitizeFileNamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "media"
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "media"
	}
	return builder.String()
}

func stableMediaID(parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "::")))
	return hex.EncodeToString(hash[:])[:16]
}

func detectMediaMime(data []byte, kind MediaKind) string {
	detected := http.DetectContentType(data)
	if detected != "application/octet-stream" {
		return detected
	}
	switch kind {
	case MediaKindImage:
		return "image/jpeg"
	case MediaKindVideo:
		return "video/mp4"
	case MediaKindGIF:
		return "image/gif"
	default:
		return detected
	}
}

func decryptMediaCBC(data, key, iv []byte) ([]byte, bool) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, false
	}
	if len(iv) < aes.BlockSize || len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, false
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, false
	}
	output := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv[:aes.BlockSize]).CryptBlocks(output, data)
	output, ok := pkcs7Unpad(output, aes.BlockSize)
	if !ok {
		return nil, false
	}
	return output, true
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, bool) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, false
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, false
	}
	for _, value := range data[len(data)-pad:] {
		if int(value) != pad {
			return nil, false
		}
	}
	return data[:len(data)-pad], true
}

func previewFromDisplayInfo(info *protos.DisplayInfo) (string, bool) {
	if info == nil {
		return "", false
	}
	if snap := info.GetSnapItem(); snap != nil {
		if snap.GetState() == protos.SnapItemState_VIEWED {
			return "Opened Snap", false
		}
		if snap.GetHasAudio() {
			return "New Snap with audio", true
		}
		return "New Snap", snap.GetState() != protos.SnapItemState_VIEWED
	}
	if chat := info.GetChatItem(); chat != nil {
		switch chat.GetState() {
		case protos.ChatItemState_CHAT_UNVIEWED, protos.ChatItemState_CHAT_SAVED_UNVIEWED,
			protos.ChatItemState_CHAT_SCREENSHOTTED_UNVIEWED, protos.ChatItemState_CHAT_RECORDED_UNVIEWED:
			return "New message", true
		case protos.ChatItemState_CHAT_VIEWED, protos.ChatItemState_CHAT_SAVED_VIEWED:
			return "Chat", false
		default:
			return chat.GetState().String(), false
		}
	}
	if call := info.GetCallItem(); call != nil {
		if call.GetIsVideo() {
			return "Missed video call", true
		}
		return "Missed call", true
	}
	if conv := info.GetConversationItem(); conv != nil {
		return conv.GetState().String(), false
	}
	return "", false
}
