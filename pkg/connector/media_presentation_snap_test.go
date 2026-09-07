package connector

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"testing"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/event"
)

const snapTypeExtraKey = "net.colej.snapchat.type"

func presentationTestPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func presentationTestMP4(t *testing.T) []byte {
	t.Helper()
	box := func(kind string, payload []byte) []byte {
		out := make([]byte, 8+len(payload))
		binary.BigEndian.PutUint32(out, uint32(8+len(payload)))
		copy(out[4:8], kind)
		copy(out[8:], payload)
		return out
	}
	var buf bytes.Buffer
	buf.Write(box("ftyp", []byte("isom")))
	buf.Write(box("moov", []byte("mvhd")))
	buf.Write(box("mdat", []byte("frame")))
	return buf.Bytes()
}

func TestConvertMessageDistinguishesSnapsFromChatMedia(t *testing.T) {
	pngData := presentationTestPNG(t)
	mp4Data := presentationTestMP4(t)
	portal := &bridgev2.Portal{Portal: &database.Portal{MXID: "!current:test"}}
	cases := []struct {
		name        string
		message     sidecar.Message
		wantBody    string
		wantMsgType event.MessageType
		wantMarker  string
	}{
		{
			name: "image snap",
			message: sidecar.Message{ID: "500", IsSnap: true, ContentType: "SNAP", Text: "New Snap",
				Media: []sidecar.MediaAttachment{{ID: "s1", Data: pngData, MimeType: "image/png"}}},
			wantBody:    "📷 Snap",
			wantMsgType: event.MsgImage,
			wantMarker:  "snap_image",
		},
		{
			name: "video snap",
			message: sidecar.Message{ID: "501", ContentType: "SNAP", Text: "New Snap",
				Media: []sidecar.MediaAttachment{{ID: "s2", Data: mp4Data, MimeType: "video/mp4"}}},
			wantBody:    "🎥 Snap",
			wantMsgType: event.MsgVideo,
			wantMarker:  "snap_video",
		},
		{
			name: "saved chat media",
			message: sidecar.Message{ID: "502", ContentType: "EXTERNAL_MEDIA", Text: "Media",
				Media: []sidecar.MediaAttachment{{ID: "c1", Data: pngData, MimeType: "image/png"}}},
			wantBody:    "snap.png",
			wantMsgType: event.MsgImage,
			wantMarker:  "chat_media",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matrix := &presentationMatrix{t: t}
			converted, err := (&SnapchatAPI{}).convertMessage(t.Context(), portal, matrix, tc.message)
			if err != nil {
				t.Fatal(err)
			}
			if len(converted.Parts) != 1 {
				t.Fatalf("part count = %d, want 1", len(converted.Parts))
			}
			part := converted.Parts[0]
			content := part.Content
			if content.MsgType != tc.wantMsgType {
				t.Fatalf("msgtype = %s, want %s", content.MsgType, tc.wantMsgType)
			}
			if content.Body != tc.wantBody {
				t.Fatalf("body = %q, want %q", content.Body, tc.wantBody)
			}
			marker, ok := part.Extra[snapTypeExtraKey].(string)
			if !ok || marker != tc.wantMarker {
				t.Fatalf("extra[%s] = %#v, want %q", snapTypeExtraKey, part.Extra[snapTypeExtraKey], tc.wantMarker)
			}
			if content.FileName == "" {
				t.Fatal("encrypted media must keep an explicit FileName")
			}
		})
	}
}

func TestSnapPlaceholderTextStaysNotice(t *testing.T) {
	for _, tc := range []struct {
		text        string
		contentType string
		isSnap      bool
		wantNotice  bool
	}{
		{text: "📷 New Snap", contentType: "SNAP", isSnap: true, wantNotice: true},
		{text: "🎥 New Snap", contentType: "SNAP", isSnap: true, wantNotice: true},
		{text: "📷 Snap", contentType: "SNAP", isSnap: true, wantNotice: true},
		{text: "🎥 Snap", contentType: "SNAP", isSnap: true, wantNotice: true},
		{text: "hello", contentType: "CHAT", wantNotice: false},
		{text: "Media", contentType: "EXTERNAL_MEDIA", wantNotice: false},
	} {
		got := isGeneratedSnapchatNotice(tc.text)
		if got != tc.wantNotice {
			t.Fatalf("isGeneratedSnapchatNotice(%q) = %t, want %t", tc.text, got, tc.wantNotice)
		}
	}
}
