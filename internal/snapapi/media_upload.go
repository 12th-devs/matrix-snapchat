package snapapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0xzer/snapper/data/headers"
	"github.com/0xzer/snapper/data/paths"
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
// chat image. Shape (Contents -> externalMedia(3) -> media(3)) recovered from
// the official web client and verified against this bridge's own incoming
// ExternalMediaEncryptionKeys extractor: Contents.3.3.5.1.1 carries the
// MediaMetadata with encryptionInfoV1 (field 4, base64) and encryptionInfoV2
// (field 19, raw). Both must carry the SAME key/IV.
func encodeExternalMediaContents(nowMillis int64, width, height int, key, iv []byte) []byte {
	metadata := bytes.Join([][]byte{
		mediaProtoBytes(5, bytes.Join([][]byte{
			mediaProtoVarint(1, uint64(width)),
			mediaProtoVarint(2, uint64(height)),
		}, nil)),
		mediaProtoBytes(18, nil),
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
	}, nil)
	mediaWrapper := bytes.Join([][]byte{
		mediaProtoVarint(8, 2),
		mediaProtoBytes(4, bytes.Join([][]byte{
			mediaProtoVarint(6, 1),
			mediaProtoString(10, "1"),
			mediaProtoVarint(8, 2),
		}, nil)),
		mediaProtoBytes(5, bytes.Join([][]byte{
			mediaProtoBytes(1, mediaProtoBytes(1, metadata)),
			mediaProtoBytes(2, bytes.Join([][]byte{
				mediaProtoBytes(7, nil),
				mediaProtoBytes(9, nil),
			}, nil)),
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

// mediaUploadClock isolates timestamps in the contents payload for tests.
var mediaUploadClock = func() int64 { return time.Now().UnixMilli() }
