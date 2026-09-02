package connector

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/networkid"
)

// remoteTypingTimeout matches Snapchat's own presence expiry cadence; the
// explicit stop event normally arrives first via the poll diff, and this
// timeout is the client-side safety net.
const remoteTypingTimeout = 8 * time.Second

// remoteTypingEvent forwards a remote participant's typing state to bridgev2,
// which renders it as a Matrix typing notification from the participant's
// ghost intent.
type remoteTypingEvent struct {
	portalKey networkid.PortalKey
	sender    bridgev2.EventSender
	typing    bool
}

func (e *remoteTypingEvent) GetType() bridgev2.RemoteEventType { return bridgev2.RemoteEventTyping }

func (e *remoteTypingEvent) GetPortalKey() networkid.PortalKey { return e.portalKey }

func (e *remoteTypingEvent) AddLogContext(c zerolog.Context) zerolog.Context {
	return c.Bool("typing", e.typing)
}

func (e *remoteTypingEvent) GetSender() bridgev2.EventSender { return e.sender }

func (e *remoteTypingEvent) GetTimeout() time.Duration {
	if e.typing {
		return remoteTypingTimeout
	}
	return 0
}

func (e *remoteTypingEvent) GetTypingType() bridgev2.TypingType { return bridgev2.TypingTypeText }

// typingChange is a single participant typing-state transition.
type typingChange struct {
	ChatID      string
	Participant string
	Typing      bool
}

// diffTypingState compares the previous per-chat typing sets against the new
// ones and returns the transitions. Known keys only; callers pass in maps they
// already filtered for self.
func diffTypingState(previous, current map[string]map[string]bool) []typingChange {
	var changes []typingChange
	seenChats := make(map[string]struct{}, len(previous)+len(current))
	for chatID, participants := range previous {
		seenChats[chatID] = struct{}{}
		for participant := range participants {
			if !current[chatID][participant] {
				changes = append(changes, typingChange{ChatID: chatID, Participant: participant, Typing: false})
			}
		}
	}
	for chatID, participants := range current {
		if _, ok := seenChats[chatID]; !ok {
			seenChats[chatID] = struct{}{}
		}
		for participant := range participants {
			if !previous[chatID][participant] {
				changes = append(changes, typingChange{ChatID: chatID, Participant: participant, Typing: true})
			}
		}
	}
	return changes
}

func (sa *SnapchatAPI) scheduleTypingStateSync(ctx context.Context) {
	if sa == nil || !sa.typingEnabled() {
		return
	}
	sa.mu.Lock()
	if sa.typingSyncRunning {
		sa.mu.Unlock()
		return
	}
	sa.typingSyncRunning = true
	sa.mu.Unlock()

	go func() {
		defer func() {
			sa.mu.Lock()
			sa.typingSyncRunning = false
			sa.mu.Unlock()
		}()
		syncCtx, cancel := context.WithTimeout(ctx, 3500*time.Millisecond)
		defer cancel()
		sa.syncTypingState(syncCtx)
	}()
}

// syncTypingState polls the connector's presence-derived typing state and
// emits bridge typing events for transitions. It is read-only, never touches
// snaps/media, and stays quiet on transient connector errors so the poll loop
// is not spammed.
func (sa *SnapchatAPI) syncTypingState(ctx context.Context) {
	if sa == nil || !sa.typingEnabled() || sa.Client == nil || sa.UserLogin == nil || sa.UserLogin.Bridge == nil {
		return
	}
	watchChatID := sa.currentTypingPresenceChat()
	resp, err := sa.Client.TypingState(ctx, watchChatID)
	if err != nil {
		return
	}
	if watchChatID != "" && len(resp.Conversations) > 0 {
		zerolog.Ctx(ctx).Debug().Str("diag", "remote_typing").Str("watch_chat_id", watchChatID).Int("conversation_count", len(resp.Conversations)).Msg("typing: connector returned presence state")
	}
	current := make(map[string]map[string]bool)
	for chatID, participants := range resp.Conversations {
		if chatID == "" {
			continue
		}
		if sa.lookupChat(chatID).ID == "" {
			// Only conversations the bridge already maps can be typed in.
			continue
		}
		for _, participant := range participants {
			if participant.UserID == "" || participant.State != "typing" {
				continue
			}
			if sa.isSelfAuthor(participant.UserID) {
				continue
			}
			if current[chatID] == nil {
				current[chatID] = make(map[string]bool)
			}
			current[chatID][participant.UserID] = true
		}
	}

	sa.mu.Lock()
	previous := sa.remoteTyping
	sa.remoteTyping = current
	sa.mu.Unlock()

	for _, change := range diffTypingState(previous, current) {
		zerolog.Ctx(ctx).Info().Str("diag", "remote_typing").Str("chat_id", change.ChatID).Str("participant", change.Participant).Bool("typing", change.Typing).Msg("typing: queue Snapchat -> Matrix transition")
		sa.queueRemoteTyping(change)
	}
}

// queueRemoteTyping queues a typing transition event for a chat participant.
func (sa *SnapchatAPI) queueRemoteTyping(change typingChange) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || change.ChatID == "" || change.Participant == "" {
		return
	}
	sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &remoteTypingEvent{
		portalKey: networkid.PortalKey{
			ID:       networkid.PortalID(change.ChatID),
			Receiver: sa.UserLogin.ID,
		},
		sender: bridgev2.EventSender{
			IsFromMe:    false,
			Sender:      makeUserID(change.Participant),
			ForceDMUser: true,
		},
		typing: change.Typing,
	})
}

var _ bridgev2.RemoteEvent = (*remoteTypingEvent)(nil)
var _ bridgev2.RemoteTyping = (*remoteTypingEvent)(nil)
var _ bridgev2.RemoteTypingWithType = (*remoteTypingEvent)(nil)
