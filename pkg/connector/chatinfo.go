package connector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/bridgev2/simplevent"
)

func (sa *SnapchatAPI) GetChatInfo(ctx context.Context, portal *bridgev2.Portal) (*bridgev2.ChatInfo, error) {
	chatID := string(portal.ID)
	name := sa.lookupChatName(chatID)
	if name == "" {
		name = chatID
	}
	info := sa.chatInfoFor(chatID, name)
	if info != nil {
		info.Avatar = sa.avatarForGhost(ctx, sa.remoteUserIDForChat(chatID, name))
	}
	return info, nil
}

func (sa *SnapchatAPI) GetUserInfo(ctx context.Context, ghost *bridgev2.Ghost) (*bridgev2.UserInfo, error) {
	name := sa.lookupGhostName(ghost.ID)
	if name == "" {
		name = sa.lookupGhostNameFromStore(ghost.ID)
	}
	if name == "" {
		name = string(ghost.ID)
	}
	identifier := fmt.Sprintf("snapchat:%s", ghost.ID)
	if username := sa.lookupGhostUsername(ghost.ID); username != "" {
		identifier = fmt.Sprintf("snapchat:%s", username)
		if name == string(ghost.ID) || snapUUIDPattern.MatchString(name) {
			name = username
		}
	}
	info := &bridgev2.UserInfo{
		Name:        &name,
		Avatar:      sa.avatarForGhost(ctx, ghost.ID),
		Identifiers: []string{identifier},
	}
	return info, nil
}

func (sa *SnapchatAPI) snapchatIdentifier(userID networkid.UserID) string {
	if username := sa.lookupGhostUsername(userID); username != "" {
		return fmt.Sprintf("snapchat:%s", username)
	}
	return fmt.Sprintf("snapchat:%s", userID)
}

func (sa *SnapchatAPI) lookupGhostNameFromStore(userID networkid.UserID) string {
	if sa.Connector == nil || sa.Connector.store == nil {
		return ""
	}
	id := strings.TrimSpace(string(userID))
	if id == "" {
		return ""
	}
	portals, err := sa.Connector.store.ListPortals()
	if err != nil {
		return ""
	}
	for _, portal := range portals {
		if strings.EqualFold(strings.TrimSpace(portal.OtherUserID), id) ||
			strings.EqualFold(strings.TrimSpace(portal.RemoteID), id) ||
			strings.EqualFold(strings.TrimSpace(portal.PortalKey), id) {
			name := strings.TrimSpace(portal.RemoteName)
			if name != "" && !strings.EqualFold(name, "unknown") {
				sa.rememberGhostName(userID, name)
				return name
			}
		}
	}
	return ""
}

func (sa *SnapchatAPI) remoteUserIDForChat(chatID, chatName string) networkid.UserID {
	chatID = strings.TrimSpace(chatID)
	otherUserID := ""
	sa.mu.Lock()
	if ref, ok := sa.chatsByID[chatID]; ok {
		otherUserID = strings.TrimSpace(ref.OtherUserID)
	}
	sa.mu.Unlock()
	if otherUserID != "" {
		log.Printf("bridgev2 map: remote user for chat_id=%s resolved from participant user_id=%s", chatID, otherUserID)
		return makeUserID(otherUserID)
	}
	if sa.Connector != nil && sa.Connector.store != nil && chatID != "" {
		if portal, err := sa.Connector.store.GetPortalByRemoteID(chatID); err == nil && portal != nil {
			otherUserID = strings.TrimSpace(portal.OtherUserID)
			if otherUserID != "" {
				log.Printf("bridgev2 map: remote user for chat_id=%s resolved from stored participant user_id=%s", chatID, otherUserID)
				return makeUserID(otherUserID)
			}
		}
	}
	if chatID != "" {
		chatName = strings.TrimSpace(chatName)
		if chatName != "" {
			log.Printf("bridgev2 map: remote user for chat_id=%s missing participant id; using sanitized display name fallback=%q", chatID, chatName)
			return makeUserID(chatName)
		}
		log.Printf("bridgev2 map: remote user for chat_id=%s missing participant id and display name; using conversation id fallback", chatID)
		return makeUserID(chatID)
	}
	chatName = strings.TrimSpace(chatName)
	log.Printf("bridgev2 map: remote user fallback has no chat id; using sanitized display name=%q", chatName)
	return makeUserID(chatName)
}

func (sa *SnapchatAPI) chatInfoFor(chatID, chatName string) *bridgev2.ChatInfo {
	return sa.chatInfoForWithAvatar(context.Background(), chatID, chatName, false)
}

func (sa *SnapchatAPI) chatInfoForWithAvatar(ctx context.Context, chatID, chatName string, fetchAvatar bool) *bridgev2.ChatInfo {
	name := strings.TrimSpace(chatName)
	if name == "" {
		name = "Snapchat"
	}
	ref := sa.lookupChatRef(chatID)
	ownUserID := makeUserID(sa.Label)
	roomType := database.RoomTypeDM
	if ref.IsGroup || len(ref.ParticipantIDs) > 1 {
		roomType = database.RoomTypeGroupDM
	}
	var avatar *bridgev2.Avatar
	memberMap := bridgev2.ChatMemberMap{
		ownUserID: {
			EventSender: bridgev2.EventSender{
				IsFromMe:    true,
				Sender:      ownUserID,
				SenderLogin: makeUserLoginID(sa.Label),
			},
		},
	}
	memberCount := 1
	otherUserID := networkid.UserID("")
	if roomType == database.RoomTypeDM {
		remoteUserID := sa.remoteUserIDForChat(chatID, name)
		otherUserID = remoteUserID
		sa.rememberGhostName(remoteUserID, name)
		if ref.Username != "" {
			sa.rememberGhostUsername(remoteUserID, ref.Username)
		}
		avatar = sa.cachedAvatarForGhost(remoteUserID)
		if avatar == nil && fetchAvatar {
			avatar = sa.avatarForGhost(ctx, remoteUserID)
		}
		remoteUserInfo := &bridgev2.UserInfo{
			Name:   &name,
			Avatar: avatar,
			Identifiers: []string{
				sa.snapchatIdentifier(remoteUserID),
			},
		}
		memberMap[remoteUserID] = bridgev2.ChatMember{
			EventSender: bridgev2.EventSender{
				Sender: remoteUserID,
			},
			UserInfo: remoteUserInfo,
		}
		memberCount = 2
	} else {
		for _, participantID := range ref.ParticipantIDs {
			participantID = strings.TrimSpace(participantID)
			if participantID == "" {
				continue
			}
			userID := makeUserID(participantID)
			display := sa.lookupGhostName(userID)
			if display == "" {
				display = sa.lookupGhostNameFromStore(userID)
			}
			if display == "" {
				display = string(userID)
			}
			memberMap[userID] = bridgev2.ChatMember{
				EventSender: bridgev2.EventSender{
					Sender: userID,
				},
				UserInfo: &bridgev2.UserInfo{
					Name:        &display,
					Avatar:      sa.cachedAvatarForGhost(userID),
					Identifiers: []string{sa.snapchatIdentifier(userID)},
				},
			}
		}
		memberCount = len(memberMap)
	}
	info := &bridgev2.ChatInfo{
		Name:   &name,
		Avatar: avatar,
		Type:   &roomType,
		Members: &bridgev2.ChatMemberList{
			IsFull:           true,
			TotalMemberCount: memberCount,
			OtherUserID:      otherUserID,
			MemberMap:        memberMap,
		},
	}
	if disappear := sa.disappearingSettingForChat(chatID); disappear != nil {
		info.Disappear = disappear
	}
	log.Printf("bridgev2 map: chat info chat_id=%s name=%q room_type=%s other_user_id=%s members=%d avatar=%t disappear=%t", chatID, name, roomType, otherUserID, memberCount, avatar != nil, info.Disappear != nil)
	return info
}

func (sa *SnapchatAPI) queueChatResync(ctx context.Context, chatID, chatName string) bool {
	return sa.queueChatResyncWithAvatar(ctx, chatID, chatName, true)
}

func (sa *SnapchatAPI) queueChatResyncWithAvatar(ctx context.Context, chatID, chatName string, fetchAvatar bool) bool {
	return sa.queueChatResyncWithInfo(chatID, chatName, fetchAvatar, sa.chatInfoForWithAvatar(ctx, chatID, chatName, fetchAvatar))
}

func (sa *SnapchatAPI) queueChatResyncWithInfo(chatID, chatName string, force bool, info *bridgev2.ChatInfo) bool {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chatID == "" {
		return false
	}
	name := strings.TrimSpace(chatName)
	if name == "" {
		name = chatID
	}
	sa.mu.Lock()
	if _, queued := sa.queuedPortalResyncs[chatID]; queued {
		if !force {
			sa.mu.Unlock()
			return false
		}
	}
	sa.queuedPortalResyncs[chatID] = struct{}{}
	sa.mu.Unlock()

	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(chatID),
		Receiver: sa.UserLogin.ID,
	}
	log.Printf("bridgev2 sync: queue portal create chat=%q id=%s", name, chatID)
	sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.ChatResync{
		EventMeta: simplevent.EventMeta{
			Type: bridgev2.RemoteEventChatResync,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("chat", name).Str("chat_id", chatID)
			},
			PortalKey:    portalKey,
			CreatePortal: true,
		},
		ChatInfo: info,
	})
	log.Printf("bridgev2 map: queued chat resync chat_id=%s name=%q force=%t existing_match=portal_key:%s", chatID, name, force, portalKey.ID)
	return true
}

func (sa *SnapchatAPI) queueStoredPortalResyncs(ctx context.Context) {
	if sa.Connector == nil || sa.Connector.store == nil {
		return
	}
	portals, err := sa.Connector.store.ListPortals()
	if err != nil {
		log.Printf("bridgev2 sync: failed to load stored portals for resync: %v", err)
		return
	}
	queued := 0
	for _, portal := range portals {
		chatID := portal.RemoteID
		if chatID == "" {
			chatID = portal.PortalKey
		}
		chatName := portal.RemoteName
		if chatName == "" {
			chatName = chatID
		}
		sa.rememberChatDetails(sidecar.Chat{
			ID:             chatID,
			Name:           chatName,
			OtherUserID:    portal.OtherUserID,
			ParticipantIDs: append([]string{}, portal.ParticipantIDs...),
			IsGroup:        portal.RoomType == "group_dm",
			Username:       portal.Username,
		})
		if sa.queueChatResyncWithAvatar(ctx, chatID, chatName, false) {
			queued++
		}
	}
	log.Printf("bridgev2 sync: queued %d stored portal creates for label=%s", queued, sa.Label)
}

func (sa *SnapchatAPI) primeStoredPortalAvatars(ctx context.Context) {
	if sa.Connector == nil || sa.Connector.store == nil || sa.UserLogin == nil {
		return
	}
	sa.mu.Lock()
	if sa.avatarBootstrapDone {
		sa.mu.Unlock()
		return
	}
	sa.avatarBootstrapDone = true
	sa.mu.Unlock()

	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		log.Printf("bridgev2 avatar: stored portal avatar bootstrap skipped, API client unavailable: %v", err)
		return
	}
	portals, err := sa.Connector.store.ListPortals()
	if err != nil {
		log.Printf("bridgev2 avatar: failed to load stored portals for avatar bootstrap: %v", err)
		return
	}

	type portalRef struct {
		ChatID      string
		ChatName    string
		OtherUserID string
	}
	refs := make([]portalRef, 0, len(portals))
	ids := make([]string, 0, len(portals))
	seenIDs := make(map[string]struct{}, len(portals))
	for _, portal := range portals {
		otherUserID := strings.ToLower(strings.TrimSpace(portal.OtherUserID))
		if otherUserID == "" || !snapUUIDPattern.MatchString(otherUserID) {
			continue
		}
		chatID := strings.TrimSpace(portal.RemoteID)
		if chatID == "" {
			chatID = strings.TrimSpace(portal.PortalKey)
		}
		if chatID == "" {
			continue
		}
		chatName := strings.TrimSpace(portal.RemoteName)
		if chatName == "" {
			chatName = chatID
		}
		refs = append(refs, portalRef{
			ChatID:      chatID,
			ChatName:    chatName,
			OtherUserID: otherUserID,
		})
		if _, ok := seenIDs[otherUserID]; ok {
			continue
		}
		seenIDs[otherUserID] = struct{}{}
		ids = append(ids, otherUserID)
	}
	if len(ids) == 0 {
		log.Printf("bridgev2 avatar: stored portal avatar bootstrap found no Snapchat user IDs")
		return
	}

	profiles, err := client.GetPublicProfiles(ctx, ids)
	if err != nil {
		log.Printf("bridgev2 avatar: stored portal profile batch failed for %d users: %v", len(ids), err)
		return
	}
	for _, id := range ids {
		sa.markGhostAvatarChecked(id)
	}

	withAvatar := 0
	queued := 0
	for _, ref := range refs {
		if ctx.Err() != nil {
			break
		}
		profile, ok := profiles[ref.OtherUserID]
		if !ok || strings.TrimSpace(profile.AvatarURL) == "" {
			continue
		}
		withAvatar++
		sa.rememberSnapAvatarURL(ref.OtherUserID, profile.AvatarURL)
		sa.rememberChat(ref.ChatID, "", ref.ChatName, ref.OtherUserID)
		sa.rememberGhostName(makeUserID(ref.OtherUserID), ref.ChatName)
		info := sa.chatInfoForWithAvatar(ctx, ref.ChatID, ref.ChatName, false)
		avatar := sa.avatarFromURL(ref.OtherUserID, profile.AvatarURL)
		if info != nil && avatar != nil {
			info.Avatar = avatar
			remoteUserID := sa.remoteUserIDForChat(ref.ChatID, ref.ChatName)
			if info.Members != nil && info.Members.MemberMap != nil {
				member := info.Members.MemberMap[remoteUserID]
				if member.UserInfo == nil {
					name := ref.ChatName
					member.UserInfo = &bridgev2.UserInfo{Name: &name}
				}
				member.UserInfo.Avatar = avatar
				info.Members.MemberMap[remoteUserID] = member
			}
		}
		if sa.queueChatResyncWithInfo(ref.ChatID, ref.ChatName, true, info) {
			queued++
			time.Sleep(250 * time.Millisecond)
		}
	}
	log.Printf("bridgev2 avatar: primed stored portal avatars users=%d with_avatar=%d queued_resyncs=%d label=%s",
		len(ids), withAvatar, queued, sa.Label)
}

func (sa *SnapchatAPI) syncStoredPortalMessages(ctx context.Context) {
	if sa.Connector == nil || sa.Connector.store == nil || sa.UserLogin == nil || sa.UserLogin.Bridge == nil {
		return
	}
	portals, err := sa.Connector.store.ListPortals()
	if err != nil {
		log.Printf("bridgev2 sync: failed to load stored portals for startup message backfill: %v", err)
		return
	}
	maxStartupBackfillChats := sa.messageSyncLimitPerPoll(true)
	synced := 0
	for _, portal := range portals {
		if synced >= maxStartupBackfillChats || ctx.Err() != nil {
			break
		}
		chatID := portal.RemoteID
		if chatID == "" {
			chatID = portal.PortalKey
		}
		if chatID == "" {
			continue
		}
		chatName := portal.RemoteName
		if chatName == "" {
			chatName = chatID
		}
		chat := sidecar.Chat{
			ID:             chatID,
			URL:            snapchatConversationURL(chatID),
			Name:           chatName,
			Preview:        portal.Preview,
			LastMessage:    portal.LastMessage,
			Unread:         portal.Unread,
			OtherUserID:    portal.OtherUserID,
			ParticipantIDs: append([]string{}, portal.ParticipantIDs...),
			IsGroup:        portal.RoomType == string(database.RoomTypeGroupDM) || len(portal.ParticipantIDs) > 1,
			Username:       portal.Username,
		}
		if err := sa.syncChatMessagesAPI(ctx, chat, 0, "startup stored portal backfill", false, false); err != nil {
			log.Printf("bridgev2 sync: startup message backfill failed chat=%q id=%s: %v", chat.Name, chat.ID, err)
			continue
		}
		synced++
	}
	log.Printf("bridgev2 sync: startup message backfilled %d stored portals for label=%s", synced, sa.Label)
}

func (sa *SnapchatAPI) restoreChatMappings() {
	if sa.Connector == nil || sa.Connector.store == nil {
		return
	}

	portals, err := sa.Connector.store.ListPortals()
	if err != nil {
		return
	}
	for _, portal := range portals {
		if portal.RemoteID == "" || portal.RemoteName == "" {
			continue
		}
		sa.rememberChatDetails(sidecar.Chat{
			ID:             portal.RemoteID,
			Name:           portal.RemoteName,
			OtherUserID:    portal.OtherUserID,
			ParticipantIDs: append([]string{}, portal.ParticipantIDs...),
			IsGroup:        portal.RoomType == "group_dm",
			Username:       portal.Username,
		})
		sa.rememberChatState(portal.RemoteID, portal.Unread, portal.Preview, portal.LastMessage)
	}
}
