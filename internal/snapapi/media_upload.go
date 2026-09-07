package snapapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0xzer/snapper/data/headers"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
	"google.golang.org/protobuf/encoding/protowire"
)

// Outgoing ordinary chat image upload, mirroring the proven Snapchat Web
// pipeline (documented reference: snapcap-native src/api/_media_send.ts and
// the live web bundle's MediaDeliveryService client):
//
//  1. MediaDeliveryService/getUploadLocations -> pre-signed PUT URL plus a
//     media content object (token + base id).
//  2. AES-256-CBC encrypt the clear bytes with a fresh random key/IV and PUT
//     the ciphertext to the pre-signed URL.
//  3. Reference the content object from CreateContentMessage with the key/IV
//     embedded in the message contents (EXTERNAL_MEDIA).
//
// The gRPC-web plumbing (framing, cookies, messenger headers) is shared with
// the rest of this client; only the content-service call below needs the
// explicit cross-site browser headers, exactly like ResolveMediaDescriptor.

const (
	maxOutgoingMediaSize = 16 * 1024 * 1024
	// maxOutgoingImageDimension guards the full-image validation decode
	// against decompression bombs (a small PNG can expand to gigabytes).
	maxOutgoingImageDimension = 10000
)

// mediaUploadURL is the content-service RPC the official web client calls for
// upload locations; not a constant because paths.WEB_BASE_URL is a var.
func mediaUploadURL() string {
	return paths.WEB_BASE_URL + "/snapchat.content.v2.MediaDeliveryService/getUploadLocations"
}

// UploadLocation is step 1 of the upload pipeline: the pre-signed PUT URL and
// the two media ids the message must reference.
type UploadLocation struct {
	// PutURL is the pre-signed HTTPS PUT URL (x-amz-acl already in the query
	// string; never send it as a header or the AWS4 signature breaks).
	PutURL string
	// MediaIDToken is the contentObjectId with the _<index> suffix.
	MediaIDToken string
	// MediaIDBase is the nested ContentDescriptor contentId without suffix.
	MediaIDBase string
}

// encodeGetUploadLocationsRequest is the proven minimal request shape:
// batchSize=1 (field 2), contentReferenceResultOption=CONTENT_OBJECT_ONLY
// (field 4), packed uploadUrlTypes=[DIRECT] (field 7 bytes {0}), and
// useZeroRatingDomainForGcsLocations=true (field 16).
func encodeGetUploadLocationsRequest() []byte {
	return bytes.Join([][]byte{
		mediaProtoVarint(2, 1),
		mediaProtoVarint(4, 1),
		mediaProtoBytes(7, []byte{0}),
		mediaProtoVarint(16, 1),
	}, nil)
}

// GetUploadLocations asks Snapchat for one DIRECT upload location.
func (c *Client) GetUploadLocations(ctx context.Context) (*UploadLocation, error) {
	body := encodeGetUploadLocationsRequest()
	framed := frameGRPC(body)
	header := headers.NewCoreHeaders(c.cookies, c.tokens, c.device)
	c.applyUserAgentHeaders(header)
	c.applyBrowserGRPCHeaders(header)
	header.Set("origin", "https://www.snapchat.com")
	header.Set("referer", "https://www.snapchat.com/")
	header.Set("sec-fetch-dest", "empty")
	header.Set("sec-fetch-mode", "cors")
	header.Set("sec-fetch-site", "cross-site")
	resp, meta, err := c.doHTTPDetailed(ctx, mediaUploadURL(), http.MethodPost, header, framed)
	if err != nil {
		return nil, fmt.Errorf("getUploadLocations request framed_len=%d payload_len=%d: %w (%s)", len(framed), len(body), err, meta)
	}
	payload, err := mediaGRPCPayload(resp)
	if err != nil {
		return nil, fmt.Errorf("%w (response frames: %s; rpc %s)", err, mediaRPCFrameDiagnostics(resp), mediaUploadURL())
	}
	return parseUploadLocations(payload)
}

func parseUploadLocations(payload []byte) (*UploadLocation, error) {
	locations, err := mediaProtoField(payload, 1)
	if err != nil {
		return nil, err
	}
	if len(locations) != 1 {
		return nil, fmt.Errorf("getUploadLocations returned %d upload locations", len(locations))
	}
	putURLs, err := mediaProtoField(locations[0], 1)
	if err != nil || len(putURLs) != 1 {
		return nil, fmt.Errorf("getUploadLocations response missing uploadUrl")
	}
	putURL := string(putURLs[0])
	parsed, err := url.Parse(putURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("getUploadLocations returned an invalid upload URL")
	}
	references, err := mediaProtoField(locations[0], 4)
	if err != nil || len(references) != 1 {
		return nil, fmt.Errorf("getUploadLocations response missing contentReference")
	}
	objects, err := mediaProtoField(references[0], 3)
	if err != nil || len(objects) != 1 {
		return nil, fmt.Errorf("getUploadLocations response missing contentObject")
	}
	tokens, err := mediaProtoField(objects[0], 1)
	if err != nil || len(tokens) != 1 {
		return nil, fmt.Errorf("getUploadLocations contentObject missing media id token")
	}
	descriptors, err := mediaProtoField(objects[0], 2)
	if err != nil || len(descriptors) != 1 {
		return nil, fmt.Errorf("getUploadLocations contentObject missing content descriptor")
	}
	bases, err := mediaProtoField(descriptors[0], 2)
	if err != nil || len(bases) != 1 {
		return nil, fmt.Errorf("getUploadLocations content descriptor missing base media id")
	}
	token, base := string(tokens[0]), string(bases[0])
	if token == "" || base == "" || !printableProtoText(tokens[0]) || !printableProtoText(bases[0]) {
		return nil, fmt.Errorf("getUploadLocations returned implausible media ids")
	}
	return &UploadLocation{PutURL: putURL, MediaIDToken: token, MediaIDBase: base}, nil
}

// EncryptedMedia is the result of encrypting clear media bytes for upload.
type EncryptedMedia struct {
	Ciphertext []byte
	Key        []byte
	IV         []byte
}

// EncryptOutgoingMedia encrypts clear bytes with a fresh random AES-256-CBC
// key (32 bytes) and IV (16 bytes), the media-at-rest layer Snapchat Web uses
// for ordinary chat images. Recipients recover the key/IV from the message
// contents, so no envelope-level encryption is involved.
func EncryptOutgoingMedia(plain []byte) (*EncryptedMedia, error) {
	key := make([]byte, 32)
	iv := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(plain, block.BlockSize())
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, padded)
	return &EncryptedMedia{Ciphertext: encrypted, Key: key, IV: iv}, nil
}

// UploadEncryptedMedia PUTs ciphertext to the pre-signed URL and returns the
// canonical content object descriptor the CreateContentMessage must carry.
func (c *Client) UploadEncryptedMedia(ctx context.Context, location *UploadLocation, encrypted *EncryptedMedia) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, location.PutURL, bytes.NewReader(encrypted.Ciphertext))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/octet-stream")
	req.Header.Set("origin", "https://www.snapchat.com")
	req.Header.Set("referer", "https://www.snapchat.com/")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("media upload PUT failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyPrefix, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return nil, fmt.Errorf("media upload PUT returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyPrefix)))
	}
	return encodeOutgoingContentObject(location), nil
}

// encodeOutgoingContentObject rebuilds the v2 ContentObject descriptor the web
// client attaches to outgoing media references: field 1 = contentObjectId
// (token), field 2 = ContentDescriptor{contentId (base), readyLocationIds [4],
// claimId 1, useCase 11 (image), hostPatternVersion 1}. These constants are
// part of the proven wire shape for image uploads.
func encodeOutgoingContentObject(location *UploadLocation) []byte {
	descriptor := bytes.Join([][]byte{
		mediaProtoString(2, location.MediaIDBase),
		mediaProtoBytes(6, []byte{0x04}),
		mediaProtoVarint(9, 1),
		mediaProtoVarint(10, 11),
		mediaProtoVarint(12, 1),
	}, nil)
	return bytes.Join([][]byte{
		mediaProtoString(1, location.MediaIDToken),
		mediaProtoBytes(2, descriptor),
	}, nil)
}

// encodeExternalMediaContents encodes the Contents payload for one ordinary
// chat image or MP4 video. Shape (Contents -> externalMedia(3) -> media(3))
// recovered from the official web client and verified against this bridge's
// own incoming ExternalMediaEncryptionKeys extractor:
// Contents.3.3.5.1.1.1 carries the MediaMetadata with type (field 2, video
// only: META_DATA_MEDIA_TYPE_VIDEO=1; images omit it since 0=IMAGE is the
// default), dimensions (field 5), hasSound (field 12, video only when the
// track has audio), encryptionInfoV1 (field 4, base64) and encryptionInfoV2
// (field 19, raw). Both must carry the SAME key/IV. The video fields mirror
// the web bundle's uploadMedia pipeline (mediaType = VIDEO/VIDEONOAUDIO by
// hasAudio; metadata.type = VIDEO; hasSound set from the audio handler).
func encodeExternalMediaContents(nowMillis int64, info *OutgoingMediaInfo, key, iv []byte) []byte {
	isVideo := info.MediaType == protos.MediaType_MEDIA_TYPE_VIDEO || info.MediaType == protos.MediaType_MEDIA_TYPE_VIDEONOAUDIO
	metadataFields := [][]byte{}
	if isVideo {
		// The proven image capture never carries these (0/false default out);
		// they mirror the web composer's video metadata (metadata.type=VIDEO).
		metadataFields = append(metadataFields, mediaProtoVarint(2, 1))
	}
	metadataFields = append(metadataFields,
		mediaProtoBytes(5, bytes.Join([][]byte{
			mediaProtoVarint(1, uint64(info.Width)),
			mediaProtoVarint(2, uint64(info.Height)),
		}, nil)),
	)
	if isVideo {
		// Web composer (ce): mediaDurationMs from the measured video duration
		// (omitted when unknown) and MediaId{mediaListId:1} tagging the
		// 1-based list position.
		if info.DurationMs > 0 {
			metadataFields = append(metadataFields, mediaProtoVarint(15, uint64(info.DurationMs)))
		}
		metadataFields = append(metadataFields, mediaProtoBytes(18, mediaProtoVarint(1, 1)))
	} else {
		metadataFields = append(metadataFields, mediaProtoBytes(18, nil))
	}
	metadataFields = append(metadataFields,
		mediaProtoBytes(4, bytes.Join([][]byte{
			mediaProtoString(1, base64.StdEncoding.EncodeToString(key)),
			mediaProtoString(2, base64.StdEncoding.EncodeToString(iv)),
		}, nil)),
		mediaProtoBytes(19, bytes.Join([][]byte{
			mediaProtoBytes(1, key),
			mediaProtoBytes(2, iv),
		}, nil)),
		mediaProtoVarint(20, 3),
		mediaProtoVarint(22, 1),
	)
	if isVideo && info.HasAudio {
		metadataFields = append(metadataFields, mediaProtoVarint(12, 1))
	}
	metadata := bytes.Join(metadataFields, nil)
	// Playback.playbackCharacteristics (item field 5.2). The proven image
	// capture carries playOnce(7) + empty seekPointsMs(9) and MUST stay
	// byte-identical. For video the web composer (ce) sends playOnce(7) with
	// hasSound(5) set only when the track carries audio — the recipient
	// derives audio playback from this structure (viewer:
	// mediaMetadata.hasSound || playbackCharacteristics.hasSound).
	characteristicsFields := [][]byte{
		mediaProtoBytes(7, nil),
		mediaProtoBytes(9, nil),
	}
	if isVideo {
		characteristicsFields = [][]byte{
			mediaProtoBytes(7, nil),
		}
		if info.HasAudio {
			characteristicsFields = append(characteristicsFields, mediaProtoVarint(5, 1))
		}
	}
	characteristics := bytes.Join(characteristicsFields, nil)
	mediaWrapper := bytes.Join([][]byte{
		mediaProtoVarint(8, 2),
		mediaProtoBytes(4, bytes.Join([][]byte{
			mediaProtoVarint(6, 1),
			mediaProtoString(10, "1"),
			mediaProtoVarint(8, 2),
		}, nil)),
		mediaProtoBytes(5, bytes.Join([][]byte{
			mediaProtoBytes(1, mediaProtoBytes(1, metadata)),
			mediaProtoBytes(2, characteristics),
		}, nil)),
		mediaProtoBytes(13, nil),
		mediaProtoBytes(17, mediaProtoVarint(4, uint64(nowMillis))),
		mediaProtoBytes(22, mediaProtoVarint(1, 7)),
	}, nil)
	// Contents { externalMedia: field 3 } where the outer field 3 is the
	// Contents oneof and the inner field 3 is the media list inside
	// ExternalMedia.
	return mediaProtoBytes(3, mediaProtoBytes(3, mediaWrapper))
}

func mediaProtoVarint(field protowire.Number, value uint64) []byte {
	return protowire.AppendVarint(protowire.AppendTag(nil, field, protowire.VarintType), value)
}

func mediaProtoString(field protowire.Number, value string) []byte {
	return mediaProtoBytes(field, []byte(value))
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

// imageDimensions reports the intrinsic pixel size of supported image formats.
func imageDimensions(data []byte) (width, height int, err error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, fmt.Errorf("decode image dimensions: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return 0, 0, fmt.Errorf("image dimensions invalid %dx%d", config.Width, config.Height)
	}
	return config.Width, config.Height, nil
}

func isJPEGData(data []byte) bool {
	return len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF
}

func isPNGData(data []byte) bool {
	return len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A
}

func isWebPData(data []byte) bool {
	return len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP"
}

// webpDimensions parses the intrinsic size straight out of the WebP RIFF
// container without decoding the image (Go's stdlib has no WebP decoder). It
// walks the chunk list and parses the first mandatory image chunk: VP8
// (lossy), VP8L (lossless), or VP8X (extended canvas). This mirrors the
// Snapchat Web reference parseImageDimensions, which auto-detects only VP8
// and documents that WebP bytes are uploaded as-is.
func webpDimensions(data []byte) (width, height int, err error) {
	if !isWebPData(data) {
		return 0, 0, fmt.Errorf("invalid WEBP container")
	}
	riffSize := binary.LittleEndian.Uint32(data[4:8])
	if riffSize < 4 || uint64(riffSize) > uint64(len(data)-8) {
		return 0, 0, fmt.Errorf("truncated WEBP container: RIFF size %d exceeds %d payload bytes", riffSize, len(data)-8)
	}
	offset := 12
	for offset+8 <= len(data) {
		fourCC := string(data[offset : offset+4])
		chunkSize := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		payload := offset + 8
		if uint64(chunkSize) > uint64(len(data)-payload) {
			return 0, 0, fmt.Errorf("truncated WEBP chunk %q: needs %d bytes, have %d", fourCC, chunkSize, len(data)-payload)
		}
		switch fourCC {
		case "VP8 ", "VP8L", "VP8X":
			width, height, err = webpChunkDimensions(fourCC, data[payload:payload+int(chunkSize)])
			if err != nil {
				return 0, 0, err
			}
			if width <= 0 || height <= 0 {
				return 0, 0, fmt.Errorf("WebP dimensions invalid %dx%d", width, height)
			}
			return width, height, nil
		}
		offset = payload + int(chunkSize) + int(chunkSize)%2
	}
	return 0, 0, fmt.Errorf("WEBP container has no VP8/VP8L/VP8X image chunk")
}

func webpChunkDimensions(fourCC string, payload []byte) (int, int, error) {
	switch fourCC {
	case "VP8 ": // lossy: 3-byte frame tag, sync 0x9D 0x01 0x2A, LE16 dims (low 14 bits)
		if len(payload) < 10 || payload[3] != 0x9D || payload[4] != 0x01 || payload[5] != 0x2A {
			return 0, 0, fmt.Errorf("malformed WebP: bad VP8 lossy frame header")
		}
		return int(binary.LittleEndian.Uint16(payload[6:8])) & 0x3FFF,
			int(binary.LittleEndian.Uint16(payload[8:10])) & 0x3FFF, nil
	case "VP8L": // lossless: 0x2F signature, then 14-bit width-1 and height-1
		if len(payload) < 5 || payload[0] != 0x2F {
			return 0, 0, fmt.Errorf("malformed WebP: bad VP8L frame header")
		}
		bits := binary.LittleEndian.Uint32(payload[1:5])
		return int(bits&0x3FFF) + 1, int((bits>>14)&0x3FFF) + 1, nil
	case "VP8X": // extended: 4 flag bytes, LE24 canvas width-1, LE24 canvas height-1
		if len(payload) < 10 {
			return 0, 0, fmt.Errorf("malformed WebP: truncated VP8X canvas header")
		}
		return webpLE24(payload[4:7]) + 1, webpLE24(payload[7:10]) + 1, nil
	}
	return 0, 0, fmt.Errorf("unsupported WebP image chunk %q", fourCC)
}

func webpLE24(b []byte) int {
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16
}

// OutgoingMediaInfo captures everything the EXTERNAL_MEDIA send path needs to
// know about one outgoing image. Snapchat Web uploads JPEG/PNG/WebP bytes
// as-is over the identical EXTERNAL_MEDIA wire shape (documented reference:
// snapcap-native src/api/_media_send.ts encodeCreateContentMessageMedia and
// sendImageDirect — "Raw image bytes (PNG / JPEG / WebP). Sent as-is; no
// resizing or re-encoding" — the contents carry only decoded width/height
// plus the key/IV, and the media reference carries the shared image
// useCase/media type; there is no MIME or file-type field), so the only
// per-format variation is validation and dimension parsing.
type OutgoingMediaInfo struct {
	MIME      string
	Width     int
	Height    int
	MediaType protos.MediaType
	// HasAudio distinguishes MEDIA_TYPE_VIDEO from MEDIA_TYPE_VIDEONOAUDIO on
	// outgoing video; it is always false for images.
	HasAudio bool
	// DurationMs carries the video track duration for mediaDurationMs(15);
	// it is always 0 (omitted) for images.
	DurationMs int64
}

// classifyOutgoingImage sniffs the real file signature (never the declared
// filename/extension), cross-checks an optional declared MIME, parses decoded
// dimensions, and fully validates PNG payloads. JPEG keeps its proven
// header-only validation; PNG additionally runs a complete decode because
// Go's png decoder verifies chunk structure, CRCs and the zlib checksum, so
// truncated/malformed PNGs that pass header-only parsing are rejected; WebP
// gets structural RIFF/chunk validation plus VP8/VP8L/VP8X header parsing
// (the stdlib has no WebP decoder); MP4 gets box-structure validation plus
// tkhd/hdlr metadata parsing. It is the single classification entry for the
// ordinary outbound media sender (images and MP4 video).
func classifyOutgoingImage(data []byte, declaredMIME string) (*OutgoingMediaInfo, error) {
	var isWebP, isMP4 bool
	switch {
	case isJPEGData(data):
		if declaredMIME != "" && declaredMIME != "image/jpeg" && declaredMIME != "image/jpg" {
			return nil, fmt.Errorf("media bytes are image/jpeg but declared %q", declaredMIME)
		}
	case isPNGData(data):
		if declaredMIME != "" && declaredMIME != "image/png" {
			return nil, fmt.Errorf("media bytes are image/png but declared %q", declaredMIME)
		}
	case isWebPData(data):
		if declaredMIME != "" && declaredMIME != "image/webp" {
			return nil, fmt.Errorf("media bytes are image/webp but declared %q", declaredMIME)
		}
		isWebP = true
	case isMP4Data(data):
		if declaredMIME != "" && declaredMIME != "video/mp4" {
			return nil, fmt.Errorf("media bytes are video/mp4 but declared %q", declaredMIME)
		}
		isMP4 = true
	default:
		return nil, fmt.Errorf("outbound Snapchat media supports only image/jpeg, image/png, image/webp, and video/mp4 (declared %q)", declaredMIME)
	}
	var width, height int
	var hasAudio bool
	var durationMs int64
	var err error
	switch {
	case isWebP:
		width, height, err = webpDimensions(data)
	case isMP4:
		width, height, hasAudio, durationMs, err = mp4MediaInfo(data)
	default:
		width, height, err = imageDimensions(data)
	}
	if err != nil {
		return nil, err
	}
	if width > maxOutgoingImageDimension || height > maxOutgoingImageDimension {
		return nil, fmt.Errorf("image dimensions %dx%d exceed the %dpx outgoing limit", width, height, maxOutgoingImageDimension)
	}
	if isPNGData(data) {
		if _, err := png.Decode(bytes.NewReader(data)); err != nil {
			return nil, fmt.Errorf("invalid PNG data: %w", err)
		}
		return &OutgoingMediaInfo{MIME: "image/png", Width: width, Height: height, MediaType: protos.MediaType_MEDIA_TYPE_IMAGE}, nil
	}
	if isWebP {
		return &OutgoingMediaInfo{MIME: "image/webp", Width: width, Height: height, MediaType: protos.MediaType_MEDIA_TYPE_IMAGE}, nil
	}
	if isMP4 {
		mediaType := protos.MediaType_MEDIA_TYPE_VIDEO
		if !hasAudio {
			mediaType = protos.MediaType_MEDIA_TYPE_VIDEONOAUDIO
		}
		return &OutgoingMediaInfo{MIME: "video/mp4", Width: width, Height: height, MediaType: mediaType, HasAudio: hasAudio, DurationMs: durationMs}, nil
	}
	return &OutgoingMediaInfo{MIME: "image/jpeg", Width: width, Height: height, MediaType: protos.MediaType_MEDIA_TYPE_IMAGE}, nil
}

// mediaUploadClock isolates timestamps in the contents payload for tests.
var mediaUploadClock = func() int64 { return time.Now().UnixMilli() }
