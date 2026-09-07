package connector

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/bridgev2/status"
	"maunium.net/go/mautrix/event"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/snapapi"
)

type chatRef struct {
	ID                    string
	URL                   string
	Name                  string
	OtherUserID           string
	ParticipantIDs        []string
	IsGroup               bool
	Username              string
	DisappearAfterSeconds int64
}

type pendingOutgoing struct {
	RemoteID string
	Body     string
	SentAt   time.Time
}

type typingUpdateState struct {
	Typing bool
	SentAt time.Time
}

type SnapchatAPI struct {
	Connector *SnapchatConnector
	UserLogin *bridgev2.UserLogin
	Label     string
	Client    *sidecar.Client

	apiMu                    sync.Mutex
	resyncMu                 sync.Mutex
	apiClient                *snapapi.Client
	apiCookieString          string
	apiAuthCheckedAt         time.Time
	consecutiveUnauthorized  int
	reLoginRequired          bool
	nextRecoveryCheckAt      time.Time
	readWatermarks           map[string]map[string]int64
	apiSelfUserID            string
	mu                       sync.Mutex
	seenByChat               map[string]map[string]struct{}
	chatsByID                map[string]chatRef
	ghostNames               map[string]string
	ghostUsernames           map[string]string
	chatState                map[string]string
	lastMessageID            map[string]int64
	lastMessageVersion       map[string]int64
	messageVersions          map[string]map[int64]int64
	messageFailures          map[string]int
	messageRetryAfter        map[string]time.Time
	lastReadReceiptSync      map[string]time.Time
	lastTypingSent           map[string]typingUpdateState
	typingPresenceChatID     string
	typingSyncRunning        bool
	remoteTyping             map[string]map[string]bool
	ghostAvatarCheckedAt     map[string]time.Time
	avatarURLBySnapID        map[string]string
	queuedPortalResyncs      map[string]struct{}
	appliedChatInfoState     map[string]chatInfoState
	lastSkippedChatInfoState map[string]chatInfoState
	recentOutgoing           map[string][]pendingOutgoing
	chatDisappearAfter       map[string]int64
	avatarBootstrapDone      bool
	sidebarBaselineReady     bool
	sidebarBaselineAt        time.Time
	hybridBackfillStarted    bool
}

type connectorEELDecrypter struct {
	client *sidecar.Client
	api    *SnapchatAPI
}

func (d connectorEELDecrypter) DecryptEEL(ctx context.Context, req snapapi.EELDecryptRequest) ([]byte, error) {
	decrypted, _, err := d.DecryptEELWithCandidates(ctx, req)
	return decrypted, err
}

// DecryptEELWithCandidates carries ambiguous snap timestamp-match candidates
// through the connector so the bridge can try every candidate media key.
func (d connectorEELDecrypter) DecryptEELWithCandidates(ctx context.Context, req snapapi.EELDecryptRequest) ([]byte, [][]byte, error) {
	if d.client == nil {
		return nil, nil, fmt.Errorf("connector client is not initialized")
	}
	decrypted, response, err := d.client.DecryptEELDetailed(ctx, sidecar.EELDecryptRequest{
		ConversationID:        req.ConversationID,
		MessageID:             req.MessageID,
		ContentBase64:         base64.StdEncoding.EncodeToString(req.Content),
		CEKBase64:             base64.StdEncoding.EncodeToString(req.CEK),
		CEKIVBase64:           base64.StdEncoding.EncodeToString(req.CEKIV),
		NonceBase64:           base64.StdEncoding.EncodeToString(req.Nonce),
		SenderPublicKeyBase64: base64.StdEncoding.EncodeToString(req.SenderPublicKey),
		SenderVersion:         req.SenderVersion,
		MediaIDs:              req.MediaIDs,
		TimestampMs:           req.TimestampMs,
	})
	if err != nil {
		return nil, nil, err
	}
	var candidates [][]byte
	for _, encoded := range response.DecryptedCandidatesBase64 {
		if value, decodeErr := base64.StdEncoding.DecodeString(encoded); decodeErr == nil && len(value) > 0 {
			candidates = append(candidates, value)
		}
	}
	return decrypted, candidates, nil
}

var _ bridgev2.NetworkAPI = (*SnapchatAPI)(nil)
var _ bridgev2.ReadReceiptHandlingNetworkAPI = (*SnapchatAPI)(nil)
var _ bridgev2.TypingHandlingNetworkAPI = (*SnapchatAPI)(nil)

var relativeTimestampPattern = regexp.MustCompile(`(?i)^(\d+)\s*([mhdwy])$`)
var snapUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

const minGhostAvatarRefreshInterval = 24 * time.Hour

var sidebarRelativeAgePattern = regexp.MustCompile(`(?i)(^|[\s\-·|:])\d+\s*[mhdwy]\b`)

func NewSnapchatAPI(sc *SnapchatConnector, login *bridgev2.UserLogin, label string) *SnapchatAPI {
	return &SnapchatAPI{
		Connector:                sc,
		UserLogin:                login,
		Label:                    label,
		Client:                   sc.newClient(),
		seenByChat:               make(map[string]map[string]struct{}),
		chatsByID:                make(map[string]chatRef),
		ghostNames:               make(map[string]string),
		ghostUsernames:           make(map[string]string),
		chatState:                make(map[string]string),
		lastMessageID:            make(map[string]int64),
		lastMessageVersion:       make(map[string]int64),
		messageVersions:          make(map[string]map[int64]int64),
		messageFailures:          make(map[string]int),
		messageRetryAfter:        make(map[string]time.Time),
		lastReadReceiptSync:      make(map[string]time.Time),
		lastTypingSent:           make(map[string]typingUpdateState),
		remoteTyping:             make(map[string]map[string]bool),
		ghostAvatarCheckedAt:     make(map[string]time.Time),
		avatarURLBySnapID:        make(map[string]string),
		queuedPortalResyncs:      make(map[string]struct{}),
		appliedChatInfoState:     make(map[string]chatInfoState),
		lastSkippedChatInfoState: make(map[string]chatInfoState),
		recentOutgoing:           make(map[string][]pendingOutgoing),
		chatDisappearAfter:       make(map[string]int64),
		readWatermarks:           make(map[string]map[string]int64),
	}
}

func (sa *SnapchatAPI) Connect(ctx context.Context) {
	sa.restoreChatMappings()
	if sa.UserLogin != nil {
		sa.UserLogin.BridgeState.Send(status.BridgeState{StateEvent: status.StateConnected})
	}
	if sa.Connector.apiMode() == "hybrid" {
		// Hybrid mode uses the browser sidebar as the authoritative portal list.
		// Do not race that baseline with legacy stored-portal resync, avatar, or
		// message backfill jobs; those can retain obsolete mappings and contend
		// with the main SQLite connection during startup.
		go sa.pollLoop(ctx)
		return
	}
	sa.queueStoredPortalResyncs(ctx)
	go sa.primeStoredPortalAvatars(ctx)
	if sa.autoFetchMessagesEnabled() {
		go sa.syncStoredPortalMessages(ctx)
	}
	go sa.pollLoop(ctx)
}

func (sa *SnapchatAPI) Disconnect() {}

func (sa *SnapchatAPI) IsLoggedIn() bool {
	status, err := sa.Client.Status(context.Background())
	if err == nil {
		sa.updateLoginState(status)
		return status.Authenticated
	}

	meta, _ := sa.UserLoginMetadata()
	if meta != nil && meta.Authenticated {
		return true
	}
	if sa.Connector != nil && sa.Connector.store != nil {
		state, stateErr := sa.Connector.store.GetLogin(string(makeUserLoginID(sa.Label)))
		if stateErr == nil && state != nil && strings.EqualFold(state.LastSeenState, "ready") {
			return true
		}
	}
	return false
}

func (sa *SnapchatAPI) LogoutRemote(ctx context.Context) {}

func (sa *SnapchatAPI) GetCapabilities(ctx context.Context, portal *bridgev2.Portal) *event.RoomFeatures {
	return sa.Connector.chatCapabilities()
}

func (sa *SnapchatAPI) IsThisUser(ctx context.Context, userID networkid.UserID) bool {
	if userID == makeUserID(sa.Label) {
		return true
	}
	selfUserID := sa.currentSelfUserID()
	return selfUserID != "" && strings.EqualFold(normalizeIDPart(string(userID)), normalizeIDPart(selfUserID))
}

func (sa *SnapchatAPI) UserLoginMetadata() (*UserLoginMetadata, bool) {
	if sa == nil || sa.UserLogin == nil {
		return nil, false
	}
	meta, ok := sa.UserLogin.Metadata.(*UserLoginMetadata)
	return meta, ok
}

func sidebarUpdateText(chat sidecar.Chat) (string, bool) {
	detail := strings.TrimSpace(chat.Preview)
	if detail == "" {
		detail = strings.TrimSpace(chat.LastMessage)
	}
	lower := strings.ToLower(detail)
	if isPassiveSidebarStatus(lower) {
		return "", false
	}
	if text, ok := systemEventPreviewText(lower); ok {
		return text, true
	}
	// Client-rendered system lines reach the sidebar as their exact rendered
	// text; pass them through instead of the generic "New message".
	if text, ok := clientRenderedSystemLine(lower, detail); ok {
		return text, true
	}
	if chat.Unread || looksLikeSnapOrMediaStatus(lower) {
		return "New Snap", true
	}
	if detail == "" {
		return "", false
	}
	return "New message", true
}

// clientRenderedSystemLine recognizes the sidebar previews of client-rendered
// system lines whose exact texts are confirmed in the web bundle:
// "YOU ARE USING SNAPCHAT FOR WEB", "{sender} IS USING SNAPCHAT FOR WEB",
// "YOU AND {name} STARTED A SNAPSTREAK 🔥" and "YOUR {n}-DAY SNAPSTREAK
// ENDED". The preview already is the exact client-rendered line, so it is
// passed through verbatim.
func clientRenderedSystemLine(lower, detail string) (string, bool) {
	switch {
	case strings.Contains(lower, "using snapchat for web"),
		strings.Contains(lower, "snapstreak"),
		strings.Contains(lower, "-day snapstreak ended"):
		return detail, true
	}
	return "", false
}

// systemEventPreviewText maps recognizable Snapchat sidebar previews of
// client-rendered system lines (screenshots, screen recordings, missed
// calls) to the friendly text used by the message path, so these events
// become gray m.notice room events instead of generic "New message"
// placeholders. Snapchat renders these lines client-side; only the sidebar
// preview reaches the bridge.
func systemEventPreviewText(lower string) (string, bool) {
	switch {
	case strings.Contains(lower, "screenshot"):
		return "Screenshot captured", true
	case strings.Contains(lower, "screen record"), strings.Contains(lower, "screenrecord"):
		return "Screen recorded", true
	case strings.Contains(lower, "missed"):
		if strings.Contains(lower, "video") {
			return "Missed video call", true
		}
		return "Missed audio call", true
	}
	return "", false
}

func isPassiveSidebarStatus(lower string) bool {
	lower = strings.TrimSpace(lower)
	if lower == "" {
		return false
	}
	status := strings.FieldsFunc(lower, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == 183 || r == ':' || r == '-' || r == '|'
	})
	if len(status) == 0 {
		return false
	}
	switch status[0] {
	case "opened", "delivered", "sent", "read", "seen":
		return true
	default:
		return false
	}
}

func looksLikeSnapOrMediaStatus(lower string) bool {
	lower = strings.TrimSpace(lower)
	return strings.Contains(lower, "snap") ||
		strings.Contains(lower, "received") ||
		strings.Contains(lower, "photo") ||
		strings.Contains(lower, "video")
}

func dedupeChatsByName(chats []sidecar.Chat) []sidecar.Chat {
	if len(chats) < 2 {
		return chats
	}
	deduped := make([]sidecar.Chat, 0, len(chats))
	seen := make(map[string]struct{}, len(chats))
	for _, chat := range chats {
		key := strings.ToLower(strings.TrimSpace(chat.ID))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(chat.Name))
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, chat)
	}
	return deduped
}

func chatFingerprint(unread bool, preview, lastMessage string) string {
	return fmt.Sprintf("%t|%s|%s", unread, normalizeSidebarFingerprintPart(preview), normalizeSidebarFingerprintPart(lastMessage))
}

func normalizeSidebarFingerprintPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = sidebarRelativeAgePattern.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(value), " ")
}

func parseMessageTimestamp(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts
	}

	now := time.Now()
	location := now.Location()
	lower := strings.ToLower(raw)
	noon := func(ts time.Time) time.Time {
		return time.Date(ts.Year(), ts.Month(), ts.Day(), 12, 0, 0, 0, location)
	}
	switch lower {
	case "today":
		return noon(now)
	case "yesterday":
		return noon(now.AddDate(0, 0, -1))
	}
	if ts, ok := parseRelativeDateWithClock(lower, raw, now, location); ok {
		return ts
	}
	if match := relativeTimestampPattern.FindStringSubmatch(raw); match != nil {
		amount, err := strconv.Atoi(match[1])
		if err == nil {
			switch strings.ToLower(match[2]) {
			case "m":
				return now.Add(-time.Duration(amount) * time.Minute)
			case "h":
				return now.Add(-time.Duration(amount) * time.Hour)
			case "d":
				return now.AddDate(0, 0, -amount)
			case "w":
				return now.AddDate(0, 0, -7*amount)
			case "y":
				return now.AddDate(-amount, 0, 0)
			}
		}
	}
	if weekday, ok := parseWeekday(lower); ok {
		daysBack := (int(now.Weekday()) - int(weekday) + 7) % 7
		return noon(now.AddDate(0, 0, -daysBack))
	}

	layouts := []string{
		"January 2, 2006 3:04 PM",
		"January 2, 2006 at 3:04 PM",
		"January 2 2006 3:04 PM",
		"Jan 2, 2006 3:04 PM",
		"Jan 2, 2006 at 3:04 PM",
		"January 2 2006",
		"January 2, 2006",
		"January 2",
		"January 2 3:04 PM",
		"January 2 at 3:04 PM",
		"Jan 2 2006",
		"Jan 2, 2006",
		"Jan 2",
		"Jan 2 3:04 PM",
		"Jan 2 at 3:04 PM",
		"3:04 PM",
	}
	for _, layout := range layouts {
		if ts, err := time.ParseInLocation(layout, raw, location); err == nil {
			hasYear := strings.Contains(layout, "2006")
			hasMonth := strings.Contains(layout, "Jan") || strings.Contains(layout, "January")
			if !hasYear && hasMonth {
				ts = time.Date(now.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond(), location)
				if ts.After(now.Add(24 * time.Hour)) {
					ts = ts.AddDate(-1, 0, 0)
				}
				if !strings.Contains(layout, "3:04") {
					ts = noon(ts)
				}
			} else if !hasYear && !hasMonth {
				ts = time.Date(now.Year(), now.Month(), now.Day(), ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond(), location)
				if ts.After(now.Add(2 * time.Hour)) {
					ts = ts.AddDate(0, 0, -1)
				}
			}
			return ts
		}
	}
	return time.Time{}
}

func parseRelativeDateWithClock(lower, raw string, now time.Time, location *time.Location) (time.Time, bool) {
	parseClock := func(value string) (time.Time, bool) {
		value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "at "))
		value = strings.ReplaceAll(value, ".", "")
		layouts := []string{"3:04 PM", "3 PM", "15:04", "15"}
		for _, layout := range layouts {
			if ts, err := time.ParseInLocation(layout, value, location); err == nil {
				return ts, true
			}
		}
		return time.Time{}, false
	}
	withClock := func(day time.Time, rest string) (time.Time, bool) {
		clock, ok := parseClock(rest)
		if !ok {
			return time.Time{}, false
		}
		return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, location), true
	}
	for _, prefix := range []string{"today", "yesterday"} {
		if strings.HasPrefix(lower, prefix+" ") {
			day := now
			if prefix == "yesterday" {
				day = now.AddDate(0, 0, -1)
			}
			return withClock(day, strings.TrimSpace(raw[len(prefix):]))
		}
	}
	for _, field := range strings.Fields(lower) {
		weekday, ok := parseWeekday(strings.TrimSuffix(field, ","))
		if !ok {
			continue
		}
		prefixLen := len(field)
		if len(raw) < prefixLen {
			return time.Time{}, false
		}
		daysBack := (int(now.Weekday()) - int(weekday) + 7) % 7
		return withClock(now.AddDate(0, 0, -daysBack), strings.TrimSpace(raw[prefixLen:]))
	}
	return time.Time{}, false
}

func parseWeekday(raw string) (time.Weekday, bool) {
	switch raw {
	case "sunday", "sun":
		return time.Sunday, true
	case "monday", "mon":
		return time.Monday, true
	case "tuesday", "tue", "tues":
		return time.Tuesday, true
	case "wednesday", "wed":
		return time.Wednesday, true
	case "thursday", "thu", "thur", "thurs":
		return time.Thursday, true
	case "friday", "fri":
		return time.Friday, true
	case "saturday", "sat":
		return time.Saturday, true
	default:
		return time.Sunday, false
	}
}

func mediaMessageID(base string, media []sidecar.MediaAttachment) string {
	parts := make([]string, 0, len(media))
	for _, item := range media {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = fmt.Sprintf("%s:%d", strings.TrimSpace(item.FileName), len(item.Data))
		}
		if id != "" {
			parts = append(parts, id)
		}
	}
	if len(parts) == 0 {
		return base + "-media"
	}
	return base + "-media-" + stableTextID(strings.Join(parts, "|"))
}

func mediaUploadIntent(portal *bridgev2.Portal, fallback bridgev2.MatrixAPI) bridgev2.MatrixAPI {
	if portal != nil && portal.Bridge != nil && portal.Bridge.Bot != nil {
		return portal.Bridge.Bot
	}
	return fallback
}

// isBridgeGeneratedSnapchatMarker reports whether a Matrix-authored body is
// unmistakably a bridge-generated synthetic notice. This is intentionally
// narrow: it is the only body check allowed to drop outgoing Matrix messages,
// so ordinary user-authored words must never match.
func isBridgeGeneratedSnapchatMarker(body string) bool {
	trimmed := strings.TrimSpace(body)
	return strings.HasPrefix(trimmed, "[Snapchat") || strings.HasPrefix(trimmed, "[Unsupported Snapchat")
}

// isSystemEventContentType reports whether a Snapchat content type is a
// conversation event (status lines like screenshots, missed calls, saves, and
// shared items) that must render as a gray m.notice system line instead of a
// chatty m.text that triggers "New Message" notifications.
func isSystemEventContentType(contentType string) bool {
	return strings.HasPrefix(contentType, "STATUS") || contentType == "SHARE"
}

func isGeneratedSnapchatNotice(body string) bool {
	trimmed := strings.TrimSpace(body)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(trimmed, "[Snapchat") || strings.HasPrefix(trimmed, "[Unsupported Snapchat") {
		return true
	}
	if isPassiveSidebarStatus(lower) {
		return true
	}
	switch lower {
	case "received", "read", "seen", "opened", "delivered", "sent", "new snap", "new message", "snap unavailable",
		"📷 new snap", "🎥 new snap", "📷 snap", "🎥 snap":
		return true
	default:
		return false
	}
}

func matrixMsgTypeForMedia(mimeType string) event.MessageType {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return event.MsgImage
	case strings.HasPrefix(mimeType, "video/"):
		return event.MsgVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return event.MsgAudio
	default:
		return event.MsgFile
	}
}

// snapTypeMarker classifies an incoming media message for the Matrix
// representation so clients and tooling can immediately distinguish a
// disappearing image Snap, a disappearing video Snap, and ordinary saved chat
// media. It rides the event's unsigned extra data as
// "net.colej.snapchat.type".
func snapTypeMarker(isSnap bool, msgType event.MessageType) string {
	if !isSnap {
		return "chat_media"
	}
	switch msgType {
	case event.MsgImage:
		return "snap_image"
	case event.MsgVideo:
		return "snap_video"
	default:
		return "snap"
	}
}

// snapMediaCaption is the visible caption for hydrated disappearing Snaps.
// Ordinary chat media keeps its filename body; Snaps carry the emoji caption
// so the distinction survives in every client, including ones that ignore
// extra data.
func snapMediaCaption(msgType event.MessageType) string {
	switch msgType {
	case event.MsgImage:
		return "📷 Snap"
	case event.MsgVideo:
		return "🎥 Snap"
	default:
		return "Snap"
	}
}

func normalizeMediaMIME(data []byte, hinted string) string {
	hinted = strings.ToLower(strings.TrimSpace(strings.Split(hinted, ";")[0]))
	switch hinted {
	case "image/jpg":
		hinted = "image/jpeg"
	}
	detected := ""
	if len(data) > 0 {
		detected = strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0]))
		if detected == "image/jpg" {
			detected = "image/jpeg"
		}
	}
	if detected != "" && detected != "application/octet-stream" {
		return detected
	}
	if hinted != "" {
		return hinted
	}
	return "application/octet-stream"
}

func mediaFileNameForMIME(fileName string, msgType event.MessageType, mimeType string) string {
	fileName = sanitizeOutgoingFileName(fileName)
	lower := strings.ToLower(fileName)
	if fileName == "" || fileName == "snap-media.bin" || strings.HasSuffix(lower, ".bin") || !strings.Contains(fileName, ".") {
		return defaultMediaFileName(msgType, mimeType)
	}
	return fileName
}

func defaultMediaFileName(msgType event.MessageType, mimeType string) string {
	lowerMime := strings.ToLower(mimeType)
	switch {
	case msgType == event.MsgImage && strings.Contains(lowerMime, "gif"):
		return "snap.gif"
	case msgType == event.MsgImage && strings.Contains(lowerMime, "png"):
		return "snap.png"
	case msgType == event.MsgImage && strings.Contains(lowerMime, "webp"):
		return "snap.webp"
	case msgType == event.MsgImage:
		return "snap.jpg"
	case msgType == event.MsgVideo && strings.Contains(lowerMime, "quicktime"):
		return "snap.mov"
	case msgType == event.MsgVideo:
		return "snap.mp4"
	case msgType == event.MsgAudio && strings.Contains(lowerMime, "mp4"):
		return "snap-audio.m4a"
	case strings.HasPrefix(lowerMime, "audio/"):
		return "snap-audio.mp3"
	default:
		return "snap-media.bin"
	}
}

func sanitizeOutgoingFileName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
		} else if r == ' ' {
			builder.WriteByte('-')
		}
	}
	if builder.Len() == 0 {
		return ""
	}
	return builder.String()
}

func stableTextID(value string) string {
	hash := sha1.Sum([]byte(value))
	return hex.EncodeToString(hash[:])[:16]
}

func normalizeOutgoingBody(body string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(body), " "))
}

func connectorChatFromAPI(chat snapapi.Chat) sidecar.Chat {
	name := strings.TrimSpace(chat.Name)
	if name == "" {
		name = chat.ID
	}
	preview := strings.TrimSpace(chat.Preview)
	if preview == "" {
		preview = strings.TrimSpace(chat.LastMessage)
	}
	participantIDs := append([]string{}, chat.ParticipantIDs...)
	if len(participantIDs) == 0 && strings.TrimSpace(chat.OtherUserID) != "" {
		participantIDs = []string{strings.TrimSpace(chat.OtherUserID)}
	}
	return sidecar.Chat{
		ID:                    chat.ID,
		OtherUserID:           chat.OtherUserID,
		ParticipantIDs:        participantIDs,
		IsGroup:               chat.IsGroup || len(participantIDs) > 1,
		URL:                   snapchatConversationURL(chat.ID),
		Name:                  name,
		Username:              chat.Username,
		Preview:               preview,
		Unread:                chat.Unread,
		LastMessage:           preview,
		DisappearAfterSeconds: int64(chat.DisappearAfter / time.Second),
		ReadWatermarks:        chat.ReadWatermarks,
	}
}

func connectorMessageFromAPI(message snapapi.Message) sidecar.Message {
	body := strings.TrimSpace(message.Text)
	if body == "" && message.IsSnap {
		body = "New Snap"
	}
	if body == "" {
		body = "[Unsupported Snapchat event]"
	}
	author := strings.TrimSpace(message.Author)
	if author == "" {
		author = message.AuthorID
	}
	ts := ""
	if !message.Timestamp.IsZero() {
		ts = message.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return sidecar.Message{
		ID:                    message.ID,
		AuthorID:              message.AuthorID,
		Author:                author,
		Text:                  body,
		Timestamp:             ts,
		Outgoing:              message.Outgoing,
		IsSnap:                message.IsSnap,
		ContentType:           message.ContentType,
		Saved:                 message.Saved,
		DisappearAfterSeconds: int64(message.DisappearAfter / time.Second),
		QuotedMessageID:       quotedMessageIDString(message.QuotedMessageID),
		Tombstone:             message.Tombstone,
	}
}

// quotedMessageIDString renders the numeric quoted-message reference; 0 means
// "not a reply" and becomes empty.
func quotedMessageIDString(id int64) string {
	if id <= 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}

func messageKindFromAPI(message snapapi.Message) string {
	if message.IsSnap {
		return "snap"
	}
	switch strings.ToUpper(strings.TrimSpace(message.ContentType)) {
	case "CHAT":
		return "chat"
	case "EXTERNAL_MEDIA":
		return "media"
	case "SNAP", "SNAP_NOT_VIEWABLE":
		return "snap"
	}
	if len(message.Media) > 0 {
		return "media"
	}
	return "text"
}

func parseSnapchatMessageID(raw string) (int64, bool) {
	raw = baseSnapchatMessageID(raw)
	raw = strings.TrimPrefix(raw, "client-")
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func scopedSnapchatMessageID(chatID, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, ".msg.") {
		return raw
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return raw
	}
	return chatID + ".msg." + raw
}

func baseSnapchatMessageID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, "-media-"); idx > 0 {
		raw = raw[:idx]
	}
	if idx := strings.Index(raw, "-snap-unavailable"); idx > 0 {
		raw = raw[:idx]
	}
	if idx := strings.Index(raw, "-edit-"); idx > 0 {
		raw = raw[:idx]
	}
	if idx := strings.LastIndex(raw, ".msg."); idx >= 0 {
		raw = raw[idx+len(".msg."):]
	}
	return raw
}

func snapchatConversationURL(chatID string) string {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return ""
	}
	return "https://www.snapchat.com/web/" + chatID
}
