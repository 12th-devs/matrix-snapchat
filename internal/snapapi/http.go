package snapapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/0xzer/snapper/data/headers"
	"github.com/0xzer/snapper/protos"
	"google.golang.org/protobuf/proto"
)

// ErrUnauthorized marks an API response that Snapchat rejected with HTTP
// 401/403 or a PERMISSION_DENIED/UNAUTHENTICATED gRPC status, typically meaning
// the web messaging session is no longer valid server-side.
var ErrUnauthorized = errors.New("snapchat session unauthorized")

// messagingEndpointPrefix marks RPCs that belong to the messaging core service.
// Both SyncConversations and DeltaSync (plus BatchDeltaSync/QueryConversations)
// live under this single service, so auth failures on any of them surface through
// the same doHTTP/doGRPC path and can be normalized consistently.
const messagingEndpointPrefix = "messagingcoreservice.MessagingCoreService"

func isMessagingEndpoint(endpoint string) bool {
	return strings.Contains(endpoint, messagingEndpointPrefix)
}

// grpcAuthRejected reports whether a grpc-web trailer block carries an
// auth-rejection status (PERMISSION_DENIED or UNAUTHENTICATED).
func grpcAuthRejected(trailer string) bool {
	for _, line := range strings.Split(trailer, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "grpc-status:") {
			continue
		}
		code := strings.TrimSpace(strings.TrimPrefix(line, "grpc-status:"))
		// 7 = PERMISSION_DENIED, 16 = UNAUTHENTICATED.
		return code == "7" || code == "16"
	}
	return false
}

func (c *Client) doGRPC(ctx context.Context, endpoint string, req proto.Message, target proto.Message) error {
	payloadBytes, err := protos.EncodeProtoMessage(req)
	if err != nil {
		return err
	}
	framed := frameGRPC(payloadBytes)
	header := headers.NewCoreHeaders(c.cookies, c.tokens, c.device)
	c.applyUserAgentHeaders(header)
	c.applyBrowserGRPCHeaders(header)
	body, err := c.doHTTP(ctx, endpoint, http.MethodPost, header, framed)
	if err != nil {
		return err
	}
	if len(body) >= 5 && body[0]&0x80 != 0 {
		trailerLength := int(body[1])<<24 | int(body[2])<<16 | int(body[3])<<8 | int(body[4])
		if trailerLength <= len(body)-5 {
			trailer := strings.TrimSpace(string(body[5 : 5+trailerLength]))
			if isMessagingEndpoint(endpoint) && grpcAuthRejected(trailer) {
				return fmt.Errorf("%w: grpc from %s rejected: %s", ErrUnauthorized, endpoint, trailer)
			}
			return fmt.Errorf("grpc error from %s: %s", endpoint, trailer)
		}
	}
	if err = protos.DecodeGRPCProtoMessage(body, target); err != nil {
		return fmt.Errorf("decode grpc response from %s len=%d: %w", endpoint, len(body), err)
	}
	return nil
}

func (c *Client) applyUserAgentHeaders(header http.Header) {
	if c.userAgent != "" {
		setHeaderValue(header, "user-agent", c.userAgent)
		if c.secChUA != "" {
			setHeaderValue(header, "sec-ch-ua", c.secChUA)
		}
		if c.secChUAPlatform != "" {
			setHeaderValue(header, "sec-ch-ua-platform", c.secChUAPlatform)
		}
	}
	if c.snapClientUA != "" {
		setHeaderValue(header, "x-snap-client-user-agent", c.snapClientUA)
	}
	if c.grpcWebUA != "" {
		setHeaderValue(header, "x-user-agent", c.grpcWebUA)
	} else {
		setHeaderValue(header, "x-user-agent", "grpc-web-javascript/0.1")
	}
	if c.mcsCOFIDsBin != "" {
		setHeaderValue(header, "mcs-cof-ids-bin", c.mcsCOFIDsBin)
	}
}

func (c *Client) applyBrowserGRPCHeaders(header http.Header) {
	for _, key := range []string{
		"accept-language",
		"connection",
		"origin",
		"sec-ch-ua-mobile",
		"sec-fetch-dest",
		"sec-fetch-mode",
		"sec-fetch-site",
	} {
		deleteHeaderValue(header, key)
	}
	if c.secChUA == "" {
		deleteHeaderValue(header, "sec-ch-ua")
	}
	if c.secChUAPlatform == "" {
		deleteHeaderValue(header, "sec-ch-ua-platform")
	}
	setHeaderValue(header, "accept", "*/*")
	setHeaderValue(header, "content-type", "application/grpc-web+proto")
	setHeaderValue(header, "x-grpc-web", "1")
	if c.grpcWebUA == "" {
		setHeaderValue(header, "x-user-agent", "grpc-web-javascript/0.1")
	}
}

func setHeaderValue(header http.Header, key, value string) {
	deleteHeaderValue(header, key)
	header.Set(key, value)
}

func deleteHeaderValue(header http.Header, key string) {
	header.Del(key)
	delete(header, key)
	delete(header, strings.ToLower(key))
}

func chromeMajorFromUserAgent(userAgent string) string {
	for _, marker := range []string{"Chrome/", "Chromium/", "HeadlessChrome/"} {
		idx := strings.Index(userAgent, marker)
		if idx < 0 {
			continue
		}
		version := userAgent[idx+len(marker):]
		if dot := strings.IndexByte(version, '.'); dot > 0 {
			return version[:dot]
		}
		if version != "" {
			return version
		}
	}
	return ""
}

// HTTPCallMeta carries safe transport facts for diagnostics: no header values,
// cookies, tokens, or body contents.
type HTTPCallMeta struct {
	Status        int
	FinalURLPath  string
	Redirects     int
	ContentType   string
	ContentLength int64
	BodyLen       int
}

func (m HTTPCallMeta) String() string {
	return fmt.Sprintf("status=%d final_path=%s redirects=%d content_type=%q content_length=%d resp_bytes=%d",
		m.Status, m.FinalURLPath, m.Redirects, m.ContentType, m.ContentLength, m.BodyLen)
}

func (c *Client) doHTTPDetailed(ctx context.Context, endpoint, method string, header http.Header, body []byte) ([]byte, HTTPCallMeta, error) {
	var meta HTTPCallMeta
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, meta, err
	}
	req.Header = header
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, meta, err
	}
	defer resp.Body.Close()
	meta.Status = resp.StatusCode
	if resp.Request != nil {
		if resp.Request.URL != nil {
			meta.FinalURLPath = resp.Request.URL.Path
		}
		for prev := resp.Request.Response; prev != nil; prev = prev.Request.Response {
			meta.Redirects++
		}
	}
	meta.ContentType = resp.Header.Get("Content-Type")
	meta.ContentLength = resp.ContentLength
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, meta, err
	}
	meta.BodyLen = len(data)
	if resp.StatusCode >= 300 {
		msg := fmt.Sprintf("%s %s returned %s: %s", method, endpoint, resp.Status, strings.TrimSpace(string(data[:min(len(data), 500)])))
		if isMessagingEndpoint(endpoint) && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
			return nil, meta, fmt.Errorf("%w: %s", ErrUnauthorized, msg)
		}
		return nil, meta, fmt.Errorf("%s", msg)
	}
	return data, meta, nil
}

func (c *Client) doHTTP(ctx context.Context, endpoint, method string, header http.Header, body []byte) ([]byte, error) {
	data, _, err := c.doHTTPDetailed(ctx, endpoint, method, header, body)
	return data, err
}

func (c *Client) selfUUID() *protos.UUID {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.selfEncoded
}
