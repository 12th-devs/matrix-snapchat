package snapapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/0xzer/snapper/data/headers"
	"github.com/0xzer/snapper/protos"
	"google.golang.org/protobuf/proto"
)

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

func (c *Client) doHTTP(ctx context.Context, endpoint, method string, header http.Header, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header = header
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s returned %s: %s", method, endpoint, resp.Status, strings.TrimSpace(string(data[:min(len(data), 500)])))
	}
	return data, nil
}

func (c *Client) selfUUID() *protos.UUID {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.selfEncoded
}
