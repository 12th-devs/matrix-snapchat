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
	Name           string
	AvatarURL      string
	Preview        string
	LastMessage    string
	Unread         bool
	Version        int64
	LastActivityAt time.Time
	DisappearAfter time.Duration
	Status         ConversationStatus
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
	Status         MessageStatus
}

type ParticipantStatus struct {
	UserID                    string
	ReadHighWatermark         int64
	ReleaseHighWatermark      int64
	SnapReleaseHighWatermark  int64
	ReactionReadHighWatermark int64
	ReleaseWatermark          int64
}

type ConversationStatus struct {
	ConversationID                    string
	Participants                      []ParticipantStatus
	FeedOpenedMessageDisplayTimestamp int64
}

type MessageStatus struct {
	ReadTimestamp       int64
	ReadBy              []string
	ReleasedBy          []string
	SavedBy             []string
	ScreenshottedBy     []string
	ScreenRecordedBy    []string
	ReplayedBy          []string
	ConversationVersion int64
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

	mu               sync.Mutex
	selfUserID       string
	selfEncoded      *protos.UUID
	conversations    map[string]*protos.Conversation
	conversationIDs  map[string]*protos.UUID
	namesByUserID    map[string]string
	avatarsByUserID  map[string]string
	profileCheckedAt map[string]time.Time
	failedEEL        map[string]time.Time
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
		eelDecrypter:      cfg.EELDecrypter,
		conversations:     make(map[string]*protos.Conversation),
		conversationIDs:   make(map[string]*protos.UUID),
		namesByUserID:     make(map[string]string),
		avatarsByUserID:   make(map[string]string),
		profileCheckedAt:  make(map[string]time.Time),
		failedEEL:         make(map[string]time.Time),
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
	Name      string
	AvatarURL string
}

// PublicProfile is the safe subset of Snapchat profile data that bridge code may expose.
type PublicProfile struct {
	UserID    string
	Name      string
	AvatarURL string
}
