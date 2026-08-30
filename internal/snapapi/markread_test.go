package snapapi

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/0xzer/snapper/data/methods"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
)

// TestMarkReadUsesFreshConversationVersion verifies that MarkRead sends the
// CURRENT conversation version (from the freshly synced conversation) rather
// than the stale version carried by receipt metadata. Snapchat rejects read
// updates with an outdated CurrentVersion (success=false).
func TestMarkReadUsesFreshConversationVersion(t *testing.T) {
	const (
		chatID       = "aaaaaaaa-0000-0000-0000-000000000001"
		staleVersion = int64(100)
		freshVersion = int64(500)
		messageID    = int64(42)
	)
	var updateRequests []*protos.UpdateContentMessageRequest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		switch r.URL.Path {
		case urlPath(paths.DELTA_SYNC_CONVERSATIONS):
			// conversation() fetches the CURRENT conversation state.
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("delta frame unreadable: %v", err2)
			}
			var req protos.DeltaSyncRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("delta proto undecodable: %v", err2)
			}
			convID := req.GetConversationId()
			payload, _ := protos.EncodeProtoMessage(&protos.DeltaSyncResponse{
				Metadata: &protos.DeltaSyncResponse_Conversation{
					Conversation: &protos.Conversation{
						ConversationId: convID,
						Version:        freshVersion,
						Participants: []*protos.Participant{
							{UserId: convID, ReadHighWatermark: messageID},
						},
					},
				},
			})
			w.Write(grpcWebData(payload))
		case urlPath(paths.UPDATE_CONTENT_MESSAGE):
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("update frame unreadable: %v", err2)
			}
			var req protos.UpdateContentMessageRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("update proto undecodable: %v", err2)
			}
			updateRequests = append(updateRequests, &req)
			payload, _ := protos.EncodeProtoMessage(&protos.UpdateContentMessageResponse{Success: true})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTestClient(t, handler)
	defer srv.Close()
	overrideListingURL(t, srv)
	// Pre-seed a STALE cached conversation: MarkRead must bypass the cache and
	// DeltaSync the CURRENT version (a cached version is stale by definition).
	c.mu.Lock()
	c.conversations[chatID] = &protos.Conversation{Version: staleVersion}
	c.mu.Unlock()

	if err := c.MarkRead(context.Background(), chatID, messageID, staleVersion); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if len(updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(updateRequests))
	}
	req := updateRequests[0]
	if req.GetCurrentVersion() != freshVersion {
		t.Fatalf("CurrentVersion = %d, want fresh %d (not stale %d)", req.GetCurrentVersion(), freshVersion, staleVersion)
	}
	if req.GetUpdate().GetMessageId() != messageID {
		t.Fatalf("Update.MessageId = %d, want %d", req.GetUpdate().GetMessageId(), messageID)
	}
	if req.GetUpdate().GetRead() == nil {
		t.Fatal("Update must be a Read action")
	}
}

// TestMarkReadFallsBackWhenConversationMissingVersion verifies the receipt
// version is used when the fresh conversation carries no version.
func TestMarkReadFallsBackWhenConversationMissingVersion(t *testing.T) {
	const (
		chatID       = "aaaaaaaa-0000-0000-0000-000000000002"
		staleVersion = int64(123)
		messageID    = int64(7)
	)
	var updateRequests []*protos.UpdateContentMessageRequest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		switch r.URL.Path {
		case urlPath(paths.DELTA_SYNC_CONVERSATIONS):
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("delta frame unreadable: %v", err2)
			}
			var req protos.DeltaSyncRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("delta proto undecodable: %v", err2)
			}
			payload, _ := protos.EncodeProtoMessage(&protos.DeltaSyncResponse{
				Metadata: &protos.DeltaSyncResponse_Conversation{
					Conversation: &protos.Conversation{
						ConversationId: req.GetConversationId(),
						Version:        0, // missing version
					},
				},
			})
			w.Write(grpcWebData(payload))
		case urlPath(paths.UPDATE_CONTENT_MESSAGE):
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("update frame unreadable: %v", err2)
			}
			var req protos.UpdateContentMessageRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("update proto undecodable: %v", err2)
			}
			updateRequests = append(updateRequests, &req)
			payload, _ := protos.EncodeProtoMessage(&protos.UpdateContentMessageResponse{Success: true})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTestClient(t, handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	if err := c.MarkRead(context.Background(), chatID, messageID, staleVersion); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if len(updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(updateRequests))
	}
	if req := updateRequests[0]; req.GetCurrentVersion() != staleVersion {
		t.Fatalf("CurrentVersion = %d, want receipt fallback %d", req.GetCurrentVersion(), staleVersion)
	}
}
