package connector

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"

	"github.com/colej/mautrix-snapchat/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	db, err := store.New(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func matrixMessageReplyingTo(target string) *bridgev2.MatrixMessage {
	return &bridgev2.MatrixMessage{
		ReplyTo: &database.Message{ID: networkid.MessageID(target)},
	}
}

// TestMatrixReplyTarget verifies the Beeper reply relation resolves to the
// numeric Snapchat message ID, and that unresolvable targets fall back to 0
// (send without reply) instead of failing.
func TestMatrixReplyTarget(t *testing.T) {
	sa := &SnapchatAPI{}
	if got := sa.matrixReplyTarget("chat-1", nil); got != 0 {
		t.Fatalf("nil message returned %d, want 0", got)
	}
	if got := sa.matrixReplyTarget("chat-1", &bridgev2.MatrixMessage{}); got != 0 {
		t.Fatalf("no-reply message returned %d, want 0", got)
	}
	msg := matrixMessageReplyingTo("chat-1.msg.456")
	if got := sa.matrixReplyTarget("chat-1", msg); got != 456 {
		t.Fatalf("reply target = %d, want 456", got)
	}
	// Media message ID form strips to the base Snapchat ID.
	if got := sa.matrixReplyTarget("chat-1", matrixMessageReplyingTo("chat-1.msg.123-media-abc")); got != 123 {
		t.Fatalf("media reply target = %d, want 123", got)
	}
	// Non-Snapchat target (e.g. ignored IDs) -> no reply.
	if got := sa.matrixReplyTarget("chat-1", matrixMessageReplyingTo("ignored-abc")); got != 0 {
		t.Fatalf("non-snapchat reply target resolved to %d, want 0", got)
	}
}

// TestResolveReplyTargetUsesPersistedMapping verifies incoming replies map to
// the Matrix event via the persistent store, and a missing target falls back
// cleanly (nil, no crash) so the message still arrives.
func TestResolveReplyTargetUsesPersistedMapping(t *testing.T) {
	db, err := store.New(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertMessages([]store.MessageState{{
		PortalKey:  "chat-1",
		RemoteID:   "42",
		Text:       "original",
		Kind:       "chat",
		LastSeenAt: time.Now(),
	}}); err != nil {
		t.Fatal(err)
	}
	sa := &SnapchatAPI{Connector: &SnapchatConnector{store: db}}

	resolved := sa.resolveReplyTarget("chat-1", "42")
	if resolved == nil {
		t.Fatal("stored quoted message should resolve to a reply target")
	}
	if string(resolved.MessageID) != "chat-1.msg.42" {
		t.Fatalf("resolved reply target = %s, want chat-1.msg.42", resolved.MessageID)
	}

	if missing := sa.resolveReplyTarget("chat-1", "999999"); missing != nil {
		t.Fatal("missing quoted message must fall back to no reply relation")
	}
	// Zero/empty quoted IDs never resolve.
	if sa.resolveReplyTarget("chat-1", "") != nil || sa.resolveReplyTarget("chat-1", "0") != nil {
		t.Fatal("empty/zero quoted id must produce no reply relation")
	}
	// Missing portal scope: safe nil.
	if missing := sa.resolveReplyTarget("", "42"); missing != nil {
		t.Fatal("missing portal id must fall back to no reply")
	}
}

// TestHandleMatrixEditExplicitlyRejected verifies edits return a clear error
// (Snapchat has no edit operation) instead of faking anything.
func TestHandleMatrixEditExplicitlyRejected(t *testing.T) {
	sa := &SnapchatAPI{}
	err := sa.HandleMatrixEdit(context.Background(), &bridgev2.MatrixEdit{
		EditTarget: &database.Message{ID: networkid.MessageID("chat-1.msg.7")},
	})
	if err == nil {
		t.Fatal("expected an explicit rejection for edits")
	}
	if !strings.Contains(err.Error(), "does not support editing") {
		t.Fatalf("unexpected rejection error: %v", err)
	}
}

// TestHandleMatrixEditNilTarget verifies nil targets are no-ops.
func TestHandleMatrixEditNilTarget(t *testing.T) {
	sa := &SnapchatAPI{}
	if err := sa.HandleMatrixEdit(context.Background(), nil); err != nil {
		t.Fatalf("nil edit must be a no-op, got: %v", err)
	}
	if err := sa.HandleMatrixEdit(context.Background(), &bridgev2.MatrixEdit{}); err != nil {
		t.Fatalf("nil target edit must be a no-op, got: %v", err)
	}
}

// TestChatIDFromScopedMessageID verifies chat scope extraction used by deletes.
func TestChatIDFromScopedMessageID(t *testing.T) {
	cases := map[string]string{
		"chat-1.msg.123":       "chat-1",
		"abc.def-uuid.msg.999": "abc.def-uuid",
		"no-scope":             "",
		"":                     "",
	}
	for input, want := range cases {
		if got := chatIDFromScopedMessageID(input); got != want {
			t.Errorf("chatIDFromScopedMessageID(%q) = %q, want %q", input, got, want)
		}
	}
}
