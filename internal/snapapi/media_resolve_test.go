package snapapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"google.golang.org/protobuf/encoding/protowire"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestExternalMediaEncryptionKeys(t *testing.T) {
	key, iv := bytes.Repeat([]byte{7}, 32), bytes.Repeat([]byte{8}, 16)
	raw := mediaProtoBytes(19, append(mediaProtoBytes(1, key), mediaProtoBytes(2, iv)...))
	legacy := mediaProtoBytes(4, append(mediaProtoBytes(1, []byte(base64.StdEncoding.EncodeToString(key))), mediaProtoBytes(2, []byte(base64.StdEncoding.EncodeToString(iv)))...))
	wrap := func(data []byte) []byte {
		for _, field := range []protowire.Number{1, 1, 5, 3, 3} {
			data = mediaProtoBytes(field, data)
		}
		return data
	}
	for _, data := range [][]byte{raw, legacy, append(append([]byte{}, raw...), legacy...)} {
		gotKey, gotIV, err := ExternalMediaEncryptionKeys(wrap(data))
		if err != nil || !bytes.Equal(gotKey, key) || !bytes.Equal(gotIV, iv) {
			t.Fatalf("key extraction failed: %v", err)
		}
	}
	badRaw := mediaProtoBytes(19, append(mediaProtoBytes(1, bytes.Repeat([]byte{9}, 32)), mediaProtoBytes(2, iv)...))
	if _, _, err := ExternalMediaEncryptionKeys(wrap(append(badRaw, legacy...))); err == nil {
		t.Fatal("conflicting metadata accepted")
	}
	if _, _, err := ExternalMediaEncryptionKeys(append(wrap(raw), wrap(raw)...)); err == nil {
		t.Fatal("ambiguous items accepted")
	}
}

func resolverResponse(url string) []byte {
	data := []byte(url)
	// Mn.responses / Ln.resolvedContentObject / Fn.resolvedContents /
	// xn.resolvedUrls / Bn.url.
	for i := 0; i < 5; i++ {
		data = mediaProtoBytes(1, data)
	}
	return data
}

func TestMediaResolverResponseValidation(t *testing.T) {
	valid := resolverResponse("https://cdn.example.invalid/image")
	for _, tc := range []struct {
		name  string
		data  []byte
		valid bool
	}{
		{"URL", valid, true},
		{"non-HTTPS", resolverResponse("file:///private"), false},
		{"credentials", resolverResponse("https://user:pass@cdn.example.invalid/image"), false},
		{"truncated", valid[:len(valid)-1], false},
		{"error", mediaProtoBytes(1, mediaProtoBytes(2, mediaProtoBytes(1, []byte("expired")))), false},
		{"multiple responses", append(append([]byte{}, valid...), valid...), false},
		{"empty", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			urls, err := resolvedMediaURLs(tc.data)
			if (err == nil) != tc.valid {
				t.Fatalf("URLs=%v error=%v", urls, err)
			}
		})
	}
}

func TestMediaResolverChecksTrailingGRPCError(t *testing.T) {
	data := frameGRPC(resolverResponse("https://cdn.example.invalid/image"))
	trailer := frameGRPC([]byte("grpc-status: 7\r\n"))
	trailer[0] = 0x80
	if _, err := mediaGRPCPayload(append(data, trailer...)); err == nil {
		t.Fatal("ignored trailing permission denial")
	}
	if _, err := mediaGRPCPayload([]byte{0, 0, 0, 0, 9, 1}); err == nil {
		t.Fatal("accepted truncated frame")
	}
}

type resolverTransport func(*http.Request) (*http.Response, error)

func (f resolverTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMediaResolverUsesOpaqueDescriptorAndExistingAuthentication(t *testing.T) {
	client, err := New(Config{CookieString: "__Host-sc-a-session=test; __Host-X-Snap-Client-Cookie=test; __Host-sc-a-nonce=test", SSOToken: "test"})
	if err != nil {
		t.Fatal(err)
	}
	descriptor := mediaProtoBytes(1, []byte("ordinary-media_1"))
	client.http.Transport = resolverTransport(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/snapchat.content.v2.MediaDeliveryService/resolveContentObjects") || r.Method != http.MethodPost {
			t.Fatalf("unexpected RPC %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		want := frameGRPC(mediaProtoBytes(1, mediaProtoBytes(1, descriptor)))
		if !bytes.Equal(body, want) {
			t.Fatal("descriptor not encoded as the contentObject reference")
		}
		if r.Header.Get("Content-Type") != "application/grpc-web+proto" {
			t.Fatal("missing existing grpc headers")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(frameGRPC(resolverResponse("https://cdn.example.invalid/image"))))}, nil
	})
	urls, err := client.ResolveMediaDescriptor(context.Background(), descriptor)
	if err != nil || len(urls) != 1 {
		t.Fatalf("URLs=%v error=%v", urls, err)
	}
}
