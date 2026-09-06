package snapapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	contents := encodeExternalMediaContents(1725000000000, 640, 480, key, iv)
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

func TestSendMediaRejectsNonJPEG(t *testing.T) {
	c := &Client{}
	png := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{0}, 64)...)
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "image/png", Data: png}, ""); err == nil {
		t.Fatal("PNG must be rejected")
	} else if !strings.Contains(err.Error(), "image/jpeg") {
		t.Fatalf("unexpected rejection error: %v", err)
	}
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "image/jpeg", Data: nil}, ""); err == nil {
		t.Fatal("empty media must be rejected")
	}
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "image/jpeg", Data: bytes.Repeat([]byte{0x01}, maxOutgoingMediaSize+1)}, ""); err == nil {
		t.Fatal("oversized media must be rejected")
	}
	// JPEG magic with a non-JPEG declared MIME is rejected too.
	broken := append([]byte{0xFF, 0xD8, 0xFF}, bytes.Repeat([]byte{0}, 64)...)
	if _, err := c.SendMedia(context.Background(), "chat-1", MediaAttachment{MimeType: "video/mp4", Data: broken}, ""); err == nil {
		t.Fatal("declared non-JPEG MIME must be rejected")
	}
}

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
