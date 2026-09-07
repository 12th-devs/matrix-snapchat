package snapapi

import (
	"testing"

	"github.com/0xzer/snapper/protos"
)

// TestStatusEventText pins the friendly system-line text for known status
// content types: these render as gray m.notice room events in Beeper (like
// Snapchat's own "Screenshot captured" / "Missed video call" lines), never as
// snaps or chatty text.
func TestStatusEventText(t *testing.T) {
	cases := map[protos.ContentType]string{
		protos.ContentType_STATUS_SAVE_TO_CAMERA_ROLL:             "Saved to camera roll",
		protos.ContentType_STATUS_CONVERSATION_CAPTURE_SCREENSHOT: "Screenshot captured",
		protos.ContentType_STATUS_CONVERSATION_CAPTURE_RECORD:     "Screen recorded",
		protos.ContentType_STATUS_CALL_MISSED_VIDEO:               "Missed video call",
		protos.ContentType_STATUS_CALL_MISSED_AUDIO:               "Missed audio call",
		protos.ContentType_STATUS_INVITE_LINK_CHANGE:              "Invite link changed",
		protos.ContentType_SHARE:                                  "Shared content",
	}
	for contentType, want := range cases {
		text, isSnap, known := statusEventText(contentType)
		if !known {
			t.Fatalf("statusEventText(%s) not marked known", contentType)
		}
		if isSnap {
			t.Fatalf("statusEventText(%s) must not be a snap", contentType)
		}
		if text != want {
			t.Fatalf("statusEventText(%s) = %q, want %q", contentType, text, want)
		}
	}

	// Stickers ride the media path: empty text, not a snap, but known.
	text, isSnap, known := statusEventText(protos.ContentType_STICKER)
	if !known || isSnap || text != "" {
		t.Fatalf("statusEventText(STICKER) = (%q, %t, %t), want empty non-snap known", text, isSnap, known)
	}

	// Unmapped content types (including generic STATUS) stay unknown so the
	// caller keeps its existing rendering.
	for _, contentType := range []protos.ContentType{
		protos.ContentType_STATUS,
		protos.ContentType_CHAT,
		protos.ContentType_SNAP,
		protos.ContentType_LOCATION,
		protos.ContentType(99),
	} {
		if _, _, known := statusEventText(contentType); known {
			t.Fatalf("statusEventText(%s) must be unknown", contentType)
		}
	}
}

// TestPreviewEnvelopeBytes pins the log helper's truncation and binary-safe
// escaping.
func TestPreviewEnvelopeBytes(t *testing.T) {
	if got := previewEnvelopeBytes([]byte("hello\x00world")); got != "hello.world" {
		t.Fatalf("previewEnvelopeBytes = %q", got)
	}
	long := make([]byte, 128)
	for i := range long {
		long[i] = 'a'
	}
	if got := previewEnvelopeBytes(long); len(got) != 64 {
		t.Fatalf("previewEnvelopeBytes truncated to %d bytes, want 64", len(got))
	}
}
