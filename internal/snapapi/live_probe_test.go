package snapapi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0xzer/snapper/data/headers"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
)

func TestLiveSyncProbe(t *testing.T) {
	authURL := os.Getenv("SNAPAPI_LIVE_AUTH_URL")
	if authURL == "" {
		t.Skip("set SNAPAPI_LIVE_AUTH_URL to probe a live Snapchat web session")
	}
	secret := os.Getenv("SNAPAPI_LIVE_SECRET")
	if secret == "" {
		secret = "1225"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Bridge-Secret", secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var auth struct {
		CookieString        string `json:"cookieString"`
		SelfUserID          string `json:"selfUserID"`
		BrowserUserAgent    string `json:"browserUserAgent"`
		SnapClientUserAgent string `json:"snapClientUserAgent"`
		SecChUA             string `json:"secChUa"`
		SecChUAPlatform     string `json:"secChUaPlatform"`
		GRPCWebUserAgent    string `json:"grpcWebUserAgent"`
		MCSCOFIDsBin        string `json:"mcsCofIdsBin"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}
	if override := strings.TrimSpace(os.Getenv("SNAPAPI_LIVE_SELF_USER_ID")); override != "" {
		auth.SelfUserID = override
	}

	client, err := New(Config{
		CookieString:        auth.CookieString,
		SelfUserID:          auth.SelfUserID,
		UserAgent:           auth.BrowserUserAgent,
		SnapClientUserAgent: auth.SnapClientUserAgent,
		SecChUA:             auth.SecChUA,
		SecChUAPlatform:     auth.SecChUAPlatform,
		GRPCWebUserAgent:    auth.GRPCWebUserAgent,
		MCSCOFIDsBin:        auth.MCSCOFIDsBin,
		Timeout:             30 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = client.Authenticate(ctx); err != nil {
		t.Fatal(err)
	}

	payloadBytes, err := protos.EncodeProtoMessage(&protos.SyncConversationsRequest{
		SelfUserId: client.selfUUID(),
		SyncToken:  []byte("useV3"),
	})
	if err != nil {
		t.Fatal(err)
	}
	grpcBody := frameGRPC(payloadBytes)
	grpcReq, err := http.NewRequestWithContext(ctx, http.MethodPost, paths.SYNC_CONVERSATIONS, bytes.NewReader(grpcBody))
	if err != nil {
		t.Fatal(err)
	}
	grpcReq.Header = headers.NewCoreHeaders(client.cookies, client.tokens, client.device)
	client.applyUserAgentHeaders(grpcReq.Header)
	client.applyBrowserGRPCHeaders(grpcReq.Header)
	grpcReq.Header.Set("te", "trailers")

	grpcResp, err := client.http.Do(grpcReq)
	if err != nil {
		t.Fatal(err)
	}
	defer grpcResp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(grpcResp.Body, 1024*1024))
	if err != nil {
		t.Fatal(err)
	}

	snippetLen := min(len(body), 96)
	t.Logf("status=%s content-type=%q grpc-status=%q grpc-message=%q len=%d first_hex=%s first_text=%q self=%q",
		grpcResp.Status,
		grpcResp.Header.Get("content-type"),
		grpcResp.Header.Get("grpc-status"),
		grpcResp.Header.Get("grpc-message"),
		len(body),
		hex.EncodeToString(body[:snippetLen]),
		string(body[:snippetLen]),
		auth.SelfUserID,
	)
	if len(body) >= 5 {
		var decoded protos.SyncConversationsResponse
		err = protos.DecodeGRPCProtoMessage(body, &decoded)
		if err != nil {
			t.Logf("decode error: %v", err)
		} else {
			t.Logf("decoded conversations=%d incomplete=%v token_len=%d", len(decoded.GetConversations()), decoded.GetIncomplete(), len(decoded.GetSyncToken()))
		}
	}
}
