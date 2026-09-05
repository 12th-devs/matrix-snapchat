package snapapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"

	"github.com/0xzer/snapper/protos"
	"google.golang.org/protobuf/encoding/protowire"
)

type MediaDownloadInfo struct {
	Source      string
	DecryptPath string
	Bytes       int
	ContentType string
	MediaID string
	DescriptorShape string
	CiphertextBytes int
}

// MediaDescriptor shapes observed in live Snapchat Web envelopes. The opaque
// payload is a serialized v2 ContentObject (bundle definitions rt/je):
//   - field1-direct: field 1 = contentObjectId (ASCII) — live 61-byte capture.
//   - field2-nested: field 2 = contentDescriptor whose field 2 = contentId
//     (ASCII), plus readyLocationIds/claimId/useCase/hostPatternVersion —
//     live 34-byte capture from message 4114.
//
// Both are opaque references that must be resolved through
// snapchat.content.v2.MediaDeliveryService/resolveContentObjects; they must
// never be treated as media bytes.
type MediaDescriptor struct {
	Detected        bool
	Shape           string
	ContentObjectID string
}

const (
	mediaDescriptorShapeField1Direct = "field1-direct"
	mediaDescriptorShapeField2Nested = "field2-nested"
)

const (
	maxMediaDescriptorBytes     = 512
	minMediaDescriptorIDBytes   = 8
	maxMediaDescriptorIDBytes   = 96
	maxMediaDescriptorFieldNum  = 15
)

const (
	minUnsniffedRenderableMediaBytes = 128
)

// MediaDescriptorID reports the Snapchat content object ID when the inline
// payload is not renderable media but a small protobuf descriptor that must be
// resolved through the media delivery service. Snapchat web sends these
// ~34-61 byte descriptors for chat media instead of bytes or URLs.
func MediaDescriptorID(data []byte) string {
	return ParseMediaDescriptor(data).ContentObjectID
}

func descriptorIDFromProtoBytes(value []byte) bool {
	if len(value) < minMediaDescriptorIDBytes || len(value) > maxMediaDescriptorIDBytes {
		return false
	}
	for _, b := range value {
		if b < 0x21 || b > 0x7e {
			return false
		}
	}
	return true
}

// parseProtoFields walks one protobuf message, invoking seen for every field
// until the buffer is exhausted. It rejects malformed or implausible data so
// real media bytes are never mistaken for descriptors.
func parseProtoFields(data []byte, seen func(field protowire.Number, kind protowire.Type, value []byte) error) error {
	for len(data) > 0 {
		number, kind, n := protowire.ConsumeTag(data)
		if n < 0 || number == 0 || number > maxMediaDescriptorFieldNum {
			return fmt.Errorf("invalid descriptor protobuf tag")
		}
		data = data[n:]
		var value []byte
		switch kind {
		case protowire.VarintType:
			_, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("invalid descriptor protobuf varint")
			}
			data = data[n:]
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("invalid descriptor protobuf bytes")
			}
			value, data = v, data[n:]
		default:
			return fmt.Errorf("unsupported descriptor protobuf wire type")
		}
		if err := seen(number, kind, value); err != nil {
			return err
		}
	}
	return nil
}

// ParseMediaDescriptor recognizes the descriptor shapes actually observed on
// the wire. Raw descriptor bytes stay the source of truth for the resolver;
// the extracted ID is informational only.
func ParseMediaDescriptor(data []byte) MediaDescriptor {
	if len(data) < 2+minMediaDescriptorIDBytes || len(data) > maxMediaDescriptorBytes {
		return MediaDescriptor{}
	}
	// Shape A: field 1 holds the ASCII content object ID directly; the live
	// capture carries a nested auxiliary field after it.
	if data[0] == 0x0a {
		if length, offset, ok := readProtoVarint(data, 1); ok &&
			int(length) <= len(data)-offset &&
			descriptorIDFromProtoBytes(data[offset:offset+int(length)]) {
			rest := data[offset+int(length):]
			if len(rest) == 0 || parseProtoFields(rest, func(protowire.Number, protowire.Type, []byte) error { return nil }) == nil {
				return MediaDescriptor{
					Detected:        true,
					Shape:           mediaDescriptorShapeField1Direct,
					ContentObjectID: string(data[offset : offset+int(length)]),
				}
			}
		}
	}
	// Shape B: field 2 wraps an inner protobuf whose field 2 holds the ASCII
	// content object ID (with small auxiliary fields alongside).
	if data[0] == 0x12 {
		if length, offset, ok := readProtoVarint(data, 1); ok && int(length) == len(data)-offset {
			inner := data[offset : offset+int(length)]
			var candidate []byte
			err := parseProtoFields(inner, func(field protowire.Number, kind protowire.Type, value []byte) error {
				if field == 2 && kind == protowire.BytesType && descriptorIDFromProtoBytes(value) {
					if candidate != nil {
						return fmt.Errorf("ambiguous descriptor content object ID")
					}
					candidate = value
				}
				return nil
			})
			if err == nil && candidate != nil {
				return MediaDescriptor{
					Detected:        true,
					Shape:           mediaDescriptorShapeField2Nested,
					ContentObjectID: string(candidate),
				}
			}
		}
	}
	return MediaDescriptor{}
}

func (c *Client) DownloadMedia(ctx context.Context, media MediaAttachment) ([]byte, string, error) {
	data, contentType, _, err := c.DownloadMediaWithInfo(ctx, media)
	return data, contentType, err
}

func (c *Client) DownloadMediaWithInfo(ctx context.Context, media MediaAttachment) ([]byte, string, MediaDownloadInfo, error) {
	info := MediaDownloadInfo{DecryptPath: "none"}
	descriptor := ParseMediaDescriptor(media.Data)
	if descriptor.Detected && strings.TrimSpace(media.URL) == "" {
		info.Source = "descriptor-rpc"
		info.MediaID = descriptor.ContentObjectID
		info.DescriptorShape = descriptor.Shape
		urls, err := c.ResolveMediaDescriptor(ctx, media.Data)
		if err != nil { return nil, "", info, err }
		for _, mediaURL := range urls {
			resolved := media
			resolved.Data = nil
			resolved.URL = mediaURL
			data, mime, downloaded, err := c.DownloadMediaWithInfo(ctx, resolved)
			if err != nil { continue }
			downloaded.Source = info.Source
			downloaded.MediaID = descriptor.ContentObjectID
			downloaded.DescriptorShape = descriptor.Shape
			return data, mime, downloaded, nil
		}
		return nil, "", info, fmt.Errorf("all resolved Snapchat media downloads failed validation or retrieval")
	}
	var data []byte
	contentType := strings.TrimSpace(media.MimeType)
	if len(media.Data) > 0 && !descriptor.Detected {
		info.Source = "inline"
		data = append([]byte(nil), media.Data...)
	} else if strings.TrimSpace(media.URL) != "" {
		info.Source = "url"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, media.URL, nil)
		if err != nil {
			return nil, "", info, err
		}
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, "", info, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, "", info, fmt.Errorf("download Snapchat media returned %s", resp.Status)
		}
		if contentType == "" {
			contentType = strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, 64*1024*1024+1))
		if err != nil {
			return nil, "", info, err
		}
		if len(data) > 64*1024*1024 {
			return nil, "", info, fmt.Errorf("Snapchat media exceeds 64 MiB download limit")
		}
	} else {
		return nil, "", info, fmt.Errorf("Snapchat media has no URL or inline data")
	}
	if len(data) == 0 {
		return nil, "", info, fmt.Errorf("Snapchat media was empty")
	}
	if len(media.Key) > 0 || len(media.IV) > 0 {
		info.CiphertextBytes = len(data)
		decrypted, ok := decryptMediaCBC(data, media.Key, media.IV)
		if !ok {
			return nil, "", info, fmt.Errorf("Snapchat media clear-key decryption failed")
		}
		data = decrypted
		info.DecryptPath = "cbc-clear-media-key"
	}
	if err := ValidateRenderableMediaPayload(data, contentType, media.Kind); err != nil {
		return nil, "", info, err
	}
	if detected := mediaMagicMime(data); detected != "" {
		contentType = detected
	} else if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectMediaMime(data, media.Kind)
	}
	info.Bytes = len(data)
	info.ContentType = contentType
	return data, contentType, info, nil
}

func ValidateRenderableMediaPayload(data []byte, mimeType string, kind MediaKind) error {
	if len(data) == 0 {
		return fmt.Errorf("Snapchat media was empty")
	}
	if id := MediaDescriptorID(data); id != "" {
		return fmt.Errorf("snapchat media is a descriptor reference (media_id=%s) with no fetchable URL; downloading it requires the media service integration", id)
	}
	detected := mediaMagicMime(data)
	if detected == "image/jpeg" || detected == "image/png" || detected == "image/gif" {
		config, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || config.Width <= 0 || config.Height <= 0 {
			return fmt.Errorf("invalid Snapchat image structure: %v", err)
		}
		if int64(config.Width)*int64(config.Height) > 40_000_000 {
			return fmt.Errorf("Snapchat image exceeds 40 megapixel validation limit")
		}
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("invalid Snapchat image data: %w", err)
		}
		return nil
	}
	if detected == "video/mp4" {
		return validateMP4(data)
	}
	mimeType = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	if len(data) < minUnsniffedRenderableMediaBytes && (kind == MediaKindImage || kind == MediaKindVideo || kind == MediaKindGIF || strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/")) {
		return fmt.Errorf("snapchat media payload is too small and lacks image/video magic bytes (bytes=%d mime=%q kind=%s)", len(data), mimeType, kind)
	}
	if kind == MediaKindImage || kind == MediaKindGIF || kind == MediaKindVideo || strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/") {
		return fmt.Errorf("Snapchat media lacks a supported, valid image/video structure (bytes=%d)", len(data))
	}
	return nil
}

// Reject header-only and truncated ISO BMFF payloads. Track metadata and actual
// media data independently: an ftyp signature alone is not a playable video.
func validateMP4(data []byte) error {
	var metadata, payload bool
	for offset := 0; offset < len(data); {
		if len(data)-offset < 8 {
			return fmt.Errorf("truncated MP4 box header")
		}
		size := uint64(binary.BigEndian.Uint32(data[offset:]))
		kind := string(data[offset+4 : offset+8])
		header := uint64(8)
		if size == 1 {
			if len(data)-offset < 16 {
				return fmt.Errorf("truncated MP4 extended box")
			}
			size = binary.BigEndian.Uint64(data[offset+8:])
			header = 16
		} else if size == 0 {
			size = uint64(len(data) - offset)
		}
		if size < header || size > uint64(len(data)-offset) {
			return fmt.Errorf("invalid MP4 box length")
		}
		metadata = metadata || (kind == "moov" && size > header)
		payload = payload || (kind == "mdat" && size > header)
		offset += int(size)
	}
	if !metadata || !payload {
		return fmt.Errorf("MP4 missing movie metadata or media payload")
	}
	return nil
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
	if mimeType := mediaMagicMime(data); mimeType != "" {
		return mimeType
	}
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

func mediaMagicMime(data []byte) string {
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "image/jpeg"
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return "image/gif"
	case len(data) >= 16 && string(data[4:8]) == "ftyp":
		return "video/mp4"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE":
		return "audio/wav"
	case len(data) >= 3 && string(data[:3]) == "ID3":
		return "audio/mpeg"
	case len(data) >= 2 && data[0] == 0xff && (data[1] == 0xfb || data[1] == 0xf3 || data[1] == 0xf2):
		return "audio/mpeg"
	default:
		return ""
	}
}

func decryptMediaCBC(data, key, iv []byte) ([]byte, bool) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, false
	}
	if len(iv) != aes.BlockSize || len(data) == 0 || len(data)%aes.BlockSize != 0 {
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
