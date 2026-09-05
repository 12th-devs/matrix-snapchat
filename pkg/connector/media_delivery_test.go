package connector

import (
	"bytes"
	"context"
	"errors"
	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"image"
	"image/png"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"testing"
)

type failingMediaMatrix struct{ recordingMatrixAPI }

func (m *failingMediaMatrix) UploadMedia(context.Context, id.RoomID, []byte, string, string) (id.ContentURIString, *event.EncryptedFileInfo, error) {
	return "", nil, errors.New("temporary upload failure")
}

func deliveryTestMessage(t *testing.T) sidecar.Message {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	return sidecar.Message{ID: "123-media-photo", Text: "Media", Media: []sidecar.MediaAttachment{{ID: "photo", Data: buf.Bytes(), MimeType: "image/png"}}}
}

func TestMediaUploadFailureRemainsRetryable(t *testing.T) {
	portal := &bridgev2.Portal{Portal: &database.Portal{MXID: "!room:example.com"}}
	api := &SnapchatAPI{}
	message := deliveryTestMessage(t)
	converted, err := api.convertMessage(context.Background(), portal, &failingMediaMatrix{}, message)
	if err == nil || converted != nil {
		t.Fatal("failed upload must not commit a fallback placeholder under the media ID")
	}
	converted, err = api.convertMessage(context.Background(), portal, &recordingMatrixAPI{}, message)
	if err != nil || len(converted.Parts) != 1 || converted.Parts[0].Content.MsgType != event.MsgImage {
		t.Fatalf("retry did not produce image: %v %#v", err, converted)
	}
	if converted.Parts[0].ID != "" {
		t.Fatal("first part mapping must remain unchanged")
	}
}

func TestDeliveredMediaRequiresCompletePersistedMapping(t *testing.T) {
	meta := &MediaDeliveryMetadata{MediaDelivered: true, AttachmentCount: 2}
	first := &database.Message{MXID: "$one", Metadata: meta}
	second := &database.Message{MXID: "$two", Metadata: meta}
	if deliveredMediaParts([]*database.Message{first}, 2) {
		t.Fatal("partial send cannot mark hydration complete")
	}
	if !deliveredMediaParts([]*database.Message{first, second}, 2) {
		t.Fatal("complete delivery should reconcile")
	}
	second.Metadata = &MediaDeliveryMetadata{}
	if deliveredMediaParts([]*database.Message{first, second}, 2) {
		t.Fatal("placeholder mapping cannot prove delivery")
	}
}

func TestMediaEditSkipsDeliveredPart(t *testing.T) {
	portal := &bridgev2.Portal{Portal: &database.Portal{MXID: "!room:example.com"}}
	existing := []*database.Message{{MXID: "$one", Metadata: &MediaDeliveryMetadata{MediaDelivered: true, AttachmentCount: 1}}}
	edit, err := (&SnapchatAPI{}).convertMessageEdit(context.Background(), portal, &recordingMatrixAPI{}, existing, deliveryTestMessage(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(edit.ModifiedParts) != 0 || edit.AddedParts != nil {
		t.Fatal("re-poll must not emit another hydration edit")
	}
}
