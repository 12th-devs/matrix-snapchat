package snapapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/0xzer/snapper/protos"
	"github.com/0xzer/snapper/types"
)

type Config struct {
	CookieString        string
	SSOToken            string
	SelfUserID          string
	UserAgent           string
	SnapClientUserAgent string
	SecChUA             string
	SecChUAPlatform     string
	GRPCWebUserAgent    string
	MCSCOFIDsBin        string
	Timeout             time.Duration
	FetchLimit          int
	EELDecrypter        EELDecrypter
}

type EELDecryptRequest struct {
	ConversationID  string
	MessageID       string
	Content         []byte
	CEK             []byte
	CEKIV           []byte
	Nonce           []byte
	SenderPublicKey []byte
	SenderVersion   int32
	// MediaIDs are the envelope's media content-object IDs. The connector
	// matches fetched decrypted content against them because snap messages
	// use a different analytics identifier shape in the web app.
	MediaIDs []string
	// TimestampMs is the message's server-created timestamp. Snap messages
	// embed their capture timestamp in the decrypted contents, so the
	// connector matches on it (snap ids are absent from the app state).
	TimestampMs int64
}

type EELDecrypter interface {
	DecryptEEL(ctx context.Context, req EELDecryptRequest) ([]byte, error)
}

type State struct {
	SyncToken            []byte
	ConversationVersions map[string]int64
	SelfUserID           string
}

type Chat struct {
	ID             string
	OtherUserID    string
	ParticipantIDs []string
	IsGroup        bool
	Name           string
	Username       string
	AvatarURL      string
	Preview        string
	LastMessage    string
	Unread         bool
	Version        int64
	LastActivityAt time.Time
	DisappearAfter time.Duration
	// ReadWatermarks maps participant Snapchat user ID -> last-read message ID
	// (ReadHighWatermark) from the hydrated conversation object. Absent/zero
	// entries mean the participant has not read anything.
	ReadWatermarks map[string]int64
}

type MediaKind string

const (
	MediaKindFile  MediaKind = "file"
	MediaKindImage MediaKind = "image"
	MediaKindVideo MediaKind = "video"
	MediaKindGIF   MediaKind = "gif"
	MediaKindAudio MediaKind = "audio"
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
	ID          string
	AuthorID    string
	Author      string
	Text        string
	Timestamp   time.Time
	Outgoing    bool
	IsSnap      bool
	ContentType string
	Media       []MediaAttachment
	Version     int64
	Saved       bool
	// DisappearAfter mirrors Snapchat's per-conversation retention timer for
	// unsaved messages. Saved messages intentionally leave this unset.
	DisappearAfter time.Duration
	// QuotedMessageID is the numeric Snapchat message this message replies to
	// (from MessageMetadata.QuotedMetadata), 0 when not a reply.
	QuotedMessageID int64
	// Tombstone marks a message that was erased/unsent on Snapchat
	// (MessageMetadata.Tombstone): it should be removed, not displayed.
	Tombstone bool
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
	eelDecrypter      EELDecrypter

	mu                sync.Mutex
	selfUserID        string
	selfEncoded       *protos.UUID
	conversations     map[string]*protos.Conversation
	conversationIDs   map[string]*protos.UUID
	namesByUserID     map[string]string
	usernamesByUserID map[string]string
	avatarsByUserID   map[string]string
	profileCheckedAt  map[string]time.Time
	eelPlaintext      map[string][]byte
	failedEEL         map[string]time.Time
	// eelInflight suppresses duplicate concurrent EEL decrypt attempts for
	// the same conversation+message: one connector task per target, waiters
	// simply skip and the message stays retryable on a later poll.
	eelInflight map[string]chan struct{}
	// lastEELAttempt globally throttles EEL connector work: a poll cycle with
	// many undecoded messages must not occupy the connector task queue for
	// minutes (each attempt costs ~22s). At most one attempt per interval;
	// skipped messages remain retryable on later polls.
	lastEELAttempt time.Time
	// eelWindowStart/eelWindowAttempts bound total EEL attempts to 3 per 20s
	// window while letting fresh messages bypass the retry throttle.
	eelWindowStart    time.Time
	eelWindowAttempts int

	mediaMappingMu sync.Mutex
	mediaMapping   *boltNetworkMapping
	mediaMappingAt time.Time
}

const minPublicProfileRefreshInterval = 24 * time.Hour

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
		tokens:            &types.SnapTokens{SSO_TOKEN: strings.TrimSpace(cfg.SSOToken)},
		http:              &http.Client{Timeout: cfg.Timeout},
		device:            types.NewDevice(),
		sessionCookieName: sessionCookieName,
		userAgent:         strings.TrimSpace(cfg.UserAgent),
		snapClientUA:      strings.TrimSpace(cfg.SnapClientUserAgent),
		secChUA:           strings.TrimSpace(cfg.SecChUA),
		secChUAPlatform:   strings.TrimSpace(cfg.SecChUAPlatform),
		grpcWebUA:         strings.TrimSpace(cfg.GRPCWebUserAgent),
		mcsCOFIDsBin:      strings.TrimSpace(cfg.MCSCOFIDsBin),
		eelDecrypter:      cfg.EELDecrypter,
		conversations:     make(map[string]*protos.Conversation),
		conversationIDs:   make(map[string]*protos.UUID),
		namesByUserID:     make(map[string]string),
		usernamesByUserID: make(map[string]string),
		avatarsByUserID:   make(map[string]string),
		profileCheckedAt:  make(map[string]time.Time),
		eelPlaintext:      make(map[string][]byte),
		failedEEL:         make(map[string]time.Time),
		eelInflight:       make(map[string]chan struct{}),
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

type publicProfile struct {
	Username  string
	Name      string
	AvatarURL string
}

// PublicProfile is the safe subset of Snapchat profile data that bridge code may expose.
type PublicProfile struct {
	UserID    string
	Username  string
	Name      string
	AvatarURL string
}
