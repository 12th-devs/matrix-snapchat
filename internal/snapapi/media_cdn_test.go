package snapapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func makeTestJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
	return buf.Bytes()
}

func aesCBCEncrypt(t *testing.T, key, iv, plaintext []byte) []byte {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	if len(iv) != aes.BlockSize {
		t.Fatalf("iv length %d", len(iv))
	}
	pad := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := append(append([]byte{}, plaintext...), bytes.Repeat([]byte{byte(pad)}, pad)...)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
	return out
}

func appendVarintField(b []byte, num protowire.Number, v uint64) []byte {
	return protowire.AppendVarint(protowire.AppendTag(b, num, protowire.VarintType), v)
}

func appendBytesField(b []byte, num protowire.Number, v []byte) []byte {
	return protowire.AppendBytes(protowire.AppendTag(b, num, protowire.BytesType), v)
}

// nestedDescriptor builds the live 34-byte shape from message 4114: outer
// field 2 wrapping an inner message with field 2 = ASCII content object ID
// plus the locally-resolvable auxiliary pattern f6={3}, f9=2, f10=use case,
// f12=1.
func nestedDescriptor(t *testing.T, id string, useCase uint64) []byte {
	t.Helper()
	inner := appendBytesField(nil, 2, []byte(id))
	inner = appendBytesField(inner, 6, []byte{3})
	inner = appendVarintField(inner, 9, 2)
	inner = appendVarintField(inner, 10, useCase)
	inner = appendVarintField(inner, 12, 1)
	return appendBytesField(nil, 2, inner)
}

func TestMediaContentObjectURLConstruction(t *testing.T) {
	descriptor := nestedDescriptor(t, "abcdefgh", 4)
	got, err := MediaContentObjectURL(descriptor)
	if err != nil {
		t.Fatalf("MediaContentObjectURL: %v", err)
	}
	want := "https://cf-st.sc-cdn.net/c/abcdefgh?uc=4"
	if got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
	parsed := ParseMediaDescriptor(descriptor)
	if !parsed.Detected || parsed.ContentObjectID != "abcdefgh" || parsed.UseCase != 4 {
		t.Fatalf("descriptor parse = %+v", parsed)
	}
	if parsed.CDNPathSegment != mediaCDNPathSegmentContent {
		t.Fatalf("field2-nested CDNPathSegment = %q, want %q", parsed.CDNPathSegment, mediaCDNPathSegmentContent)
	}
	if parsed.CDNObjectID != "abcdefgh" || !parsed.CDNEligible {
		t.Fatalf("field2-nested CDN fields = %+v", parsed)
	}
}

func TestMediaContentObjectURLRejectsUnsafeID(t *testing.T) {
	for _, id := range []string{"../../etc", "id with space", "id?query=1", "id#frag", "aaa"} {
		descriptor := nestedDescriptor(t, id, 4)
		if _, err := MediaContentObjectURL(descriptor); err == nil {
			t.Fatalf("expected rejection for %q", id)
		}
	}
	if _, err := MediaContentObjectURL([]byte("not-a-descriptor")); err == nil {
		t.Fatal("expected rejection for unrecognized payload")
	}
}

func TestDescriptorURLTargetsProductionCDNHost(t *testing.T) {
	// The construction must target the real proven CDN shape independent of
	// any test server; retrieval tests redirect at the HTTP client level.
	descriptor := nestedDescriptor(t, "cVRuvIYXvbkai9zfZHsZM", 4)
	got, err := MediaContentObjectURL(descriptor)
	if err != nil {
		t.Fatalf("MediaContentObjectURL: %v", err)
	}
	if got != "https://cf-st.sc-cdn.net/c/cVRuvIYXvbkai9zfZHsZM?uc=4" {
		t.Fatalf("URL = %q", got)
	}
}

// directDescriptor builds the live 61-byte shape from message 745: outer
// field 1 = ASCII content object ID, outer field 2 = nested ContentDescriptor
// with field 2 = nested contentId, field 6 = {4}, f9=1, f10=use case, f12=1,
// f14=2. Snapchat Web's bolt resolver maps this shape to
// https://cf-st.sc-cdn.net/d/{nested contentId}?uc={use case}.
func directDescriptor(t *testing.T, outerID, nestedID string, useCase uint64) []byte {
	t.Helper()
	inner := appendBytesField(nil, 2, []byte(nestedID))
	inner = appendBytesField(inner, 6, []byte{4})
	inner = appendVarintField(inner, 9, 1)
	inner = appendVarintField(inner, 10, useCase)
	inner = appendVarintField(inner, 12, 1)
	inner = appendVarintField(inner, 14, 2)
	out := appendBytesField(nil, 1, []byte(outerID))
	return append(out, appendBytesField(nil, 2, inner)...)
}

func TestField1DirectDescriptorCDNDerivation(t *testing.T) {
	descriptor := directDescriptor(t, "YdVQtQ3u6WWy5ab3cYcUm_1", "YdVQtQ3u6WWy5ab3cYcUm", 4)
	parsed := ParseMediaDescriptor(descriptor)
	if !parsed.Detected {
		t.Fatal("field1-direct descriptor not detected")
	}
	if parsed.Shape != mediaDescriptorShapeField1Direct {
		t.Fatalf("shape = %q", parsed.Shape)
	}
	if parsed.ContentObjectID != "YdVQtQ3u6WWy5ab3cYcUm_1" {
		t.Fatalf("ContentObjectID = %q", parsed.ContentObjectID)
	}
	if !parsed.CDNEligible {
		t.Fatalf("descriptor must be CDN eligible: %+v", parsed)
	}
	if parsed.CDNObjectID != "YdVQtQ3u6WWy5ab3cYcUm" {
		t.Fatalf("CDNObjectID = %q", parsed.CDNObjectID)
	}
	if parsed.CDNPathSegment != mediaCDNPathSegmentDirect {
		t.Fatalf("CDNPathSegment = %q", parsed.CDNPathSegment)
	}
	if parsed.UseCase != 4 {
		t.Fatalf("UseCase = %d", parsed.UseCase)
	}
	got, err := MediaContentObjectURL(descriptor)
	if err != nil {
		t.Fatalf("MediaContentObjectURL: %v", err)
	}
	want := "https://cf-st.sc-cdn.net/d/YdVQtQ3u6WWy5ab3cYcUm?uc=4"
	if got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}

func TestField1DirectWithoutNestedContentIDStaysResolverRPC(t *testing.T) {
	// A field1-direct payload whose nested field 2 carries no valid ASCII
	// contentId must not derive a CDN URL; it keeps the resolver fallback.
	inner := appendVarintField(nil, 10, 4)
	inner = appendVarintField(inner, 12, 1)
	descriptor := appendBytesField(nil, 1, []byte("YdVQtQ3u6WWy5ab3cYcUm_1"))
	descriptor = append(descriptor, appendBytesField(nil, 2, inner)...)
	parsed := ParseMediaDescriptor(descriptor)
	if !parsed.Detected || parsed.Shape != mediaDescriptorShapeField1Direct {
		t.Fatalf("descriptor parse = %+v", parsed)
	}
	if parsed.CDNEligible {
		t.Fatal("unexpected CDN eligibility without nested contentId")
	}
	if _, err := MediaContentObjectURL(descriptor); err == nil {
		t.Fatal("expected MediaContentObjectURL to reject descriptor without nested contentId")
	}
}

func TestField1DirectAmbiguousNestedIDsStayResolverRPC(t *testing.T) {
	first := appendBytesField(nil, 2, []byte("nestedcontentida"))
	first = appendVarintField(first, 10, 4)
	second := appendBytesField(nil, 2, []byte("nestedcontentidb"))
	second = appendVarintField(second, 10, 4)
	descriptor := appendBytesField(nil, 1, []byte("YdVQtQ3u6WWy5ab3cYcUm_1"))
	descriptor = append(descriptor, appendBytesField(nil, 2, first)...)
	descriptor = append(descriptor, appendBytesField(nil, 2, second)...)
	parsed := ParseMediaDescriptor(descriptor)
	if parsed.CDNEligible {
		t.Fatal("unexpected CDN eligibility with ambiguous nested contentIds")
	}
	if _, err := MediaContentObjectURL(descriptor); err == nil {
		t.Fatal("expected MediaContentObjectURL to reject ambiguous descriptor")
	}
}

func TestField1DirectConflictingNestedUseCasesStayResolverRPC(t *testing.T) {
	// Same nested contentId with conflicting use cases: the URL uc value is
	// ambiguous, so the descriptor must stay ineligible.
	build := func(useCase uint64) []byte {
		inner := appendBytesField(nil, 2, []byte("YdVQtQ3u6WWy5ab3cYcUm"))
		inner = appendVarintField(inner, 10, useCase)
		inner = appendVarintField(inner, 12, 1)
		return appendBytesField(nil, 2, inner)
	}
	descriptor := appendBytesField(nil, 1, []byte("YdVQtQ3u6WWy5ab3cYcUm_1"))
	descriptor = append(descriptor, build(4)...)
	descriptor = append(descriptor, build(5)...)
	parsed := ParseMediaDescriptor(descriptor)
	if parsed.CDNEligible {
		t.Fatal("unexpected CDN eligibility with conflicting nested use cases")
	}
	if _, err := MediaContentObjectURL(descriptor); err == nil {
		t.Fatal("expected MediaContentObjectURL to reject conflicting use cases")
	}
}

func TestField1DirectMalformedSiblingIgnored(t *testing.T) {
	// A malformed (non-protobuf) field-2 sibling must be ignored, not poison
	// the one recognizable nested ContentDescriptor.
	nested := appendBytesField(nil, 2, []byte("YdVQtQ3u6WWy5ab3cYcUm"))
	nested = appendVarintField(nested, 10, 4)
	nested = appendVarintField(nested, 12, 1)
	valid := appendBytesField(nil, 2, nested)
	descriptor := appendBytesField(nil, 1, []byte("YdVQtQ3u6WWy5ab3cYcUm_1"))
	descriptor = append(descriptor, appendBytesField(nil, 2, []byte{0xff, 0xff, 0xff})...)
	descriptor = append(descriptor, valid...)
	parsed := ParseMediaDescriptor(descriptor)
	if !parsed.CDNEligible || parsed.CDNObjectID != "YdVQtQ3u6WWy5ab3cYcUm" || parsed.UseCase != 4 {
		t.Fatalf("descriptor parse = %+v", parsed)
	}
	if got, err := MediaContentObjectURL(descriptor); err != nil || got != "https://cf-st.sc-cdn.net/d/YdVQtQ3u6WWy5ab3cYcUm?uc=4" {
		t.Fatalf("URL = %q, err = %v", got, err)
	}
}

func TestParseMediaDescriptorNeverPanics(t *testing.T) {
	// Structural robustness: arbitrary bytes must never panic and must never
	// produce a descriptor whose CDN ID fails the URL-safety validation.
	var payload []byte
	for i := 0; i < 4000; i++ {
		payload = append(payload, byte(i*97+((i*i)%251)))
		payload = append(payload, 0x0a, 0x12)
		parsed := ParseMediaDescriptor(payload)
		if parsed.CDNEligible {
			if parsed.CDNObjectID == "" || !validMediaContentObjectID(parsed.CDNObjectID) {
				t.Fatalf("eligible descriptor with unsafe CDNObjectID %q", parsed.CDNObjectID)
			}
		}
	}
	if _, err := MediaContentObjectURL(payload); err != nil {
		return
	}
}

// TestField1DirectLiveDescriptorURLShape is an opt-in check against the exact
// live descriptor bytes captured for message 745. It asserts and prints only
// the URL SHAPE (scheme, host, /d/ path segment, uc value, ID length) — never
// credentials, tokens, or other secret material. Content object IDs are not
// secrets, but only their length is reported here.
func TestField1DirectLiveDescriptorURLShape(t *testing.T) {
	hex := strings.TrimSpace(os.Getenv("SNAPCHAT_LIVE_DESCRIPTOR_HEX"))
	if hex == "" {
		t.Skip("SNAPCHAT_LIVE_DESCRIPTOR_HEX not set")
	}
	raw, err := os.ReadFile(filepath.Clean(hex))
	if err == nil {
		hex = strings.TrimSpace(string(raw))
	}
	if len(hex)%2 != 0 {
		t.Fatalf("descriptor hex length %d is not even", len(hex))
	}
	data := make([]byte, len(hex)/2)
	for i := 0; i < len(data); i++ {
		if _, err := fmt.Sscanf(hex[i*2:i*2+2], "%02x", &data[i]); err != nil {
			t.Fatalf("invalid descriptor hex: %v", err)
		}
	}
	parsed := ParseMediaDescriptor(data)
	if !parsed.Detected {
		t.Fatal("live descriptor not detected")
	}
	if parsed.Shape != mediaDescriptorShapeField1Direct {
		t.Fatalf("shape = %q, want field1-direct", parsed.Shape)
	}
	if !parsed.CDNEligible || parsed.CDNPathSegment != mediaCDNPathSegmentDirect {
		t.Fatalf("CDN fields = %+v", parsed)
	}
	u, err := MediaContentObjectURL(data)
	if err != nil {
		t.Fatalf("MediaContentObjectURL: %v", err)
	}
	parsedURL, err := neturl.Parse(u)
	if err != nil {
		t.Fatalf("generated URL parse: %v", err)
	}
	if parsedURL.Scheme != "https" || parsedURL.Host != mediaCDNHost {
		t.Fatalf("URL = %s://%s, want https://%s", parsedURL.Scheme, parsedURL.Host, mediaCDNHost)
	}
	if !strings.HasPrefix(parsedURL.Path, "/d/") {
		t.Fatalf("path = %q, want /d/ prefix", parsedURL.Path)
	}
	if parsedURL.Query().Get(mediaCDNUseCaseParam) != strconv.FormatUint(parsed.UseCase, 10) {
		t.Fatalf("uc = %q, want %d", parsedURL.Query().Get(mediaCDNUseCaseParam), parsed.UseCase)
	}
	t.Logf("url_shape=%s://%s%s?uc=%s nested_id_len=%d outer_id_len=%d",
		parsedURL.Scheme, parsedURL.Host, "/d/...", parsedURL.Query().Get(mediaCDNUseCaseParam),
		len(parsed.CDNObjectID), len(parsed.ContentObjectID))
}

func TestURLBackedEncryptedMediaDecryptsBeforeSniff(t *testing.T) {
	jpegBytes := makeTestJPEG(t)
	key := bytes.Repeat([]byte{0x11}, 32)
	iv := bytes.Repeat([]byte{0x22}, 16)
	ciphertext := aesCBCEncrypt(t, key, iv, jpegBytes)

	client, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/c/abcdefgh" || r.URL.Query().Get("uc") != "4" {
			t.Errorf("CDN request = path %q uc %q", r.URL.Path, r.URL.Query().Get("uc"))
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(ciphertext)
	}))
	defer srv.Close()

	data, mime, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:  "media-1",
		URL: srv.URL + "/c/abcdefgh?uc=4",
		Key: key,
		IV:  iv,
	})
	if err != nil {
		t.Fatalf("DownloadMediaWithInfo: %v", err)
	}
	if !bytes.Equal(data, jpegBytes) {
		t.Fatalf("decrypted bytes do not match original JPEG (len=%d)", len(data))
	}
	if mime != "image/jpeg" {
		t.Fatalf("mime = %q, want image/jpeg", mime)
	}
	if info.DecryptPath != "cbc-clear-media-key" || info.CiphertextBytes != len(ciphertext) {
		t.Fatalf("info = %+v", info)
	}
	if info.Source != "url" {
		t.Fatalf("source = %q, want url", info.Source)
	}
}

func TestDescriptorCDNRetrievalSkipsResolverRPC(t *testing.T) {
	jpegBytes := makeTestJPEG(t)
	key := bytes.Repeat([]byte{0x33}, 32)
	iv := bytes.Repeat([]byte{0x44}, 16)
	ciphertext := aesCBCEncrypt(t, key, iv, jpegBytes)

	rpcCalled := false
	client, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/c/") {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(ciphertext)
			return
		}
		rpcCalled = true
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("resolver rpc rejected"))
	}))
	defer srv.Close()

	// The descriptor branch must attempt direct CDN retrieval first. The
	// production URL builder targets the real CDN host, so the fallback RPC
	// against this test server would 403; a successful retrieval proves the
	// CDN attempt ran with the client redirected at the transport level.
	// Redirect the CDN host to the test server via the client transport.
	transport := srv.Client().Transport
	srv.Client().Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "cf-st.sc-cdn.net" {
			req.URL.Scheme = "http"
			req.URL.Host = srv.Listener.Addr().String()
		}
		return transport.RoundTrip(req)
	})

	data, mime, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:   "media-cdn",
		Data: nestedDescriptor(t, "abcdefgh", 4),
		Key:  key,
		IV:   iv,
	})
	if err != nil {
		t.Fatalf("DownloadMediaWithInfo: %v", err)
	}
	if !bytes.Equal(data, jpegBytes) || mime != "image/jpeg" {
		t.Fatalf("decrypt+sniff mismatch mime=%q len=%d", mime, len(data))
	}
	if info.Source != "descriptor-cdn" || info.DecryptPath != "cbc-clear-media-key" {
		t.Fatalf("info = %+v", info)
	}
	if rpcCalled {
		t.Fatal("resolver RPC must not run when direct CDN retrieval succeeds")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestInvalidKeyIVLengthsRejected(t *testing.T) {
	jpegBytes := makeTestJPEG(t)
	ciphertext := aesCBCEncrypt(t, bytes.Repeat([]byte{0x11}, 32), bytes.Repeat([]byte{0x22}, 16), jpegBytes)
	if _, ok := decryptMediaCBC(ciphertext, bytes.Repeat([]byte{0x11}, 15), bytes.Repeat([]byte{0x22}, 16)); ok {
		t.Fatal("15-byte key must be rejected")
	}
	if _, ok := decryptMediaCBC(ciphertext, bytes.Repeat([]byte{0x11}, 64), bytes.Repeat([]byte{0x22}, 16)); ok {
		t.Fatal("64-byte key must be rejected")
	}
	if _, ok := decryptMediaCBC(ciphertext, bytes.Repeat([]byte{0x11}, 32), bytes.Repeat([]byte{0x22}, 8)); ok {
		t.Fatal("8-byte IV must be rejected")
	}
	if _, ok := decryptMediaCBC(nil, bytes.Repeat([]byte{0x11}, 32), bytes.Repeat([]byte{0x22}, 16)); ok {
		t.Fatal("empty ciphertext must be rejected")
	}
	if _, ok := decryptMediaCBC(ciphertext[:len(ciphertext)-1], bytes.Repeat([]byte{0x11}, 32), bytes.Repeat([]byte{0x22}, 16)); ok {
		t.Fatal("non-block-aligned ciphertext must be rejected")
	}
}

func TestDescriptorBytesNeverReturnedAsMedia(t *testing.T) {
	descriptor := nestedDescriptor(t, "abcdefgh", 4)
	client, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(descriptor)
	}))
	defer srv.Close()
	// A descriptor served as media bytes (no key) must fail validation, never
	// be returned as renderable media.
	if _, _, _, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:  "media-2",
		URL: srv.URL + "/c/abcdefgh?uc=4",
	}); err == nil {
		t.Fatal("descriptor-only payload must not validate as media")
	}
	if err := ValidateRenderableMediaPayload(descriptor, "application/octet-stream", MediaKindFile); err == nil {
		t.Fatal("ValidateRenderableMediaPayload must reject descriptor bytes")
	}
}

func TestClearURLBackedMediaUnchanged(t *testing.T) {
	jpegBytes := makeTestJPEG(t)
	client, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpegBytes)
	}))
	defer srv.Close()
	data, mime, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:       "media-3",
		URL:      srv.URL + "/media.jpg",
		MimeType: "image/jpeg",
	})
	if err != nil {
		t.Fatalf("DownloadMediaWithInfo: %v", err)
	}
	if !bytes.Equal(data, jpegBytes) || mime != "image/jpeg" {
		t.Fatalf("clear media altered: mime=%q len=%d", mime, len(data))
	}
	if info.Source != "url" || info.DecryptPath != "none" || info.CiphertextBytes != 0 {
		t.Fatalf("info = %+v", info)
	}
}

func TestAmbiguousUseCaseDescriptorFallsBack(t *testing.T) {
	inner := appendBytesField(nil, 2, []byte("abcdefgh"))
	inner = appendVarintField(inner, 10, 4)
	inner = appendVarintField(inner, 10, 5)
	descriptor := appendBytesField(nil, 2, inner)
	if _, err := MediaContentObjectURL(descriptor); err == nil {
		t.Fatal("ambiguous use case must not produce a CDN URL")
	}
	if parsed := ParseMediaDescriptor(descriptor); parsed.Detected {
		t.Fatalf("ambiguous use case descriptor must not parse as detected: %+v", parsed)
	}
}

func TestNonLocalAuxShapesFallBackToRPC(t *testing.T) {
	// Real auxiliary variations the synthetic resolver matrix proved go to the
	// remote /bolt-http/resolve service; native code must not imitate them.
	bareID := appendBytesField(nil, 2, []byte("abcdefgh"))
	// f12=2: f6={3}, f9=2, f10=4, f12=2.
	f12TwoInner := appendBytesField(nil, 2, []byte("abcdefgh"))
	f12TwoInner = appendBytesField(f12TwoInner, 6, []byte{3})
	f12TwoInner = appendVarintField(f12TwoInner, 9, 2)
	f12TwoInner = appendVarintField(f12TwoInner, 10, 4)
	f12TwoInner = appendVarintField(f12TwoInner, 12, 2)
	f12Two := appendBytesField(nil, 2, f12TwoInner)
	// f9 absent: f6={3}, f10=4, f12=1 (real 32-byte shape).
	f9Absent := appendVarintField(appendBytesField(appendBytesField(nil, 2, []byte("abcdefgh")), 6, []byte{3}), 10, 4)
	f9Absent = appendVarintField(f9Absent, 12, 1)
	// f6 value other than 3 (real 34-byte shape carries f6={4}).
	f6FourInner := appendBytesField(nil, 2, []byte("abcdefgh"))
	f6FourInner = appendBytesField(f6FourInner, 6, []byte{4})
	f6FourInner = appendVarintField(f6FourInner, 10, 4)
	f6FourInner = appendVarintField(f6FourInner, 12, 1)
	f6Four := appendBytesField(nil, 2, f6FourInner)
	// field-1-direct shape: ID in field 1 plus aux fields.
	f1Direct := appendBytesField(nil, 1, []byte("abcdefgh"))
	f1Direct = appendBytesField(f1Direct, 6, []byte{3})
	f1Direct = appendVarintField(f1Direct, 10, 4)

	cases := map[string][]byte{
		"no-aux":    bareID,
		"f12-eq-2":  appendBytesField(nil, 2, f12Two),
		"f9-absent": appendBytesField(nil, 2, f9Absent),
		"f6-eq-4":   f6Four,
		"f1-direct": f1Direct,
	}
	// The matrix proved f12=2, f9 absence, f6!=3, and the field-1-direct shape
	// all make the Web resolver fall back to its remote service: detected but
	// never locally expandable.
	for name, descriptor := range cases {
		parsed := ParseMediaDescriptor(descriptor)
		if parsed.CDNEligible {
			t.Fatalf("%s: must not be CDN eligible: %+v", name, parsed)
		}
		if _, err := MediaContentObjectURL(descriptor); err == nil {
			t.Fatalf("%s: must not produce a CDN URL", name)
		}
	}
}

func TestCDNFetchSendsNoSnapchatAuthMaterial(t *testing.T) {
	var gotHeaders http.Header
	client, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	_, _, err := client.fetchMediaCDN(context.Background(), srv.URL+"/c/abcdefgh?uc=4")
	if err == nil {
		t.Fatal("expected non-2xx error")
	}
	if gotHeaders == nil {
		t.Fatal("request did not reach the server")
	}
	for _, banned := range []string{"Cookie", "Authorization", "X-Snap-Client-User-Agent", "X-User-Agent", "Mcs-Cof-Ids-Bin", "X-Grpc-Web", "Content-Type"} {
		if values := gotHeaders.Values(banned); len(values) > 0 {
			t.Fatalf("CDN fetch must not send %s (got %v)", banned, values)
		}
	}
	if gotHeaders.Get("Origin") != "https://www.snapchat.com" || gotHeaders.Get("Referer") != "https://www.snapchat.com/" {
		t.Fatalf("browser origin metadata missing: %v", gotHeaders)
	}
}

func TestExternalMediaEncryptionKeysStillValid(t *testing.T) {
	// The real 34-byte descriptor shape from message 4114 must keep parsing
	// with its use case extracted for CDN URL construction.
	descriptorHex := "12201215635652757649595876626b6169397a665a48735a4d320103480250046001"
	descriptor := make([]byte, len(descriptorHex)/2)
	for i := 0; i < len(descriptor); i++ {
		_, _ = fmt.Sscanf(descriptorHex[i*2:i*2+2], "%02x", &descriptor[i])
	}
	parsed := ParseMediaDescriptor(descriptor)
	if !parsed.Detected || parsed.ContentObjectID != "cVRuvIYXvbkai9zfZHsZM" || parsed.UseCase != 4 {
		t.Fatalf("4114 descriptor parse = %+v", parsed)
	}
	got, err := MediaContentObjectURL(descriptor)
	if err != nil || got != "https://cf-st.sc-cdn.net/c/cVRuvIYXvbkai9zfZHsZM?uc=4" {
		t.Fatalf("4114 CDN URL = %q err=%v", got, err)
	}
}
