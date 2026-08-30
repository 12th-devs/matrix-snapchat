package snapapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
	"github.com/0xzer/snapper/types"
)

// urlPath extracts the URL path portion from a full endpoint constant, matching
// the httptest server's r.URL.Path for exact-path router comparisons.
func urlPath(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		panic(err)
	}
	return u.Path
}

// atSrv points an endpoint at the test server while preserving the exact
// production path (e.g. /messagingcoreservice.MessagingCoreService/SyncConversations),
// so the request exercises the real doHTTP/doGRPC pipeline against the httptest
// handler instead of the real web.snapchat.com host.
func atSrv(srv *httptest.Server, endpoint string) string {
	return srv.URL + urlPath(endpoint)
}

// newTestClient wires a Client to an httptest server so tests exercise the same
// doHTTP/doGRPC path SyncConversations and DeltaSync use in production.
func newTestClient(handler http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	c := &Client{
		http:              srv.Client(),
		cookies:           &types.SnapCookies{},
		tokens:            &types.SnapTokens{},
		device:            types.NewDevice(),
		sessionCookieName: "sc-a-nonce",
		conversations:     make(map[string]*protos.Conversation),
		conversationIDs:   make(map[string]*protos.UUID),
	}
	return c, srv
}

// grpcWebTrailer builds a grpc-web trailer block body in the format doGRPC parses:
// 0x80 flag byte + 4-byte big-endian trailer length + trailer text.
func grpcWebTrailer(status string) []byte {
	trailer := "grpc-status: " + status + "\r\ngrpc-message: test\r\n"
	n := len(trailer)
	body := []byte{0x80, byte(n >> 24 & 0xff), byte(n >> 16 & 0xff), byte(n >> 8 & 0xff), byte(n & 0xff)}
	body = append(body, trailer...)
	return body
}

// http403Router rejects every endpoint with HTTP 403 while recording which path
// was served, so tests can assert the intended branch (messaging vs non-messaging)
// was actually exercised rather than silently falling through to a catch-all.
func http403Router() (http.Handler, *sync.Map) {
	seen := &sync.Map{}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.Store(r.URL.Path, struct{}{})
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"permission denied"}`))
	})
	return h, seen
}

// grpcTrailerRouter answers messaging endpoints with HTTP 200 + a grpc-web
// trailer (the way Snapchat's grpc-web gateway surfaces per-RPC status), so the
// doGRPC trailer-parsing path is exercised on the exact production endpoints.
// Each branch returns a distinct marker and the seen map records which paths were
// served, so a test cannot pass against an unexercised catch-all branch.
func grpcTrailerRouter() (http.Handler, *sync.Map) {
	seen := &sync.Map{}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.Store(r.URL.Path, struct{}{})
		switch r.URL.Path {
		case urlPath(paths.SYNC_CONVERSATIONS):
			// PERMISSION_DENIED -> dead messaging session.
			w.Write(grpcWebTrailer("7"))
		case urlPath(paths.DELTA_SYNC_CONVERSATIONS):
			w.Write(grpcWebTrailer("7"))
		case urlPath(paths.BATCH_DELTA_SYNC_CONVERSATIONS):
			w.Write(grpcWebTrailer("7"))
		case urlPath(paths.QUERY_CONVERSATIONS):
			// UNAUTHENTICATED -> also an auth rejection.
			w.Write(grpcWebTrailer("16"))
		default:
			// Any unexpected path: 403 with a body that is NOT a valid grpc-web
			// message. A 403 here cannot map to ErrUnauthorized, so a test that
			// accidentally hits this branch and expects ErrUnauthorized fails.
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"permission denied"}`))
		}
	})
	return h, seen
}

func saw(seen *sync.Map, path string) bool {
	_, ok := seen.Load(path)
	return ok
}

func TestDoHTTPMessagingForbiddenMapsToUnauthorized(t *testing.T) {
	handler, seen := http403Router()
	c, srv := newTestClient(handler)
	defer srv.Close()

	_, err := c.doHTTP(context.Background(), atSrv(srv, paths.SYNC_CONVERSATIONS), http.MethodPost, http.Header{}, nil)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for HTTP 403 on sync endpoint, got: %v", err)
	}
	if !strings.Contains(err.Error(), "SyncConversations") {
		t.Fatalf("expected endpoint context in error, got: %v", err)
	}
	if !saw(seen, urlPath(paths.SYNC_CONVERSATIONS)) {
		t.Fatal("messaging sync endpoint was never exercised")
	}
}

func TestDoHTTPUnrelatedForbiddenDoesNotMapToUnauthorized(t *testing.T) {
	handler, seen := http403Router()
	c, srv := newTestClient(handler)
	defer srv.Close()

	// GET_USER_PUBLIC_INFO is a profile service endpoint, not the messaging core.
	_, err := c.doHTTP(context.Background(), atSrv(srv, paths.GET_USER_PUBLIC_INFO), http.MethodPost, http.Header{}, nil)
	if errors.Is(err, ErrUnauthorized) {
		t.Fatalf("unrelated 403 must NOT map to ErrUnauthorized, got: %v", err)
	}
	if err == nil {
		t.Fatal("expected a plain error for unrelated 403")
	}
	if !saw(seen, urlPath(paths.GET_USER_PUBLIC_INFO)) {
		t.Fatal("unrelated non-messaging endpoint was never exercised")
	}
}

func TestDoGRPCSyncConversationsPermissionDeniedMapsToUnauthorized(t *testing.T) {
	handler, seen := grpcTrailerRouter()
	c, srv := newTestClient(handler)
	defer srv.Close()

	// Real production endpoint + doGRPC path (sync.go:36).
	var resp protos.SyncConversationsResponse
	err := c.doGRPC(context.Background(), atSrv(srv, paths.SYNC_CONVERSATIONS), &protos.SyncConversationsRequest{}, &resp)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("grpc PERMISSION_DENIED on SyncConversations must map to ErrUnauthorized, got: %v", err)
	}
	if !strings.Contains(err.Error(), "SyncConversations") {
		t.Fatalf("expected endpoint context in error, got: %v", err)
	}
	if !saw(seen, urlPath(paths.SYNC_CONVERSATIONS)) {
		t.Fatal("SyncConversations branch was never exercised")
	}
}

func TestDoGRPCDeltaSyncPermissionDeniedMapsToUnauthorized(t *testing.T) {
	handler, seen := grpcTrailerRouter()
	c, srv := newTestClient(handler)
	defer srv.Close()

	// DeltaSync follows the same doGRPC path (sync.go:123), same service.
	var resp protos.DeltaSyncResponse
	err := c.doGRPC(context.Background(), atSrv(srv, paths.DELTA_SYNC_CONVERSATIONS), &protos.DeltaSyncRequest{}, &resp)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("grpc PERMISSION_DENIED on DeltaSync must map to ErrUnauthorized, got: %v", err)
	}
	if !saw(seen, urlPath(paths.DELTA_SYNC_CONVERSATIONS)) {
		t.Fatal("DeltaSync branch was never exercised")
	}
}

func TestDoGRPCBatchDeltaSyncPermissionDeniedMapsToUnauthorized(t *testing.T) {
	handler, seen := grpcTrailerRouter()
	c, srv := newTestClient(handler)
	defer srv.Close()

	var resp protos.BatchDeltaSyncResponse
	err := c.doGRPC(context.Background(), atSrv(srv, paths.BATCH_DELTA_SYNC_CONVERSATIONS), &protos.BatchDeltaSyncRequest{}, &resp)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("grpc PERMISSION_DENIED on BatchDeltaSync must map to ErrUnauthorized, got: %v", err)
	}
	if !saw(seen, urlPath(paths.BATCH_DELTA_SYNC_CONVERSATIONS)) {
		t.Fatal("BatchDeltaSync branch was never exercised")
	}
}

func TestDoGRPCQueryConversationsUnauthenticatedMapsToUnauthorized(t *testing.T) {
	handler, seen := grpcTrailerRouter()
	c, srv := newTestClient(handler)
	defer srv.Close()

	var resp protos.QueryConversationsResponse
	err := c.doGRPC(context.Background(), atSrv(srv, paths.QUERY_CONVERSATIONS), &protos.QueryConversationsRequest{}, &resp)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("grpc UNAUTHENTICATED on QueryConversations must map to ErrUnauthorized, got: %v", err)
	}
	if !saw(seen, urlPath(paths.QUERY_CONVERSATIONS)) {
		t.Fatal("QueryConversations branch was never exercised")
	}
}

func TestDoGRPCUnrelatedServicePermissionDeniedDoesNotMapToUnauthorized(t *testing.T) {
	handler, seen := grpcTrailerRouter()
	c, srv := newTestClient(handler)
	defer srv.Close()

	// A 403 from some unrelated service must NOT become ErrUnauthorized.
	_, err := c.doHTTP(context.Background(), atSrv(srv, paths.GET_USER_PUBLIC_INFO), http.MethodPost, http.Header{}, nil)
	if errors.Is(err, ErrUnauthorized) {
		t.Fatalf("unrelated service 403 must not map to ErrUnauthorized, got: %v", err)
	}
	if !saw(seen, urlPath(paths.GET_USER_PUBLIC_INFO)) {
		t.Fatal("unrelated non-messaging endpoint was never exercised")
	}
}

func TestDoGRPCMessagingInternalErrorDoesNotMapToUnauthorized(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HTTP 200 but grpc-status 13: an internal messaging error, not an auth
		// rejection. Must NOT be treated as dead-session.
		w.Write(grpcWebTrailer("13"))
	}))
	defer srv.Close()

	var resp protos.SyncConversationsResponse
	err := c.doGRPC(context.Background(), atSrv(srv, paths.SYNC_CONVERSATIONS), &protos.SyncConversationsRequest{}, &resp)
	if errors.Is(err, ErrUnauthorized) {
		t.Fatalf("grpc-status 13 (INTERNAL) must NOT map to ErrUnauthorized, got: %v", err)
	}
	if err == nil {
		t.Fatal("expected a plain grpc error for internal status")
	}
	if !strings.Contains(err.Error(), "SyncConversations") {
		t.Fatalf("expected endpoint context in error, got: %v", err)
	}
}

func TestDoGRPCNonMessagingTrailerDoesNotMapToUnauthorized(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(grpcWebTrailer("7"))
	}))
	defer srv.Close()

	// Even a PERMISSION_DENIED trailer on a NON-messaging endpoint must not flip
	// into ErrUnauthorized (the scoping guard rejects it).
	var resp protos.Empty
	err := c.doGRPC(context.Background(), atSrv(srv, "https://example.invalid/unrelated.Service/SomeMethod"), &protos.Empty{}, &resp)
	if errors.Is(err, ErrUnauthorized) {
		t.Fatalf("grpc PERMISSION_DENIED on unrelated service must NOT map to ErrUnauthorized, got: %v", err)
	}
	if err == nil {
		t.Fatal("expected a plain grpc error for unrelated trailer")
	}
}

// overrideIdentityURL points the fetchSelf identity-query seam at the test
// server and restores the production constant when the test finishes.
func overrideIdentityURL(t *testing.T, srv *httptest.Server) {
	t.Helper()
	orig := webGraphQLURL
	webGraphQLURL = atSrv(srv, paths.WEB_GRAPHQL_URL)
	t.Cleanup(func() { webGraphQLURL = orig })
}

func TestFetchSelfHTMLBodyMapsToUnauthorized(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Dead web session: Snapchat answers the identity query with an HTML page.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<!DOCTYPE html><html><body>Log In • Snapchat</body></html>"))
	}))
	defer srv.Close()
	overrideIdentityURL(t, srv)

	err := c.fetchSelf(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("HTML response to identity query must map to ErrUnauthorized, got: %v", err)
	}
}

func TestFetchSelfBrokenJSONStaysNeutral(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Invalid JSON that is NOT an HTML page: transient/garbage, not an auth signal.
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid-json"))
	}))
	defer srv.Close()
	overrideIdentityURL(t, srv)

	err := c.fetchSelf(context.Background())
	if errors.Is(err, ErrUnauthorized) {
		t.Fatalf("non-HTML decode failure must NOT map to ErrUnauthorized, got: %v", err)
	}
	if err == nil {
		t.Fatal("expected a plain decode error")
	}
}

func TestFetchSelfEmptyUserStaysNeutral(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Valid JSON without a user id: unexpected but not an auth rejection.
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"user":null}}`))
	}))
	defer srv.Close()
	overrideIdentityURL(t, srv)

	err := c.fetchSelf(context.Background())
	if errors.Is(err, ErrUnauthorized) {
		t.Fatalf("missing user id must NOT map to ErrUnauthorized, got: %v", err)
	}
	if err == nil {
		t.Fatal("expected the missing-user-id error")
	}
}

func TestGrpcAuthRejected(t *testing.T) {
	cases := []struct {
		trailer string
		want    bool
	}{
		{"grpc-status: 7\ngrpc-message: Permission denied", true},
		{"grpc-status: 16\ngrpc-message: Credentials expired", true},
		{"grpc-status: 6\ngrpc-message: already exists", false},
		{"grpc-status: 13\ngrpc-message: internal", false},
		{"grpc-status: 5\ngrpc-message: not found", false},
		{"no status line here", false},
		{"grpc-message: 7", false},
	}
	for _, tc := range cases {
		if got := grpcAuthRejected(tc.trailer); got != tc.want {
			t.Errorf("grpcAuthRejected(%q) = %v, want %v", tc.trailer, got, tc.want)
		}
	}
}

func TestIsMessagingEndpoint(t *testing.T) {
	if !isMessagingEndpoint(paths.SYNC_CONVERSATIONS) {
		t.Error("SyncConversations should be a messaging endpoint")
	}
	if !isMessagingEndpoint(paths.DELTA_SYNC_CONVERSATIONS) {
		t.Error("DeltaSync should be a messaging endpoint")
	}
	if !isMessagingEndpoint(paths.BATCH_DELTA_SYNC_CONVERSATIONS) {
		t.Error("BatchDeltaSync should be a messaging endpoint")
	}
	if !isMessagingEndpoint(paths.QUERY_CONVERSATIONS) {
		t.Error("QueryConversations should be a messaging endpoint")
	}
	if isMessagingEndpoint(paths.GET_USER_PUBLIC_INFO) {
		t.Error("public info should NOT be a messaging endpoint")
	}
	if isMessagingEndpoint("https://example.invalid/unrelated.Service/RPC") {
		t.Error("unrelated service should NOT be a messaging endpoint")
	}
}
