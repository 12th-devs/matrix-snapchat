package snapapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"testing"
	"time"

	"github.com/0xzer/snapper/protos"
)

func TestExtractProfileAvatarURLPrefersBitmoji(t *testing.T) {
	got := extractProfileAvatarURL(map[string]any{
		"profile_picture_url": "https://example.invalid/profile.png",
		"bitmoji": map[string]any{
			"avatar_url": "https://images.bitmoji.com/3d/avatar/abc",
		},
	})
	if got != "https://images.bitmoji.com/3d/avatar/abc" {
		t.Fatalf("avatar URL = %q, want bitmoji URL", got)
	}
}

func TestBitmojiAvatarURL(t *testing.T) {
	got := bitmojiAvatarURL("selfie-id", "avatar-id")
	want := "https://sdk.bitmoji.com/render/panel/selfie-id-avatar-id-v1.png?transparent=1&scale=2"
	if got != want {
		t.Fatalf("bitmoji URL = %q, want %q", got, want)
	}
}

func TestExtractProfileAvatarURLIgnoresNonAvatarURLs(t *testing.T) {
	got := extractProfileAvatarURL(map[string]any{
		"share_url": "https://www.snapchat.com/add/tester",
		"nested": map[string]any{
			"thumbnail_url": "https://example.invalid/thumb.webp",
		},
	})
	if got != "https://example.invalid/thumb.webp" {
		t.Fatalf("avatar URL = %q, want thumbnail fallback", got)
	}
}

func TestMessageBodyChatText(t *testing.T) {
	contentBytes, err := protos.EncodeProtoMessage(&protos.Contents{
		Content: &protos.Contents_Text{
			Text: &protos.Text{Text: "hello from api"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body, isSnap := (&Client{}).messageBody(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 123,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    contentBytes,
		},
	})
	if body != "hello from api" {
		t.Fatalf("body = %q, want decoded text", body)
	}
	if isSnap {
		t.Fatal("chat text was marked as snap")
	}
}

func TestMessageBodyUndecodableChatFallsBackToUnavailableNotice(t *testing.T) {
	body, isSnap := (&Client{}).messageBody(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 456,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    []byte{0xff, 0x01, 0x02},
		},
	})
	if body != "[Snapchat message unavailable]" {
		t.Fatalf("body = %q, want safe chat notice", body)
	}
	if isSnap {
		t.Fatal("undecodable chat was marked as snap")
	}
}

func TestMessageBodyClearTextEELChatText(t *testing.T) {
	contentBytes, err := protos.EncodeProtoMessage(&protos.Contents{
		Content: &protos.Contents_Text{
			Text: &protos.Text{Text: "hello from cleartext eel"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	key := []byte("0123456789abcdef0123456789abcdef")
	nonce := []byte("123456789012")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := gcm.Seal(nil, nonce, contentBytes, nil)

	body, isSnap := (&Client{}).messageBody(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 789,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    encrypted,
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_ClearTextEelKeyEncryption{
					ClearTextEelKeyEncryption: &protos.ClearTextEelKeyEncryption{
						Cek:   key,
						CekIv: nonce,
					},
				},
			},
		},
	})
	if body != "hello from cleartext eel" {
		t.Fatalf("body = %q, want decoded cleartext EEL text", body)
	}
	if isSnap {
		t.Fatal("cleartext EEL chat text was marked as snap")
	}
}

func TestMessageBodyFullEELUsesMessageNonceBeforeCEKIV(t *testing.T) {
	contentBytes, err := protos.EncodeProtoMessage(&protos.Contents{
		Content: &protos.Contents_Text{
			Text: &protos.Text{Text: "hello from eel nonce"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	key := []byte("0123456789abcdef0123456789abcdef")
	messageNonce := []byte("123456789012")
	cekIV := []byte("abcdefghijkl")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := gcm.Seal(nil, messageNonce, contentBytes, nil)

	decrypter := &fakeEELDecrypter{err: errors.New("helper should not be called")}
	body, isSnap := (&Client{eelDecrypter: decrypter}).messageBody(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 790,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    encrypted,
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_EelEncryption{
					EelEncryption: &protos.EelEncryption{
						Cek:   key,
						CekIv: cekIV,
						Nonce: messageNonce,
					},
				},
			},
		},
	})
	if body != "hello from eel nonce" {
		t.Fatalf("body = %q, want decoded full EEL text using message nonce", body)
	}
	if isSnap {
		t.Fatal("full EEL nonce text was marked as snap")
	}
	if decrypter.calls != 0 {
		t.Fatalf("helper calls = %d, want 0", decrypter.calls)
	}
}

type fakeEELDecrypter struct {
	output []byte
	err    error
	calls  int
}

func (f *fakeEELDecrypter) DecryptEEL(_ context.Context, _ EELDecryptRequest) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.output, nil
}

func TestMessageBodyFullEELHelperChatText(t *testing.T) {
	contentBytes, err := protos.EncodeProtoMessage(&protos.Contents{
		Content: &protos.Contents_Text{
			Text: &protos.Text{Text: "hello from full eel"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	decrypter := &fakeEELDecrypter{output: contentBytes}

	body, isSnap := (&Client{eelDecrypter: decrypter}).messageBody(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 900,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    []byte{0x01, 0x02},
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_EelEncryption{
					EelEncryption: &protos.EelEncryption{
						CekIv:           []byte("123456789012"),
						Nonce:           []byte("abcdefghijklmnop"),
						SenderPublicKey: []byte("sender-public-key-placeholder"),
						SenderVersion:   7,
					},
				},
			},
		},
	})
	if body != "hello from full eel" {
		t.Fatalf("body = %q, want decoded full EEL text", body)
	}
	if isSnap {
		t.Fatal("full EEL chat text was marked as snap")
	}
	if decrypter.calls != 1 {
		t.Fatalf("helper calls = %d, want 1", decrypter.calls)
	}
}

func TestMessageBodyFullEELHelperPlaintextChatText(t *testing.T) {
	decrypter := &fakeEELDecrypter{output: []byte("hello from raw fidelius")}

	body, isSnap := (&Client{eelDecrypter: decrypter}).messageBody(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 905,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    []byte{0x01, 0x02},
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_EelEncryption{
					EelEncryption: &protos.EelEncryption{
						CekIv:           []byte("123456789012"),
						Nonce:           []byte("abcdefghijklmnop"),
						SenderPublicKey: []byte("sender-public-key-placeholder"),
						SenderVersion:   7,
					},
				},
			},
		},
	})
	if body != "hello from raw fidelius" {
		t.Fatalf("body = %q, want raw decrypted Fidelius text", body)
	}
	if isSnap {
		t.Fatal("raw decrypted Fidelius text was marked as snap")
	}
	if decrypter.calls != 1 {
		t.Fatalf("helper calls = %d, want 1", decrypter.calls)
	}
}

func TestMessageBodyFullEELHelperALEAPPMessageContentPath(t *testing.T) {
	decrypted := protoBytesField(4, protoBytesField(4, protoBytesField(2, protoBytesField(1, []byte("hello from modern message content")))))
	decrypter := &fakeEELDecrypter{output: decrypted}

	body, isSnap := (&Client{eelDecrypter: decrypter}).messageBody(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 906,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    []byte{0x01, 0x02},
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_EelEncryption{
					EelEncryption: &protos.EelEncryption{
						CekIv:           []byte("123456789012"),
						Nonce:           []byte("abcdefghijklmnop"),
						SenderPublicKey: []byte("sender-public-key-placeholder"),
						SenderVersion:   7,
					},
				},
			},
		},
	})
	if body != "hello from modern message content" {
		t.Fatalf("body = %q, want decoded modern message_content text", body)
	}
	if isSnap {
		t.Fatal("modern message_content text was marked as snap")
	}
	if decrypter.calls != 1 {
		t.Fatalf("helper calls = %d, want 1", decrypter.calls)
	}
}

func TestMessageBodyFullEELHelperCachesPlaintext(t *testing.T) {
	contentBytes, err := protos.EncodeProtoMessage(&protos.Contents{
		Content: &protos.Contents_Text{
			Text: &protos.Text{Text: "cached eel text"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	decrypter := &fakeEELDecrypter{output: contentBytes}
	client := &Client{
		eelDecrypter: decrypter,
		eelPlaintext: make(map[string][]byte),
		failedEEL:    make(map[string]time.Time),
	}
	msg := &protos.ContentMessage{
		MessageId: 907,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    []byte{0x01, 0x02},
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_EelEncryption{
					EelEncryption: &protos.EelEncryption{
						CekIv:           []byte("123456789012"),
						Nonce:           []byte("abcdefghijklmnop"),
						SenderPublicKey: []byte("sender-public-key-placeholder"),
						SenderVersion:   7,
					},
				},
			},
		},
	}
	for i := 0; i < 2; i++ {
		body, isSnap := client.messageBody(context.Background(), "chat-1", msg)
		if body != "cached eel text" || isSnap {
			t.Fatalf("decode %d = body %q isSnap %t, want cached eel text false", i, body, isSnap)
		}
	}
	if decrypter.calls != 1 {
		t.Fatalf("helper calls = %d, want 1", decrypter.calls)
	}
}

func TestMessageBodyFullEELHelperFailureFallsBackToUnavailableNotice(t *testing.T) {
	decrypter := &fakeEELDecrypter{err: errors.New("key unavailable")}
	body, isSnap := (&Client{eelDecrypter: decrypter}).messageBody(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 901,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    []byte{0x01, 0x02},
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_EelEncryption{
					EelEncryption: &protos.EelEncryption{
						CekIv:           []byte("123456789012"),
						Nonce:           []byte("abcdefghijklmnop"),
						SenderPublicKey: []byte("sender-public-key-placeholder"),
						SenderVersion:   7,
					},
				},
			},
		},
	})
	if body != "[Snapchat message unavailable]" {
		t.Fatalf("body = %q, want safe chat notice", body)
	}
	if isSnap {
		t.Fatal("full EEL failure was marked as snap")
	}
}

func protoBytesField(field int, value []byte) []byte {
	out := appendProtoVarint(nil, uint64(field<<3|2))
	out = appendProtoVarint(out, uint64(len(value)))
	return append(out, value...)
}

func appendProtoVarint(out []byte, value uint64) []byte {
	for value >= 0x80 {
		out = append(out, byte(value)|0x80)
		value >>= 7
	}
	return append(out, byte(value))
}

func TestMessageFromProtoExternalMediaIsNotSnap(t *testing.T) {
	msg := (&Client{}).messageFromProto(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 902,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_EXTERNAL_MEDIA,
			RemoteMediaInfos: []*protos.ContentEnvelope_RemoteMediaInfo{{
				MediaType: int32(protos.ContentEnvelope_RemoteMediaInfo_MediaType_IMAGE),
				MediaInfo: &protos.ContentEnvelope_RemoteMediaInfo_ContentObject{
					ContentObject: []byte("fake-image-bytes"),
				},
			}},
		},
	})
	if msg.IsSnap {
		t.Fatal("external media was incorrectly marked as snap")
	}
	if len(msg.Media) != 1 {
		t.Fatalf("external media attachment count = %d, want 1", len(msg.Media))
	}
	if msg.Text != "Media" {
		t.Fatalf("external media body = %q, want Media", msg.Text)
	}
}

func TestDetectMediaMimeUsesSnapchatMagicBytes(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "jpeg",
			data: []byte{0xff, 0xd8, 0xff, 0xe0},
			want: "image/jpeg",
		},
		{
			name: "png",
			data: []byte("\x89PNG\r\n\x1a\n"),
			want: "image/png",
		},
		{
			name: "mp4-ftyp",
			data: []byte("\x00\x00\x00\x18ftypisom\x00\x00\x00\x00"),
			want: "video/mp4",
		},
		{
			name: "snap-video-header",
			data: []byte{0x00, 0x00, 0x00, 0x1c, 0x12, 0x34},
			want: "application/octet-stream",
		},
		{
			name: "wav",
			data: []byte("RIFF\x24\x00\x00\x00WAVE"),
			want: "audio/wav",
		},
		{
			name: "mp3-id3",
			data: []byte{'I', 'D', '3', 0x04},
			want: "audio/mpeg",
		},
		{
			name: "mp3-frame-sync",
			data: []byte{0xff, 0xfb, 0x90, 0x64},
			want: "audio/mpeg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectMediaMime(tt.data, MediaKindFile); got != tt.want {
				t.Fatalf("detectMediaMime() = %q, want %q", got, tt.want)
			}
		})
	}
}
