package connector

import (
	"testing"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
)

// TestIsSystemEventContentType pins which Snapchat content types render as
// gray m.notice room events instead of chatty m.text messages: all STATUS_*
// lines and shared items. Ordinary chat, snaps, media, and voice notes must
// stay regular messages.
func TestIsSystemEventContentType(t *testing.T) {
	for _, contentType := range []string{
		"STATUS",
		"STATUS_SAVE_TO_CAMERA_ROLL",
		"STATUS_CONVERSATION_CAPTURE_SCREENSHOT",
		"STATUS_CONVERSATION_CAPTURE_RECORD",
		"STATUS_CALL_MISSED_VIDEO",
		"STATUS_CALL_MISSED_AUDIO",
		"STATUS_INVITE_LINK_CHANGE",
		"SHARE",
	} {
		if !isSystemEventContentType(contentType) {
			t.Fatalf("isSystemEventContentType(%q) = false, want true", contentType)
		}
	}
	for _, contentType := range []string{"", "CHAT", "SNAP", "EXTERNAL_MEDIA", "NOTE", "STICKER", "MYSTATUS"} {
		if isSystemEventContentType(contentType) {
			t.Fatalf("isSystemEventContentType(%q) = true, want false", contentType)
		}
	}
}

// TestSystemEventPreviewText pins the sidebar-preview mapping for
// client-rendered system lines (screenshots, screen recordings, missed
// calls): they must map to the same friendly text the message path uses.
func TestSystemEventPreviewText(t *testing.T) {
	cases := map[string]string{
		"screenshot - 2m":            "Screenshot captured",
		"took a screenshot of chat!": "Screenshot captured",
		"screen recording - 1m":      "Screen recorded",
		"screenrecord":               "Screen recorded",
		"missed video call - 3m":     "Missed video call",
		"missed audio - 4m":          "Missed audio call",
		"missed call":                "Missed audio call",
	}
	for preview, want := range cases {
		got, ok := systemEventPreviewText(preview)
		if !ok || got != want {
			t.Fatalf("systemEventPreviewText(%q) = (%q, %t), want (%q, true)", preview, got, ok, want)
		}
	}
	for _, preview := range []string{"delivered - 5m", "opened", "new snap - 8m", ""} {
		if _, ok := systemEventPreviewText(preview); ok {
			t.Fatalf("systemEventPreviewText(%q) must not map", preview)
		}
	}
}

// TestSidebarUpdateTextSystemLines pins the sidebar placeholder contract:
// screenshot-style previews map to friendly system text, passive statuses
// stay suppressed, and generic previews keep the "New message" placeholder
// (which the notice gate renders as m.notice).
func TestSidebarUpdateTextSystemLines(t *testing.T) {
	text, ok := sidebarUpdateText(sidecar.Chat{Name: "Lorelei", Preview: "Screenshot - 2m", Unread: true})
	if !ok || text != "Screenshot captured" {
		t.Fatalf("sidebarUpdateText(screenshot) = (%q, %t), want (Screenshot captured, true)", text, ok)
	}
	if text, ok := sidebarUpdateText(sidecar.Chat{Preview: "Delivered - 5m"}); ok {
		t.Fatalf("sidebarUpdateText(passive) = (%q, %t), want suppressed", text, ok)
	}
	if text, ok := sidebarUpdateText(sidecar.Chat{Preview: "Typing something"}); !ok || text != "New message" {
		t.Fatalf("sidebarUpdateText(generic) = (%q, %t), want (New message, true)", text, ok)
	}
	if text, ok := sidebarUpdateText(sidecar.Chat{Preview: "New Snap - 8m", Unread: true}); !ok || text != "New Snap" {
		t.Fatalf("sidebarUpdateText(snap) = (%q, %t), want (New Snap, true)", text, ok)
	}
}
