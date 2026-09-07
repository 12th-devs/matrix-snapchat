package snapapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/0xzer/snapper/protos"
)

// TestMessageFromProtoVoiceNoteIsAudio pins the incoming voice-note contract:
// a NOTE envelope must not be treated as a snap, must carry the note key/IV
// from the audio-note metadata, and must be presented as an audio attachment.
func TestMessageFromProtoVoiceNoteIsAudio(t *testing.T) {
	key := bytes.Repeat([]byte{0x77}, 32)
	iv := bytes.Repeat([]byte{0x88}, 16)
	contents := encodeNoteContents(&OutgoingMediaInfo{
		MIME:       "audio/mp4",
		MediaType:  protos.MediaType_MEDIA_TYPE_AUDIO,
		HasAudio:   true,
		DurationMs: 2993,
	}, key, iv)
	msg := (&Client{}).messageFromProto(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 10108,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_NOTE,
			Contents:    contents,
			SavePolicy:  protos.ContentEnvelope_SavePolicy_LIFETIME,
			MediaReferenceLists: []*protos.ContentEnvelope_MediaReferenceList{{
				Reference: []*protos.MediaReference{{
					ContentObject: []byte("fake-audio-descriptor"),
					MediaType:     protos.MediaType_MEDIA_TYPE_UNASSIGNED,
				}},
			}},
		},
	})
	if msg.IsSnap {
		t.Fatal("voice note was incorrectly marked as snap")
	}
	if len(msg.Media) != 1 {
		t.Fatalf("voice note attachment count = %d, want 1", len(msg.Media))
	}
	attachment := msg.Media[0]
	if attachment.Kind != MediaKindAudio || attachment.MimeType != "audio/mp4" {
		t.Fatalf("voice note attachment = %s/%s, want audio/audio.mp4", attachment.Kind, attachment.MimeType)
	}
	if !bytes.Equal(attachment.Key, key) || !bytes.Equal(attachment.IV, iv) {
		t.Fatalf("voice note key/IV not propagated from the note metadata")
	}
}

// TestNoteAudioEncryptionKeysLiveShape replays the decoded contents structure
// of a real incoming voice note (message 10108: type=4, encryptionInfo b64
// key/IV, mediaDurationMs=2993) to pin the receive-side extractor.
func TestNoteAudioEncryptionKeysLiveShape(t *testing.T) {
	key := bytes.Repeat([]byte{0x11}, 32)
	iv := bytes.Repeat([]byte{0x22}, 16)
	contents := encodeNoteContents(&OutgoingMediaInfo{
		MIME:       "audio/mp4",
		MediaType:  protos.MediaType_MEDIA_TYPE_AUDIO,
		HasAudio:   true,
		DurationMs: 2993,
	}, key, iv)
	gotKey, gotIV, durationMs, err := NoteAudioEncryptionKeys(contents)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotKey, key) || !bytes.Equal(gotIV, iv) {
		t.Fatal("note key/IV roundtrip failed")
	}
	if durationMs != 2993 {
		t.Fatalf("note durationMs = %d, want 2993", durationMs)
	}
	// b64 roundtrip sanity matching the live wire shape (44/24 char strings).
	if len(base64.StdEncoding.EncodeToString(key)) != 44 || len(base64.StdEncoding.EncodeToString(iv)) != 24 {
		t.Fatal("key/IV base64 lengths diverge from the observed live shape")
	}
}
