package snapapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"testing"

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

func TestExtractMessageStatus(t *testing.T) {
	msg := &protos.ContentMessage{
		MessageId: 123,
		MetaData: &protos.MessageMetadata{
			ReadTimestamp:       456,
			ConversationVersion: 789,
			ReadBy: []*protos.UUID{{
				EncodedId: []byte{0x33, 0x43, 0x2e, 0xb1, 0x90, 0x99, 0x4b, 0xda, 0x8b, 0x85, 0xe7, 0x5c, 0xc8, 0x99, 0xc2, 0xa2},
			}},
		},
	}
	status := ExtractMessageStatus(msg)
	if status.ReadTimestamp != 456 || status.ConversationVersion != 789 {
		t.Fatalf("status timestamps = %d/%d, want 456/789", status.ReadTimestamp, status.ConversationVersion)
	}
	if len(status.ReadBy) != 1 || status.ReadBy[0] == "" {
		t.Fatalf("read-by status not extracted: %#v", status.ReadBy)
	}
}

func TestExtractConversationStatusFromDelta(t *testing.T) {
	resp := &protos.DeltaSyncResponse{
		Metadata: &protos.DeltaSyncResponse_Conversation{
			Conversation: &protos.Conversation{
				ConversationId: &protos.UUID{EncodedId: []byte{0x92, 0x63, 0x9b, 0x1c, 0xc6, 0xb9, 0x5d, 0x4a, 0xb8, 0x94, 0xa6, 0x29, 0x51, 0x70, 0x8f, 0x3b}},
				Participants: []*protos.Participant{{
					UserId:            &protos.UUID{EncodedId: []byte{0x33, 0x43, 0x2e, 0xb1, 0x90, 0x99, 0x4b, 0xda, 0x8b, 0x85, 0xe7, 0x5c, 0xc8, 0x99, 0xc2, 0xa2}},
					ReadHighWatermark: 12,
				}},
			},
		},
		FeedInfo: &protos.DeltaSyncResponse_FeedOpenedMessageDisplayTimestamp{FeedOpenedMessageDisplayTimestamp: 345},
	}
	status := ExtractConversationStatusFromDelta(resp)
	if status.ConversationID == "" {
		t.Fatal("conversation id was not extracted")
	}
	if status.FeedOpenedMessageDisplayTimestamp != 345 {
		t.Fatalf("feed opened timestamp = %d, want 345", status.FeedOpenedMessageDisplayTimestamp)
	}
	if len(status.Participants) != 1 || status.Participants[0].ReadHighWatermark != 12 {
		t.Fatalf("participant status not extracted: %#v", status.Participants)
	}
}
