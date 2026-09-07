package connector

import (
	"bytes"
	"context"
	"log"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"

	"github.com/colej/mautrix-snapchat/internal/store"
)

// These tests pin the fix for "Snapchat is not told when I read an incoming
// text in an already-open Beeper room": Beeper clients only send explicit
// m.read receipts on a fresh passive open, so bridgev2's implicit read
// receipts (emitted on outgoing sends when ImplicitReadReceipts is
// advertised) are the only read signal for messages read while the room is
// already open. The handler previously discarded them outright.

// TestGetCapabilitiesAdvertisesImplicitReadReceipts pins that the capability
// follows read_receipts_enabled so the receipts-off configuration stays off.
func TestGetCapabilitiesAdvertisesImplicitReadReceipts(t *testing.T) {
	enabled := (&SnapchatConnector{Config: ConnectorConfig{ReadReceiptsEnabled: true}}).GetCapabilities()
	if !enabled.ImplicitReadReceipts {
		t.Fatal("read_receipts_enabled=true must advertise ImplicitReadReceipts, otherwise reads during active chatting never reach Snapchat")
	}
	disabled := (&SnapchatConnector{Config: ConnectorConfig{ReadReceiptsEnabled: false}}).GetCapabilities()
	if disabled.ImplicitReadReceipts {
		t.Fatal("read_receipts_enabled=false must not advertise ImplicitReadReceipts")
	}
}

func newImplicitReceiptTestAPI(t *testing.T, readReceiptsEnabled bool, rows ...store.MessageState) *SnapchatAPI {
	t.Helper()
	db, err := store.New(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	now := time.Now()
	for i := range rows {
		if rows[i].LastSeenAt.IsZero() {
			rows[i].LastSeenAt = now
		}
	}
	if len(rows) > 0 {
		if err = db.UpsertMessages(rows); err != nil {
			t.Fatal(err)
		}
	}
	sc := &SnapchatConnector{Config: ConnectorConfig{ReadReceiptsEnabled: readReceiptsEnabled}, store: db}
	sa := &SnapchatAPI{Connector: sc}
	sa.mu.Lock()
	sa.lastReadReceiptSync = make(map[string]time.Time)
	sa.lastMessageID = make(map[string]int64)
	sa.mu.Unlock()
	return sa
}

func implicitReceipt(chatID string) *bridgev2.MatrixReadReceipt {
	return &bridgev2.MatrixReadReceipt{
		Portal:   &bridgev2.Portal{Portal: &database.Portal{PortalKey: networkid.PortalKey{ID: networkid.PortalID(chatID)}}},
		Implicit: true,
		ReadUpTo: time.Now(),
	}
}

func captureHandlerLogs(t *testing.T, run func()) string {
	t.Helper()
	var buf bytes.Buffer
	flags := log.Flags()
	writer := log.Writer()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(writer)
		log.SetFlags(flags)
	}()
	run()
	return buf.String()
}

// TestHandleMatrixReadReceiptAcceptsImplicitReceipt verifies that an implicit
// read receipt (target resolved to the last known incoming message, stored as
// chat text) proceeds past the former receipt.Implicit skip all the way to the
// API client step. The test API has no client, so reaching the
// "API client unavailable" branch proves every gate was passed and MarkRead
// was the next action.
func TestHandleMatrixReadReceiptAcceptsImplicitReceipt(t *testing.T) {
	const chatID = "chat-implicit-1"
	sa := newImplicitReceiptTestAPI(t, true, store.MessageState{
		PortalKey: chatID,
		RemoteID:  "123",
		Text:      "Read this",
		Kind:      "chat",
	})
	sa.mu.Lock()
	sa.lastMessageID[chatID] = 123
	sa.mu.Unlock()

	var logs string
	logs = captureHandlerLogs(t, func() {
		if err := sa.HandleMatrixReadReceipt(context.Background(), implicitReceipt(chatID)); err != nil {
			t.Fatalf("HandleMatrixReadReceipt: %v", err)
		}
	})
	if !strings.Contains(logs, "API client unavailable") {
		t.Fatalf("implicit receipt must pass all skips and reach the MarkRead client step, logs:\n%s", logs)
	}
	if strings.Contains(logs, "action=skip") {
		t.Fatalf("implicit receipt must not be skipped, logs:\n%s", logs)
	}
}

// TestHandleMatrixReadReceiptImplicitDisabledStillSkipped verifies the
// receipts-off configuration keeps dropping implicit receipts entirely.
func TestHandleMatrixReadReceiptImplicitDisabledStillSkipped(t *testing.T) {
	const chatID = "chat-implicit-off"
	sa := newImplicitReceiptTestAPI(t, false)
	sa.mu.Lock()
	sa.lastMessageID[chatID] = 123
	sa.mu.Unlock()

	var logs string
	logs = captureHandlerLogs(t, func() {
		if err := sa.HandleMatrixReadReceipt(context.Background(), implicitReceipt(chatID)); err != nil {
			t.Fatalf("HandleMatrixReadReceipt: %v", err)
		}
	})
	if !strings.Contains(logs, "action=skip") {
		t.Fatalf("implicit receipts must stay skipped when read_receipts_enabled=false, logs:\n%s", logs)
	}
	if strings.Contains(logs, "API client unavailable") {
		t.Fatalf("implicit receipts must not reach the API step when read_receipts_enabled=false, logs:\n%s", logs)
	}
}

// TestShouldSendReadReceiptToSnapchatImplicitSnapTarget verifies the snap
// guard also holds for implicit (non-exact) receipts so a text sent right
// after a snap can never advance the watermark onto the snap (safeNoOpen).
func TestShouldSendReadReceiptToSnapchatImplicitSnapTarget(t *testing.T) {
	const chatID = "chat-implicit-snap"
	sa := newImplicitReceiptTestAPI(t, true, store.MessageState{
		PortalKey: chatID,
		RemoteID:  "124",
		Text:      "New Snap",
		Kind:      "snap",
		HasMedia:  true,
	})
	if sa.shouldSendReadReceiptToSnapchat(chatID, 124, false) {
		t.Fatal("snap target must be rejected for implicit (non-exact) receipts too")
	}
}
