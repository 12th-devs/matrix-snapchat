package connector

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/snapapi"
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/matrix"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/bridgev2/simplevent"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// ResyncStats accumulates the counters reported by one resync-all run.
type ResyncStats struct {
	ChatsScanned            int
	HealthyPortalsPreserved int
	PortalsCreated          int
	DeadPortalsReplaced     int
	PFSLinksRepaired        int
	MembershipsRepaired     int

	MessagesFetched       int
	MessagesAlreadyMapped int
	MessagesBridged       int
	MessagesBackfilled    int

	MediaCandidates int
	MediaHydrated   int
	MediaSkipped    int

	Errors   []string
	Duration time.Duration
}

// ResyncAll runs resync-all for every Snapchat login of the given Matrix user.
// progress receives short human-readable lines suitable for command replies.
// A nonempty chatID scopes the operation to that chat without falling back to all.
func (sc *SnapchatConnector) ResyncAll(ctx context.Context, user *bridgev2.User, progress func(string), chatID string) ([]*ResyncStats, error) {
	if sc == nil || user == nil {
		return nil, fmt.Errorf("connector or user not initialized")
	}
	logins := user.GetUserLogins()
	if len(logins) == 0 {
		return nil, fmt.Errorf("user has no logins")
	}
	results := make([]*ResyncStats, 0, len(logins))
	ran := 0
	for _, login := range logins {
		api, ok := login.Client.(*SnapchatAPI)
		if !ok || api == nil {
			continue
		}
		ran++
		results = append(results, api.RunResyncAll(ctx, progress, chatID))
	}
	if ran == 0 {
		return nil, fmt.Errorf("no Snapchat logins available")
	}
	return results, nil
}

// RunResyncAll reconciles every current Snapchat conversation for this login.
// It serializes against the normal poll loop via resyncMu and never persists
// the discovery Sync state: the poll loop stays the sole owner of the cursor.
func (sa *SnapchatAPI) RunResyncAll(ctx context.Context, progress func(string), chatID string) *ResyncStats {
	started := time.Now()
	stats := &ResyncStats{}
	defer func() { stats.Duration = time.Since(started) }()
	if progress == nil {
		progress = func(string) {}
	}

	sa.resyncMu.Lock()
	defer sa.resyncMu.Unlock()
	if _, err := sa.existingSpaceRoom(ctx); err != nil {
		stats.Errors = append(stats.Errors, err.Error())
		return stats
	}

	client, err := sa.ensureSnapClient(ctx)
	if err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("snapchat client: %v", err))
		return stats
	}
	// Discovery only: pass a throwaway state so the persistent sync cursor
	// owned by the normal poll loop is untouched.
	result, err := client.Sync(ctx, snapapi.State{}, sa.conversationFetchLimit())
	if err != nil {
		sa.invalidateSnapClient()
		stats.Errors = append(stats.Errors, fmt.Sprintf("discover conversations: %v", err))
		return stats
	}
	sa.recordSyncSuccess(time.Now())

	chats, err := sa.resyncCoverage(ctx, result.Chats)
	if err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("existing portal coverage: %v", err))
		return stats
	}
	progress(fmt.Sprintf("Chats discovered: %d; union with existing portals: %d", len(result.Chats), len(chats)))
	zerolog.Ctx(ctx).Info().Int("discovered", len(result.Chats)).Int("coverage", len(chats)).Msg("Resync coverage")
	for i, chat := range chats {
		if ctx.Err() != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("cancelled after %d chats", i))
			break
		}
		if chatID != "" && chat.ID != chatID {
			continue
		}
		stats.ChatsScanned++
		sa.rememberChatDetails(chat)
		sa.rememberChatDisappear(chat.ID, chat.DisappearAfterSeconds)
		sa.persistChat(chat)
		if err := sa.resyncOneChat(ctx, chat, stats); err != nil {
			log.Printf("bridgev2 resync: chat reconciliation failed chat_id=%s name=%q: %v", chat.ID, chat.Name, err)
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", chat.ID, err))
		}
		if (i+1)%10 == 0 || i+1 == len(chats) {
			progress(fmt.Sprintf("[%d/%d]", i+1, len(chats)))
		}
	}
	if chatID != "" && stats.ChatsScanned == 0 {
		stats.Errors = append(stats.Errors, fmt.Sprintf("target chat %s not found in discovery or existing login portals; no portals changed", chatID))
	}
	stats.Duration = time.Since(started)
	log.Printf("bridgev2 resync: completed label=%s scanned=%d created=%d replaced=%d repaired_pfs=%d repaired_members=%d fetched=%d bridged=%d backfilled=%d media=%d/%d/%d errors=%d duration=%s",
		sa.Label, stats.ChatsScanned, stats.PortalsCreated, stats.DeadPortalsReplaced, stats.PFSLinksRepaired, stats.MembershipsRepaired,
		stats.MessagesFetched, stats.MessagesBridged, stats.MessagesBackfilled, stats.MediaHydrated, stats.MediaCandidates, stats.MediaSkipped,
		len(stats.Errors), stats.Duration)
	return stats
}

func (sa *SnapchatAPI) resyncCoverage(ctx context.Context, discovered []snapapi.Chat) ([]sidecar.Chat, error) {
	portals, err := sa.UserLogin.Bridge.DB.Portal.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	chats := make([]sidecar.Chat, 0, len(discovered)+len(portals))
	seen := make(map[string]bool)
	for _, apiChat := range discovered {
		chat := connectorChatFromAPI(apiChat)
		if chat.ID != "" && !seen[chat.ID] {
			seen[chat.ID] = true
			chats = append(chats, chat)
		}
	}
	for _, portal := range portals {
		chatID := string(portal.ID)
		if portal.Receiver != sa.UserLogin.ID || chatID == "" || seen[chatID] {
			continue
		}
		chat := sidecar.Chat{ID: chatID, Name: portal.Name, OtherUserID: string(portal.OtherUserID)}
		if sa.Connector != nil && sa.Connector.store != nil {
			stored, err := sa.Connector.store.GetPortalByRemoteID(chatID)
			if err != nil {
				return nil, err
			}
			if stored != nil {
				chat.Name, chat.OtherUserID = stored.RemoteName, stored.OtherUserID
				chat.ParticipantIDs, chat.Username = stored.ParticipantIDs, stored.Username
				chat.IsGroup = stored.RoomType == "group_dm"
			}
		}
		seen[chatID] = true
		chats = append(chats, chat)
	}
	return chats, nil
}

// Summary renders the final resync report for a command reply.
func (s *ResyncStats) Summary() string {
	if s == nil {
		return "resync-all produced no stats"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "chats scanned: %d\n", s.ChatsScanned)
	fmt.Fprintf(&b, "healthy portals preserved: %d\n", s.HealthyPortalsPreserved)
	fmt.Fprintf(&b, "portals created: %d\n", s.PortalsCreated)
	fmt.Fprintf(&b, "dead portals replaced: %d\n", s.DeadPortalsReplaced)
	fmt.Fprintf(&b, "PFS links repaired: %d\n", s.PFSLinksRepaired)
	fmt.Fprintf(&b, "memberships repaired: %d\n", s.MembershipsRepaired)
	fmt.Fprintf(&b, "messages fetched: %d\n", s.MessagesFetched)
	fmt.Fprintf(&b, "messages already mapped: %d\n", s.MessagesAlreadyMapped)
	fmt.Fprintf(&b, "messages newly bridged: %d\n", s.MessagesBridged)
	fmt.Fprintf(&b, "messages backfilled after room replacement: %d\n", s.MessagesBackfilled)
	fmt.Fprintf(&b, "media candidates: %d\n", s.MediaCandidates)
	fmt.Fprintf(&b, "media hydrated: %d\n", s.MediaHydrated)
	fmt.Fprintf(&b, "media skipped as delivered: %d\n", s.MediaSkipped)
	fmt.Fprintf(&b, "errors: %d\n", len(s.Errors))
	for _, e := range s.Errors {
		fmt.Fprintf(&b, "  - %s\n", e)
	}
	fmt.Fprintf(&b, "duration: %s", s.Duration.Round(time.Millisecond))
	return b.String()
}

func (sa *SnapchatAPI) resyncOneChat(ctx context.Context, chat sidecar.Chat, stats *ResyncStats) error {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" {
		return nil
	}
	br := sa.UserLogin.Bridge
	key := networkid.PortalKey{ID: networkid.PortalID(chat.ID), Receiver: sa.UserLogin.ID}
	portal, err := br.GetExistingPortalByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("load portal: %w", err)
	}

	dead := false
	if portal != nil && portal.MXID != "" {
		// Drain older polling events before examining mappings or changing MXID.
		done := make(chan struct{})
		result := sa.UserLogin.QueueRemoteEvent(&simplevent.EventMeta{
			PortalKey:      key,
			PostHandleFunc: func(context.Context, *bridgev2.Portal) { close(done) },
		})
		if err := waitRemoteMessage(result, done, []context.Context{ctx}); err != nil {
			return err
		}
		dead, err = sa.classifyRoomDead(ctx, portal)
		if err != nil {
			return fmt.Errorf("preserving room %s: %w", portal.MXID, err)
		}
	}
	switch {
	case portal == nil || portal.MXID == "":
		// No usable room for this portal key: create one through the normal
		// bridgev2 creation path for the same portal row.
		if portal == nil {
			portal, err = br.GetPortalByKey(ctx, key)
			if err != nil {
				return fmt.Errorf("get portal row: %w", err)
			}
		}
		if _, err := sa.existingSpaceRoom(ctx); err != nil {
			return err
		}
		if err := portal.CreateMatrixRoom(ctx, sa.UserLogin, nil); err != nil {
			return fmt.Errorf("create portal room: %w", err)
		}
		stats.PortalsCreated++
		log.Printf("bridgev2 resync: portal created chat_id=%s mxid=%s", chat.ID, portal.MXID)
	case dead:
		oldMXID := portal.MXID
		if err := sa.replaceDeadPortalRoom(ctx, chat.ID, portal, stats); err != nil {
			return fmt.Errorf("replace dead portal room: %w", err)
		}
		log.Printf("bridgev2 resync: DEAD_PORTAL_ROOM replaced chat_id=%s old_mxid=%s new_mxid=%s reason=dead/inaccessible room", chat.ID, oldMXID, portal.MXID)
	default:
		stats.HealthyPortalsPreserved++
	}
	if repaired, err := sa.ensureSpaceChild(ctx, portal); err != nil {
		return fmt.Errorf("repair space link: %w", err)
	} else if repaired {
		stats.PFSLinksRepaired++
	}
	repaired, err := sa.repairPortalMemberships(ctx, portal)
	stats.MembershipsRepaired += repaired
	if err != nil {
		return fmt.Errorf("repair memberships: %w", err)
	}
	sa.queueChatResync(ctx, chat.ID, chat.Name)
	err = sa.syncChatMessagesAPI(snapapi.WithEELThrottleBypass(ctx), chat, 0, "resync-all", true, false, stats)
	if err != nil {
		return fmt.Errorf("message resync: %w", err)
	}
	return nil
}

// replaceDeadPortalRoom swaps an unusable Matrix room for a fresh one created
// through the normal bridgev2 creation path while keeping the same portal row
// and remote chat key.
func (sa *SnapchatAPI) replaceDeadPortalRoom(ctx context.Context, chatID string, portal *bridgev2.Portal, stats *ResyncStats) error {
	oldMXID := portal.MXID
	spaceRoom, err := sa.existingSpaceRoom(ctx)
	if err != nil {
		return err
	}
	if err := portal.RemoveMXID(ctx); err != nil {
		return fmt.Errorf("clear dead mxid on portal row: %w", err)
	}
	if err := portal.CreateMatrixRoom(ctx, sa.UserLogin, nil); err != nil {
		return fmt.Errorf("create replacement room: %w", err)
	}
	if portal.MXID == "" || portal.MXID == oldMXID {
		return fmt.Errorf("replacement did not create a distinct room (old=%s new=%s)", oldMXID, portal.MXID)
	}
	stats.DeadPortalsReplaced++
	if repaired, err := sa.ensureSpaceChild(ctx, portal); err != nil {
		return fmt.Errorf("link replacement (old link retained): %w", err)
	} else if repaired {
		stats.PFSLinksRepaired++
	}
	if err := sa.removeSpaceChild(ctx, spaceRoom, oldMXID); err != nil {
		return fmt.Errorf("remove old space child %s: %w", oldMXID, err)
	}
	log.Printf("bridgev2 resync: replacement room ready chat_id=%s old_mxid=%s new_mxid=%s", chatID, oldMXID, portal.MXID)
	return nil
}

// removeSpaceChild clears a stale m.space.child entry under the PFS.
func (sa *SnapchatAPI) removeSpaceChild(ctx context.Context, spaceRoom id.RoomID, oldMXID id.RoomID) error {
	_, err := sa.UserLogin.Bridge.Bot.SendState(ctx, spaceRoom, event.StateSpaceChild, oldMXID.String(), &event.Content{
		Parsed: &event.SpaceChildEventContent{},
	}, time.Now())
	return err
}

// ensureSpaceChild verifies the portal is linked under the login's personal
// filtering space. Never call GetSpaceRoom: it can create a new PFS.
func (sa *SnapchatAPI) ensureSpaceChild(ctx context.Context, portal *bridgev2.Portal) (bool, error) {
	if portal == nil || portal.MXID == "" {
		return false, nil
	}
	spaceRoom, err := sa.existingSpaceRoom(ctx)
	if err != nil {
		return false, fmt.Errorf("get space room: %w", err)
	}
	linked, err := sa.spaceChildLinked(ctx, spaceRoom, portal.MXID)
	if err != nil {
		return false, fmt.Errorf("probe space child: %w", err)
	}
	up, err := sa.UserLogin.Bridge.DB.UserPortal.GetOrCreate(ctx, sa.UserLogin.UserLogin, portal.PortalKey)
	if err != nil {
		return false, fmt.Errorf("get user portal: %w", err)
	}
	if !linked {
		_, err = sa.UserLogin.Bridge.Bot.SendState(ctx, spaceRoom, event.StateSpaceChild, portal.MXID.String(), &event.Content{
			Parsed: &event.SpaceChildEventContent{Via: []string{sa.UserLogin.Bridge.Matrix.ServerName()}},
		}, time.Now())
		if err != nil {
			return false, fmt.Errorf("add portal to space: %w", err)
		}
	}
	if up.InSpace == nil || !*up.InSpace {
		inSpace := true
		up.InSpace = &inSpace
		if err := sa.UserLogin.Bridge.DB.UserPortal.Put(ctx, up); err != nil {
			return !linked, fmt.Errorf("save space link: %w", err)
		}
	}
	return !linked, nil
}

func (sa *SnapchatAPI) spaceChildLinked(ctx context.Context, spaceRoom id.RoomID, portalMXID id.RoomID) (bool, error) {
	client, err := sa.liveBotClient()
	if err != nil {
		return false, err
	}
	var child event.SpaceChildEventContent
	err = client.StateEvent(ctx, spaceRoom, event.StateSpaceChild, portalMXID.String(), &child)
	if err == nil {
		return len(child.Via) > 0, nil
	}
	if conclusiveMatrixMiss(err) && errors.Is(err, mautrix.MNotFound) {
		return false, nil
	}
	return false, err
}

func (sa *SnapchatAPI) liveBotClient() (*mautrix.Client, error) {
	if sa.UserLogin != nil && sa.UserLogin.Bridge != nil {
		if conn, ok := sa.UserLogin.Bridge.Matrix.(*matrix.Connector); ok && conn.Bot != nil && conn.Bot.Client != nil {
			return conn.Bot.Client, nil
		}
	}
	return nil, fmt.Errorf("live Matrix bot client unavailable")
}

func (sa *SnapchatAPI) existingSpaceRoom(ctx context.Context) (id.RoomID, error) {
	client, err := sa.liveBotClient()
	if err != nil {
		return "", err
	}
	space := sa.UserLogin.SpaceRoom
	if space == "" {
		return "", fmt.Errorf("existing personal filtering space required; refusing to create one")
	}
	if _, err := client.Members(ctx, space); err != nil {
		return "", fmt.Errorf("existing space %s failed live validation: %w", space, err)
	}
	return space, nil
}

// Use Client methods, not cached bridgev2 GetMembers or EnsureJoined.
type liveRoomProber interface {
	JoinRoomByID(ctx context.Context, roomID id.RoomID) (*mautrix.RespJoinRoom, error)
	Members(ctx context.Context, roomID id.RoomID, req ...mautrix.ReqMembers) (*mautrix.RespMembers, error)
}

func (sa *SnapchatAPI) portalRoomClients(ctx context.Context, portal *bridgev2.Portal) ([]*mautrix.Client, error) {
	bot, err := sa.liveBotClient()
	if err != nil {
		return nil, err
	}
	clients := []*mautrix.Client{bot}
	seen := map[id.UserID]bool{bot.UserID: true}
	add := func(intent bridgev2.MatrixAPI) error {
		as, ok := intent.(*matrix.ASIntent)
		if !ok || as.Matrix == nil || as.Matrix.Client == nil {
			return fmt.Errorf("live Matrix client unavailable for portal identity")
		}
		if !seen[as.Matrix.Client.UserID] {
			clients = append(clients, as.Matrix.Client)
			seen[as.Matrix.Client.UserID] = true
		}
		return nil
	}
	info, err := sa.GetChatInfo(ctx, portal)
	if err != nil {
		return nil, err
	}
	if info == nil || info.Members == nil {
		return nil, fmt.Errorf("portal member identities unavailable")
	}
	var dp bridgev2.MatrixAPI
	if sa.UserLogin.User != nil {
		dp = sa.UserLogin.User.DoublePuppet(ctx)
	}
	for userID, member := range info.Members.MemberMap {
		if member.IsFromMe && dp != nil {
			continue
		}
		ghost, err := sa.UserLogin.Bridge.GetGhostByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if err := add(ghost.Intent); err != nil {
			return nil, err
		}
	}
	if dp != nil {
		if err := add(dp); err != nil {
			return nil, err
		}
	}
	return clients, nil
}

func (sa *SnapchatAPI) classifyRoomDead(ctx context.Context, portal *bridgev2.Portal) (bool, error) {
	clients, err := sa.portalRoomClients(ctx, portal)
	if err != nil {
		return false, err
	}
	probers := make([]liveRoomProber, len(clients))
	for i, client := range clients {
		probers[i] = client
	}
	return liveRoomDead(ctx, portal.MXID, probers)
}

func liveRoomDead(ctx context.Context, room id.RoomID, probers []liveRoomProber) (bool, error) {
	if len(probers) == 0 || room == "" {
		return false, fmt.Errorf("room or live identities unavailable")
	}
	// Every identity must miss both before and after every direct join fails.
	for phase := 0; phase < 3; phase++ {
		for _, prober := range probers {
			var err error
			if phase == 1 {
				_, err = prober.JoinRoomByID(ctx, room)
			} else {
				_, err = prober.Members(ctx, room)
			}
			if err == nil {
				return false, nil
			}
			if !conclusiveMatrixMiss(err) {
				return false, fmt.Errorf("inconclusive live room probe/join: %w", err)
			}
		}
	}
	return true, nil
}

// repairPortalMemberships re-adds only the members that a live membership read
// shows as missing. It never kicks or mass-reinvites existing members.
func (sa *SnapchatAPI) repairPortalMemberships(ctx context.Context, portal *bridgev2.Portal) (int, error) {
	if portal == nil || portal.MXID == "" {
		return 0, nil
	}
	clients, err := sa.portalRoomClients(ctx, portal)
	if err != nil {
		return 0, err
	}
	var members *mautrix.RespMembers
	for _, client := range clients {
		members, err = client.Members(ctx, portal.MXID)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, fmt.Errorf("live membership read: %w", err)
	}
	present := make(map[id.UserID]event.Membership, len(members.Chunk))
	for _, evt := range members.Chunk {
		present[id.UserID(evt.GetStateKey())] = evt.Content.AsMember().Membership
	}
	repaired := 0
	var failures []error
	for _, client := range clients {
		if present[client.UserID] == event.MembershipJoin {
			continue
		}
		// A readable room can be invite-only. Invite from a joined identity
		// before joining, without touching existing invites or bans.
		if present[client.UserID] != event.MembershipInvite && present[client.UserID] != event.MembershipBan {
			for _, inviter := range clients {
				if present[inviter.UserID] == event.MembershipJoin {
					if _, err := inviter.InviteUser(ctx, portal.MXID, &mautrix.ReqInviteUser{UserID: client.UserID}); err == nil {
						break
					}
				}
			}
		}
		if _, err := client.JoinRoomByID(ctx, portal.MXID); err != nil {
			failures = append(failures, fmt.Errorf("join %s: %w", client.UserID, err))
		} else {
			repaired++
			present[client.UserID] = event.MembershipJoin
		}
	}
	user := sa.UserLogin.UserMXID
	if user != "" && present[user] != event.MembershipJoin && present[user] != event.MembershipInvite {
		if _, err := clients[0].InviteUser(ctx, portal.MXID, &mautrix.ReqInviteUser{UserID: user}); err != nil {
			failures = append(failures, fmt.Errorf("invite user: %w", err))
		} else {
			repaired++
		}
	}
	return repaired, errors.Join(failures...)
}

// conclusiveMatrixMiss reports whether a Matrix error proves the room/event is
// inaccessible or absent (as opposed to a transient network/server failure).
func conclusiveMatrixMiss(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var httpErr mautrix.HTTPError
	if !errors.As(err, &httpErr) {
		var ptr *mautrix.HTTPError
		if !errors.As(err, &ptr) || ptr == nil {
			return false
		}
		httpErr = *ptr
	}
	return httpErr.Response != nil && httpErr.RespError != nil && !httpErr.RespError.CanRetry &&
		((httpErr.Response.StatusCode == 403 && httpErr.RespError.ErrCode == mautrix.MForbidden.ErrCode) ||
			(httpErr.Response.StatusCode == 404 && httpErr.RespError.ErrCode == mautrix.MNotFound.ErrCode))
}
