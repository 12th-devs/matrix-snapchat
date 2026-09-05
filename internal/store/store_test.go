package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMessageStateMetadataPersists(t *testing.T) {
	db, err := New(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.UpsertMessages([]MessageState{{
		PortalKey:    "chat-1",
		RemoteID:     "123",
		Author:       "Emerson",
		Text:         "New Snap",
		Kind:         "snap",
		HasMedia:     true,
		TimestampRaw: "2026-05-17T12:00:00Z",
	}})
	if err != nil {
		t.Fatal(err)
	}

	state, err := db.GetMessage("chat-1", "123")
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("message state was not persisted")
	}
	if state.Kind != "snap" || !state.HasMedia || state.HydratedAt != nil {
		t.Fatalf("unexpected state: kind=%q has_media=%t hydrated=%v", state.Kind, state.HasMedia, state.HydratedAt)
	}

	if err = db.MarkMessageHydrated("chat-1", "123"); err != nil {
		t.Fatal(err)
	}
	state, err = db.GetMessage("chat-1", "123")
	if err != nil {
		t.Fatal(err)
	}
	if state == nil || state.HydratedAt == nil {
		t.Fatalf("hydrated state was not marked: %#v", state)
	}
}

func TestMessageHydratedAtSurvivesNilUpsert(t *testing.T) {
	db, err := New(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	hydratedAt := time.Date(2026, 9, 4, 15, 0, 0, 0, time.UTC)
	if err = db.UpsertMessages([]MessageState{{
		PortalKey:  "chat-1",
		RemoteID:   "123",
		Text:       "Media",
		Kind:       "media",
		HasMedia:   true,
		HydratedAt: &hydratedAt,
	}}); err != nil {
		t.Fatal(err)
	}
	if err = db.UpsertMessages([]MessageState{{
		PortalKey: "chat-1",
		RemoteID:  "123",
		Text:      "Media",
		Kind:      "media",
		HasMedia:  true,
	}}); err != nil {
		t.Fatal(err)
	}

	state, err := db.GetMessage("chat-1", "123")
	if err != nil {
		t.Fatal(err)
	}
	if state == nil || state.HydratedAt == nil {
		t.Fatalf("hydrated_at was not preserved: %#v", state)
	}
	if !state.HydratedAt.Equal(hydratedAt) {
		t.Fatalf("hydrated_at = %s, want %s", state.HydratedAt.Format(time.RFC3339Nano), hydratedAt.Format(time.RFC3339Nano))
	}
}

func TestPortalOtherUserIDPersists(t *testing.T) {
	db, err := New(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.UpsertPortal(PortalState{
		PortalKey:   "chat-1",
		RemoteID:    "chat-1",
		RemoteName:  "Loreleiii",
		OtherUserID: "33432eb1-9099-4bda-8b85-e75cc899c2a2",
		Preview:     "hey",
		LastMessage: "hey",
		Unread:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	state, err := db.GetPortalByKey("chat-1")
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("portal state was not persisted")
	}
	if state.OtherUserID != "33432eb1-9099-4bda-8b85-e75cc899c2a2" {
		t.Fatalf("unexpected other user id: %q", state.OtherUserID)
	}
}

func TestPortalOtherUserIDSurvivesEmptySidebarRefresh(t *testing.T) {
	db, err := New(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const participantID = "33432eb1-9099-4bda-8b85-e75cc899c2a2"
	if err = db.UpsertPortal(PortalState{
		PortalKey:   "chat-1",
		RemoteID:    "chat-1",
		RemoteName:  "Loreleiii",
		OtherUserID: participantID,
		Preview:     "hey",
		LastMessage: "hey",
		Unread:      true,
	}); err != nil {
		t.Fatal(err)
	}

	if err = db.UpsertPortal(PortalState{
		PortalKey:   "chat-1",
		RemoteID:    "chat-1",
		RemoteName:  "Loreleiii",
		OtherUserID: "",
		Preview:     "new preview",
		LastMessage: "new preview",
		Unread:      false,
	}); err != nil {
		t.Fatal(err)
	}

	state, err := db.GetPortalByKey("chat-1")
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("portal state was not persisted")
	}
	if state.OtherUserID != participantID {
		t.Fatalf("empty refresh clobbered participant id: %q", state.OtherUserID)
	}
	if state.Preview != "new preview" || state.LastMessage != "new preview" || state.Unread {
		t.Fatalf("non-identity fields were not refreshed: preview=%q last=%q unread=%t", state.Preview, state.LastMessage, state.Unread)
	}
}
