package connector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type presentationMatrix struct {
	recordingMatrixAPI
	t          *testing.T
	content    *event.MessageEventContent
	getErr     error
	sendErr    error
	reads      int
	sends      int
	serialized []byte
}

func (m *presentationMatrix) UploadMedia(context.Context, id.RoomID, []byte, string, string) (id.ContentURIString, *event.EncryptedFileInfo, error) {
	m.uploadCalls++
	var file event.EncryptedFileInfo
	if err := json.Unmarshal([]byte(`{"url":"mxc://test/encrypted","v":"v2","iv":"test-iv","key":{"kty":"oct","alg":"A256CTR","k":"test-key","ext":true,"key_ops":["encrypt","decrypt"]},"hashes":{"sha256":"test-hash"}}`), &file); err != nil {
		m.t.Fatal(err)
	}
	return "", &file, nil
}

func (m *presentationMatrix) GetEvent(_ context.Context, room id.RoomID, eventID id.EventID) (*event.Event, error) {
	m.reads++
	if room != "!current:test" || eventID != "$original" {
		m.t.Errorf("unexpected event read: %s %s", room, eventID)
	}
	return &event.Event{Type: event.EventMessage, Content: event.Content{Parsed: m.content}}, m.getErr
}

func (m *presentationMatrix) SendMessage(_ context.Context, _ id.RoomID, _ event.Type, content *event.Content, _ *bridgev2.MatrixSendExtra) (*mautrix.RespSendEvent, error) {
	m.sends++
	var err error
	m.serialized, err = json.Marshal(content)
	if err != nil {
		m.t.Fatal(err)
	}
	return &mautrix.RespSendEvent{EventID: "$edit"}, m.sendErr
}

func (m *presentationMatrix) DownloadMedia(context.Context, id.ContentURIString, *event.EncryptedFileInfo) ([]byte, error) {
	m.t.Fatal("presentation repair must not download media")
	return nil, nil
}

func TestMediaPresentationConversion(t *testing.T) {
	for _, body := range []string{"Media", "A real caption", "media", "Media coverage"} {
		t.Run(body, func(t *testing.T) {
			matrix := &presentationMatrix{t: t}
			message := deliveryTestMessage(t)
			message.Text = body
			portal := &bridgev2.Portal{Portal: &database.Portal{MXID: "!current:test"}}
			converted, err := (&SnapchatAPI{}).convertMessage(t.Context(), portal, matrix, message)
			if err != nil {
				t.Fatal(err)
			}
			part := converted.Parts[0]
			wantBody := body
			if body == "Media" {
				wantBody = "snap.png"
			}
			if part.Content.Body != wantBody || part.Content.FileName != "snap.png" || part.Content.MsgType != event.MsgImage {
				t.Fatalf("wrong media presentation: %+v", part.Content)
			}
			if matrix.uploadCalls != 1 || part.DBMetadata.(*MediaDeliveryMetadata).PresentationVersion != 1 {
				t.Fatal("new media must upload once and carry presentation version 1")
			}
			raw, err := json.Marshal(part.Content)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]any
			if err := json.Unmarshal(raw, &wire); err != nil {
				t.Fatal(err)
			}
			if _, plainURL := wire["url"]; plainURL || wire["file"].(map[string]any)["url"] != "mxc://test/encrypted" {
				t.Fatalf("encrypted upload must use file.url, not url: %s", raw)
			}
			info := wire["info"].(map[string]any)
			if info["mimetype"] != "image/png" || info["size"] != float64(len(message.Media[0].Data)) {
				t.Fatalf("missing media info: %s", raw)
			}
		})
	}
	converted, err := (&SnapchatAPI{}).convertMessage(t.Context(), nil, nil, sidecar.Message{Text: "Media"})
	if err != nil || converted.Parts[0].Content.Body != "Media" || converted.Parts[0].Content.MsgType != event.MsgText {
		t.Fatalf("plain text Media must remain text: %+v, %v", converted, err)
	}
}

func TestMediaPresentationRepair(t *testing.T) {
	for _, tc := range []struct {
		name, body, fileName, wantBody string
		version                        int
		failRead, failSend             bool
	}{
		{name: "encrypted image", body: "Media", fileName: "photo.png", wantBody: "photo.png"},
		{name: "filename fallback", body: "Media", wantBody: "snap.png"},
		{name: "caption preserved", body: "A real caption", fileName: "photo.png"},
		{name: "already filename", body: "photo.png", fileName: "photo.png"},
		{name: "already versioned", body: "Media", version: 1},
		{name: "failed read", body: "Media", failRead: true},
		{name: "failed send", body: "Media", wantBody: "snap.png", failSend: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sa, portal := newMessageProofAPI(t, func(http.ResponseWriter, *http.Request) {
				t.Error("repair must use Bot.GetEvent, not a raw Matrix request")
			})
			matrix := &presentationMatrix{t: t}
			_, file, _ := matrix.UploadMedia(t.Context(), "", nil, "", "")
			matrix.uploadCalls = 0
			matrix.content = &event.MessageEventContent{
				MsgType: event.MsgImage, Body: tc.body, FileName: tc.fileName, File: file,
				Info: &event.FileInfo{MimeType: "image/png", Size: 123, Width: 10, Height: 20},
			}
			before, _ := json.Marshal(matrix.content)
			if tc.failRead {
				matrix.getErr = errors.New("decryption unavailable")
			}
			if tc.failSend {
				matrix.sendErr = errors.New("send failed")
			}
			portal.Bridge.Bot = matrix
			message := sidecar.Message{Media: []sidecar.MediaAttachment{{ID: "photo"}}}
			mappingID := "4114"
			if tc.name == "encrypted image" {
				mappingID = mediaMessageID(mappingID, message.Media)
			}
			target := &database.Message{
				ID:   makeMessageID(scopedSnapchatMessageID("chat-1", mappingID)),
				MXID: "$original", Room: portal.PortalKey, Timestamp: time.Now(),
				Metadata: &MediaDeliveryMetadata{MediaDelivered: true, AttachmentCount: 1, PresentationVersion: tc.version},
			}
			mq := portal.Bridge.DB.Message
			if err := mq.Insert(t.Context(), target); err != nil {
				t.Fatal(err)
			}
			edit, err := sa.convertMediaPresentationEdit(t.Context(), portal, matrix, []*database.Message{target}, nil)
			if (err != nil) != tc.failRead {
				t.Fatalf("unexpected conversion result: %v", err)
			}
			needsSend := tc.version == 0 && tc.body == "Media" && !tc.failRead
			if err == nil {
				if (len(edit.ModifiedParts) == 1) != needsSend || edit.AddedParts != nil || len(edit.DeletedParts) != 0 {
					t.Fatalf("unexpected edit parts: %+v", edit)
				}
				result := (*bridgev2.PortalInternals)(portal).SendConvertedEdit(t.Context(), target.ID, "", edit, matrix, time.Now(), 0)
				if result.Success == tc.failSend {
					t.Fatalf("send result must reflect failure: %+v", result)
				}
			}
			persisted, err := mq.GetPartByMXID(t.Context(), target.MXID)
			if err != nil || persisted == nil {
				t.Fatalf("original mapping lost: %v", err)
			}
			wantVersion := 1
			if tc.failRead || tc.failSend {
				wantVersion = 0
			}
			if persisted.Metadata.(*MediaDeliveryMetadata).PresentationVersion != wantVersion || matrix.uploadCalls != 0 {
				t.Fatalf("incorrect persisted version or unexpected upload: %+v", persisted.Metadata)
			}
			if target.Metadata.(*MediaDeliveryMetadata).PresentationVersion != tc.version {
				t.Fatal("conversion mutated the original mapping metadata")
			}
			after, _ := json.Marshal(matrix.content)
			if string(before) != string(after) {
				t.Fatal("repair mutated original event content")
			}
			if needsSend {
				var sent event.MessageEventContent
				if err := json.Unmarshal(matrix.serialized, &sent); err != nil {
					t.Fatal(err)
				}
				if sent.RelatesTo.EventID != target.MXID || sent.RelatesTo.Type != event.RelReplace || sent.NewContent.Body != tc.wantBody {
					t.Fatalf("invalid replacement: %s", matrix.serialized)
				}
				var originalWire, sentWire map[string]any
				if err := json.Unmarshal(before, &originalWire); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(matrix.serialized, &sentWire); err != nil {
					t.Fatal(err)
				}
				newWire := sentWire["m.new_content"].(map[string]any)
				if !reflect.DeepEqual(newWire["file"], originalWire["file"]) || !reflect.DeepEqual(newWire["info"], originalWire["info"]) || sent.NewContent.URL != "" {
					t.Fatal("replacement changed encryption or image info")
				}
			} else if matrix.sends != 0 {
				t.Fatal("unchanged or unreadable content must not send")
			}
			if tc.failSend {
				matrix.sendErr = nil
				retry, err := sa.convertMediaPresentationEdit(t.Context(), portal, matrix, []*database.Message{persisted}, nil)
				if err != nil {
					t.Fatal(err)
				}
				result := (*bridgev2.PortalInternals)(portal).SendConvertedEdit(t.Context(), target.ID, "", retry, matrix, time.Now(), 0)
				persisted, err = mq.GetPartByMXID(t.Context(), target.MXID)
				if !result.Success || err != nil || persisted.Metadata.(*MediaDeliveryMetadata).PresentationVersion != 1 {
					t.Fatalf("failed edit was not retryable: %+v, %v", result, err)
				}
				wantVersion = 1
			}
			if wantVersion == 1 {
				reads := matrix.reads
				if err := sa.repairMediaPresentation(t.Context(), sidecar.Chat{ID: "chat-1"}, "4114", message); err != nil {
					t.Fatal(err)
				}
				if matrix.reads != reads {
					t.Fatal("version 1 must skip event inspection")
				}
			}
			wantReads := 1
			if tc.version == 1 {
				wantReads = 0
			}
			if tc.failSend {
				wantReads++
			}
			if matrix.reads != wantReads {
				t.Fatalf("unexpected reads (including post-send): %d", matrix.reads)
			}
		})
	}
}
