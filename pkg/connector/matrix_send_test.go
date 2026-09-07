package connector

import (
	"strings"
	"testing"

	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/event"
)

func textMessage(body string) *bridgev2.MatrixMessage {
	return &bridgev2.MatrixMessage{
		MatrixEventBase: bridgev2.MatrixEventBase[*event.MessageEventContent]{
			Content: &event.MessageEventContent{
				MsgType: event.MsgText,
				Body:    body,
			},
		},
	}
}

func TestMatrixTextToSendAcceptsUserAuthoredWords(t *testing.T) {
	sa := &SnapchatAPI{}
	bodies := []string{
		"sent",
		"read",
		"seen",
		"opened",
		"delivered",
		"new snap",
		"received",
		"Sent from Beeper",
		"read this now",
		"delivered: 2m",
		"hello there, arbitrary user text",
	}
	for _, body := range bodies {
		got, ok := sa.matrixTextToSend(textMessage(body))
		if !ok {
			t.Fatalf("matrixTextToSend(%q) was rejected; user-authored text must send", body)
		}
		if got != body {
			t.Fatalf("matrixTextToSend(%q) = %q, want the trimmed-free body preserved", body, got)
		}
	}
}

func TestMatrixTextToSendStillRejectsBridgeGeneratedMarkers(t *testing.T) {
	sa := &SnapchatAPI{}
	bodies := []string{
		"[Snapchat status] STATUS",
		"[Snapchat message unavailable]",
		"[Unsupported Snapchat event]",
		"  ",
	}
	for _, body := range bodies {
		if _, ok := sa.matrixTextToSend(textMessage(body)); ok {
			t.Fatalf("matrixTextToSend(%q) was accepted; bridge-generated markers must stay suppressed", body)
		}
	}
}

func TestMatrixTextToSendRejectsNonTextMsgTypes(t *testing.T) {
	sa := &SnapchatAPI{}
	notice := &bridgev2.MatrixMessage{MatrixEventBase: bridgev2.MatrixEventBase[*event.MessageEventContent]{
		Content: &event.MessageEventContent{MsgType: event.MsgNotice, Body: "hello"},
	}}
	if _, ok := sa.matrixTextToSend(notice); ok {
		t.Fatal("m.notice must not be sent as a Snapchat chat text")
	}
}

func TestBridgeGeneratedMarkerOnlyMatchesBracketedNotices(t *testing.T) {
	accepted := []string{"sent", "read", "seen", "opened", "delivered", "sent a message about new snap"}
	for _, body := range accepted {
		if isBridgeGeneratedSnapchatMarker(body) {
			t.Fatalf("isBridgeGeneratedSnapchatMarker(%q) = true; ordinary words must never match", body)
		}
	}
	rejected := []string{"[Snapchat status] STATUS", "[Unsupported Snapchat event]", "[Snapchat message unavailable]"}
	for _, body := range rejected {
		if !isBridgeGeneratedSnapchatMarker(body) {
			t.Fatalf("isBridgeGeneratedSnapchatMarker(%q) = false; bracketed notices must match", body)
		}
	}
}

func TestMatrixVoiceNotePassesMsgTypeGate(t *testing.T) {
	sa := &SnapchatAPI{Connector: &SnapchatConnector{Config: ConnectorConfig{SendMediaEnabled: true}}}
	voice := &bridgev2.MatrixMessage{MatrixEventBase: bridgev2.MatrixEventBase[*event.MessageEventContent]{
		Content: &event.MessageEventContent{MsgType: event.MsgAudio, Body: "voice-message.m4a", URL: "mxc://test/audio"},
	}}
	_, _, ok, err := sa.matrixMediaToSend(t.Context(), voice)
	// m.audio must pass the msgtype gate and the send-media switch; with no
	// Matrix client stub it must then stop at the missing downloader rather
	// than being silently ignored.
	if !ok || err == nil || !strings.Contains(err.Error(), "media downloader") {
		t.Fatalf("m.audio gate: ok=%v err=%v; voice notes must reach the media pipeline", ok, err)
	}
	notice := &bridgev2.MatrixMessage{MatrixEventBase: bridgev2.MatrixEventBase[*event.MessageEventContent]{
		Content: &event.MessageEventContent{MsgType: event.MsgNotice, Body: "hello"},
	}}
	if _, _, ok, err := sa.matrixMediaToSend(t.Context(), notice); ok || err != nil {
		t.Fatalf("m.notice must stay ignored: ok=%v err=%v", ok, err)
	}
}

func TestDefaultMediaFileNameVoiceNote(t *testing.T) {
	if got := defaultMediaFileName(event.MsgAudio, "audio/mp4"); got != "snap-audio.m4a" {
		t.Fatalf("defaultMediaFileName(audio/mp4) = %q, want snap-audio.m4a", got)
	}
	if got := defaultMediaFileName(event.MsgAudio, "audio/ogg"); got != "snap-audio.mp3" {
		t.Fatalf("defaultMediaFileName(audio/ogg) = %q, want snap-audio.mp3", got)
	}
}
