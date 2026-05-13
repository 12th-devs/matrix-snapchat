package snapapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	snapcrypto "github.com/0xzer/snapper/crypto"
	"github.com/0xzer/snapper/data/headers"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/payload"
	"github.com/0xzer/snapper/protos"
	"github.com/0xzer/snapper/types"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type Config struct {
	CookieString        string
	SelfUserID          string
	UserAgent           string
	SnapClientUserAgent string
	SecChUA             string
	SecChUAPlatform     string
	GRPCWebUserAgent    string
	MCSCOFIDsBin        string
	Timeout             time.Duration
	FetchLimit          int
}

type State struct {
	SyncToken            []byte
	ConversationVersions map[string]int64
	SelfUserID           string
}

type Chat struct {
	ID             string
	OtherUserID    string
	Name           string
	Preview        string
	LastMessage    string
	Unread         bool
	Version        int64
	LastActivityAt time.Time
}

type MediaKind string

const (
	MediaKindFile  MediaKind = "file"
	MediaKindImage MediaKind = "image"
	MediaKindVideo MediaKind = "video"
	MediaKindGIF   MediaKind = "gif"
)

type MediaAttachment struct {
	ID       string
	URL      string
	FileName string
	MimeType string
	Kind     MediaKind
	Data     []byte
	Key      []byte
	IV       []byte
}

type Message struct {
	ID        string
	AuthorID  string
	Author    string
	Text      string
	Timestamp time.Time
	Outgoing  bool
	IsSnap    bool
	Media     []MediaAttachment
	Version   int64
}

type SyncResult struct {
	Chats          []Chat
	ChangedChatIDs map[string]struct{}
	State          State
}

type Client struct {
	cookies           *types.SnapCookies
	tokens            *types.SnapTokens
	http              *http.Client
	device            string
	sessionCookieName string
	userAgent         string
	snapClientUA      string
	secChUA           string
	secChUAPlatform   string
	grpcWebUA         string
	mcsCOFIDsBin      string

	mu              sync.Mutex
	selfUserID      string
	selfEncoded     *protos.UUID
	conversations   map[string]*protos.Conversation
	conversationIDs map[string]*protos.UUID
	namesByUserID   map[string]string
}

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.CookieString) == "" {
		return nil, fmt.Errorf("missing Snapchat cookie string")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	cookies := types.NewCookiesFromString(cfg.CookieString)
	if cookies.HOST_SC_A_NONCE == "" {
		cookies.HOST_SC_A_NONCE = extractCookieValue(cfg.CookieString, "sc-a-nonce")
	}
	sessionCookieName := "__Host-sc-a-session"
	if cookies.HOST_SC_A_SESSION == "" {
		if value := extractCookieValue(cfg.CookieString, "__Host-sc-a-auth-session"); value != "" {
			cookies.HOST_SC_A_SESSION = value
			sessionCookieName = "__Host-sc-a-auth-session"
		}
	}
	if cookies.HOST_SC_A_SESSION == "" || cookies.HOST_X_SNAP_CLIENT_COOKIE == "" || cookies.HOST_SC_A_NONCE == "" {
		return nil, fmt.Errorf("missing required Snapchat web cookies")
	}
	client := &Client{
		cookies:           cookies,
		tokens:            &types.SnapTokens{},
		http:              &http.Client{Timeout: cfg.Timeout},
		device:            types.NewDevice(),
		sessionCookieName: sessionCookieName,
		userAgent:         strings.TrimSpace(cfg.UserAgent),
		snapClientUA:      strings.TrimSpace(cfg.SnapClientUserAgent),
		secChUA:           strings.TrimSpace(cfg.SecChUA),
		secChUAPlatform:   strings.TrimSpace(cfg.SecChUAPlatform),
		grpcWebUA:         strings.TrimSpace(cfg.GRPCWebUserAgent),
		mcsCOFIDsBin:      strings.TrimSpace(cfg.MCSCOFIDsBin),
		conversations:     make(map[string]*protos.Conversation),
		conversationIDs:   make(map[string]*protos.UUID),
		namesByUserID:     make(map[string]string),
	}
	if selfUserID := strings.TrimSpace(cfg.SelfUserID); selfUserID != "" {
		encoded, err := encodeUUIDString(selfUserID)
		if err != nil {
			return nil, fmt.Errorf("invalid Snapchat self user id: %w", err)
		}
		client.selfUserID = strings.ToLower(selfUserID)
		client.selfEncoded = encoded
	}
	return client, nil
}

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

func (c *Client) Sync(ctx context.Context, state State, fetchLimit int) (*SyncResult, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}
	if fetchLimit <= 0 {
		fetchLimit = 40
	}
	if state.ConversationVersions == nil {
		state.ConversationVersions = make(map[string]int64)
	}

	versionInfos := conversationVersionInfos(state.ConversationVersions)
	syncToken := state.SyncToken
	if len(syncToken) == 0 {
		syncToken = []byte("useV3")
	}

	req := &protos.SyncConversationsRequest{
		SelfUserId:   c.selfUUID(),
		SyncToken:    syncToken,
		VersionInfos: versionInfos,
	}
	var resp protos.SyncConversationsResponse
	if err := c.doGRPC(ctx, paths.SYNC_CONVERSATIONS, req, &resp); err != nil {
		return nil, err
	}

	entries := append([]*protos.ConversationEntry{}, resp.GetConversations()...)
	nextToken := resp.GetSyncToken()
	if len(nextToken) == 0 {
		nextToken = syncToken
	}
	if len(state.SyncToken) == 0 || resp.GetIncomplete() {
		paged, pagedToken, err := c.queryAllConversations(ctx, nextToken, fetchLimit)
		if err == nil && len(paged) > 0 {
			entries = mergeEntries(entries, paged)
			if len(pagedToken) > 0 {
				nextToken = pagedToken
			}
		}
	}

	newVersions := cloneVersions(state.ConversationVersions)
	changed := make(map[string]struct{})
	needsConversation := make([]*protos.ConversationEntry, 0, len(entries))
	for _, entry := range entries {
		id := uuidToString(entry.GetVersionInfo().GetConversationId())
		if id == "" || entry.GetIsDeleted() || entry.GetHideFromSendtoAndSearch() {
			continue
		}
		version := entry.GetVersionInfo().GetConversationVersion()
		if prev, ok := newVersions[id]; !ok || prev != version || entry.GetNeedsSync() {
			changed[id] = struct{}{}
			needsConversation = append(needsConversation, entry)
		}
		newVersions[id] = version
	}
	if len(needsConversation) > 0 {
		if err := c.hydrateConversations(ctx, needsConversation); err != nil {
			return nil, err
		}
	}
	if err := c.resolveEntryNames(ctx, entries); err != nil {
		// Names are important but not important enough to block sync. The bridge
		// can still create stable portals using conversation IDs.
	}

	chats := make([]Chat, 0, len(entries))
	for _, entry := range entries {
		id := uuidToString(entry.GetVersionInfo().GetConversationId())
		if id == "" || entry.GetIsDeleted() || entry.GetHideFromSendtoAndSearch() {
			continue
		}
		chat := c.chatFromEntry(entry)
		if chat.ID == "" {
			continue
		}
		chats = append(chats, chat)
	}
	sort.SliceStable(chats, func(i, j int) bool {
		return chats[i].LastActivityAt.After(chats[j].LastActivityAt)
	})

	return &SyncResult{
		Chats:          chats,
		ChangedChatIDs: changed,
		State: State{
			SyncToken:            nextToken,
			ConversationVersions: newVersions,
			SelfUserID:           c.SelfUserID(),
		},
	}, nil
}

func (c *Client) QueryMessages(ctx context.Context, chatID string, version int64, limit int) ([]Message, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	convID, err := encodeUUIDString(chatID)
	if err != nil {
		return nil, err
	}
	messages, err := c.queryMessages(ctx, convID, version, limit)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 && version != 0 {
		messages, err = c.queryMessages(ctx, convID, 0, limit)
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
	return messages, nil
}

func (c *Client) SendText(ctx context.Context, chatID, text string) (string, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return "", err
	}
	conv, err := c.conversation(ctx, chatID)
	if err != nil {
		return "", err
	}
	content := &protos.Contents{
		Content: &protos.Contents_Text{
			Text: &protos.Text{Text: text},
		},
	}
	contentBytes, err := protos.EncodeProtoMessage(content)
	if err != nil {
		return "", err
	}
	attemptID, _ := snapcrypto.EncodeUUID(uuid.NewString())
	req := &protos.CreateContentMessageRequest{
		SenderId:           c.selfUUID(),
		ClientResolutionId: randomUint64(),
		Destinations: []*protos.DeliveryDestination{{
			EncryptionInfo: &protos.EncryptionInfo{
				Method: &protos.EncryptionInfo_Fidelius{Fidelius: &protos.Empty{}},
			},
			Destination: &protos.DeliveryDestination_ConversationDestination{
				ConversationDestination: &protos.ConversationDestination{
					ConversationId: conv.GetConversationId(),
					CurrentVersion: conv.GetVersion(),
				},
			},
		}},
		Content: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    contentBytes,
			SavePolicy:  protos.ContentEnvelope_SavePolicy_LIFETIME,
			EnvelopeEncryption: &protos.EnvelopeEncryption{
				Method: &protos.EnvelopeEncryption_None{None: &protos.Empty{}},
			},
		},
		CreateContentMessageBlizzardData: &protos.CreateContentMessageBlizzardData{
			SendMessageAttemptId: &protos.UUID{EncodedId: attemptID},
		},
	}
	var resp protos.CreateContentMessageResponse
	if err = c.doGRPC(ctx, paths.CREATE_CONTENT_MESSAGE, req, &resp); err != nil {
		return "", err
	}
	for _, result := range resp.GetResult() {
		if !result.GetSuccess() {
			return "", fmt.Errorf("snapchat api send failed")
		}
	}
	return fmt.Sprintf("%d", resp.GetClientResolutionId()), nil
}

func (c *Client) MarkRead(ctx context.Context, chatID string, messageID int64, version int64) error {
	if messageID <= 0 {
		return nil
	}
	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}
	conv, err := c.conversation(ctx, chatID)
	if err != nil {
		return err
	}
	if version <= 0 {
		version = conv.GetVersion()
	}
	req := &protos.UpdateContentMessageRequest{
		ClientResolutionId: randomUint64(),
		CurrentVersion:     version,
		Update: &protos.UpdateAction{
			MessageId:       messageID,
			SenderId:        c.selfUUID(),
			ConversationId:  conv.GetConversationId(),
			UpdateTimestamp: time.Now().UnixMilli(),
			Update:          &protos.UpdateAction_Read{Read: &protos.Read{}},
		},
	}
	var resp protos.UpdateContentMessageResponse
	if err = c.doGRPC(ctx, paths.UPDATE_CONTENT_MESSAGE, req, &resp); err != nil {
		return err
	}
	if !resp.GetSuccess() && !resp.GetRetryable() {
		return fmt.Errorf("snapchat api read receipt failed")
	}
	return nil
}

func (c *Client) DownloadMedia(ctx context.Context, media MediaAttachment) ([]byte, string, error) {
	var data []byte
	contentType := strings.TrimSpace(media.MimeType)
	if len(media.Data) > 0 {
		data = append([]byte(nil), media.Data...)
	} else if strings.TrimSpace(media.URL) != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, media.URL, nil)
		if err != nil {
			return nil, "", err
		}
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, "", fmt.Errorf("download Snapchat media returned %s", resp.Status)
		}
		if contentType == "" {
			contentType = strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, 64*1024*1024))
		if err != nil {
			return nil, "", err
		}
	} else {
		return nil, "", fmt.Errorf("Snapchat media has no URL or inline data")
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("Snapchat media was empty")
	}
	if decrypted, ok := decryptMediaCBC(data, media.Key, media.IV); ok {
		data = decrypted
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectMediaMime(data, media.Kind)
	}
	return data, contentType, nil
}

func (c *Client) conversation(ctx context.Context, chatID string) (*protos.Conversation, error) {
	c.mu.Lock()
	if conv := c.conversations[chatID]; conv != nil {
		c.mu.Unlock()
		return conv, nil
	}
	c.mu.Unlock()
	encoded, err := encodeUUIDString(chatID)
	if err != nil {
		return nil, err
	}
	req := &protos.DeltaSyncRequest{
		SelfUserId:     c.selfUUID(),
		ConversationId: encoded,
	}
	var resp protos.DeltaSyncResponse
	if err = c.doGRPC(ctx, paths.DELTA_SYNC_CONVERSATIONS, req, &resp); err != nil {
		return nil, err
	}
	conv := resp.GetConversation()
	if conv == nil {
		return nil, fmt.Errorf("conversation %s not found", chatID)
	}
	c.mu.Lock()
	c.conversations[chatID] = conv
	c.conversationIDs[chatID] = conv.GetConversationId()
	c.mu.Unlock()
	return conv, nil
}

func (c *Client) queryMessages(ctx context.Context, convID *protos.UUID, version int64, limit int) ([]Message, error) {
	req := &protos.QueryMessagesRequest{
		SelfUserId:         c.selfUUID(),
		ConversationId:     convID,
		RequestedCountSize: int32(limit),
		CurrentVersion:     version,
	}
	var resp protos.QueryMessagesResponse
	if err := c.doGRPC(ctx, paths.QUERY_MESSAGES, req, &resp); err != nil {
		return nil, err
	}
	result := make([]Message, 0, len(resp.GetMessages()))
	for _, msg := range resp.GetMessages() {
		converted := c.messageFromProto(msg)
		if converted.ID != "" {
			result = append(result, converted)
		}
	}
	return result, nil
}

func (c *Client) hydrateConversations(ctx context.Context, entries []*protos.ConversationEntry) error {
	requests := make([]*protos.DeltaSyncRequest, 0, len(entries))
	for _, entry := range entries {
		convID := entry.GetVersionInfo().GetConversationId()
		id := uuidToString(convID)
		if id == "" {
			continue
		}
		requests = append(requests, &protos.DeltaSyncRequest{
			SelfUserId:     c.selfUUID(),
			ConversationId: convID,
		})
	}
	if len(requests) == 0 {
		return nil
	}
	req := &protos.BatchDeltaSyncRequest{DeltaSyncRequests: requests}
	var resp protos.BatchDeltaSyncResponse
	if err := c.doGRPC(ctx, paths.BATCH_DELTA_SYNC_CONVERSATIONS, req, &resp); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, item := range resp.GetDeltaSyncResponses() {
		conv := item.GetSuccessResponse().GetConversation()
		if conv == nil {
			continue
		}
		id := uuidToString(conv.GetConversationId())
		if id == "" {
			continue
		}
		c.conversations[id] = conv
		c.conversationIDs[id] = conv.GetConversationId()
	}
	return nil
}

func (c *Client) queryAllConversations(ctx context.Context, syncToken []byte, pageSize int) ([]*protos.ConversationEntry, []byte, error) {
	if pageSize <= 0 {
		pageSize = 100
	}
	var entries []*protos.ConversationEntry
	var pagination *protos.QueryConversationsRequest_PaginationInfo
	token := syncToken
	for page := 0; page < 5; page++ {
		req := &protos.QueryConversationsRequest{
			SelfUserId:        c.selfUUID(),
			RequestedPageSize: int32(pageSize),
			PaginationInfo:    pagination,
			SyncToken:         token,
		}
		var resp protos.QueryConversationsResponse
		if err := c.doGRPC(ctx, paths.QUERY_CONVERSATIONS, req, &resp); err != nil {
			return entries, token, err
		}
		entries = append(entries, resp.GetConversations()...)
		if len(resp.GetSyncToken()) > 0 {
			token = resp.GetSyncToken()
		}
		last := resp.GetLastConversation()
		if resp.GetNoMore() || last == nil {
			break
		}
		pagination = &protos.QueryConversationsRequest_PaginationInfo{
			OldestConversationOrderTimestamp: last.GetOldestConversationOrderTimestamp(),
			OldestConversationId:             last.GetOldestConversationId(),
		}
	}
	return entries, token, nil
}

func (c *Client) resolveEntryNames(ctx context.Context, entries []*protos.ConversationEntry) error {
	unknown := make([]string, 0)
	seen := make(map[string]struct{})
	for _, entry := range entries {
		if strings.TrimSpace(entry.GetTitle()) != "" {
			continue
		}
		for _, participant := range entry.GetParticipants() {
			id := uuidToString(participant)
			if id == "" || id == c.SelfUserID() {
				continue
			}
			if c.lookupName(id) != "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			unknown = append(unknown, id)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	names, err := c.fetchPublicNames(ctx, unknown)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, name := range names {
		c.namesByUserID[id] = name
	}
	return nil
}

func (c *Client) chatFromEntry(entry *protos.ConversationEntry) Chat {
	id := uuidToString(entry.GetVersionInfo().GetConversationId())
	name := strings.TrimSpace(entry.GetTitle())
	otherUserID := ""
	nonSelfParticipants := 0
	for _, participant := range entry.GetParticipants() {
		userID := uuidToString(participant)
		if userID == "" || userID == c.SelfUserID() {
			continue
		}
		nonSelfParticipants++
		if otherUserID == "" {
			otherUserID = userID
		}
		if name == "" {
			name = c.lookupName(userID)
			if name == "" {
				name = shortID(userID)
			}
		}
	}
	if nonSelfParticipants != 1 {
		otherUserID = ""
	}
	if name == "" {
		name = shortID(id)
	}
	info := entry.GetLastFeedUpdateInfo()
	preview, unread := previewFromDisplayInfo(info)
	lastAt := timeFromMillis(entry.GetLastEventTimestamp())
	if info.GetDisplayTimestamp() > 0 {
		lastAt = timeFromMillis(info.GetDisplayTimestamp())
	}
	return Chat{
		ID:             id,
		OtherUserID:    otherUserID,
		Name:           name,
		Preview:        preview,
		LastMessage:    preview,
		Unread:         unread,
		Version:        entry.GetVersionInfo().GetConversationVersion(),
		LastActivityAt: lastAt,
	}
}

func (c *Client) messageFromProto(msg *protos.ContentMessage) Message {
	id := fmt.Sprintf("%d", msg.GetMessageId())
	if msg.GetMessageId() == 0 {
		id = fmt.Sprintf("client-%d", msg.GetClientResolutionId())
	}
	authorID := uuidToString(msg.GetSenderId())
	body, isSnap := messageBody(msg)
	media := mediaAttachmentsFromEnvelope(msg.GetContents(), id)
	if body == "" && len(media) > 0 {
		body = "New Snap"
		isSnap = true
	}
	if body == "" {
		return Message{}
	}
	createdAt := timeFromMillis(msg.GetMetaData().GetServerCreatedAt())
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	authorName := c.lookupName(authorID)
	if authorName == "" && authorID == c.SelfUserID() {
		authorName = "You"
	}
	if authorName == "" {
		authorName = shortID(authorID)
	}
	return Message{
		ID:        id,
		AuthorID:  authorID,
		Author:    authorName,
		Text:      body,
		Timestamp: createdAt,
		Outgoing:  authorID != "" && authorID == c.SelfUserID(),
		IsSnap:    isSnap,
		Media:     media,
		Version:   msg.GetMetaData().GetConversationVersion(),
	}
}

func messageBody(msg *protos.ContentMessage) (string, bool) {
	envelope := msg.GetContents()
	if envelope == nil {
		return "New Snap", true
	}
	var contents protos.Contents
	if len(envelope.GetContents()) > 0 {
		if err := protos.DecodeProtoMessage(envelope.GetContents(), &contents); err == nil {
			if text := contents.GetText().GetText(); strings.TrimSpace(text) != "" {
				return text, false
			}
		}
	}
	switch envelope.GetContentType() {
	case protos.ContentType_CHAT:
		return "[Unsupported Snapchat chat]", false
	case protos.ContentType_SNAP, protos.ContentType_EXTERNAL_MEDIA, protos.ContentType_SNAP_NOT_VIEWABLE:
		return "New Snap", true
	case protos.ContentType_STATUS, protos.ContentType_STATUS_SAVE_TO_CAMERA_ROLL,
		protos.ContentType_STATUS_CONVERSATION_CAPTURE_SCREENSHOT,
		protos.ContentType_STATUS_CONVERSATION_CAPTURE_RECORD,
		protos.ContentType_STATUS_CALL_MISSED_VIDEO,
		protos.ContentType_STATUS_CALL_MISSED_AUDIO:
		return fmt.Sprintf("[Snapchat status] %s", envelope.GetContentType().String()), false
	default:
		return "New Snap", true
	}
}

func mediaAttachmentsFromEnvelope(envelope *protos.ContentEnvelope, messageID string) []MediaAttachment {
	if envelope == nil {
		return nil
	}
	key, iv := mediaKeyFromEnvelope(envelope)
	media := make([]MediaAttachment, 0)
	for index, info := range envelope.GetRemoteMediaInfos() {
		kind, mimeType := mediaKindFromRemote(info.GetMediaType(), info.GetHasAudio())
		attachment := MediaAttachment{
			ID:       stableMediaID(messageID, "remote", fmt.Sprint(index), info.GetContentUrl(), info.GetLegacyMediaId()),
			URL:      strings.TrimSpace(info.GetContentUrl()),
			FileName: mediaFileName(messageID, index, kind, mimeType),
			MimeType: mimeType,
			Kind:     kind,
			Data:     append([]byte(nil), info.GetContentObject()...),
			Key:      key,
			IV:       iv,
		}
		if attachment.URL != "" || len(attachment.Data) > 0 {
			media = append(media, attachment)
		}
	}
	for listIndex, list := range envelope.GetMediaReferenceLists() {
		for refIndex, ref := range list.GetReference() {
			kind, mimeType := mediaKindFromReference(ref.GetMediaType())
			attachment := MediaAttachment{
				ID:       stableMediaID(messageID, "ref", fmt.Sprint(listIndex), fmt.Sprint(refIndex), ref.GetUrl(), fmt.Sprint(ref.GetMediaListId())),
				URL:      strings.TrimSpace(ref.GetUrl()),
				FileName: mediaFileName(messageID, len(media), kind, mimeType),
				MimeType: mimeType,
				Kind:     kind,
				Data:     append([]byte(nil), ref.GetContentObject()...),
				Key:      key,
				IV:       iv,
			}
			if attachment.URL != "" || len(attachment.Data) > 0 {
				media = append(media, attachment)
			}
		}
	}
	return media
}

func mediaKeyFromEnvelope(envelope *protos.ContentEnvelope) ([]byte, []byte) {
	if envelope == nil {
		return nil, nil
	}
	clear := envelope.GetEnvelopeEncryption().GetClearTextMediaKey()
	if clear == nil {
		return nil, nil
	}
	return append([]byte(nil), clear.GetMediaKey()...), append([]byte(nil), clear.GetMediaIv()...)
}

func mediaKindFromRemote(raw int32, hasAudio bool) (MediaKind, string) {
	switch protos.ContentEnvelope_RemoteMediaInfo_MediaType(raw) {
	case protos.ContentEnvelope_RemoteMediaInfo_MediaType_IMAGE:
		return MediaKindImage, "image/jpeg"
	case protos.ContentEnvelope_RemoteMediaInfo_MediaType_VIDEO:
		return MediaKindVideo, "video/mp4"
	case protos.ContentEnvelope_RemoteMediaInfo_MediaType_GIF:
		return MediaKindGIF, "image/gif"
	default:
		if hasAudio {
			return MediaKindVideo, "video/mp4"
		}
		return MediaKindFile, "application/octet-stream"
	}
}

func mediaKindFromReference(raw protos.MediaType) (MediaKind, string) {
	switch raw {
	case protos.MediaType_MEDIA_TYPE_IMAGE:
		return MediaKindImage, "image/jpeg"
	case protos.MediaType_MEDIA_TYPE_VIDEO, protos.MediaType_MEDIA_TYPE_VIDEONOAUDIO:
		return MediaKindVideo, "video/mp4"
	case protos.MediaType_MEDIA_TYPE_ANIMATEDIMAGE:
		return MediaKindGIF, "image/gif"
	case protos.MediaType_MEDIA_TYPE_AUDIO:
		return MediaKindFile, "audio/mpeg"
	default:
		return MediaKindFile, "application/octet-stream"
	}
}

func mediaFileName(messageID string, index int, kind MediaKind, mimeType string) string {
	ext := ".bin"
	switch {
	case strings.Contains(mimeType, "jpeg"):
		ext = ".jpg"
	case strings.Contains(mimeType, "png"):
		ext = ".png"
	case strings.Contains(mimeType, "gif"):
		ext = ".gif"
	case strings.Contains(mimeType, "mp4"):
		ext = ".mp4"
	case strings.Contains(mimeType, "mpeg"):
		ext = ".mp3"
	case kind == MediaKindImage:
		ext = ".jpg"
	case kind == MediaKindVideo:
		ext = ".mp4"
	}
	return fmt.Sprintf("snap-%s-%d%s", sanitizeFileNamePart(messageID), index+1, ext)
}

func sanitizeFileNamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "media"
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "media"
	}
	return builder.String()
}

func stableMediaID(parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "::")))
	return hex.EncodeToString(hash[:])[:16]
}

func detectMediaMime(data []byte, kind MediaKind) string {
	detected := http.DetectContentType(data)
	if detected != "application/octet-stream" {
		return detected
	}
	switch kind {
	case MediaKindImage:
		return "image/jpeg"
	case MediaKindVideo:
		return "video/mp4"
	case MediaKindGIF:
		return "image/gif"
	default:
		return detected
	}
}

func decryptMediaCBC(data, key, iv []byte) ([]byte, bool) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, false
	}
	if len(iv) < aes.BlockSize || len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, false
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, false
	}
	output := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv[:aes.BlockSize]).CryptBlocks(output, data)
	output, ok := pkcs7Unpad(output, aes.BlockSize)
	if !ok {
		return nil, false
	}
	return output, true
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, bool) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, false
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, false
	}
	for _, value := range data[len(data)-pad:] {
		if int(value) != pad {
			return nil, false
		}
	}
	return data[:len(data)-pad], true
}

func previewFromDisplayInfo(info *protos.DisplayInfo) (string, bool) {
	if info == nil {
		return "", false
	}
	if snap := info.GetSnapItem(); snap != nil {
		if snap.GetState() == protos.SnapItemState_VIEWED {
			return "Opened Snap", false
		}
		if snap.GetHasAudio() {
			return "New Snap with audio", true
		}
		return "New Snap", snap.GetState() != protos.SnapItemState_VIEWED
	}
	if chat := info.GetChatItem(); chat != nil {
		switch chat.GetState() {
		case protos.ChatItemState_CHAT_UNVIEWED, protos.ChatItemState_CHAT_SAVED_UNVIEWED,
			protos.ChatItemState_CHAT_SCREENSHOTTED_UNVIEWED, protos.ChatItemState_CHAT_RECORDED_UNVIEWED:
			return "New Chat", true
		case protos.ChatItemState_CHAT_VIEWED, protos.ChatItemState_CHAT_SAVED_VIEWED:
			return "Chat", false
		default:
			return chat.GetState().String(), false
		}
	}
	if call := info.GetCallItem(); call != nil {
		if call.GetIsVideo() {
			return "Missed video call", true
		}
		return "Missed call", true
	}
	if conv := info.GetConversationItem(); conv != nil {
		return conv.GetState().String(), false
	}
	return "", false
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

func (c *Client) fetchPublicNames(ctx context.Context, userIDs []string) (map[string]string, error) {
	form, err := payload.GetPublicUserInfo(userIDs, "CHAT")
	if err != nil {
		return nil, err
	}
	header := headers.NewDefaultHeaders(c.cookies, c.tokens, true)
	c.applyUserAgentHeaders(header)
	header.Set("content-type", "application/x-www-form-urlencoded; charset=utf-8")
	body, err := c.doHTTP(ctx, paths.GET_USER_PUBLIC_INFO, http.MethodPost, header, form)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Snapchatters []struct {
			ID              string `json:"user_id"`
			Username        string `json:"username"`
			DisplayName     string `json:"display_name"`
			MutableUsername string `json:"mutable_username"`
		} `json:"snapchatters"`
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	names := make(map[string]string, len(resp.Snapchatters))
	for _, user := range resp.Snapchatters {
		name := user.DisplayName
		if name == "" {
			name = user.MutableUsername
		}
		if name == "" {
			name = user.Username
		}
		if user.ID != "" && name != "" {
			names[user.ID] = name
		}
	}
	return names, nil
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

func (c *Client) lookupName(userID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.namesByUserID[userID]
}

func conversationVersionInfos(versions map[string]int64) []*protos.ConversationVersionInfo {
	infos := make([]*protos.ConversationVersionInfo, 0, len(versions))
	for id, version := range versions {
		encoded, err := encodeUUIDString(id)
		if err != nil {
			continue
		}
		infos = append(infos, &protos.ConversationVersionInfo{
			ConversationId:      encoded,
			ConversationVersion: version,
		})
	}
	return infos
}

func mergeEntries(primary, secondary []*protos.ConversationEntry) []*protos.ConversationEntry {
	byID := make(map[string]*protos.ConversationEntry, len(primary)+len(secondary))
	for _, entry := range secondary {
		if id := uuidToString(entry.GetVersionInfo().GetConversationId()); id != "" {
			byID[id] = entry
		}
	}
	for _, entry := range primary {
		if id := uuidToString(entry.GetVersionInfo().GetConversationId()); id != "" {
			byID[id] = entry
		}
	}
	merged := make([]*protos.ConversationEntry, 0, len(byID))
	for _, entry := range byID {
		merged = append(merged, entry)
	}
	return merged
}

func cloneVersions(in map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func encodeUUIDString(value string) (*protos.UUID, error) {
	encoded, err := snapcrypto.EncodeUUID(value)
	if err != nil {
		return nil, err
	}
	return &protos.UUID{EncodedId: encoded}, nil
}

func uuidToString(value *protos.UUID) string {
	if value == nil || len(value.GetEncodedId()) != 16 {
		return ""
	}
	decoded, err := snapcrypto.DecodeUUID(value.GetEncodedId())
	if err != nil {
		return ""
	}
	return decoded
}

func frameGRPC(message []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(0)
	_ = binary.Write(&buf, binary.BigEndian, int32(len(message)))
	buf.Write(message)
	return buf.Bytes()
}

func randomUint64() uint64 {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return uint64(time.Now().UnixNano())
	}
	return binary.BigEndian.Uint64(data[:])
}

func timeFromMillis(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(value)
}

func shortID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Snapchat"
	}
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func extractCookieValue(cookieString, key string) string {
	for _, rawPart := range strings.Split(cookieString, ";") {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		name, value, ok := strings.Cut(part, "=")
		if ok && strings.TrimSpace(name) == key {
			unescaped, err := url.QueryUnescape(value)
			if err == nil {
				return unescaped
			}
			return value
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
