package snapapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xzer/snapper/data/methods"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
	"github.com/0xzer/snapper/types"
	"google.golang.org/protobuf/encoding/protowire"
)

// testJPEG is a freshly encoded 3x2 RGBA JPEG.
var testJPEG = func() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		panic(err)
	}
	return buf.Bytes()
}()

// testPNGBytes is a freshly encoded 4x3 RGBA PNG.
var testPNGBytes = func() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}()

func TestEncodeGetUploadLocationsRequest(t *testing.T) {
	body := encodeGetUploadLocationsRequest()
	want := []byte{
		0x10, 0x01, // field 2 varint 1 (batchSize)
		0x20, 0x01, // field 4 varint 1 (CONTENT_OBJECT_ONLY)
		0x3A, 0x01, 0x00, // field 7 packed [0] (DIRECT)
		0x80, 0x01, 0x01, // field 16 varint 1 (useZeroRatingDomainForGcsLocations)
	}
	if !bytes.Equal(body, want) {
		t.Fatalf("unexpected getUploadLocations request bytes: got %x want %x", body, want)
	}
}

func encodeTestUploadLocationsResponse(putURL, token, base string, includeURL bool) []byte {
	var contentObject []byte
	contentObject = append(contentObject, mediaProtoString(1, token)...)
	contentObject = append(contentObject, mediaProtoBytes(2, mediaProtoString(2, base))...)
	var reference []byte
	reference = append(reference, mediaProtoBytes(3, contentObject)...)
	var location []byte
	if includeURL {
		location = append(location, mediaProtoString(1, putURL)...)
	}
	location = append(location, mediaProtoBytes(4, reference)...)
	return mediaProtoBytes(1, location)
}

func TestParseUploadLocations(t *testing.T) {
	location, err := parseUploadLocations(encodeTestUploadLocationsResponse(
		"https://upload.example.com/bucket?sig=abc", "wO87suY3U6Ss473nIx9ba_1", "wO87suY3U6Ss473nIx9ba", true,
	))
	if err != nil {
		t.Fatal(err)
	}
	if location.PutURL != "https://upload.example.com/bucket?sig=abc" {
		t.Fatalf("unexpected put url: %s", location.PutURL)
	}
	if location.MediaIDToken != "wO87suY3U6Ss473nIx9ba_1" || location.MediaIDBase != "wO87suY3U6Ss473nIx9ba" {
		t.Fatalf("unexpected media ids: token=%q base=%q", location.MediaIDToken, location.MediaIDBase)
	}
}

func TestParseUploadLocationsRejectsIncomplete(t *testing.T) {
	if _, err := parseUploadLocations(encodeTestUploadLocationsResponse("", "tok_1", "base", false)); err == nil {
		t.Fatal("missing upload URL must be rejected")
	}
	// Non-HTTPS URL must be rejected.
	if _, err := parseUploadLocations(encodeTestUploadLocationsResponse(
		"http://insecure.example.com/x", "tok_1", "base", true,
	)); err == nil {
		t.Fatal("insecure upload URL must be rejected")
	}
	// Non-ASCII ids are implausible.
	if _, err := parseUploadLocations(encodeTestUploadLocationsResponse(
		"https://upload.example.com/x", "\x00\x01bad", "base", true,
	)); err == nil {
		t.Fatal("implausible media ids must be rejected")
	}
}

func TestEncryptOutgoingMediaRoundtrip(t *testing.T) {
	plain := bytes.Repeat([]byte("snapchat-outbound-jpeg-probe"), 973)
	encrypted, err := EncryptOutgoingMedia(plain)
	if err != nil {
		t.Fatal(err)
	}
	if len(encrypted.Key) != 32 || len(encrypted.IV) != 16 {
		t.Fatalf("unexpected key/iv sizes: %d/%d", len(encrypted.Key), len(encrypted.IV))
	}
	if len(encrypted.Ciphertext) == 0 || len(encrypted.Ciphertext)%aes.BlockSize != 0 {
		t.Fatalf("ciphertext length not block aligned: %d", len(encrypted.Ciphertext))
	}
	if bytes.Contains(encrypted.Ciphertext, plain[:32]) {
		t.Fatal("ciphertext leaks plaintext")
	}
	block, err := aes.NewCipher(encrypted.Key)
	if err != nil {
		t.Fatal(err)
	}
	decrypted := make([]byte, len(encrypted.Ciphertext))
	cipher.NewCBCDecrypter(block, encrypted.IV).CryptBlocks(decrypted, encrypted.Ciphertext)
	pad := int(decrypted[len(decrypted)-1])
	decrypted = decrypted[:len(decrypted)-pad]
	if !bytes.Equal(decrypted, plain) {
		t.Fatal("roundtrip decryption mismatch")
	}
	// Fresh randomness per call.
	second, err := EncryptOutgoingMedia(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(second.Key, encrypted.Key) || bytes.Equal(second.IV, encrypted.IV) {
		t.Fatal("key/iv must be freshly random per send")
	}
}

func TestEncodeOutgoingContentObject(t *testing.T) {
	descriptor := encodeOutgoingContentObject(&UploadLocation{
		PutURL:       "https://upload.example.com/x",
		MediaIDToken: "wO87suY3U6Ss473nIx9ba_1",
		MediaIDBase:  "wO87suY3U6Ss473nIx9ba",
	})
	tokens, err := mediaProtoField(descriptor, 1)
	if err != nil || len(tokens) != 1 || string(tokens[0]) != "wO87suY3U6Ss473nIx9ba_1" {
		t.Fatalf("contentObjectId mismatch: %v %v", tokens, err)
	}
	descriptors, err := mediaProtoField(descriptor, 2)
	if err != nil || len(descriptors) != 1 {
		t.Fatalf("content descriptor missing: %v", err)
	}
	ids, err := mediaProtoField(descriptors[0], 2)
	if err != nil || len(ids) != 1 || string(ids[0]) != "wO87suY3U6Ss473nIx9ba" {
		t.Fatalf("descriptor contentId mismatch: %v %v", ids, err)
	}
	locations, err := mediaProtoField(descriptors[0], 6)
	if err != nil || len(locations) != 1 || !bytes.Equal(locations[0], []byte{0x04}) {
		t.Fatalf("readyLocationIds mismatch: %v %v", locations, err)
	}
	for field, want := range map[protowire.Number]uint64{9: 1, 10: 11, 12: 1} {
		got, err := readUploadVarintField(descriptors[0], field)
		if err != nil || got != want {
			t.Fatalf("descriptor field %d = %d (%v), want %d", field, got, err, want)
		}
	}
}

func readUploadVarintField(data []byte, wanted protowire.Number) (uint64, error) {
	for len(data) > 0 {
		number, kind, n := protowire.ConsumeTag(data)
		if n < 0 {
			return 0, protowire.ParseError(n)
		}
		data = data[n:]
		if number == wanted && kind == protowire.VarintType {
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return 0, protowire.ParseError(n)
			}
			return value, nil
		}
		n = protowire.ConsumeFieldValue(number, kind, data)
		if n < 0 {
			return 0, protowire.ParseError(n)
		}
		data = data[n:]
	}
	return 0, nil
}

func TestEncodeExternalMediaContentsKeys(t *testing.T) {
	key := bytes.Repeat([]byte{0x11}, 32)
	iv := bytes.Repeat([]byte{0x22}, 16)
	contents := encodeExternalMediaContents(1725000000000, &OutgoingMediaInfo{MIME: "image/png", Width: 640, Height: 480, MediaType: protos.MediaType_MEDIA_TYPE_IMAGE}, key, iv)
	gotKey, gotIV, err := ExternalMediaEncryptionKeys(contents)
	if err != nil {
		t.Fatalf("incoming extractor rejected outgoing contents: %v", err)
	}
	if !bytes.Equal(gotKey, key) || !bytes.Equal(gotIV, iv) {
		t.Fatal("outgoing contents key/IV do not roundtrip through the incoming extractor")
	}
	// Both encodings must agree: field 4 carries base64, field 19 raw bytes.
	external, err := mediaProtoField(contents, 3)
	if err != nil || len(external) != 1 {
		t.Fatalf("externalMedia missing: %v", err)
	}
	media, err := mediaProtoField(external[0], 3)
	if err != nil || len(media) != 1 {
		t.Fatalf("media wrapper missing: %v", err)
	}
	wrapper, err := mediaProtoField(media[0], 5)
	if err != nil || len(wrapper) != 1 {
		t.Fatalf("descriptor wrapper missing: %v", err)
	}
	d1, err := mediaProtoField(wrapper[0], 1)
	if err != nil || len(d1) != 1 {
		t.Fatalf("descriptor outer message missing: %v", err)
	}
	d2, err := mediaProtoField(d1[0], 1)
	if err != nil || len(d2) != 1 {
		t.Fatalf("metadata missing: %v", err)
	}
	v1, err := mediaProtoField(d2[0], 4)
	if err != nil || len(v1) != 1 {
		t.Fatalf("encryptionInfoV1 missing: %v", err)
	}
	keys, err := mediaProtoField(v1[0], 1)
	if err != nil || len(keys) != 1 || base64.StdEncoding.EncodeToString(key) != string(keys[0]) {
		t.Fatalf("encryptionInfoV1 key mismatch: %v", err)
	}
	v2, err := mediaProtoField(d2[0], 19)
	if err != nil || len(v2) != 1 {
		t.Fatalf("encryptionInfoV2 missing: %v", err)
	}
	rawKeys, err := mediaProtoField(v2[0], 1)
	if err != nil || len(rawKeys) != 1 || !bytes.Equal(rawKeys[0], key) {
		t.Fatalf("encryptionInfoV2 key mismatch: %v", err)
	}
	dimensions, err := mediaProtoField(d2[0], 5)
	if err != nil || len(dimensions) != 1 {
		t.Fatalf("dimensions missing: %v", err)
	}
	width, err := readUploadVarintField(dimensions[0], 1)
	if err != nil || width != 640 {
		t.Fatalf("width mismatch: %d %v", width, err)
	}
	height, err := readUploadVarintField(dimensions[0], 2)
	if err != nil || height != 480 {
		t.Fatalf("height mismatch: %d %v", height, err)
	}
}

func TestSendMediaRejectsUnsupportedFormats(t *testing.T) {
	c := &Client{}
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "image/jpeg", Data: nil}, ""); err == nil {
		t.Fatal("empty media must be rejected")
	}
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "image/jpeg", Data: bytes.Repeat([]byte{0x01}, maxOutgoingMediaSize+1)}, ""); err == nil {
		t.Fatal("oversized media must be rejected")
	}
	// Non-image formats stay rejected (GIF is future work; WebP is accepted).
	gif := append([]byte("GIF89a"), bytes.Repeat([]byte{0}, 64)...)
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "image/gif", Data: gif}, ""); err == nil {
		t.Fatal("GIF must still be rejected")
	}
	garbage := bytes.Repeat([]byte{0x01}, 64)
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "image/jpeg", Data: garbage}, ""); err == nil {
		t.Fatal("malformed JPEG (bad magic) must still be rejected")
	}
	// Declared MIME must agree with the real file signature, both ways.
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "image/png", Data: testJPEG}, ""); err == nil {
		t.Fatal("JPEG bytes declared as image/png must be rejected")
	}
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "image/jpeg", Data: testPNGBytes}, ""); err == nil {
		t.Fatal("PNG bytes declared as image/jpeg must be rejected")
	}
}

func TestClassifyOutgoingImagePNG(t *testing.T) {
	info, err := classifyOutgoingImage(testPNGBytes, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if info.MIME != "image/png" {
		t.Fatalf("mime = %q, want image/png", info.MIME)
	}
	if info.Width != 4 || info.Height != 3 {
		t.Fatalf("unexpected PNG dimensions %dx%d", info.Width, info.Height)
	}
	if info.MediaType != protos.MediaType_MEDIA_TYPE_IMAGE {
		t.Fatalf("PNG media type = %v, want MEDIA_TYPE_IMAGE", info.MediaType)
	}
	// Declared MIME is optional; sniffing alone must classify.
	if info2, err := classifyOutgoingImage(testPNGBytes, ""); err != nil || info2.MIME != "image/png" {
		t.Fatalf("sniff-only classification failed: %v %v", info2, err)
	}
}

func TestClassifyOutgoingImageRejectsBrokenPNG(t *testing.T) {
	// Signature only: no IHDR -> dimension parsing fails.
	sigOnly := testPNGBytes[:8]
	if _, err := classifyOutgoingImage(sigOnly, "image/png"); err == nil {
		t.Fatal("signature-only PNG must be rejected")
	}
	// Header-valid but truncated body: full decode must fail.
	if len(testPNGBytes) < 2 {
		t.Fatal("testPNGBytes unexpectedly tiny")
	}
	truncated := testPNGBytes[:len(testPNGBytes)/2]
	if _, err := classifyOutgoingImage(truncated, "image/png"); err == nil {
		t.Fatal("truncated PNG must be rejected")
	}
	// Valid signature followed by garbage.
	garbage := append(append([]byte{}, testPNGBytes[:8]...), bytes.Repeat([]byte{0xAB}, 64)...)
	if _, err := classifyOutgoingImage(garbage, "image/png"); err == nil {
		t.Fatal("garbage PNG body must be rejected")
	}
}

func TestClassifyOutgoingImageRejectsOversizeDimensions(t *testing.T) {
	huge := image.NewRGBA(image.Rect(0, 0, maxOutgoingImageDimension+1, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, huge); err != nil {
		t.Fatal(err)
	}
	if _, err := classifyOutgoingImage(buf.Bytes(), "image/png"); err == nil {
		t.Fatal("oversize-dimension PNG must be rejected")
	}
}

// webpChunk frames one RIFF chunk with its FourCC, LE32 size, and even-byte
// padding.
func webpChunk(fourCC string, payload []byte) []byte {
	chunk := make([]byte, 8, 8+len(payload)+len(payload)%2)
	copy(chunk[0:4], fourCC)
	binary.LittleEndian.PutUint32(chunk[4:8], uint32(len(payload)))
	chunk = append(chunk, payload...)
	if len(payload)%2 == 1 {
		chunk = append(chunk, 0)
	}
	return chunk
}

// webpFile wraps chunks in the 12-byte RIFF/WEBP container header.
func webpFile(chunks ...[]byte) []byte {
	body := append([]byte("WEBP"), bytes.Join(chunks, nil)...)
	out := make([]byte, 8, 8+len(body))
	copy(out[0:4], "RIFF")
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(body)))
	return append(out, body...)
}

func testWebPVP8(w, h int, breakSync bool) []byte {
	payload := make([]byte, 10)
	payload[3], payload[4], payload[5] = 0x9D, 0x01, 0x2A
	binary.LittleEndian.PutUint16(payload[6:8], uint16(w))
	binary.LittleEndian.PutUint16(payload[8:10], uint16(h))
	if breakSync {
		payload[3] = 0x00
	}
	return webpFile(webpChunk("VP8 ", payload))
}

func testWebPVP8L(w, h int) []byte {
	bits := uint32(w-1) | uint32(h-1)<<14
	payload := []byte{0x2F, byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24)}
	return webpFile(webpChunk("VP8L", payload))
}

func testWebPVP8X(w, h int) []byte {
	payload := make([]byte, 10)
	payload[4] = byte(w - 1)
	payload[5] = byte((w - 1) >> 8)
	payload[6] = byte((w - 1) >> 16)
	payload[7] = byte(h - 1)
	payload[8] = byte((h - 1) >> 8)
	payload[9] = byte((h - 1) >> 16)
	return webpFile(webpChunk("VP8X", payload))
}

func TestClassifyOutgoingImageWebP(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"vp8", testWebPVP8(640, 480, false)},
		{"vp8l", testWebPVP8L(100, 50)},
		{"vp8x", testWebPVP8X(800, 600)},
	}
	for _, tc := range cases {
		info, err := classifyOutgoingImage(tc.data, "image/webp")
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if info.MIME != "image/webp" {
			t.Fatalf("%s: mime = %q, want image/webp", tc.name, info.MIME)
		}
		if info.MediaType != protos.MediaType_MEDIA_TYPE_IMAGE {
			t.Fatalf("%s: media type = %v, want MEDIA_TYPE_IMAGE", tc.name, info.MediaType)
		}
		// Declared MIME is optional; sniffing alone must classify.
		if _, err := classifyOutgoingImage(tc.data, ""); err != nil {
			t.Fatalf("%s sniff-only: %v", tc.name, err)
		}
	}
	if w, h, err := webpDimensions(testWebPVP8(640, 480, false)); err != nil || w != 640 || h != 480 {
		t.Fatalf("vp8 dimensions = %dx%d, %v", w, h, err)
	}
	if w, h, err := webpDimensions(testWebPVP8L(100, 50)); err != nil || w != 100 || h != 50 {
		t.Fatalf("vp8l dimensions = %dx%d, %v", w, h, err)
	}
	if w, h, err := webpDimensions(testWebPVP8X(800, 600)); err != nil || w != 800 || h != 600 {
		t.Fatalf("vp8x dimensions = %dx%d, %v", w, h, err)
	}
	if !isWebPData(testWebPVP8(2, 2, false)) || isWebPData(testPNGBytes) || isWebPData(testJPEG) {
		t.Fatal("WebP magic check must not cross-match JPEG/PNG")
	}
}

func TestClassifyOutgoingImageRejectsBrokenWebP(t *testing.T) {
	// VP8 lossy frame header with a corrupted sync code.
	if _, err := classifyOutgoingImage(testWebPVP8(10, 10, true), "image/webp"); err == nil {
		t.Fatal("malformed VP8 frame header must be rejected")
	}
	// RIFF header truncated below the container minimum.
	if _, err := classifyOutgoingImage(testWebPVP8(10, 10, false)[:11], "image/webp"); err == nil {
		t.Fatal("short RIFF header must be rejected")
	}
	// Declared RIFF size beyond the actual bytes.
	trunc := testWebPVP8(10, 10, false)
	binary.LittleEndian.PutUint32(trunc[4:8], uint32(len(trunc)))
	if _, err := classifyOutgoingImage(trunc[:len(trunc)-4], "image/webp"); err == nil {
		t.Fatal("truncated WEBP payload must be rejected")
	}
	// Chunk size beyond the actual bytes.
	bigChunk := testWebPVP8(10, 10, false)
	binary.LittleEndian.PutUint32(bigChunk[16:20], 1<<20)
	if _, err := classifyOutgoingImage(bigChunk, "image/webp"); err == nil {
		t.Fatal("truncated WEBP chunk must be rejected")
	}
	// Container with no image chunk at all.
	if _, err := classifyOutgoingImage(webpFile(webpChunk("AAAA", []byte{0})), "image/webp"); err == nil {
		t.Fatal("WEBP without an image chunk must be rejected")
	}
	// Zero dimensions are invalid.
	if _, err := classifyOutgoingImage(testWebPVP8(0, 10, false), "image/webp"); err == nil {
		t.Fatal("zero-width WebP must be rejected")
	}
	if _, err := classifyOutgoingImage(nil, "image/webp"); err == nil {
		t.Fatal("empty WebP must be rejected")
	}
}

func TestClassifyOutgoingImageWebPMIMEMismatch(t *testing.T) {
	webp := testWebPVP8(10, 10, false)
	if _, err := classifyOutgoingImage(webp, "image/png"); err == nil {
		t.Fatal("WebP bytes declared as image/png must be rejected")
	}
	if _, err := classifyOutgoingImage(webp, "image/jpeg"); err == nil {
		t.Fatal("WebP bytes declared as image/jpeg must be rejected")
	}
	if _, err := classifyOutgoingImage(testPNGBytes, "image/webp"); err == nil {
		t.Fatal("PNG bytes declared as image/webp must be rejected")
	}
	if _, err := classifyOutgoingImage(testJPEG, "image/webp"); err == nil {
		t.Fatal("JPEG bytes declared as image/webp must be rejected")
	}
}

func TestClassifyOutgoingImageRejectsOversizeWebP(t *testing.T) {
	// 14-bit VP8 dims allow up to 16383, which exceeds the 10000px limit.
	if _, err := classifyOutgoingImage(testWebPVP8(16383, 16383, false), "image/webp"); err == nil {
		t.Fatal("oversize-dimension WebP must be rejected")
	}
}

// mp4Box frames one ISO-BMFF box with its 4-byte big-endian size and type.
func mp4Box(boxType string, payload []byte) []byte {
	out := make([]byte, 8, 8+len(payload))
	binary.BigEndian.PutUint32(out[0:4], uint32(8+len(payload)))
	copy(out[4:8], boxType)
	return append(out, payload...)
}

// testMP4 builds a minimal structurally valid MP4: ftyp + moov{trak{tkhd
// dims, mdia{hdlr vide, mdhd 2000ms}}[, trak{tkhd 0x0, mdia{hdlr soun}}]}.
// width or height 0 models an audio-only file.
func testMP4(width, height int, withAudio bool) []byte {
	return testMP4Rotated(width, height, withAudio, false)
}

// testMP4Rotated optionally marks the video track with a 90-degree tkhd
// transformation matrix (coded dims stay unrotated, display dims swap).
func testMP4Rotated(width, height int, withAudio, rotated bool) []byte {
	tkhd := make([]byte, 96)
	if width > 0 && height > 0 {
		binary.BigEndian.PutUint32(tkhd[88:92], uint32(width)<<16)
		binary.BigEndian.PutUint32(tkhd[92:96], uint32(height)<<16)
	}
	if rotated {
		// 90-degree rotation: a=0, b=+1, c=-1, d=0 in 16.16 fixed point.
		binary.BigEndian.PutUint32(tkhd[52:56], 0)
		binary.BigEndian.PutUint32(tkhd[56:60], 1<<16)
		binary.BigEndian.PutUint32(tkhd[64:68], 0xFFFF0000)
		binary.BigEndian.PutUint32(tkhd[68:72], 0)
	}
	handler := make([]byte, 12)
	copy(handler[8:12], "vide")
	// mdhd v0: ver/flags(4) creation(4) modification(4) timescale(4)=1000
	// duration(4)=2000 -> 2000ms.
	mdhd := make([]byte, 20)
	binary.BigEndian.PutUint32(mdhd[12:16], 1000)
	binary.BigEndian.PutUint32(mdhd[16:20], 2000)
	videoTrak := mp4Box("trak", bytes.Join([][]byte{
		mp4Box("tkhd", tkhd),
		mp4Box("mdia", bytes.Join([][]byte{mp4Box("hdlr", handler), mp4Box("mdhd", mdhd)}, nil)),
	}, nil))
	traks := [][]byte{videoTrak}
	if withAudio {
		audioTkhd := make([]byte, 96)
		audioHandler := make([]byte, 12)
		copy(audioHandler[8:12], "soun")
		traks = append(traks, mp4Box("trak", bytes.Join([][]byte{
			mp4Box("tkhd", audioTkhd),
			mp4Box("mdia", mp4Box("hdlr", audioHandler)),
		}, nil)))
	}
	ftyp := mp4Box("ftyp", []byte("isom\x00\x00\x00\x00isom"))
	return bytes.Join(append([][]byte{ftyp, mp4Box("moov", bytes.Join(traks, nil))}, nil...), nil)
}

func TestClassifyOutgoingMediaMP4(t *testing.T) {
	info, err := classifyOutgoingImage(testMP4(640, 360, true), "video/mp4")
	if err != nil {
		t.Fatal(err)
	}
	if info.MIME != "video/mp4" {
		t.Fatalf("mime = %q, want video/mp4", info.MIME)
	}
	if info.Width != 640 || info.Height != 360 {
		t.Fatalf("unexpected MP4 dimensions %dx%d", info.Width, info.Height)
	}
	if info.MediaType != protos.MediaType_MEDIA_TYPE_VIDEO || !info.HasAudio {
		t.Fatalf("audio MP4 media type = %v hasAudio=%v, want VIDEO/true", info.MediaType, info.HasAudio)
	}
	if info.DurationMs != 2000 {
		t.Fatalf("MP4 duration = %dms, want 2000", info.DurationMs)
	}
	silent, err := classifyOutgoingImage(testMP4(320, 240, false), "video/mp4")
	if err != nil {
		t.Fatal(err)
	}
	if silent.MediaType != protos.MediaType_MEDIA_TYPE_VIDEONOAUDIO || silent.HasAudio {
		t.Fatalf("silent MP4 media type = %v hasAudio=%v, want VIDEONOAUDIO/false", silent.MediaType, silent.HasAudio)
	}
	// Declared MIME is optional; sniffing alone must classify.
	if _, err := classifyOutgoingImage(testMP4(2, 2, true), ""); err != nil {
		t.Fatal(err)
	}
}

func TestClassifyOutgoingMediaRejectsBrokenMP4(t *testing.T) {
	// Not an MP4: bytes 4-8 must read ftyp.
	if _, err := classifyOutgoingImage(append([]byte{0, 0, 0, 0}, []byte("XtypJUNK")...), "video/mp4"); err == nil {
		t.Fatal("missing ftyp must be rejected")
	}
	// ftyp only: no moov box.
	ftypOnly := mp4Box("ftyp", []byte("isom\x00\x00\x00\x00isom"))
	if _, err := classifyOutgoingImage(ftypOnly, "video/mp4"); err == nil {
		t.Fatal("MP4 without moov must be rejected")
	}
	// moov box whose declared size runs past the available bytes.
	trunc := append(append([]byte{}, ftypOnly...), mp4Box("moov", mp4Box("trak", []byte{}))...)
	binary.BigEndian.PutUint32(trunc[len(ftypOnly):len(ftypOnly)+4], 1<<20)
	if _, err := classifyOutgoingImage(trunc, "video/mp4"); err == nil {
		t.Fatal("truncated moov must be rejected")
	}
	// Degenerate box header (size < 8).
	bad := append(append([]byte{}, ftypOnly...), []byte{0, 0, 0, 3, 'm', 'o', 'o', 'v'}...)
	if _, err := classifyOutgoingImage(bad, "video/mp4"); err == nil {
		t.Fatal("bad box header must be rejected")
	}
	// Audio-only file: no video track with usable dimensions.
	if _, err := classifyOutgoingImage(testMP4(0, 0, true), "video/mp4"); err == nil {
		t.Fatal("MP4 without a video track must be rejected")
	}
	// Empty data.
	if _, err := classifyOutgoingImage(nil, "video/mp4"); err == nil {
		t.Fatal("empty MP4 must be rejected")
	}
}

func TestClassifyOutgoingMediaMP4MIMEMismatch(t *testing.T) {
	if _, err := classifyOutgoingImage(testMP4(10, 10, true), "image/png"); err == nil {
		t.Fatal("MP4 bytes declared as image/png must be rejected")
	}
	if _, err := classifyOutgoingImage(testPNGBytes, "video/mp4"); err == nil {
		t.Fatal("PNG bytes declared as video/mp4 must be rejected")
	}
	if _, err := classifyOutgoingImage(testJPEG, "video/mp4"); err == nil {
		t.Fatal("JPEG bytes declared as video/mp4 must be rejected")
	}
}

func TestClassifyOutgoingMediaRejectsOversizeMP4(t *testing.T) {
	if _, err := classifyOutgoingImage(testMP4(maxOutgoingImageDimension+1, 100, true), "video/mp4"); err == nil {
		t.Fatal("oversize-dimension MP4 must be rejected")
	}
}

func TestSendMediaMP4HappyPath(t *testing.T) {
	runSendMediaHappyPath(t, testMP4(640, 360, true), "video/mp4", 640, 360, protos.MediaType_MEDIA_TYPE_VIDEO)
}

func TestClassifyOutgoingMediaMP4Orientation(t *testing.T) {
	// Portrait phone video: coded 1920x1080 track with a 90-degree tkhd
	// matrix must report swapped DISPLAY dimensions.
	info, err := classifyOutgoingImage(testMP4Rotated(1920, 1080, true, true), "video/mp4")
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 1080 || info.Height != 1920 {
		t.Fatalf("rotated MP4 dimensions = %dx%d, want 1080x1920", info.Width, info.Height)
	}
	// Landscape identity matrix keeps the coded dimensions.
	info, err = classifyOutgoingImage(testMP4(1920, 1080, true), "video/mp4")
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 1920 || info.Height != 1080 {
		t.Fatalf("landscape MP4 dimensions = %dx%d, want 1920x1080", info.Width, info.Height)
	}
	// Degenerate (zeroed) and 180-degree matrices fall back to / keep the
	// coded dimensions (unit-tested against the helper directly).
	tkhd := make([]byte, 96)
	binary.BigEndian.PutUint32(tkhd[88:92], 1280<<16)
	binary.BigEndian.PutUint32(tkhd[92:96], 720<<16)
	if w, h := mp4DisplayDimensions(1280, 720, tkhd); w != 1280 || h != 720 {
		t.Fatalf("degenerate matrix fallback = %dx%d, want 1280x720", w, h)
	}
	binary.BigEndian.PutUint32(tkhd[52:56], 0xFFFF0000)
	binary.BigEndian.PutUint32(tkhd[68:72], 0xFFFF0000)
	if w, h := mp4DisplayDimensions(1280, 720, tkhd); w != 1280 || h != 720 {
		t.Fatalf("180-degree matrix = %dx%d, want 1280x720", w, h)
	}
}

func TestEncodeExternalMediaContentsVideoCharacteristics(t *testing.T) {
	key := bytes.Repeat([]byte{0x11}, 32)
	iv := bytes.Repeat([]byte{0x22}, 16)
	build := func(mediaType protos.MediaType, hasAudio bool) []byte {
		return encodeExternalMediaContents(0, &OutgoingMediaInfo{
			MIME: "video/mp4", Width: 320, Height: 240,
			MediaType: mediaType, HasAudio: hasAudio, DurationMs: 1500,
		}, key, iv)
	}
	silent, err := contentsPlaybackCharacteristics(build(protos.MediaType_MEDIA_TYPE_VIDEONOAUDIO, false))
	if err != nil {
		t.Fatal(err)
	}
	if po, err := mediaProtoField(silent, 7); err != nil || len(po) != 1 {
		t.Fatal("silent video must carry playOnce(7)")
	}
	if got, err := readUploadVarintField(silent, 5); err != nil || got != 0 {
		t.Fatalf("silent video must not carry characteristics.hasSound, got %d err %v", got, err)
	}
	audio, err := contentsPlaybackCharacteristics(build(protos.MediaType_MEDIA_TYPE_VIDEO, true))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := readUploadVarintField(audio, 5); err != nil || got != 1 {
		t.Fatalf("audio video must carry characteristics.hasSound=1, got %d err %v", got, err)
	}
	contents := build(protos.MediaType_MEDIA_TYPE_VIDEO, true)
	meta, err := contentsMetadataNode(contents)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := readUploadVarintField(meta, 15); err != nil || got != 1500 {
		t.Fatalf("video metadata.mediaDurationMs = %d, %v; want 1500", got, err)
	}
	mediaID, err := mediaProtoField(meta, 18)
	if err != nil || len(mediaID) != 1 {
		t.Fatalf("video metadata.mediaId missing: %v", err)
	}
	if got, err := readUploadVarintField(mediaID[0], 1); err != nil || got != 1 {
		t.Fatalf("video metadata.mediaId.mediaListId = %d, %v; want 1", got, err)
	}
}

func TestClassifyOutgoingImageJPEGUnchanged(t *testing.T) {
	info, err := classifyOutgoingImage(testJPEG, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if info.MIME != "image/jpeg" || info.Width != 3 || info.Height != 2 {
		t.Fatalf("unexpected JPEG classification: %+v", info)
	}
	if info.MediaType != protos.MediaType_MEDIA_TYPE_IMAGE {
		t.Fatalf("JPEG media type = %v, want MEDIA_TYPE_IMAGE", info.MediaType)
	}
}

func TestImageDimensionsPNG(t *testing.T) {
	width, height, err := imageDimensions(testPNGBytes)
	if err != nil {
		t.Fatal(err)
	}
	if width != 4 || height != 3 {
		t.Fatalf("unexpected PNG dimensions %dx%d", width, height)
	}
	if !isPNGData(testPNGBytes) {
		t.Fatal("PNG magic not detected")
	}
	if isPNGData(testJPEG) || isJPEGData(testPNGBytes) {
		t.Fatal("JPEG/PNG magic checks must not cross-match")
	}
}

// newAuthedTLSMediaClient is newAuthedTestClient over TLS: getUploadLocations
// requires https upload URLs, so the media happy-path tests need the TLS
// httptest server and its trusting client.
func newAuthedTLSMediaClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	tlsMediaServerURL = srv.URL
	t.Cleanup(func() { tlsMediaServerURL = "" })
	c := &Client{
		http:              srv.Client(),
		cookies:           &types.SnapCookies{},
		tokens:            &types.SnapTokens{},
		device:            types.NewDevice(),
		sessionCookieName: "sc-a-nonce",
		conversations:     make(map[string]*protos.Conversation),
		conversationIDs:   make(map[string]*protos.UUID),
	}
	c.tokens.SSO_TOKEN = "test-sso-token"
	c.selfUserID = "99999999-9999-9999-9999-999999999999"
	selfUUID, err := encodeUUIDString(c.selfUserID)
	if err != nil {
		t.Fatalf("encode self uuid: %v", err)
	}
	c.selfEncoded = selfUUID
	return c, srv
}

// runSendMediaHappyPath drives the FULL proven pipeline (getUploadLocations ->
// PUT -> CreateContentMessage) against a fake server and asserts the wire
// shape, encryption roundtrip, upload content type, and the nested
// createdMessageId that the outgoing RemoteID mapping depends on.
func runSendMediaHappyPath(t *testing.T, mediaData []byte, declaredMIME string, wantWidth, wantHeight int, wantMediaType protos.MediaType) {
	t.Helper()
	const createdMessageID = uint64(9089)
	var (
		putContentType  string
		putBody         []byte
		createRequests  []*protos.CreateContentMessageRequest
		uploadLocations = 0
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/put":
			putContentType = r.Header.Get("Content-Type")
			putBody = append([]byte{}, body...)
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/getUploadLocations"):
			uploadLocations++
			w.Write(grpcWebData(encodeTestUploadLocationsResponse(srvURL(t)+"/put", "wO87suY3U6Ss473nIx9ba_1", "wO87suY3U6Ss473nIx9ba", true)))
		case r.URL.Path == urlPath(paths.CREATE_CONTENT_MESSAGE):
			frame, err := methods.ReadResponseFrame(body)
			if err != nil {
				t.Fatalf("create frame unreadable: %v", err)
			}
			var req protos.CreateContentMessageRequest
			if err := protos.DecodeProtoMessage(frame, &req); err != nil {
				t.Fatalf("create proto undecodable: %v", err)
			}
			createRequests = append(createRequests, &req)
			resp := &protos.CreateContentMessageResponse{
				Result: []*protos.CreateContentMessageResult{{
					Success: true,
					DestinationRes: &protos.CreateContentMessageResult_ConversationDestinationResult{
						ConversationDestinationResult: &protos.ConversationDestinationResult{
							CreatedMessageId: createdMessageID,
						},
					},
				}},
			}
			payload, _ := protos.EncodeProtoMessage(resp)
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTLSMediaClient(t, handler)
	overrideListingURL(t, srv)
	// Point the media delivery RPC at the test server.
	origWebBase := paths.WEB_BASE_URL
	paths.WEB_BASE_URL = srv.URL
	t.Cleanup(func() { paths.WEB_BASE_URL = origWebBase })

	messageID, err := c.SendMedia(context.Background(), "aaaaaaaa-0000-0000-0000-000000000001", MediaAttachment{
		MimeType: declaredMIME,
		Data:     mediaData,
	}, "")
	if err != nil {
		t.Fatalf("SendMedia: %v", err)
	}
	if messageID != "9089" {
		t.Fatalf("SendMedia returned %q, want nested createdMessageId %d", messageID, createdMessageID)
	}
	if uploadLocations != 1 {
		t.Fatalf("getUploadLocations calls = %d, want 1", uploadLocations)
	}
	if putContentType != "application/octet-stream" {
		t.Fatalf("upload content type = %q, want application/octet-stream", putContentType)
	}
	if len(putBody) == 0 || len(putBody) == len(mediaData) {
		t.Fatalf("uploaded payload must be encrypted ciphertext, got %d bytes for %d clear bytes", len(putBody), len(mediaData))
	}
	if len(createRequests) != 1 {
		t.Fatalf("CreateContentMessage calls = %d, want 1", len(createRequests))
	}
	req := createRequests[0]
	if req.GetContent().GetContentType() != protos.ContentType_EXTERNAL_MEDIA {
		t.Fatalf("content type = %v, want EXTERNAL_MEDIA", req.GetContent().GetContentType())
	}
	if req.GetContent().GetSavePolicy() != protos.ContentEnvelope_SavePolicy_LIFETIME {
		t.Fatalf("save policy = %v, want LIFETIME", req.GetContent().GetSavePolicy())
	}
	refs := req.GetContent().GetMediaReferenceLists()
	if len(refs) != 1 || len(refs[0].GetReference()) != 1 {
		t.Fatalf("media reference list shape unexpected: %#v", refs)
	}
	if refs[0].GetReference()[0].GetMediaType() != wantMediaType {
		t.Fatalf("media type = %v, want %v", refs[0].GetReference()[0].GetMediaType(), wantMediaType)
	}
	// The contents must carry the key/IV and the decoded dimensions, and the
	// uploaded ciphertext must decrypt back to the original bytes with them.
	contents := req.GetContent().GetContents()
	key, iv, err := ExternalMediaEncryptionKeys(contents)
	if err != nil {
		t.Fatalf("outgoing contents rejected by the incoming extractor: %v", err)
	}
	if len(key) != 32 || len(iv) != 16 {
		t.Fatalf("unexpected key/iv sizes: %d/%d", len(key), len(iv))
	}
	dimensions, err := contentsDimensions(contents)
	if err != nil {
		t.Fatalf("contents dimensions: %v", err)
	}
	if dimensions[0] != wantWidth || dimensions[1] != wantHeight {
		t.Fatalf("contents dimensions = %dx%d, want %dx%d", dimensions[0], dimensions[1], wantWidth, wantHeight)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	decrypted := make([]byte, len(putBody))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, putBody)
	pad := int(decrypted[len(decrypted)-1])
	if pad <= 0 || pad > aes.BlockSize {
		t.Fatalf("invalid PKCS#7 padding %d", pad)
	}
	decrypted = decrypted[:len(decrypted)-pad]
	if !bytes.Equal(decrypted, mediaData) {
		t.Fatal("uploaded ciphertext does not decrypt to the original media bytes with the embedded key/IV")
	}
	// Video carries metadata.type=VIDEO and hasSound when the track has
	// audio; images must keep the proven shape without them.
	if wantMediaType == protos.MediaType_MEDIA_TYPE_VIDEO {
		if got, err := contentsMetadataVarint(contents, 2); err != nil || got != 1 {
			t.Fatalf("video metadata.type = %d, %v; want 1 (VIDEO)", got, err)
		}
		if got, err := contentsMetadataVarint(contents, 12); err != nil || got != 1 {
			t.Fatalf("video metadata.hasSound = %d, %v; want 1", got, err)
		}
		if got, err := contentsMetadataVarint(contents, 15); err != nil || got != 2000 {
			t.Fatalf("video metadata.mediaDurationMs = %d, %v; want 2000", got, err)
		}
		meta, err := contentsMetadataNode(contents)
		if err != nil {
			t.Fatal(err)
		}
		mediaID, err := mediaProtoField(meta, 18)
		if err != nil || len(mediaID) != 1 {
			t.Fatalf("video metadata.mediaId missing: %v", err)
		}
		if got, err := readUploadVarintField(mediaID[0], 1); err != nil || got != 1 {
			t.Fatalf("video metadata.mediaId.mediaListId = %d, %v; want 1", got, err)
		}
		charac, err := contentsPlaybackCharacteristics(contents)
		if err != nil {
			t.Fatal(err)
		}
		if po, err := mediaProtoField(charac, 7); err != nil || len(po) != 1 {
			t.Fatalf("video characteristics must carry playOnce(7): %v", err)
		}
		if inf, err := mediaProtoField(charac, 6); err != nil || len(inf) != 0 {
			t.Fatal("ordinary video characteristics must not carry displayDuration=infinite(6)")
		}
		if got, err := readUploadVarintField(charac, 5); err != nil || got != 1 {
			t.Fatalf("video characteristics.hasSound = %d, %v; want 1", got, err)
		}
	} else {
		if got, err := contentsMetadataVarint(contents, 2); err != nil || got != 0 {
			t.Fatalf("image contents must not carry metadata.type, got %d err %v", got, err)
		}
		charac, err := contentsPlaybackCharacteristics(contents)
		if err != nil {
			t.Fatal(err)
		}
		if po, err := mediaProtoField(charac, 7); err != nil || len(po) != 1 {
			t.Fatalf("image characteristics must carry playOnce(7): %v", err)
		}
		if inf, err := mediaProtoField(charac, 6); err != nil || len(inf) != 0 {
			t.Fatal("image characteristics must not carry displayDuration=infinite(6)")
		}
		if got, err := readUploadVarintField(charac, 5); err != nil || got != 0 {
			t.Fatalf("image characteristics must not carry hasSound, got %d err %v", got, err)
		}
	}
}

// contentsMetadataNode digs the MediaMetadata message out of the outgoing
// Contents payload (Contents.3.3.5.1.1).
func contentsMetadataNode(contents []byte) ([]byte, error) {
	external, err := mediaProtoField(contents, 3)
	if err != nil || len(external) != 1 {
		return nil, fmt.Errorf("externalMedia: %w", err)
	}
	media, err := mediaProtoField(external[0], 3)
	if err != nil || len(media) != 1 {
		return nil, fmt.Errorf("media wrapper: %w", err)
	}
	wrapper, err := mediaProtoField(media[0], 5)
	if err != nil || len(wrapper) != 1 {
		return nil, fmt.Errorf("descriptor wrapper: %w", err)
	}
	d1, err := mediaProtoField(wrapper[0], 1)
	if err != nil || len(d1) != 1 {
		return nil, fmt.Errorf("descriptor outer: %w", err)
	}
	d2, err := mediaProtoField(d1[0], 1)
	if err != nil || len(d2) != 1 {
		return nil, fmt.Errorf("metadata: %w", err)
	}
	return d2[0], nil
}

func contentsMetadataVarint(contents []byte, field protowire.Number) (uint64, error) {
	node, err := contentsMetadataNode(contents)
	if err != nil {
		return 0, err
	}
	return readUploadVarintField(node, field)
}

// contentsPlaybackCharacteristics digs the Playback.playbackCharacteristics
// message out of the outgoing Contents payload (Contents.3.3.5.2).
func contentsPlaybackCharacteristics(contents []byte) ([]byte, error) {
	external, err := mediaProtoField(contents, 3)
	if err != nil || len(external) != 1 {
		return nil, fmt.Errorf("externalMedia: %w", err)
	}
	media, err := mediaProtoField(external[0], 3)
	if err != nil || len(media) != 1 {
		return nil, fmt.Errorf("media wrapper: %w", err)
	}
	playback, err := mediaProtoField(media[0], 5)
	if err != nil || len(playback) != 1 {
		return nil, fmt.Errorf("playback: %w", err)
	}
	charac, err := mediaProtoField(playback[0], 2)
	if err != nil || len(charac) != 1 {
		return nil, fmt.Errorf("playbackCharacteristics: %w", err)
	}
	return charac[0], nil
}

// contentsDimensions digs the width/height pair out of the outgoing Contents
// payload (Contents.3.3.5.1.1.5.{1,2}).
func contentsDimensions(contents []byte) ([2]int, error) {
	external, err := mediaProtoField(contents, 3)
	if err != nil || len(external) != 1 {
		return [2]int{}, fmt.Errorf("externalMedia: %w", err)
	}
	media, err := mediaProtoField(external[0], 3)
	if err != nil || len(media) != 1 {
		return [2]int{}, fmt.Errorf("media wrapper: %w", err)
	}
	wrapper, err := mediaProtoField(media[0], 5)
	if err != nil || len(wrapper) != 1 {
		return [2]int{}, fmt.Errorf("descriptor wrapper: %w", err)
	}
	d1, err := mediaProtoField(wrapper[0], 1)
	if err != nil || len(d1) != 1 {
		return [2]int{}, fmt.Errorf("descriptor outer: %w", err)
	}
	d2, err := mediaProtoField(d1[0], 1)
	if err != nil || len(d2) != 1 {
		return [2]int{}, fmt.Errorf("metadata: %w", err)
	}
	dim, err := mediaProtoField(d2[0], 5)
	if err != nil || len(dim) != 1 {
		return [2]int{}, fmt.Errorf("dimensions: %w", err)
	}
	width, err := readUploadVarintField(dim[0], 1)
	if err != nil {
		return [2]int{}, err
	}
	height, err := readUploadVarintField(dim[0], 2)
	if err != nil {
		return [2]int{}, err
	}
	return [2]int{int(width), int(height)}, nil
}

func TestSendMediaPNGHappyPath(t *testing.T) {
	runSendMediaHappyPath(t, testPNGBytes, "image/png", 4, 3, protos.MediaType_MEDIA_TYPE_IMAGE)
}

func TestSendMediaWebPHappyPath(t *testing.T) {
	runSendMediaHappyPath(t, testWebPVP8(640, 480, false), "image/webp", 640, 480, protos.MediaType_MEDIA_TYPE_IMAGE)
}

func TestSendMediaJPEGHappyPathUnchanged(t *testing.T) {
	runSendMediaHappyPath(t, testJPEG, "image/jpeg", 3, 2, protos.MediaType_MEDIA_TYPE_IMAGE)
}

// srvURL returns the URL of the test server currently registered for t.
// The happy-path handler needs it before newAuthedTLSMediaClient returns, so
// it is stashed via a package-level test var set inside that helper.
func srvURL(t *testing.T) string {
	t.Helper()
	if tlsMediaServerURL == "" {
		t.Fatal("TLS media server URL not initialized")
	}
	return tlsMediaServerURL
}

var tlsMediaServerURL string

func TestImageDimensionsJPEG(t *testing.T) {
	width, height, err := imageDimensions(testJPEG)
	if err != nil {
		t.Fatal(err)
	}
	if width != 3 || height != 2 {
		t.Fatalf("unexpected dimensions %dx%d", width, height)
	}
	if !isJPEGData(testJPEG) {
		t.Fatal("JPEG magic not detected")
	}
}

func TestUploadEncryptedMedia(t *testing.T) {
	var received []byte
	var receivedType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		receivedType = r.Header.Get("Content-Type")
		received = make([]byte, r.ContentLength)
		_, _ = readFull(r, received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	c := &Client{http: server.Client()}
	encrypted := &EncryptedMedia{Ciphertext: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Key: make([]byte, 32), IV: make([]byte, 16)}
	descriptor, err := c.UploadEncryptedMedia(context.Background(), &UploadLocation{
		PutURL:       server.URL + "/bucket/object?x-amz-acl=public-read",
		MediaIDToken: "tok_1",
		MediaIDBase:  "tok",
	}, encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if receivedType != "application/octet-stream" {
		t.Fatalf("unexpected upload content type: %q", receivedType)
	}
	if !bytes.Equal(received, encrypted.Ciphertext) {
		t.Fatal("uploaded bytes mismatch")
	}
	tokens, err := mediaProtoField(descriptor, 1)
	if err != nil || len(tokens) != 1 || string(tokens[0]) != "tok_1" {
		t.Fatalf("descriptor token mismatch: %v %v", tokens, err)
	}
	// Server failure must surface as an error.
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	}))
	defer failing.Close()
	if _, err := c.UploadEncryptedMedia(context.Background(), &UploadLocation{PutURL: failing.URL + "/x", MediaIDToken: "t_1", MediaIDBase: "t"}, encrypted); err == nil {
		t.Fatal("failed PUT must be rejected")
	}
}

func readFull(r *http.Request, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Body.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// --- Outgoing voice notes (experimental audio/mp4) ---

// testMP4VoiceNote builds a minimal structurally valid audio-only MP4:
// ftyp + moov{trak{tkhd 0x0, mdia{hdlr soun, mdhd 1000Hz/durationMs}}}.
func testMP4VoiceNote(durationMs int) []byte {
	tkhd := make([]byte, 96)
	handler := make([]byte, 12)
	copy(handler[8:12], "soun")
	mdhd := make([]byte, 20)
	binary.BigEndian.PutUint32(mdhd[12:16], 1000)
	binary.BigEndian.PutUint32(mdhd[16:20], uint32(durationMs))
	ftyp := mp4Box("ftyp", []byte("isom\x00\x00\x00\x00isom"))
	audioTrak := mp4Box("trak", bytes.Join([][]byte{
		mp4Box("tkhd", tkhd),
		mp4Box("mdia", bytes.Join([][]byte{mp4Box("hdlr", handler), mp4Box("mdhd", mdhd)}, nil)),
	}, nil))
	return bytes.Join([][]byte{ftyp, mp4Box("moov", audioTrak)}, nil)
}

func TestClassifyOutgoingVoiceNote(t *testing.T) {
	info, err := classifyOutgoingMedia(testMP4VoiceNote(2993), "audio/mp4")
	if err != nil {
		t.Fatal(err)
	}
	if info.MIME != "audio/mp4" || info.MediaType != protos.MediaType_MEDIA_TYPE_AUDIO || !info.HasAudio {
		t.Fatalf("voice note classification = %+v, want audio/mp4 MEDIA_TYPE_AUDIO", info)
	}
	if info.DurationMs != 2993 {
		t.Fatalf("voice note duration = %dms, want 2993", info.DurationMs)
	}
	if info.Width != 0 || info.Height != 0 {
		t.Fatalf("voice note must not carry dimensions: %+v", info)
	}
	// m4a aliases are the same container.
	if _, err := classifyOutgoingMedia(testMP4VoiceNote(1000), "audio/x-m4a"); err != nil {
		t.Fatal(err)
	}
	// Without an audio/* declaration the bytes follow the proven video path
	// (audio-only MP4 is not a valid video), so no silent reclassification.
	if _, err := classifyOutgoingMedia(testMP4VoiceNote(2993), ""); err == nil {
		t.Fatal("audio-only MP4 without audio MIME must not be accepted")
	}
}

func TestClassifyOutgoingVoiceNoteRejectsOthers(t *testing.T) {
	// Non-MP4 containers are rejected cleanly (no transcoding).
	if _, err := classifyOutgoingMedia(bytes.Repeat([]byte("OggS"), 40), "audio/ogg"); err == nil {
		t.Fatal("OGG voice notes must be rejected")
	}
	if _, err := classifyOutgoingMedia(bytes.Repeat([]byte{0x1A, 0x45, 0xDF, 0xA3}, 40), "audio/webm"); err == nil {
		t.Fatal("WebM voice notes must be rejected")
	}
	if _, err := classifyOutgoingMedia([]byte("ID3 garbage mp3 frames"), "audio/mpeg"); err == nil {
		t.Fatal("MP3 voice notes must be rejected")
	}
	// MP4 declared as image/* keeps the MIME mismatch rejection.
	if _, err := classifyOutgoingMedia(testMP4VoiceNote(2993), "image/png"); err == nil {
		t.Fatal("audio/mp4 declared image/png must be rejected")
	}
}

func TestClassifyOutgoingVoiceNoteRejectsVideoTrack(t *testing.T) {
	// A full video MP4 sent as m.audio must not become a voice note.
	if _, err := classifyOutgoingMedia(testMP4(640, 360, true), "audio/mp4"); err == nil {
		t.Fatal("MP4 with a video track must be rejected as a voice note")
	}
}

func TestEncodeNoteContentsStructure(t *testing.T) {
	key := bytes.Repeat([]byte{0x33}, 32)
	iv := bytes.Repeat([]byte{0x44}, 16)
	info := &OutgoingMediaInfo{MIME: "audio/mp4", MediaType: protos.MediaType_MEDIA_TYPE_AUDIO, HasAudio: true, DurationMs: 2993}
	contents := encodeNoteContents(info, key, iv)
	note, err := mediaProtoField(contents, 6)
	if err != nil || len(note) != 1 {
		t.Fatalf("Contents.note missing: %v", err)
	}
	audioOneof, err := mediaProtoField(note[0], 1)
	if err != nil || len(audioOneof) != 1 {
		t.Fatalf("Note.note audio oneof missing: %v", err)
	}
	audioNote, err := mediaProtoField(audioOneof[0], 1)
	if err != nil || len(audioNote) != 1 {
		t.Fatalf("AudioNote.note missing: %v", err)
	}
	userLocale, err := mediaProtoField(audioOneof[0], 3)
	if err != nil || len(userLocale) != 1 || string(userLocale[0]) != "en" {
		t.Fatalf("AudioNote.userLocale mismatch: %v", err)
	}
	details := audioNote[0]
	typeField, err := readUploadVarintField(details, 2)
	if err != nil || typeField != 4 {
		t.Fatalf("note type = %d %v, want 4 (AUDIO)", typeField, err)
	}
	hasSound, err := readUploadVarintField(details, 8)
	if err != nil || hasSound != 1 {
		t.Fatalf("note hasSound = %d %v, want 1", hasSound, err)
	}
	enc, err := mediaProtoField(details, 4)
	if err != nil || len(enc) != 1 {
		t.Fatalf("note encryptionInfo missing: %v", err)
	}
	gotKeyB64, err := mediaProtoField(enc[0], 1)
	if err != nil || len(gotKeyB64) != 1 {
		t.Fatalf("note encryptionInfo key missing: %v", err)
	}
	gotIVB64, err := mediaProtoField(enc[0], 2)
	if err != nil || len(gotIVB64) != 1 {
		t.Fatalf("note encryptionInfo iv missing: %v", err)
	}
	gotKey, err := base64.StdEncoding.DecodeString(string(gotKeyB64[0]))
	if err != nil || !bytes.Equal(gotKey, key) {
		t.Fatalf("note key mismatch: %v", err)
	}
	gotIV, err := base64.StdEncoding.DecodeString(string(gotIVB64[0]))
	if err != nil || !bytes.Equal(gotIV, iv) {
		t.Fatalf("note iv mismatch: %v", err)
	}
	mediaDurationMs, err := readUploadVarintField(details, 13)
	if err != nil || mediaDurationMs != 2993 {
		t.Fatalf("note mediaDurationMs = %d %v, want 2993", mediaDurationMs, err)
	}
	duration, err := mediaProtoField(details, 12)
	if err != nil || len(duration) != 1 {
		t.Fatalf("note duration missing: %v", err)
	}
	durationSeconds, err := readUploadVarintField(duration[0], 3)
	if err != nil || durationSeconds != 2 {
		t.Fatalf("note durationSeconds = %d %v, want 2 (2993ms truncated like web durationInSec)", durationSeconds, err)
	}
	if dims, err := mediaProtoField(details, 5); err == nil && len(dims) > 0 {
		t.Fatal("voice notes must not carry dimensions")
	}
}

func TestSendMediaVoiceNoteHappyPath(t *testing.T) {
	const createdMessageID = uint64(9090)
	mediaData := testMP4VoiceNote(2993)
	var (
		putContentType  string
		putBody         []byte
		createRequests  []*protos.CreateContentMessageRequest
		uploadLocations = 0
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/put":
			putContentType = r.Header.Get("Content-Type")
			putBody = append([]byte{}, body...)
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/getUploadLocations"):
			uploadLocations++
			w.Write(grpcWebData(encodeTestUploadLocationsResponse(srvURL(t)+"/put", "wO87suY3U6Ss473nIx9ba_1", "wO87suY3U6Ss473nIx9ba", true)))
		case r.URL.Path == urlPath(paths.CREATE_CONTENT_MESSAGE):
			frame, err := methods.ReadResponseFrame(body)
			if err != nil {
				t.Fatalf("create frame unreadable: %v", err)
			}
			var req protos.CreateContentMessageRequest
			if err := protos.DecodeProtoMessage(frame, &req); err != nil {
				t.Fatalf("create proto undecodable: %v", err)
			}
			createRequests = append(createRequests, &req)
			resp := &protos.CreateContentMessageResponse{
				Result: []*protos.CreateContentMessageResult{{
					Success: true,
					DestinationRes: &protos.CreateContentMessageResult_ConversationDestinationResult{
						ConversationDestinationResult: &protos.ConversationDestinationResult{
							CreatedMessageId: createdMessageID,
						},
					},
				}},
			}
			payload, _ := protos.EncodeProtoMessage(resp)
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTLSMediaClient(t, handler)
	overrideListingURL(t, srv)
	origWebBase := paths.WEB_BASE_URL
	paths.WEB_BASE_URL = srv.URL
	t.Cleanup(func() { paths.WEB_BASE_URL = origWebBase })
	messageID, err := c.SendMedia(context.Background(), "aaaaaaaa-0000-0000-0000-000000000001", MediaAttachment{
		MimeType: "audio/mp4",
		Data:     mediaData,
	}, "")
	if err != nil {
		t.Fatalf("SendMedia: %v", err)
	}
	if messageID != "9090" {
		t.Fatalf("SendMedia returned %q, want nested createdMessageId 9090", messageID)
	}
	if uploadLocations != 1 || putContentType != "application/octet-stream" || len(putBody) == 0 || len(putBody) == len(mediaData) {
		t.Fatalf("upload pipeline mismatch: locations=%d put=%s cipher=%d clear=%d", uploadLocations, putContentType, len(putBody), len(mediaData))
	}
	if len(createRequests) != 1 {
		t.Fatalf("CreateContentMessage calls = %d, want 1", len(createRequests))
	}
	env := createRequests[0].GetContent()
	if env.GetContentType() != protos.ContentType_NOTE {
		t.Fatalf("content type = %v, want NOTE", env.GetContentType())
	}
	if env.GetSavePolicy() != protos.ContentEnvelope_SavePolicy_LIFETIME {
		t.Fatalf("save policy = %v, want LIFETIME", env.GetSavePolicy())
	}
	if !env.GetAudioNote().GetAllowsTranscription() {
		t.Fatal("voice note must carry audioNote.allowsTranscription")
	}
	refs := env.GetMediaReferenceLists()
	if len(refs) != 1 || len(refs[0].GetReference()) != 1 || refs[0].GetReference()[0].GetMediaType() != protos.MediaType_MEDIA_TYPE_AUDIO {
		t.Fatalf("voice note media reference mismatch: %#v", refs)
	}
	// The note contents carry the same key/IV the ciphertext used.
	contents := env.GetContents()
	note, err := mediaProtoField(contents, 6)
	if err != nil || len(note) != 1 {
		t.Fatalf("Contents.note missing: %v", err)
	}
	audioOneof, err := mediaProtoField(note[0], 1)
	if err != nil || len(audioOneof) != 1 {
		t.Fatalf("Note.note missing: %v", err)
	}
	audioNote, err := mediaProtoField(audioOneof[0], 1)
	if err != nil || len(audioNote) != 1 {
		t.Fatalf("AudioNote.note missing: %v", err)
	}
	enc, err := mediaProtoField(audioNote[0], 4)
	if err != nil || len(enc) != 1 {
		t.Fatalf("note encryptionInfo missing: %v", err)
	}
	keyB64, err := mediaProtoField(enc[0], 1)
	if err != nil || len(keyB64) != 1 {
		t.Fatalf("note key missing: %v", err)
	}
	ivB64, err := mediaProtoField(enc[0], 2)
	if err != nil || len(ivB64) != 1 {
		t.Fatalf("note iv missing: %v", err)
	}
	key, err := base64.StdEncoding.DecodeString(string(keyB64[0]))
	if err != nil {
		t.Fatal(err)
	}
	iv, err := base64.StdEncoding.DecodeString(string(ivB64[0]))
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	decrypted := make([]byte, len(putBody))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, putBody)
	pad := int(decrypted[len(decrypted)-1])
	if pad <= 0 || pad > aes.BlockSize {
		t.Fatalf("invalid PKCS#7 padding %d", pad)
	}
	decrypted = decrypted[:len(decrypted)-pad]
	if !bytes.Equal(decrypted, mediaData) {
		t.Fatal("uploaded ciphertext does not decrypt to the original audio bytes with the note key/IV")
	}
}

func TestSendMediaRejectsUnsupportedVoiceFormats(t *testing.T) {
	c := &Client{}
	// OGG/Opus voice notes are rejected cleanly, not transcoded.
	ogg := bytes.Repeat([]byte("OggS"), 64)
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "audio/ogg", Data: ogg}, ""); err == nil {
		t.Fatal("audio/ogg voice note must be rejected")
	}
	// A real video MP4 declared as audio must not become a voice note.
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "audio/mp4", Data: testMP4(640, 360, true)}, ""); err == nil {
		t.Fatal("video MP4 declared audio/mp4 must be rejected")
	}
	// Malformed MP4 bytes declared as audio must be rejected.
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "audio/mp4", Data: bytes.Repeat([]byte{0x00}, 64)}, ""); err == nil {
		t.Fatal("malformed audio/mp4 must be rejected")
	}
}
