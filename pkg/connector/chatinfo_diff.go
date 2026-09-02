package connector

import (
	"context"
	"log"
	"sort"
	"strings"

	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
)

// chatInfoState is a comparable snapshot of the effective portal metadata a
// ChatResync would apply. Only fields bridgev2 would turn into Matrix room
// state or portal bookkeeping participate: display name, avatar, room type,
// DM participant identity, full member identity set (participant IDs plus
// ghost display names), disappearing timer, and capability version.
type chatInfoState struct {
	Name           string
	AvatarID       string
	RoomType       database.RoomType
	OtherUser      string
	Participants   string
	HasDisappear   bool
	DisappearType  string
	DisappearTimer int64
	CapID          string
}

// disappearAbsent is the sentinel timer for chats with no disappearing
// setting so a nil Disappear and a zero timer are never conflated.
const disappearAbsent = int64(-1)

// effectiveChatInfoState captures the metadata the connector is about to send
// in a ChatResync event. A nil ChatInfo field means "do not touch" in
// bridgev2, so it is compared as don't-care here too.
func (sa *SnapchatAPI) effectiveChatInfoState(info *bridgev2.ChatInfo) chatInfoState {
	state := chatInfoState{DisappearTimer: disappearAbsent}
	if info == nil {
		return state
	}
	if info.Name != nil {
		state.Name = strings.TrimSpace(*info.Name)
	}
	if info.Avatar != nil {
		state.AvatarID = string(info.Avatar.ID)
	}
	if info.Type != nil {
		state.RoomType = *info.Type
	}
	if info.Members != nil {
		state.OtherUser = string(info.Members.OtherUserID)
		if info.Members.MemberMap != nil {
			entries := make([]string, 0, len(info.Members.MemberMap))
			for userID, member := range info.Members.MemberMap {
				display := ""
				if member.UserInfo != nil && member.UserInfo.Name != nil {
					display = strings.TrimSpace(*member.UserInfo.Name)
				}
				entries = append(entries, string(userID)+"\x1f"+display)
			}
			sort.Strings(entries)
			state.Participants = strings.Join(entries, "\x1e")
		}
	}
	if info.Disappear != nil {
		state.HasDisappear = true
		state.DisappearType = string(info.Disappear.Type)
		state.DisappearTimer = int64(info.Disappear.Timer)
	}
	if sa.Connector != nil {
		state.CapID = sa.Connector.chatCapabilities().ID
	}
	return state
}

// storedPortalChatInfoState captures the metadata bridgev2 last applied to
// the portal, from the authoritative mautrix portal row.
func storedPortalChatInfoState(portal *database.Portal) chatInfoState {
	state := chatInfoState{DisappearTimer: disappearAbsent}
	if portal == nil {
		return state
	}
	state.Name = strings.TrimSpace(portal.Name)
	state.AvatarID = string(portal.AvatarID)
	state.RoomType = portal.RoomType
	state.OtherUser = string(portal.OtherUserID)
	if portal.Disappear.Type != "" || portal.Disappear.Timer != 0 {
		state.HasDisappear = true
		state.DisappearType = string(portal.Disappear.Type)
		state.DisappearTimer = int64(portal.Disappear.Timer)
	}
	state.CapID = portal.CapState.ID
	return state
}

// chatInfoStateMatchesPortal reports whether the ChatInfo the connector wants
// to apply is already reflected in the stored portal row. Fields the ChatInfo
// leaves nil are ignored, mirroring bridgev2's UpdateInfo semantics.
func (state chatInfoState) matchesPortal(portal *database.Portal) bool {
	stored := storedPortalChatInfoState(portal)
	if state.Name != stored.Name && state.Name != "" {
		return false
	}
	if state.AvatarID != stored.AvatarID && state.AvatarID != "" && portal.AvatarSet {
		return false
	}
	if state.RoomType != "" && state.RoomType != stored.RoomType {
		return false
	}
	if state.OtherUser != "" && state.OtherUser != stored.OtherUser {
		return false
	}
	if state.HasDisappear != stored.HasDisappear {
		return false
	}
	if state.HasDisappear && (state.DisappearType != stored.DisappearType || state.DisappearTimer != stored.DisappearTimer) {
		return false
	}
	if state.CapID != "" && stored.CapID != "" && state.CapID != stored.CapID {
		return false
	}
	return true
}

// chatResyncUnchanged reports whether a ChatResync for this chat can be
// skipped because the effective ChatInfo is identical to what was last queued
// and is already reflected in the stored portal row. Message syncing, read
// receipts, and typing are independent code paths and are never gated here.
func (sa *SnapchatAPI) chatResyncUnchanged(ctx context.Context, chatID string, info *bridgev2.ChatInfo) bool {
	if info == nil {
		return false
	}
	current := sa.effectiveChatInfoState(info)
	sa.mu.Lock()
	recorded, recordedOK := sa.appliedChatInfoState[chatID]
	sa.mu.Unlock()
	if !recordedOK || recorded != current {
		return false
	}
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil {
		return false
	}
	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(chatID),
		Receiver: sa.UserLogin.ID,
	}
	portal, err := sa.UserLogin.Bridge.DB.Portal.GetByKey(ctx, portalKey)
	if err != nil || portal == nil || portal.MXID == "" {
		return false
	}
	return current.matchesPortal(portal)
}

// noteChatResyncSkipped records the skipped state and logs the first skip per
// distinct state so steady-state polls do not spam the log.
func (sa *SnapchatAPI) noteChatResyncSkipped(chatID, chatName string, info *bridgev2.ChatInfo) {
	state := sa.effectiveChatInfoState(info)
	sa.mu.Lock()
	alreadyLogged := sa.lastSkippedChatInfoState[chatID] == state
	sa.lastSkippedChatInfoState[chatID] = state
	sa.mu.Unlock()
	if !alreadyLogged {
		log.Printf("bridgev2 sync: skipping portal resync, metadata unchanged chat=%q id=%s", chatName, chatID)
	}
}

// recordQueuedChatResync stores the state that was queued so later identical
// ChatInfos can be skipped until something actually changes.
func (sa *SnapchatAPI) recordQueuedChatResync(chatID string, info *bridgev2.ChatInfo) {
	if info == nil {
		return
	}
	state := sa.effectiveChatInfoState(info)
	sa.mu.Lock()
	sa.appliedChatInfoState[chatID] = state
	delete(sa.lastSkippedChatInfoState, chatID)
	sa.mu.Unlock()
}
