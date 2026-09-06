package snapapi

import (
	"bytes"
	"context"
	"encoding/hex"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xzer/snapper/protos"
)

func TestDownloadMediaWithInfoRejectsDescriptorOnlyAttachment(t *testing.T) {
	// Live capture: 61-byte contentObject proto holding a media ID.
	const descriptorHex = "0a17753930436a34386878644b733968576c794a734b635f3112221215753930436a34386878644b733968576c794a734b633201034801500460017002"
	descriptor := testHexBytes(t, descriptorHex)
	if got := MediaDescriptorID(descriptor); got != "u90Cj48hxdKs9hWlyJsKc_1" {
		t.Fatalf("MediaDescriptorID = %q, want u90Cj48hxdKs9hWlyJsKc_1", got)
	}
	if got := MediaDescriptorID([]byte{0xff, 0xd8, 0xff, 0xe0, 'j', 'p', 'e', 'g'}); got != "" {
		t.Fatalf("MediaDescriptorID(jpeg) = %q, want empty", got)
	}
	if got := MediaDescriptorID(bytes.Repeat([]byte{0x0a, 'a'}, 400)); got != "" {
		t.Fatalf("MediaDescriptorID(oversized) = %q, want empty", got)
	}

	client := &Client{}
	_, _, _, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:       "media-1",
		Data:     append([]byte(nil), descriptor...),
		MimeType: "image/jpeg",
		Kind:     MediaKindImage,
	})
	if err == nil {
		t.Fatal("expected descriptor-only attachment to be rejected")
	}
	if !strings.Contains(err.Error(), "resolver") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMediaDescriptorRecognizesBothLiveShapes(t *testing.T) {
	// Live capture (message 4114): 34-byte nested field-2 descriptor.
	const nestedHex = "12201215635652757649595876626b6169397a665a48735a4d320103480250046001"
	nested := testHexBytes(t, nestedHex)
	parsed := ParseMediaDescriptor(nested)
	if !parsed.Detected || parsed.Shape != "field2-nested" || parsed.ContentObjectID != "cVRuvIYXvbkai9zfZHsZM" {
		t.Fatalf("nested descriptor parse = %+v", parsed)
	}
	if got := MediaDescriptorID(nested); got != "cVRuvIYXvbkai9zfZHsZM" {
		t.Fatalf("MediaDescriptorID(nested) = %q", got)
	}

	// Live capture: 61-byte direct field-1 descriptor.
	const directHex = "0a17753930436a34386878644b733968576c794a734b635f3112221215753930436a34386878644b733968576c794a734b633201034801500460017002"
	direct := testHexBytes(t, directHex)
	parsed = ParseMediaDescriptor(direct)
	if !parsed.Detected || parsed.Shape != "field1-direct" || parsed.ContentObjectID != "u90Cj48hxdKs9hWlyJsKc_1" {
		t.Fatalf("direct descriptor parse = %+v", parsed)
	}

	// The descriptor must be preserved verbatim for the resolver request: both
	// shapes are accepted as-is by ResolveMediaDescriptor.
	for _, descriptor := range [][]byte{nested, direct} {
		if MediaDescriptorID(descriptor) == "" {
			t.Fatal("recognized descriptor lost")
		}
	}

	// Real media magic bytes and garbage must never parse as descriptors.
	for _, payload := range [][]byte{
		[]byte("\xff\xd8\xff\xe0jpeg"),
		[]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDRabcdabcd"),
		[]byte("\x00\x00\x00\x18ftypisom"),
		bytes.Repeat([]byte{0x12, 0xff}, 64),
		[]byte{0x12, 0x20, 0x12, 0x15},
	} {
		if got := ParseMediaDescriptor(payload); got.Detected {
			t.Fatalf("misdetected %x as descriptor %+v", payload[:8], got)
		}
	}
}

func TestDownloadMediaWithInfoNeverDecryptsRecognizedDescriptor(t *testing.T) {
	// Even with key material present, a recognized descriptor must go to the
	// resolver path, never to inline-media decryption.
	const nestedHex = "12201215635652757649595876626b6169397a665a48735a4d320103480250046001"
	client := &Client{}
	_, _, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:       "media-1",
		Data:     testHexBytes(t, nestedHex),
		Key:      bytes.Repeat([]byte{7}, 32),
		IV:       bytes.Repeat([]byte{8}, 16),
		MimeType: "image/jpeg",
		Kind:     MediaKindImage,
	})
	if err == nil {
		t.Fatal("descriptor without an authenticated client must fail at the resolver")
	}
	if info.Source != "descriptor-rpc" || info.DescriptorShape != "field2-nested" {
		t.Fatalf("unexpected info: %+v", info)
	}
	if info.CiphertextBytes != 0 {
		t.Fatalf("descriptor treated as ciphertext: %+v", info)
	}
}

func TestDownloadMediaWithInfoRejectsTinyUnsniffableImageHint(t *testing.T) {
	client := &Client{}
	_, _, _, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:       "media-1",
		Data:     bytes.Repeat([]byte{0x42}, 61),
		MimeType: "image/jpeg",
		Kind:     MediaKindImage,
	})
	if err == nil {
		t.Fatal("expected tiny image-labeled payload without magic bytes to be rejected")
	}
	if !strings.Contains(err.Error(), "too small") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testHexBytes(t *testing.T, value string) []byte {
	t.Helper()
	data, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDownloadMediaWithInfoClassifiesURLDownload(t *testing.T) {
	body := testPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg") // Deliberately incorrect hint.
		_, _ = w.Write(body)
	}))
	defer server.Close()

	client := &Client{http: server.Client()}
	data, mimeType, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:  "media-1",
		URL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte(body)) {
		t.Fatalf("downloaded bytes = %x, want %x", data, []byte(body))
	}
	if mimeType != "image/png" {
		t.Fatalf("mimeType = %q, want image/png", mimeType)
	}
	if info.Source != "url" || info.DecryptPath != "none" || info.Bytes != len(body) || info.ContentType != "image/png" {
		t.Fatalf("unexpected download info: %#v", info)
	}
}

func TestDownloadMediaWithInfoClassifiesInlineDownload(t *testing.T) {
	client := &Client{}
	payload := testPNG(t)
	data, mimeType, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:       "media-1",
		Data:     payload,
		MimeType: "application/octet-stream",
		Kind:     MediaKindImage,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("downloaded bytes = %x", data)
	}
	if mimeType != "image/png" {
		t.Fatalf("mimeType = %q, want image/png", mimeType)
	}
	if info.Source != "inline" || info.DecryptPath != "none" || info.Bytes != len(payload) || info.ContentType != "image/png" {
		t.Fatalf("unexpected download info: %#v", info)
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRejectOpaqueAndTruncatedMedia(t *testing.T) {
	for _, payload := range [][]byte{bytes.Repeat([]byte{0x42}, 4096), []byte("\xff\xd8\xff\xe0jpeg"), []byte("\x89PNG\r\n\x1a\n"), []byte("\x00\x00\x00\x18ftypisom\x00\x00\x00\x00")} {
		if err := ValidateRenderableMediaPayload(payload, "image/jpeg", MediaKindImage); err == nil {
			t.Fatalf("accepted invalid media %x", payload[:8])
		}
	}
}

func TestDescriptorWithURLFetchesRealBytes(t *testing.T) {
	payload := testPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload) }))
	defer server.Close()
	client := &Client{http: server.Client()}
	data, mime, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{Data: append([]byte{10, 8}, []byte("media-id")...), URL: server.URL, Kind: MediaKindImage})
	if err != nil || !bytes.Equal(data, payload) || mime != "image/png" || info.Source != "url" {
		t.Fatalf("descriptor URL result: mime=%s info=%+v err=%v", mime, info, err)
	}
}

func TestMediaDecryptionFailureNeverReturnsCiphertext(t *testing.T) {
	client := &Client{}
	data, _, _, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{Data: bytes.Repeat([]byte{0x42}, 4096), Key: make([]byte, 32), IV: make([]byte, 17), Kind: MediaKindFile})
	if err == nil || len(data) != 0 {
		t.Fatal("failed decryption exposed ciphertext")
	}
}

// referenceEnvelope builds an EXTERNAL_MEDIA reference-list envelope like
// message 772: mediaListId 0 carries the original media and mediaListId 2 a
// server-derived thumbnail rendition (descriptor ID suffix ".1020") declared
// by the envelope's thumbnails field.
func referenceEnvelope(originalListID, thumbnailListID uint64, declareThumbnail bool) *protos.ContentEnvelope {
	envelope := &protos.ContentEnvelope{
		ContentType: protos.ContentType_EXTERNAL_MEDIA,
		MediaReferenceLists: []*protos.ContentEnvelope_MediaReferenceList{{
			Reference: []*protos.MediaReference{
				{MediaType: protos.MediaType_MEDIA_TYPE_IMAGE, MediaListId: originalListID, ContentObject: []byte("original-descriptor")},
				{MediaType: protos.MediaType_MEDIA_TYPE_IMAGE, MediaListId: thumbnailListID, ContentObject: []byte("thumbnail-descriptor")},
			},
		}},
	}
	if declareThumbnail {
		envelope.Thumbnails = &protos.ContentEnvelope_Thumbnails{
			Thumbnails: []*protos.ThumbnailInfo{{MediaId: &protos.MediaId{MediaListId: thumbnailListID}}},
		}
	}
	return envelope
}

func TestThumbnailRenditionReferenceExcludedFromAttachments(t *testing.T) {
	media := mediaAttachmentsFromEnvelope(referenceEnvelope(0, 2, true), "772")
	if len(media) != 1 {
		t.Fatalf("attachment count = %d, want 1 (thumbnail rendition excluded)", len(media))
	}
	if !bytes.Equal(media[0].Data, []byte("original-descriptor")) {
		t.Fatalf("kept attachment = %q, want the original descriptor", media[0].Data)
	}
}

func TestUndeclaredThumbnailReferencesStayAttachments(t *testing.T) {
	// Without the envelope thumbnail declaration, both references must stay:
	// the exclusion follows only the envelope's own structural statement.
	media := mediaAttachmentsFromEnvelope(referenceEnvelope(0, 2, false), "772")
	if len(media) != 2 {
		t.Fatalf("attachment count = %d, want 2", len(media))
	}
}
