package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/colej/mautrix-snapchat/internal/config"
)

type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

type SessionStatus struct {
	State            string `json:"state"`
	Authenticated    bool   `json:"authenticated"`
	ActiveChatName   string `json:"activeChatName,omitempty"`
	URL              string `json:"url,omitempty"`
	VisibleChatCount int    `json:"visibleChatCount,omitempty"`
}

type Chat struct {
	ID          string `json:"id"`
	URL         string `json:"url,omitempty"`
	Name        string `json:"name"`
	Preview     string `json:"preview,omitempty"`
	Unread      bool   `json:"unread"`
	LastMessage string `json:"lastMessage,omitempty"`
}

type Message struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp,omitempty"`
	Outgoing  bool   `json:"outgoing"`
}

type SendMessageRequest struct {
	ChatID   string `json:"chatId,omitempty"`
	ChatName string `json:"chatName,omitempty"`
	ChatURL  string `json:"chatUrl,omitempty"`
	Text     string `json:"text"`
}

type SendMessageResponse struct {
	Queued    bool   `json:"queued"`
	Confirmed bool   `json:"confirmed"`
	ChatID    string `json:"chatID,omitempty"`
	ChatName  string `json:"chatName,omitempty"`
	ChatURL   string `json:"chatURL,omitempty"`
	Text      string `json:"text,omitempty"`
}

type Diagnostics struct {
	State         string            `json:"state"`
	Authenticated bool              `json:"authenticated"`
	URL           string            `json:"url,omitempty"`
	Title         string            `json:"title,omitempty"`
	ProfileDir    string            `json:"profileDir,omitempty"`
	Headless      bool              `json:"headless"`
	ChatNames     []string          `json:"chatNames,omitempty"`
	VisibleTexts  []string          `json:"visibleTexts,omitempty"`
	Snapshot      map[string]string `json:"snapshot,omitempty"`
	SelectorHints map[string]string `json:"selectorHints,omitempty"`
}

func New(cfg config.ConnectorConfig) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		secret:  cfg.SharedSecret,
		http: &http.Client{
			Timeout: time.Duration(cfg.RequestTimeoutSeconds) * time.Second,
		},
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) StartSession(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/session/start", nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("connector session start %s returned %s: %s", req.URL.String(), resp.Status, readResponseSnippet(resp.Body))
	}

	return nil
}

func (c *Client) Status(ctx context.Context) (*SessionStatus, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/session/status", nil)
	if err != nil {
		return nil, err
	}

	var status SessionStatus
	if err = c.doJSON(req, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

func (c *Client) ListChats(ctx context.Context) ([]Chat, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/chats", nil)
	if err != nil {
		return nil, err
	}

	var chats []Chat
	if err = c.doJSON(req, &chats); err != nil {
		return nil, err
	}

	return chats, nil
}

func (c *Client) Messages(ctx context.Context, chatID, chatName, chatURL string) ([]Message, error) {
	query := url.Values{}
	if chatID != "" {
		query.Set("chatId", chatID)
	}
	if chatName != "" {
		query.Set("chatName", chatName)
	}
	if chatURL != "" {
		query.Set("chatUrl", chatURL)
	}

	req, err := c.newRequest(ctx, http.MethodGet, "/messages?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var messages []Message
	if err = c.doJSON(req, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID, chatName, chatURL, text string) error {
	payload := SendMessageRequest{
		ChatID:   chatID,
		ChatName: chatName,
		ChatURL:  chatURL,
		Text:     text,
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/messages", payload)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("connector send %s returned %s: %s", req.URL.String(), resp.Status, readResponseSnippet(resp.Body))
	}

	var result SendMessageResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode connector send response: %w", err)
	}
	if !result.Confirmed {
		return fmt.Errorf("connector send did not confirm delivery to chat %q", result.ChatName)
	}

	return nil
}

func (c *Client) Diagnostics(ctx context.Context) (*Diagnostics, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/debug/diagnostics", nil)
	if err != nil {
		return nil, err
	}

	var diagnostics Diagnostics
	if err = c.doJSON(req, &diagnostics); err != nil {
		return nil, err
	}

	return &diagnostics, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.secret != "" {
		req.Header.Set("X-Bridge-Secret", c.secret)
	}

	return req, nil
}

func (c *Client) doJSON(req *http.Request, target any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", req.Method, req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s returned %s: %s", req.Method, req.URL.String(), resp.Status, readResponseSnippet(resp.Body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func readResponseSnippet(body io.Reader) string {
	const maxLen = 300
	data, err := io.ReadAll(io.LimitReader(body, maxLen))
	if err != nil {
		return "<failed to read response body>"
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "<empty response body>"
	}
	return text
}
