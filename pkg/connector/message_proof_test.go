package connector

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/colej/mautrix-snapchat/internal/config"
	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/snapapi"
	"github.com/colej/mautrix-snapchat/internal/store"
	"github.com/rs/zerolog"
	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/matrix"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/id"
)

// Initialize bridge caches without starting Matrix or the connector runtime.
type messageProofMatrix struct{ bridgev2.MatrixConnector }

func (*messageProofMatrix) Init(*bridgev2.Bridge)         {}
func (*messageProofMatrix) BotIntent() bridgev2.MatrixAPI { return nil }

func newMessageProofAPI(t *testing.T, handler http.HandlerFunc) (*SnapchatAPI, *bridgev2.Portal) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := mautrix.NewClient(server.URL, "@bot:test", "test")
	if err != nil {
		t.Fatal(err)
	}
	client.DefaultHTTPRetries = 0
	client.Client = server.Client()
	client.Client.Timeout = time.Second
	raw, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "bridge.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	db, err := dbutil.NewWithDB(raw, "sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	sc := &SnapchatConnector{}
	br := bridgev2.NewBridge("proof", db, zerolog.Nop(), nil, &messageProofMatrix{}, sc,
		func(*bridgev2.Bridge) bridgev2.CommandProcessor { return nil })
	br.BackgroundCtx = t.Context()
	br.Matrix = &matrix.Connector{Bot: &appservice.IntentAPI{Client: client}}
	if err := br.DB.Upgrade(t.Context()); err != nil {
		t.Fatal(err)
	}
	key := networkid.PortalKey{ID: "chat-1", Receiver: "login-1"}
	if err := br.DB.Portal.Insert(t.Context(), &database.Portal{PortalKey: key, MXID: "!current:test"}); err != nil {
		t.Fatal(err)
	}
	portal, err := br.GetExistingPortalByKey(t.Context(), key)
	if err != nil {
		t.Fatal(err)
	}
	return &SnapchatAPI{Connector: sc, UserLogin: &bridgev2.UserLogin{
		UserLogin: &database.UserLogin{ID: key.Receiver}, Bridge: br,
	}}, portal
}

func TestCurrentMessagePartsLiveProof(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		stale  bool
	}{
		{"present", 200, `{"event_id":"$probe","type":"m.room.message","content":{}}`, false},
		{"not found", 404, `{"errcode":"M_NOT_FOUND","error":"gone"}`, true},
		{"user forbidden", 403, `{"errcode":"M_FORBIDDEN","error":"forbidden"}`, true},
		{"unknown token", 403, `{"errcode":"M_UNKNOWN_TOKEN"}`, false},
		{"missing token", 403, `{"errcode":"M_MISSING_TOKEN"}`, false},
		{"unauthorized", 401, `{"errcode":"M_UNKNOWN_TOKEN"}`, false},
		{"rate limited", 429, `{"errcode":"M_LIMIT_EXCEEDED"}`, false},
		{"server error", 503, `{"errcode":"M_NOT_FOUND"}`, false},
		{"unstructured miss", 404, `not a Matrix error`, false},
		{"transport failure", 0, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			sa, portal := newMessageProofAPI(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet {
					t.Errorf("unexpected method %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/_matrix/client/v3/rooms/!current:test/event/$probe":
					if tc.status == 0 {
						conn, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						_ = conn.Close()
						return
					}
					w.WriteHeader(tc.status)
					_, _ = fmt.Fprint(w, tc.body)
				case "/_matrix/client/v3/rooms/!current:test/event/$kept":
					_, _ = fmt.Fprint(w, `{"event_id":"$kept","type":"m.room.message","content":{}}`)
				default:
					t.Errorf("event proof did not use current portal MXID: %s", r.URL.Path)
					http.Error(w, "unexpected path", 500)
				}
			})
			mq := sa.UserLogin.Bridge.DB.Message
			for i, eventID := range []id.EventID{"$probe", "$kept"} {
				if err := mq.Insert(t.Context(), &database.Message{
					ID: makeMessageID(scopedSnapchatMessageID("chat-1", "123")), PartID: mediaPartID(i),
					MXID: eventID, Room: portal.PortalKey, Timestamp: time.Now(),
				}); err != nil {
					t.Fatal(err)
				}
			}
			parts, err := sa.currentMessageParts(t.Context(), "chat-1", "123")
			if err != nil {
				t.Fatal(err)
			}
			want := 2
			if tc.stale {
				want = 1
			}
			if len(parts) != want || requests.Load() != 2 {
				t.Fatalf("parts=%d requests=%d, want parts=%d requests=2", len(parts), requests.Load(), want)
			}
			if tc.stale && parts[0].MXID != "$kept" {
				t.Fatalf("wrong part survived: %s", parts[0].MXID)
			}
			for _, eventID := range []id.EventID{"$probe", "$kept"} {
				persisted, err := mq.GetPartByMXID(t.Context(), eventID)
				if err != nil {
					t.Fatal(err)
				}
				if wantPresent := eventID == "$kept" || !tc.stale; (persisted != nil) != wantPresent {
					t.Fatalf("mapping %s present=%t, want %t", eventID, persisted != nil, wantPresent)
				}
			}
		})
	}
}

func TestMediaDeliveryPostHandleDoesNotGetEvent(t *testing.T) {
	var requests atomic.Int32
	sa, portal := newMessageProofAPI(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"errcode":"M_NOT_FOUND"}`)
	})
	state, err := store.New(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = state.Close() })
	sa.Connector.store = state
	message := sidecar.Message{ID: "123-media-photo", Media: []sidecar.MediaAttachment{{ID: "photo"}}}
	sa.seenByChat = map[string]map[string]struct{}{"chat-1": {message.ID: {}}}
	if err := sa.UserLogin.Bridge.DB.Message.Insert(t.Context(), &database.Message{
		ID:   makeMessageID(scopedSnapchatMessageID("chat-1", message.ID)),
		MXID: "$sent", Room: portal.PortalKey, Timestamp: time.Now(),
		Metadata: &MediaDeliveryMetadata{MediaDelivered: true, AttachmentCount: 1},
	}); err != nil {
		t.Fatal(err)
	}
	handle := sa.mediaDeliveryPostHandle(sidecar.Chat{ID: "chat-1"}, message.ID, message)
	if handle == nil {
		t.Fatal("missing media post-handle")
	}
	handle(t.Context(), portal)
	stored, err := state.GetMessage("chat-1", "123")
	if err != nil || stored == nil || stored.HydratedAt == nil {
		t.Fatalf("persisted send did not hydrate message: state=%+v err=%v", stored, err)
	}
	if requests.Load() != 0 {
		t.Fatalf("post-handle performed %d HTTP requests", requests.Load())
	}
	if _, seen := sa.seenByChat["chat-1"][message.ID]; seen {
		t.Fatal("post-handle did not clear seen state")
	}
}

func TestPollOnceReturnsWhileResyncLocked(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/chats" {
			t.Errorf("unexpected poll request: %s %s", r.Method, r.URL.Path)
		}
		http.Error(w, "test poll reached connector", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	sa := &SnapchatAPI{
		UserLogin: &bridgev2.UserLogin{Bridge: &bridgev2.Bridge{}},
		Connector: &SnapchatConnector{Config: ConnectorConfig{APIMode: "hybrid"}},
		Client:    sidecar.New(config.ConnectorConfig{BaseURL: server.URL, RequestTimeoutSeconds: 1}),
	}
	sa.resyncMu.Lock()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		sa.pollOnce(t.Context())
	}()
	select {
	case <-finished:
		sa.resyncMu.Unlock()
	case <-time.After(time.Second):
		sa.resyncMu.Unlock()
		<-finished
		t.Fatal("poll blocked on resyncMu instead of returning")
	}
	if requests.Load() != 0 {
		t.Fatal("poll reached connector while resyncMu was held")
	}
	// The same initialized API must actually poll once the lock is available.
	sa.pollOnce(context.Background())
	if requests.Load() != 1 {
		t.Fatalf("unlocked poll made %d requests, want 1", requests.Load())
	}
	if !sa.resyncMu.TryLock() {
		t.Fatal("poll did not release resyncMu")
	}
	sa.resyncMu.Unlock()
}

func TestResyncCoverageIncludesOnlyExistingLoginPortals(t *testing.T) {
	sa, _ := newMessageProofAPI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("coverage must not perform network requests")
	})
	state, err := store.New(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = state.Close() })
	sa.Connector.store = state
	if err := state.UpsertPortal(store.PortalState{PortalKey: "chat-1", RemoteID: "chat-1", RemoteName: "Stored name", OtherUserID: "friend"}); err != nil {
		t.Fatal(err)
	}
	if err := state.UpsertPortal(store.PortalState{PortalKey: "not-a-portal", RemoteID: "not-a-portal"}); err != nil {
		t.Fatal(err)
	}
	if err := sa.UserLogin.Bridge.DB.Portal.Insert(t.Context(), &database.Portal{PortalKey: networkid.PortalKey{ID: "other-login", Receiver: "someone-else"}}); err != nil {
		t.Fatal(err)
	}
	chats, err := sa.resyncCoverage(t.Context(), []snapapi.Chat{{ID: "discovered", Name: "Live"}, {ID: "discovered"}})
	if err != nil || len(chats) != 2 {
		t.Fatalf("coverage=%+v err=%v", chats, err)
	}
	if chats[0].ID != "discovered" || chats[1].ID != "chat-1" || chats[1].Name != "Stored name" || chats[1].OtherUserID != "friend" {
		t.Fatalf("wrong union: %+v", chats)
	}
	chats, err = sa.resyncCoverage(t.Context(), []snapapi.Chat{{ID: "chat-1", Name: "Fresh"}})
	if err != nil || len(chats) != 1 || chats[0].Name != "Fresh" {
		t.Fatalf("discovery must win: %+v %v", chats, err)
	}
}
