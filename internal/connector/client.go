package connector

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/colej/mautrix-snapchat/internal/config"
)

type Client struct {
	baseURL     string
	secret      string
	http        *http.Client
	browserHTTP *http.Client
}

type SessionStatus struct {
	State            string `json:"state"`
	Authenticated    bool   `json:"authenticated"`
	ReLoginRequired  bool   `json:"reLoginRequired,omitempty"`
	Message          string `json:"message,omitempty"`
	ActiveChatName   string `json:"activeChatName,omitempty"`
	URL              string `json:"url,omitempty"`
	VisibleChatCount int    `json:"visibleChatCount,omitempty"`
}

type Chat struct {
	ID                    string   `json:"id"`
	OtherUserID           string   `json:"otherUserId,omitempty"`
	ParticipantIDs        []string `json:"participantIds,omitempty"`
	IsGroup               bool     `json:"isGroup,omitempty"`
	URL                   string   `json:"url,omitempty"`
	Name                  string   `json:"name"`
	Username              string   `json:"username,omitempty"`
	Preview               string   `json:"preview,omitempty"`
	Unread                bool     `json:"unread"`
	LastMessage           string   `json:"lastMessage,omitempty"`
	DisappearAfterSeconds int64    `json:"disappearAfterSeconds,omitempty"`
	// ReadWatermarks maps participant Snapchat user ID -> last-read message ID
	// as reported by the conversation object (snapchat-side read receipts).
	ReadWatermarks map[string]int64 `json:"readWatermarks,omitempty"`
}

type Message struct {
	ID                    string            `json:"id"`
	AuthorID              string            `json:"authorId,omitempty"`
	Author                string            `json:"author"`
	Text                  string            `json:"text"`
	Timestamp             string            `json:"timestamp,omitempty"`
	Outgoing              bool              `json:"outgoing"`
	IsSnap                bool              `json:"isSnap,omitempty"`
	ContentType           string            `json:"contentType,omitempty"`
	Media                 []MediaAttachment `json:"media,omitempty"`
	Saved                 bool              `json:"saved,omitempty"`
	DisappearAfterSeconds int64             `json:"disappearAfterSeconds,omitempty"`
	// QuotedMessageID is the RAW numeric Snapchat message ID this message
	// replies to (empty when not a reply). Scoping to the chat happens where
	// the chat ID is known.
	QuotedMessageID string `json:"quotedMessageId,omitempty"`
	// Tombstone marks a message that was erased/unsent on Snapchat.
	Tombstone bool `json:"tombstone,omitempty"`
}

type MediaAttachment struct {
	ID       string `json:"id,omitempty"`
	FileName string `json:"fileName,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     []byte `json:"-"`
}

type SendMessageRequest struct {
	ChatID   string `json:"chatId,omitempty"`
	ChatName string `json:"chatName,omitempty"`
	ChatURL  string `json:"chatUrl,omitempty"`
	Text     string `json:"text"`
}

type SendMediaRequest struct {
	ChatID   string `json:"chatId,omitempty"`
	ChatName string `json:"chatName,omitempty"`
	ChatURL  string `json:"chatUrl,omitempty"`
	FileName string `json:"fileName,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data"`
	Caption  string `json:"caption,omitempty"`
}

type TypingRequest struct {
	ChatID     string `json:"chatId"`
	Typing     bool   `json:"typing"`
	DurationMs int    `json:"durationMs,omitempty"`
}

type SendMessageResponse struct {
	Queued    bool   `json:"queued"`
	Confirmed bool   `json:"confirmed"`
	ChatID    string `json:"chatID,omitempty"`
	ChatName  string `json:"chatName,omitempty"`
	ChatURL   string `json:"chatURL,omitempty"`
	Text      string `json:"text,omitempty"`
}

type TypingResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
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

type APIAuth struct {
	State               string `json:"state"`
	Authenticated       bool   `json:"authenticated"`
	CookieString        string `json:"cookieString"`
	SSOToken            string `json:"ssoToken,omitempty"`
	SelfUserID          string `json:"selfUserID,omitempty"`
	Username            string `json:"username,omitempty"`
	DisplayName         string `json:"displayName,omitempty"`
	IdentitySource      string `json:"identitySource,omitempty"`
	BrowserUserAgent    string `json:"browserUserAgent,omitempty"`
	SnapClientUserAgent string `json:"snapClientUserAgent,omitempty"`
	SecChUA             string `json:"secChUa,omitempty"`
	SecChUAPlatform     string `json:"secChUaPlatform,omitempty"`
	GRPCWebUserAgent    string `json:"grpcWebUserAgent,omitempty"`
	MCSCOFIDsBin        string `json:"mcsCofIdsBin,omitempty"`
	WebVersion          string `json:"webVersion,omitempty"`
	URL                 string `json:"url,omitempty"`
}

type EELDecryptRequest struct {
	ConversationID        string `json:"conversationId,omitempty"`
	MessageID             string `json:"messageId,omitempty"`
	ContentBase64         string `json:"contentBase64"`
	CEKBase64             string `json:"cekBase64,omitempty"`
	CEKIVBase64           string `json:"cekIvBase64,omitempty"`
	NonceBase64           string `json:"nonceBase64,omitempty"`
	SenderPublicKeyBase64 string `json:"senderPublicKeyBase64,omitempty"`
	SenderVersion         int32  `json:"senderVersion,omitempty"`
}

type EELDecryptResponse struct {
	OK                     bool   `json:"ok"`
	DecryptedContentBase64 string `json:"decryptedContentBase64,omitempty"`
	Error                  string `json:"error,omitempty"`
	Retryable              bool   `json:"retryable,omitempty"`
}

func New(cfg config.ConnectorConfig) *Client {
	requestTimeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	browserTimeout := maxDuration(requestTimeout, 180*time.Second)
	return &Client{
		baseURL:     cfg.BaseURL,
		secret:      cfg.SharedSecret,
		http:        &http.Client{Timeout: requestTimeout},
		browserHTTP: &http.Client{Timeout: browserTimeout},
	}
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) StartSession(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/session/start", nil)
	if err != nil {
		return err
	}

	resp, err := c.browserHTTP.Do(req)
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

func (c *Client) APIAuth(ctx context.Context) (*APIAuth, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/session/api-auth", nil)
	if err != nil {
		return nil, err
	}

	var auth APIAuth
	if err = c.doJSONWithClient(c.browserHTTP, req, &auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

func (c *Client) DecryptEEL(ctx context.Context, payload EELDecryptRequest) ([]byte, error) {
	if strings.TrimSpace(payload.ContentBase64) == "" {
		return nil, fmt.Errorf("missing EEL content")
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/session/eel-decrypt", payload)
	if err != nil {
		return nil, err
	}

	var response EELDecryptResponse
	if err = c.doJSON(req, &response); err != nil {
		return nil, err
	}
	if !response.OK {
		if response.Error == "" {
			response.Error = "EEL decrypt failed"
		}
		return nil, errors.New(response.Error)
	}
	if response.DecryptedContentBase64 == "" {
		return nil, fmt.Errorf("connector returned empty EEL plaintext")
	}
	decrypted, err := base64.StdEncoding.DecodeString(response.DecryptedContentBase64)
	if err != nil {
		return nil, fmt.Errorf("decode connector EEL plaintext: %w", err)
	}
	return decrypted, nil
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

func (c *Client) SendMedia(ctx context.Context, chatID, chatName, chatURL string, media MediaAttachment, caption string) error {
	if len(media.Data) == 0 {
		return fmt.Errorf("missing media data")
	}
	payload := SendMediaRequest{
		ChatID:   chatID,
		ChatName: chatName,
		ChatURL:  chatURL,
		FileName: media.FileName,
		MimeType: media.MimeType,
		Data:     base64.StdEncoding.EncodeToString(media.Data),
		Caption:  caption,
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/media", payload)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("connector media send %s returned %s: %s", req.URL.String(), resp.Status, readResponseSnippet(resp.Body))
	}
	var result SendMessageResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode connector media send response: %w", err)
	}
	if !result.Confirmed {
		return fmt.Errorf("connector media send did not confirm delivery to chat %q", result.ChatName)
	}
	return nil
}

func (c *Client) SetTyping(ctx context.Context, chatID string, typing bool, durationMs int) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("missing chat ID")
	}
	payload := TypingRequest{
		ChatID:     chatID,
		Typing:     typing,
		DurationMs: durationMs,
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/typing", payload)
	if err != nil {
		return err
	}
	var result TypingResponse
	if err = c.doJSON(req, &result); err != nil {
		return err
	}
	if !result.OK {
		if result.Error == "" {
			result.Error = "typing update was not accepted"
		}
		return errors.New(result.Error)
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
	return c.doJSONWithClient(c.http, req, target)
}

func (c *Client) doJSONWithClient(client *http.Client, req *http.Request, target any) error {
	resp, err := client.Do(req)
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
