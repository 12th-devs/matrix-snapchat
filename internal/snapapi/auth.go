package snapapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/0xzer/snapper/data/headers"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/payload"
)

func (c *Client) Authenticate(ctx context.Context) error {
	if c.tokens.SSO_TOKEN == "" {
		token, err := c.fetchSSOToken(ctx)
		if err != nil {
			return err
		}
		c.tokens.SSO_TOKEN = token
	}
	c.mu.Lock()
	hasSelf := c.selfEncoded != nil && c.selfUserID != ""
	c.mu.Unlock()
	if hasSelf {
		return nil
	}
	if err := c.fetchSelf(ctx); err != nil {
		return fmt.Errorf("fetch Snapchat self profile: %w", err)
	}
	return nil
}

func (c *Client) SelfUserID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.selfUserID
}

func (c *Client) ensureAuthenticated(ctx context.Context) error {
	c.mu.Lock()
	hasToken := c.tokens.SSO_TOKEN != "" && c.selfEncoded != nil
	c.mu.Unlock()
	if hasToken {
		return nil
	}
	return c.Authenticate(ctx)
}

func (c *Client) fetchSSOToken(ctx context.Context) (string, error) {
	header := headers.NewDefaultHeaders(c.cookies, c.tokens, false)
	c.applyUserAgentHeaders(header)
	header.Del("Cookie")
	header.Set("Cookie", fmt.Sprintf("__Host-X-Snap-Client-Cookie=%s; %s=%s; __Host-sc-a-nonce=%s",
		c.cookies.HOST_X_SNAP_CLIENT_COOKIE,
		c.sessionCookieName,
		c.cookies.HOST_SC_A_SESSION,
		c.cookies.HOST_SC_A_NONCE,
	))
	header.Set("content-length", "0")
	body, err := c.doHTTP(ctx, paths.ACCOUNTS_SSO, http.MethodPost, header, nil)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(body))
	if token == "" || len(token) > 500 {
		return "", fmt.Errorf("invalid Snapchat SSO token response")
	}
	return token, nil
}

func (c *Client) fetchSelf(ctx context.Context) error {
	header := headers.NewDefaultHeaders(c.cookies, c.tokens, true)
	c.applyUserAgentHeaders(header)
	header.Set("content-type", "application/json")
	body, err := c.doHTTP(ctx, paths.WEB_GRAPHQL_URL, http.MethodPost, header, payload.USER_QUERY)
	if err != nil {
		return err
	}
	var resp struct {
		Data struct {
			User struct {
				ID          string `json:"id"`
				Username    string `json:"username"`
				DisplayName string `json:"displayName"`
			} `json:"user"`
		} `json:"data"`
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if resp.Data.User.ID == "" {
		return fmt.Errorf("Snapchat account info response did not include user id")
	}
	encoded, err := encodeUUIDString(resp.Data.User.ID)
	if err != nil {
		return err
	}
	name := resp.Data.User.DisplayName
	if name == "" {
		name = resp.Data.User.Username
	}
	c.mu.Lock()
	c.selfUserID = resp.Data.User.ID
	c.selfEncoded = encoded
	if name != "" {
		c.namesByUserID[resp.Data.User.ID] = name
	}
	c.mu.Unlock()
	return nil
}
