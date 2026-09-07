package connector

import (
	"testing"

	"maunium.net/go/mautrix/event"
)

// TestChatCapabilitiesMediaFlag pins the outbound media capability contract:
// with send_media_enabled the connector must advertise m.image (including
// image/png and image/webp) so the mautrix-go pre-handler capability check
// passes; with the flag off the File map is empty and the framework rejects
// media events with "unsupported message type" before the connector ever
// sees them.
func TestChatCapabilitiesMediaFlag(t *testing.T) {
	enabled := (&SnapchatConnector{Config: ConnectorConfig{SendMediaEnabled: true}}).chatCapabilities()
	if got := enabled.File[event.MsgImage]; got == nil || got.MimeTypes["image/png"] != event.CapLevelFullySupported || got.MimeTypes["image/jpeg"] != event.CapLevelFullySupported || got.MimeTypes["image/webp"] != event.CapLevelFullySupported {
		t.Fatalf("SendMediaEnabled=true must fully support m.image (jpeg+png+webp), got %#v", enabled.File[event.MsgImage])
	}
	if got := enabled.File[event.MsgVideo]; got == nil || got.MimeTypes["video/mp4"] != event.CapLevelFullySupported {
		t.Fatalf("SendMediaEnabled=true must fully support m.video/mp4, got %#v", enabled.File[event.MsgVideo])
	}
	// Sticker-shaped events (m.sticker / msgtype-less media) ride the image
	// path; without this key the framework rejects them pre-handler with
	// "unsupported message type".
	if got := enabled.File[event.CapMsgSticker]; got == nil {
		t.Fatal("SendMediaEnabled=true must advertise sticker support (rides the image path)")
	}
	if got := enabled.File[event.MsgFile]; got == nil {
		t.Fatal("SendMediaEnabled=true must advertise m.file (connector still rejects it later by design)")
	}
	if got := enabled.File[event.MsgVideo]; got == nil {
		t.Fatal("SendMediaEnabled=true must advertise m.video (connector still rejects it later by design)")
	}
	voice := enabled.File[event.CapMsgVoice]
	if voice == nil || voice.MimeTypes["audio/mp4"] != event.CapLevelFullySupported {
		t.Fatalf("SendMediaEnabled=true must advertise MSC3245 voice with audio/mp4, got %#v", voice)
	}
	// OGG/Opus and WebM recordings from Beeper clients are transcoded to
	// audio/mp4 by the connector, so they are advertised fully supported.
	if voice.MimeTypes["audio/ogg"] != event.CapLevelFullySupported || voice.MimeTypes["audio/webm"] != event.CapLevelFullySupported {
		t.Fatalf("voice capability must fully support audio/ogg and audio/webm (ffmpeg-transcoded), got %#v", voice.MimeTypes)
	}

	disabled := (&SnapchatConnector{Config: ConnectorConfig{SendMediaEnabled: false}}).chatCapabilities()
	if len(disabled.File) != 0 {
		t.Fatalf("SendMediaEnabled=false must advertise no file capabilities, got %#v", disabled.File)
	}
	if disabled.ID != enabled.ID {
		t.Fatalf("capabilities ID must not depend on the media flag: %q vs %q", enabled.ID, disabled.ID)
	}
}
