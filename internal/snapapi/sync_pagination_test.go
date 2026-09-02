package snapapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xzer/snapper/data/methods"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
)

// grpcWebData frames a proto response payload as a grpc-web data frame:
// 1-byte flag (0x00 = data) + 4-byte big-endian length + payload.
func grpcWebData(payload []byte) []byte {
	n := len(payload)
	body := []byte{0x00, byte(n >> 24 & 0xff), byte(n >> 16 & 0xff), byte(n >> 8 & 0xff), byte(n & 0xff)}
	return append(body, payload...)
}

// decodeQueryRequest decodes the framed QueryConversationsRequest a doGRPC call sends.
func decodeQueryRequest(t *testing.T, body []byte) *protos.QueryConversationsRequest {
	t.Helper()
	frame, err := methods.ReadResponseFrame(body)
	if err != nil {
		t.Fatalf("request frame unreadable: %v", err)
	}
	var req protos.QueryConversationsRequest
	if err := protos.DecodeProtoMessage(frame, &req); err != nil {
		t.Fatalf("request proto undecodable: %v", err)
	}
	return &req
}

func mustEntry(t *testing.T, id string) *protos.ConversationEntry {
	t.Helper()
	return &protos.ConversationEntry{
		VersionInfo: &protos.ConversationVersionInfo{
			ConversationId:      mustUUID(t, id),
			ConversationVersion: 1,
		},
	}
}

func mustUUID(t *testing.T, s string) *protos.UUID {
	t.Helper()
	u, err := encodeUUIDString(s)
	if err != nil {
		t.Fatalf("encode uuid: %v", err)
	}
	return u
}

// newAuthedTestClient is a test client with SSO token and self identity
// pre-populated so Sync skips the SSO/identity fetches (no network).
func newAuthedTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	c, srv := newTestClient(handler)
	c.tokens.SSO_TOKEN = "test-sso-token"
	c.selfUserID = "99999999-9999-9999-9999-999999999999"
	selfUUID, err := encodeUUIDString(c.selfUserID)
	if err != nil {
		t.Fatalf("encode self uuid: %v", err)
	}
	c.selfEncoded = selfUUID
	return c, srv
}

// overrideListingURL points the sync endpoints used by the full-listing path
// at the test server and restores the production constants when the test
// finishes.
func overrideListingURL(t *testing.T, srv *httptest.Server) {
	t.Helper()
	origSync := syncConversationsURL
	origQuery := queryConversationsURL
	origBatch := batchDeltaSyncURL
	origDelta := deltaSyncURL
	origUpdate := updateContentMessageURL
	origCreate := createContentMessageURL
	origUpdateConv := updateConversationURL
	syncConversationsURL = atSrv(srv, paths.SYNC_CONVERSATIONS)
	queryConversationsURL = atSrv(srv, paths.QUERY_CONVERSATIONS)
	batchDeltaSyncURL = atSrv(srv, paths.BATCH_DELTA_SYNC_CONVERSATIONS)
	deltaSyncURL = atSrv(srv, paths.DELTA_SYNC_CONVERSATIONS)
	updateContentMessageURL = atSrv(srv, paths.UPDATE_CONTENT_MESSAGE)
	createContentMessageURL = atSrv(srv, paths.CREATE_CONTENT_MESSAGE)
	updateConversationURL = atSrv(srv, paths.UPDATE_CONVERSATION)
	t.Cleanup(func() {
		syncConversationsURL = origSync
		queryConversationsURL = origQuery
		batchDeltaSyncURL = origBatch
		deltaSyncURL = origDelta
		updateContentMessageURL = origUpdate
		createContentMessageURL = origCreate
		updateConversationURL = origUpdateConv
	})
}

// TestQueryAllConversationsPagination verifies the full-sync conversation
// listing: fresh (token-less) start, indicator-based paging, and the stop
// conditions. This locks in the fix for first syncs returning only the
// SyncConversations page (40 of 74 chats).
func TestQueryAllConversationsPagination(t *testing.T) {
	var requests []*protos.QueryConversationsRequest
	pageCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		requests = append(requests, decodeQueryRequest(t, body))
		var resp *protos.QueryConversationsResponse
		switch pageCount {
		case 0:
			// First page: two entries, more pages available.
			resp = &protos.QueryConversationsResponse{
				Conversations: []*protos.ConversationEntry{
					mustEntry(t, "11111111-1111-1111-1111-111111111111"),
					mustEntry(t, "22222222-2222-2222-2222-222222222222"),
				},
				LastConversation: &protos.LastConversationIndicator{
					OldestConversationOrderTimestamp: 1234,
					OldestConversationId:             mustUUID(t, "22222222-2222-2222-2222-222222222222"),
				},
				SyncToken: []byte("page0-token"),
			}
		case 1:
			// Second page: one entry, no more.
			resp = &protos.QueryConversationsResponse{
				Conversations: []*protos.ConversationEntry{mustEntry(t, "33333333-3333-3333-3333-333333333333")},
				NoMore:        true,
				SyncToken:     []byte("page1-token"),
			}
		default:
			t.Error("server served more pages than expected")
			resp = &protos.QueryConversationsResponse{NoMore: true}
		}
		pageCount++
		payload, err := protos.EncodeProtoMessage(resp)
		if err != nil {
			t.Fatalf("encode response: %v", err)
		}
		w.Write(grpcWebData(payload))
	})
	c, srv := newTestClient(handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	entries, token, err := c.queryAllConversations(context.Background(), nil, 100)
	if err != nil {
		t.Fatalf("queryAllConversations: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	if string(token) != "page1-token" {
		t.Fatalf("token = %q, want page1-token", token)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(requests))
	}
	// The fresh listing must start with NO sync token: carrying the
	// incremental token made the server return nothing (the 40-of-74 bug).
	if len(requests[0].GetSyncToken()) != 0 {
		t.Fatalf("page 0 request carried a sync token (%q), want empty", requests[0].GetSyncToken())
	}
	if requests[0].GetPaginationInfo() != nil {
		t.Fatal("page 0 request must not carry pagination info")
	}
	// Page 1 must continue from page 0's indicator + token.
	if requests[1].GetPaginationInfo() == nil {
		t.Fatal("page 1 request missing pagination info")
	}
	if got := requests[1].GetPaginationInfo().GetOldestConversationOrderTimestamp(); got != 1234 {
		t.Fatalf("page 1 cursor timestamp = %d, want 1234", got)
	}
	if string(requests[1].GetSyncToken()) != "page0-token" {
		t.Fatalf("page 1 token = %q, want page0-token", requests[1].GetSyncToken())
	}
}

// TestQueryAllConversationsStopsOnEmptyPage verifies an empty page (without
// no_more) ends pagination instead of looping.
func TestQueryAllConversationsStopsOnEmptyPage(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, _ := protos.EncodeProtoMessage(&protos.QueryConversationsResponse{})
		w.Write(grpcWebData(payload))
	})
	c, srv := newTestClient(handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	entries, _, err := c.queryAllConversations(context.Background(), nil, 100)
	if err != nil {
		t.Fatalf("queryAllConversations: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %d, want 0", len(entries))
	}
}

// TestQueryAllConversationsStopsWithoutIndicator verifies that a full page
// without a last-conversation indicator ends pagination (no cursor to continue
// from) without an error.
func TestQueryAllConversationsStopsWithoutIndicator(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, _ := protos.EncodeProtoMessage(&protos.QueryConversationsResponse{
			Conversations: []*protos.ConversationEntry{mustEntry(t, "44444444-4444-4444-4444-444444444444")},
		})
		w.Write(grpcWebData(payload))
	})
	c, srv := newTestClient(handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	entries, _, err := c.queryAllConversations(context.Background(), nil, 100)
	if err != nil {
		t.Fatalf("queryAllConversations: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
}

// TestQueryAllConversationsSurfacesErrors verifies page errors are returned
// (with page context) instead of silently truncating the listing, and that an
// auth rejection on the listing endpoint is classified as ErrUnauthorized.
func TestQueryAllConversationsSurfacesErrors(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"permission denied"}`))
	}))
	defer srv.Close()
	overrideListingURL(t, srv)

	_, _, err := c.queryAllConversations(context.Background(), nil, 100)
	if err == nil {
		t.Fatal("expected an error from the failing listing")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

// TestHydrateConversationsChunksBatches verifies BatchDeltaSync is chunked to
// at most 20 requests (Snapchat rejects oversized batches with grpc-status:3)
// and that a rejected chunk falls back to per-conversation DeltaSync.
func TestHydrateConversationsChunksBatches(t *testing.T) {
	batchSizes := []int{}
	batchedIDs := map[string]bool{}
	individuallySynced := map[string]bool{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case urlPath(paths.BATCH_DELTA_SYNC_CONVERSATIONS):
			frame, err := methods.ReadResponseFrame(body)
			if err != nil {
				t.Fatalf("batch frame unreadable: %v", err)
			}
			var req protos.BatchDeltaSyncRequest
			if err := protos.DecodeProtoMessage(frame, &req); err != nil {
				t.Fatalf("batch proto undecodable: %v", err)
			}
			batchSizes = append(batchSizes, len(req.GetDeltaSyncRequests()))
			// Reject any batch larger than 20 like the real server does.
			if len(req.GetDeltaSyncRequests()) > 20 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			for _, dr := range req.GetDeltaSyncRequests() {
				batchedIDs[uuidToString(dr.GetConversationId())] = true
			}
			var items []*protos.DeltaSyncResponseWrapper
			for _, dr := range req.GetDeltaSyncRequests() {
				items = append(items, &protos.DeltaSyncResponseWrapper{
					Response: &protos.DeltaSyncResponseWrapper_SuccessResponse{
						SuccessResponse: &protos.DeltaSyncResponse{
							Metadata: &protos.DeltaSyncResponse_Conversation{
								Conversation: &protos.Conversation{ConversationId: dr.GetConversationId()},
							},
						},
					},
				})
			}
			payload, _ := protos.EncodeProtoMessage(&protos.BatchDeltaSyncResponse{DeltaSyncResponses: items})
			w.Write(grpcWebData(payload))
		case urlPath(paths.DELTA_SYNC_CONVERSATIONS):
			frame, err := methods.ReadResponseFrame(body)
			if err != nil {
				t.Fatalf("delta frame unreadable: %v", err)
			}
			var req protos.DeltaSyncRequest
			if err := protos.DecodeProtoMessage(frame, &req); err != nil {
				t.Fatalf("delta proto undecodable: %v", err)
			}
			individuallySynced[uuidToString(req.GetConversationId())] = true
			payload, _ := protos.EncodeProtoMessage(&protos.DeltaSyncResponse{
				Metadata: &protos.DeltaSyncResponse_Conversation{
					Conversation: &protos.Conversation{ConversationId: req.GetConversationId()},
				},
			})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newTestClient(handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	// 63 conversations: 4 chunks (20/20/20/3) like the observed production stall.
	var entries []*protos.ConversationEntry
	for i := 0; i < 63; i++ {
		entries = append(entries, mustEntry(t, fmt.Sprintf("00000000-0000-0000-0000-%012d", i)))
	}
	if err := c.hydrateConversations(context.Background(), entries); err != nil {
		t.Fatalf("hydrateConversations: %v", err)
	}
	for i, size := range batchSizes {
		if size > 20 {
			t.Fatalf("batch %d had %d requests, want <= 20", i, size)
		}
	}
	total := 0
	for _, size := range batchSizes {
		total += size
	}
	if total != 63 {
		t.Fatalf("batched requests total = %d, want 63", total)
	}
	for id := range batchedIDs {
		if individuallySynced[id] {
			t.Fatalf("conversation %s was both batched and individually synced", id)
		}
	}
}

// TestHydrateChunkFallbackSyncsIndividually verifies that when a chunk is
// rejected, its conversations are retried one-by-one and failures are skipped
// without failing the rest.
func TestHydrateChunkFallbackSyncsIndividually(t *testing.T) {
	batchCalls := 0
	deltaCalls := map[string]int{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case urlPath(paths.BATCH_DELTA_SYNC_CONVERSATIONS):
			batchCalls++
			// Reject every batch like the production grpc-status:3 failure.
			w.WriteHeader(http.StatusForbidden)
		case urlPath(paths.DELTA_SYNC_CONVERSATIONS):
			frame, err := methods.ReadResponseFrame(body)
			if err != nil {
				t.Fatalf("delta frame unreadable: %v", err)
			}
			var req protos.DeltaSyncRequest
			if err := protos.DecodeProtoMessage(frame, &req); err != nil {
				t.Fatalf("delta proto undecodable: %v", err)
			}
			deltaCalls[uuidToString(req.GetConversationId())]++
			payload, _ := protos.EncodeProtoMessage(&protos.DeltaSyncResponse{
				Metadata: &protos.DeltaSyncResponse_Conversation{
					Conversation: &protos.Conversation{ConversationId: req.GetConversationId()},
				},
			})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newTestClient(handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	entries := []*protos.ConversationEntry{
		mustEntry(t, "aaaaaaaa-0000-0000-0000-000000000001"),
		mustEntry(t, "aaaaaaaa-0000-0000-0000-000000000002"),
		mustEntry(t, "aaaaaaaa-0000-0000-0000-000000000003"),
	}
	// Hydration must NOT return an error even when every batch is rejected:
	// the fallback covers the conversations and the sync must continue.
	if err := c.hydrateConversations(context.Background(), entries); err != nil {
		t.Fatalf("hydrateConversations returned an error despite fallback: %v", err)
	}
	if batchCalls != 1 {
		t.Fatalf("batch calls = %d, want 1", batchCalls)
	}
	if len(deltaCalls) != 3 {
		t.Fatalf("individually synced conversations = %d, want 3", len(deltaCalls))
	}
	c.mu.Lock()
	cached := len(c.conversations)
	c.mu.Unlock()
	if cached != 3 {
		t.Fatalf("cached conversations = %d, want 3", cached)
	}
}

// TestSyncConversationListingUsesFreshToken verifies through the public Sync
// entry point that the full-listing request carries no incremental token on a
// first sync.
func TestSyncConversationListingUsesFreshToken(t *testing.T) {
	var listingRequests []*protos.QueryConversationsRequest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case urlPath(paths.QUERY_CONVERSATIONS):
			body, _ := io.ReadAll(r.Body)
			listingRequests = append(listingRequests, decodeQueryRequest(t, body))
			payload, _ := protos.EncodeProtoMessage(&protos.QueryConversationsResponse{
				Conversations: []*protos.ConversationEntry{mustEntry(t, "55555555-5555-5555-5555-555555555555")},
				NoMore:        true,
			})
			w.Write(grpcWebData(payload))
		case urlPath(paths.SYNC_CONVERSATIONS):
			payload, _ := protos.EncodeProtoMessage(&protos.SyncConversationsResponse{})
			w.Write(grpcWebData(payload))
		case urlPath(paths.BATCH_DELTA_SYNC_CONVERSATIONS):
			payload, _ := protos.EncodeProtoMessage(&protos.BatchDeltaSyncResponse{})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTestClient(t, handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	// Empty state => first sync (full listing path).
	_, err := c.Sync(context.Background(), State{}, 100)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(listingRequests) != 1 {
		t.Fatalf("listing requests = %d, want 1", len(listingRequests))
	}
	if len(listingRequests[0].GetSyncToken()) != 0 {
		t.Fatalf("listing request carried sync token %q, want empty", listingRequests[0].GetSyncToken())
	}
}

func TestChatFromEntryUsesCachedUsernameWhenDisplayNameMissing(t *testing.T) {
	selfID := "99999999-9999-9999-9999-999999999999"
	otherID := "11111111-1111-1111-1111-111111111111"
	c := &Client{
		selfUserID:        selfID,
		namesByUserID:     map[string]string{},
		usernamesByUserID: map[string]string{otherID: "snapuser"},
		avatarsByUserID:   map[string]string{},
	}
	entry := mustEntry(t, "55555555-5555-5555-5555-555555555555")
	entry.Participants = []*protos.UUID{mustUUID(t, selfID), mustUUID(t, otherID)}

	chat := c.chatFromEntry(entry)
	if chat.Name != "snapuser" {
		t.Fatalf("chat name = %q, want username fallback", chat.Name)
	}
	if chat.Username != "snapuser" {
		t.Fatalf("chat username = %q, want cached username", chat.Username)
	}
	if chat.OtherUserID != otherID {
		t.Fatalf("other user ID = %q, want %q", chat.OtherUserID, otherID)
	}
}
