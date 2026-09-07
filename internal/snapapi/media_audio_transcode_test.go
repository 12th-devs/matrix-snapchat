package snapapi

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/0xzer/snapper/protos"
)

// TestTranscodableAudioMIME pins which outbound voice-note MIME types the
// send path routes through ffmpeg transcoding instead of rejecting.
func TestTranscodableAudioMIME(t *testing.T) {
	for _, mime := range []string{"audio/ogg", "application/ogg", "audio/opus", "audio/webm"} {
		if !transcodableAudioMIME(mime) {
			t.Fatalf("transcodableAudioMIME(%q) = false; Beeper clients record OGG/Opus and WebM", mime)
		}
	}
	for _, mime := range []string{"audio/mp4", "audio/mpeg", "video/mp4", "image/jpeg", ""} {
		if transcodableAudioMIME(mime) {
			t.Fatalf("transcodableAudioMIME(%q) = true; only OGG/Opus and WebM need transcoding", mime)
		}
	}
}

// TestTranscodeVoiceNoteToMP4WithFFmpeg runs a real ffmpeg round trip:
// synthesize a one-second OGG/Opus note (what Beeper Android uploads), run
// it through transcodeVoiceNoteToMP4, and verify the result classifies as a
// valid audio-only Snapchat voice note with a usable duration. Skipped when
// ffmpeg is not installed.
func TestTranscodeVoiceNoteToMP4WithFFmpeg(t *testing.T) {
	if _, err := ffmpegPath(); err != nil {
		t.Skipf("ffmpeg not available: %v", err)
	}
	ctx := context.Background()
	ogg, err := runFFmpegOutput(ctx,
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:a", "libopus", "-b:a", "32k", "-f", "ogg", "-",
	)
	if err != nil {
		t.Fatalf("failed to synthesize test OGG/Opus: %v", err)
	}
	if !bytes.Contains(ogg, []byte("OpusHead")) {
		t.Fatalf("synthesized fixture is not an Opus stream (%d bytes)", len(ogg))
	}
	converted, err := transcodeVoiceNoteToMP4(ctx, ogg, "application/ogg")
	if err != nil {
		t.Fatalf("transcodeVoiceNoteToMP4: %v", err)
	}
	info, err := classifyOutgoingAudio(converted, "audio/mp4")
	if err != nil {
		t.Fatalf("transcoded bytes failed voice-note classification: %v", err)
	}
	if info.MediaType != protos.MediaType_MEDIA_TYPE_AUDIO {
		t.Fatalf("transcoded voice note MediaType = %s, want MEDIA_TYPE_AUDIO", info.MediaType)
	}
	if info.DurationMs <= 0 || info.DurationMs > 3000 {
		t.Fatalf("transcoded voice note duration = %dms, want ~1000ms", info.DurationMs)
	}
	// classifyOutgoingAudio rejects MP4s with video tracks before returning,
	// so a successful classification with AUDIO type proves audio-only.
	if info.Width != 0 || info.Height != 0 {
		t.Fatalf("transcoded voice note must not carry dimensions: %+v", info)
	}
}

// runFFmpegOutput is a small helper for test fixture synthesis: it runs
// ffmpeg with the given args and returns stdout.
func runFFmpegOutput(ctx context.Context, args ...string) ([]byte, error) {
	bin, err := ffmpegPath()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}
