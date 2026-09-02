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
// than the stale version carried by receipt metadata, and that reads go
// through UpdateConversation with the UpdateConversationRead action (the
// operation Snapchat Web uses to advance the ReadHighWatermark;
// UpdateContentMessage+UpdateAction_Read is rejected with
// UPDATE_NOT_APPLICABLE).
func TestMarkReadUsesFreshConversationVersion(t *testing.T) {
	const (
		chatID       = "aaaaaaaa-0000-0000-0000-000000000001"
		staleVersion = int64(100)
		freshVersion = int64(500)
		messageID    = int64(42)
	)
	var updateRequests []*protos.UpdateConversationRequest
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
		case urlPath(paths.UPDATE_CONVERSATION):
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("update frame unreadable: %v", err2)
			}
			var req protos.UpdateConversationRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("update proto undecodable: %v", err2)
			}
			updateRequests = append(updateRequests, &req)
			payload, _ := protos.EncodeProtoMessage(&protos.UpdateConversationResponse{
				UpdateData: &protos.UpdateConversationResult{
					Success:        true,
					CurrentVersion: freshVersion,
				},
			})
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
	read := req.GetRead()
	if read == nil {
		t.Fatal("UpdateConversationAction must be Read")
	}
	if read.GetSelfUserId() == nil {
		t.Fatal("UpdateConversationRead.SelfUserId must be set")
	}
	if got := read.GetReadConversationMessageData().GetLastMessageId(); got != messageID {
		t.Fatalf("ReadConversationMessageData.LastMessageId = %d, want %d", got, messageID)
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
	var updateRequests []*protos.UpdateConversationRequest
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
		case urlPath(paths.UPDATE_CONVERSATION):
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("update frame unreadable: %v", err2)
			}
			var req protos.UpdateConversationRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("update proto undecodable: %v", err2)
			}
			updateRequests = append(updateRequests, &req)
			payload, _ := protos.EncodeProtoMessage(&protos.UpdateConversationResponse{
				UpdateData: &protos.UpdateConversationResult{Success: true},
			})
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

// TestMarkReadSkipsWhenSelfWatermarkAlreadyCoversTarget verifies duplicate
// Matrix read receipts do not send no-op updates that Snapchat may reject.
func TestMarkReadSkipsWhenSelfWatermarkAlreadyCoversTarget(t *testing.T) {
	const (
		chatID       = "aaaaaaaa-0000-0000-0000-000000000004"
		freshVersion = int64(300)
		messageID    = int64(42)
	)
	updateCalls := 0
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
						Version:        freshVersion,
						Participants: []*protos.Participant{{
							UserId:            req.GetSelfUserId(),
							ReadHighWatermark: messageID,
						}},
					},
				},
			})
			w.Write(grpcWebData(payload))
		case urlPath(paths.UPDATE_CONVERSATION):
			updateCalls++
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTestClient(t, handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	if err := c.MarkRead(context.Background(), chatID, messageID, freshVersion); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", updateCalls)
	}
}

// TestMarkReadRejectsUnsuccessfulResponse verifies a response without
// UpdateConversationResult.Success=true is treated as a failure so a read is
// never claimed without a server commit.
func TestMarkReadRejectsUnsuccessfulResponse(t *testing.T) {
	const (
		chatID       = "aaaaaaaa-0000-0000-0000-000000000003"
		freshVersion = int64(300)
		messageID    = int64(9)
	)
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
						Version:        freshVersion,
					},
				},
			})
			w.Write(grpcWebData(payload))
		case urlPath(paths.UPDATE_CONVERSATION):
			payload, _ := protos.EncodeProtoMessage(&protos.UpdateConversationResponse{
				UpdateData: &protos.UpdateConversationResult{Success: false},
			})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTestClient(t, handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	if err := c.MarkRead(context.Background(), chatID, messageID, freshVersion); err == nil {
		t.Fatal("MarkRead must fail when Success=false")
	}
}
