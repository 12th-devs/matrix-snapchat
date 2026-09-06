package snapapi

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
)

// boltMappingEntry encodes one network-mapping bucket entry: field 1 =
// location id, field 2 = bucket, field 4 = storage tier, field 5 = template,
// repeated field 6 = positional arguments with the CDN path prefix last.
func boltMappingEntry(locationID uint64, bucket string, tier uint64, prefix string) []byte {
	entry := appendVarintField(nil, 1, locationID)
	entry = appendBytesField(entry, 2, []byte(bucket))
	entry = appendVarintField(entry, 4, tier)
	entry = appendBytesField(entry, 5, []byte("{0}{1}{contentId}"))
	entry = appendBytesField(entry, 6, []byte(bucket))
	entry = appendBytesField(entry, 6, nil)
	entry = appendBytesField(entry, 6, []byte(prefix))
	return appendBytesField(nil, 2, entry)
}

// boltMappingFor79 builds a mapping covering the three live-proven locations:
// 3 -> cf-st /c/, 4 -> cf-st /d/, 79 -> bolt-gcdn /bq/.
func boltMappingFor79(t *testing.T) []byte {
	t.Helper()
	bin := boltMappingEntry(3, "bolt-prod-s3-us-east-2", boltLocationTierS3, "c")
	bin = append(bin, boltMappingEntry(4, "bolt-prod-s3-us-east-1", boltLocationTierS3, "d")...)
	bin = append(bin, boltMappingEntry(79, "bolt-prod-gcs-us-east-5", boltLocationTierGCS, "bq")...)
	return bin
}

func TestParseBoltNetworkMappingResolvesLiveLocations(t *testing.T) {
	mapping, err := parseBoltNetworkMapping(boltMappingFor79(t))
	if err != nil {
		t.Fatalf("parseBoltNetworkMapping: %v", err)
	}
	cases := []struct {
		location uint64
		id       string
		useCase  uint64
		want     string
	}{
		{79, "bDeVLqJU75PVY0Z46aAfP", 4, "https://bolt-gcdn.sc-cdn.net/bq/bDeVLqJU75PVY0Z46aAfP?uc=4"},
		{3, "abcdefgh", 4, "https://cf-st.sc-cdn.net/c/abcdefgh?uc=4"},
		{4, "abcdefgh", 4, "https://cf-st.sc-cdn.net/d/abcdefgh?uc=4"},
	}
	for _, tc := range cases {
		location, ok := mapping.locations[tc.location]
		if !ok {
			t.Fatalf("location %d missing from mapping", tc.location)
		}
		got, ok := mediaCDNURLFromLocation(location, tc.id, tc.useCase)
		if !ok || got != tc.want {
			t.Fatalf("location %d URL = %q ok=%t, want %q", tc.location, got, ok, tc.want)
		}
	}
	if _, ok := mapping.locations[999]; ok {
		t.Fatal("unexpected location 999 in mapping")
	}
}

func TestMediaCDNURLFromLocationRejectsUnsafeInputs(t *testing.T) {
	location := boltLocation{Bucket: "bolt-prod-gcs-us-east-5", Tier: boltLocationTierGCS, PathPrefix: "bq"}
	for _, prefix := range []string{"", "BQ", "b/q", "bq?x=1", "verylongprefix"} {
		if _, ok := mediaCDNURLFromLocation(boltLocation{Tier: boltLocationTierGCS, PathPrefix: prefix}, "abcdefgh", 4); ok {
			t.Fatalf("prefix %q must be rejected", prefix)
		}
	}
	for _, tier := range []uint64{0, 3, 99} {
		if _, ok := mediaCDNURLFromLocation(boltLocation{Tier: tier, PathPrefix: "bq"}, "abcdefgh", 4); ok {
			t.Fatalf("tier %d must be rejected", tier)
		}
	}
	if _, ok := mediaCDNURLFromLocation(location, "../../etc/passwd", 4); ok {
		t.Fatal("unsafe content ID must be rejected")
	}
}

// The live 34-byte descriptor from message 79 (evie chat): content ID
// bDeVLqJU75PVY0Z46aAfP, readyLocationIds [79], claimId 2, useCase 4,
// hostPatternVersion 1. It must NOT satisfy the static /c/ rule (location 3
// only) but must expose the location for the network-mapping resolver.
func TestMessage79DescriptorParsesReadyLocationIDs(t *testing.T) {
	descriptorHex := "12201215624465564c714a553735505659305a34366141665032014f480250046001"
	descriptorBytes := mustHex(t, descriptorHex)
	if _, urlErr := MediaContentObjectURL(descriptorBytes); urlErr == nil {
		t.Fatal("static /c/ rule must not claim message 79's descriptor")
	}
	parsed := ParseMediaDescriptor(descriptorBytes)
	if !parsed.Detected || parsed.Shape != mediaDescriptorShapeField2Nested {
		t.Fatalf("descriptor parse = %+v", parsed)
	}
	if parsed.ContentObjectID != "bDeVLqJU75PVY0Z46aAfP" || parsed.CDNObjectID != parsed.ContentObjectID {
		t.Fatalf("descriptor IDs = %+v", parsed)
	}
	if parsed.UseCase != 4 {
		t.Fatalf("useCase = %d, want 4", parsed.UseCase)
	}
	if len(parsed.ReadyLocationIDs) != 1 || parsed.ReadyLocationIDs[0] != 79 {
		t.Fatalf("readyLocationIds = %v, want [79]", parsed.ReadyLocationIDs)
	}
	if parsed.CDNEligible {
		t.Fatal("static CDNEligible must stay false for location 79")
	}
}

func TestDownloadUsesMappedCDNForMessage79Descriptor(t *testing.T) {
	jpegBytes := makeTestJPEG(t)
	key := bytes.Repeat([]byte{0x55}, 32)
	iv := bytes.Repeat([]byte{0x66}, 16)
	ciphertext := aesCBCEncrypt(t, key, iv, jpegBytes)

	rpcCalled := false
	client, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/sc-bolt-nm-prod/network_mapping.bin":
			_, _ = w.Write(boltMappingFor79(t))
		case r.URL.Path == "/bq/bDeVLqJU75PVY0Z46aAfP" && r.URL.Query().Get("uc") == "4":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(ciphertext)
		default:
			rpcCalled = true
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("resolver rpc rejected"))
		}
	}))
	defer srv.Close()

	transport := srv.Client().Transport
	srv.Client().Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "bolt-gcdn.sc-cdn.net" {
			req.URL.Scheme = "http"
			req.URL.Host = srv.Listener.Addr().String()
		}
		return transport.RoundTrip(req)
	})

	data, mime, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:   "media-79",
		Data: mustHex(t, "12201215624465564c714a553735505659305a34366141665032014f480250046001"),
		Key:  key,
		IV:   iv,
	})
	if err != nil {
		t.Fatalf("DownloadMediaWithInfo: %v", err)
	}
	if !bytes.Equal(data, jpegBytes) || mime != "image/jpeg" {
		t.Fatalf("decrypt+sniff mismatch mime=%q len=%d", mime, len(data))
	}
	if info.Source != "descriptor-mapped-cdn" || info.DecryptPath != "cbc-clear-media-key" {
		t.Fatalf("info = %+v", info)
	}
	if rpcCalled {
		t.Fatal("resolver RPC must not run when the mapped CDN retrieval succeeds")
	}
}

func TestMappedCDNFailureFallsBackToRPC(t *testing.T) {
	client, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sc-bolt-nm-prod/network_mapping.bin":
			_, _ = w.Write(boltMappingFor79(t))
		case "/snapchat.content.v2.MediaDeliveryService/resolveContentObjects":
			// Valid gRPC-web frame carrying an empty response, matching the
			// live empty-200 behavior for unresolvable objects.
			w.Header().Set("Content-Type", "application/grpc-web+proto")
			_, _ = w.Write([]byte{0, 0, 0, 0, 0})
		default:
			// Mapped CDN path 404s.
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	transport := srv.Client().Transport
	srv.Client().Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "bolt-gcdn.sc-cdn.net" || req.URL.Host == "web.snapchat.com" {
			req.URL.Scheme = "http"
			req.URL.Host = srv.Listener.Addr().String()
		}
		return transport.RoundTrip(req)
	})

	_, _, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:   "media-79-miss",
		Data: mustHex(t, "12201215624465564c714a553735505659305a34366141665032014f480250046001"),
		Key:  bytes.Repeat([]byte{0x55}, 32),
		IV:   bytes.Repeat([]byte{0x66}, 16),
	})
	if err == nil {
		t.Fatal("expected the empty RPC response to fail loudly")
	}
	if info.Source != "descriptor-rpc" {
		t.Fatalf("source = %q, want descriptor-rpc", info.Source)
	}
	if !strings.Contains(info.CDNFallbackReason, "404") {
		t.Fatalf("CDNFallbackReason = %q, want the mapped-CDN 404", info.CDNFallbackReason)
	}
}

// The live 60-byte field1-direct descriptor from message 9983: outer content
// object 8faeOUT1frbOlSNkJI5e7_1, nested contentId 8faeOUT1frbOlSNkJI5e7 with
// readyLocationIds [3] (the /c/ location), claimId 1, useCase 4,
// hostPatternVersion 1, attributes 2. The static shape-A rule still derives
// /d/, but the network mapping must make the /c/ URL available.
func TestMessage9983Field1DirectDescriptorParsesReadyLocationIDs(t *testing.T) {
	descriptorBytes := mustHex(t, "0a17386661654f5554316672624f6c534e6b4a493565375f3112211215386661654f5554316672624f6c534e6b4a4935653730034801500460017002")
	parsed := ParseMediaDescriptor(descriptorBytes)
	if !parsed.Detected || parsed.Shape != mediaDescriptorShapeField1Direct {
		t.Fatalf("descriptor parse = %+v", parsed)
	}
	if parsed.ContentObjectID != "8faeOUT1frbOlSNkJI5e7_1" || parsed.CDNObjectID != "8faeOUT1frbOlSNkJI5e7" {
		t.Fatalf("descriptor IDs = %+v", parsed)
	}
	if parsed.UseCase != 4 {
		t.Fatalf("useCase = %d, want 4", parsed.UseCase)
	}
	if len(parsed.ReadyLocationIDs) != 1 || parsed.ReadyLocationIDs[0] != 3 {
		t.Fatalf("readyLocationIds = %v, want [3]", parsed.ReadyLocationIDs)
	}
	staticURL, err := MediaContentObjectURL(descriptorBytes)
	if err != nil || staticURL != "https://cf-st.sc-cdn.net/d/8faeOUT1frbOlSNkJI5e7?uc=4" {
		t.Fatalf("static shape-A URL = %q err=%v, want the unchanged /d/ URL", staticURL, err)
	}
}

func TestDownloadFallsBackFromShapeAStaticCDNToMappedCDN(t *testing.T) {
	jpegBytes := makeTestJPEG(t)
	key := bytes.Repeat([]byte{0x77}, 32)
	iv := bytes.Repeat([]byte{0x88}, 16)
	ciphertext := aesCBCEncrypt(t, key, iv, jpegBytes)

	rpcCalled := false
	client, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/sc-bolt-nm-prod/network_mapping.bin":
			_, _ = w.Write(boltMappingFor79(t))
		case r.URL.Path == "/d/8faeOUT1frbOlSNkJI5e7":
			// The static shape-A URL 404s exactly like live message 9983.
			w.WriteHeader(http.StatusNotFound)
		case r.URL.Path == "/c/8faeOUT1frbOlSNkJI5e7" && r.URL.Query().Get("uc") == "4":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(ciphertext)
		default:
			rpcCalled = true
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("resolver rpc rejected"))
		}
	}))
	defer srv.Close()

	transport := srv.Client().Transport
	srv.Client().Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "bolt-gcdn.sc-cdn.net" || req.URL.Host == "cf-st.sc-cdn.net" {
			req.URL.Scheme = "http"
			req.URL.Host = srv.Listener.Addr().String()
		}
		return transport.RoundTrip(req)
	})

	data, mime, info, err := client.DownloadMediaWithInfo(context.Background(), MediaAttachment{
		ID:   "media-9983",
		Data: mustHex(t, "0a17386661654f5554316672624f6c534e6b4a493565375f3112211215386661654f5554316672624f6c534e6b4a4935653730034801500460017002"),
		Key:  key,
		IV:   iv,
	})
	if err != nil {
		t.Fatalf("DownloadMediaWithInfo: %v", err)
	}
	if !bytes.Equal(data, jpegBytes) || mime != "image/jpeg" {
		t.Fatalf("decrypt+sniff mismatch mime=%q len=%d", mime, len(data))
	}
	if info.Source != "descriptor-mapped-cdn" {
		t.Fatalf("source = %q, want descriptor-mapped-cdn", info.Source)
	}
	if !strings.Contains(info.CDNFallbackReason, "404") {
		t.Fatalf("CDNFallbackReason = %q, want the static /d/ 404", info.CDNFallbackReason)
	}
	if rpcCalled {
		t.Fatal("resolver RPC must not run when the mapped CDN retrieval succeeds")
	}
}

func mustHex(t *testing.T, value string) []byte {
	t.Helper()
	data := make([]byte, len(value)/2)
	for i := 0; i < len(data); i++ {
		high := hexDigit(t, value[2*i])
		low := hexDigit(t, value[2*i+1])
		data[i] = high<<4 | low
	}
	return data
}

func hexDigit(t *testing.T, c byte) byte {
	t.Helper()
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	default:
		t.Fatalf("invalid hex digit %q", c)
		return 0
	}
}
