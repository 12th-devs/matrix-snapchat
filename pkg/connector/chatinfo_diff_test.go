package connector

import (
	"context"
	"testing"
	"time"

	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/event"
)

func testChatInfo(name string, disappearSeconds int64, avatarID string, memberNames map[string]string) *bridgev2.ChatInfo {
	roomType := database.RoomTypeDM
	info := &bridgev2.ChatInfo{
		Name: &name,
		Type: &roomType,
		Members: &bridgev2.ChatMemberList{
			IsFull:           true,
			TotalMemberCount: len(memberNames) + 1,
			MemberMap:        bridgev2.ChatMemberMap{},
		},
	}
	if avatarID != "" {
		info.Avatar = &bridgev2.Avatar{ID: networkid.AvatarID(avatarID)}
	}
	for uid, display := range memberNames {
		displayName := display
		info.Members.MemberMap[networkid.UserID(uid)] = bridgev2.ChatMember{
			EventSender: bridgev2.EventSender{Sender: networkid.UserID(uid)},
			UserInfo:    &bridgev2.UserInfo{Name: &displayName},
		}
	}
	if disappearSeconds > 0 {
		info.Disappear = &database.DisappearingSetting{
			Type:  event.DisappearingTypeAfterSend,
			Timer: time.Duration(disappearSeconds) * time.Second,
		}
	}
	return info
}

func testPortal(name, avatarID string, disappearSeconds int64, capID string) *database.Portal {
	return &database.Portal{
		MXID:      "!room:beeper.local",
		Name:      name,
		AvatarID:  networkid.AvatarID(avatarID),
		AvatarSet: true,
		RoomType:  database.RoomTypeDM,
		Disappear: database.DisappearingSetting{
			Type:  event.DisappearingTypeAfterSend,
			Timer: time.Duration(disappearSeconds) * time.Second,
		},
		CapState: database.CapabilityState{ID: capID},
	}
}

func testFingerprintAPI() *SnapchatAPI {
	sc := &SnapchatConnector{}
	return &SnapchatAPI{
		Connector:                sc,
		appliedChatInfoState:     make(map[string]chatInfoState),
		lastSkippedChatInfoState: make(map[string]chatInfoState),
	}
}

func TestChatResyncUnchangedWhenInfoMatchesPortal(t *testing.T) {
	sa := testFingerprintAPI()
	info := testChatInfo("evie", 86400, "snapchat-bitmoji-abc", map[string]string{
		"remote-user-1": "evie",
	})
	state := sa.effectiveChatInfoState(info)
	sa.appliedChatInfoState["chat-1"] = state
	portal := &database.Portal{
		MXID:        "!room:beeper.local",
		Name:        "evie",
		AvatarID:    networkid.AvatarID("snapchat-bitmoji-abc"),
		AvatarSet:   true,
		RoomType:    database.RoomTypeDM,
		OtherUserID: "remote-user-1",
		Disappear: database.DisappearingSetting{
			Type:  event.DisappearingTypeAfterSend,
			Timer: 86400 * time.Second,
		},
	}
	if !state.matchesPortal(portal) {
		t.Fatal("identical ChatInfo should match the stored portal row")
	}
}

func TestChatResyncQueuedWhenNameChanged(t *testing.T) {
	sa := testFingerprintAPI()
	info := testChatInfo("new name", 86400, "snapchat-bitmoji-abc", nil)
	state := sa.effectiveChatInfoState(info)
	sa.appliedChatInfoState["chat-1"] = state
	portal := testPortal("old name", "snapchat-bitmoji-abc", 86400, "fi.mau.snapchat.capabilities.test")
	if state.matchesPortal(portal) {
		t.Fatal("changed display name must not match the stored portal row")
	}
}

func TestChatResyncQueuedWhenAvatarChanged(t *testing.T) {
	sa := testFingerprintAPI()
	info := testChatInfo("evie", 86400, "snapchat-bitmoji-new", nil)
	state := sa.effectiveChatInfoState(info)
	sa.appliedChatInfoState["chat-1"] = state
	portal := testPortal("evie", "snapchat-bitmoji-old", 86400, "fi.mau.snapchat.capabilities.test")
	if state.matchesPortal(portal) {
		t.Fatal("changed avatar must not match the stored portal row")
	}
}

func TestChatResyncQueuedWhenDisappearTimerChanged(t *testing.T) {
	sa := testFingerprintAPI()
	info := testChatInfo("evie", 3600, "snapchat-bitmoji-abc", nil)
	state := sa.effectiveChatInfoState(info)
	sa.appliedChatInfoState["chat-1"] = state
	portal := testPortal("evie", "snapchat-bitmoji-abc", 86400, "fi.mau.snapchat.capabilities.test")
	if state.matchesPortal(portal) {
		t.Fatal("changed disappearing timer must not match the stored portal row")
	}
}

func TestChatResyncQueuedWhenDisappearSettingRemoved(t *testing.T) {
	sa := testFingerprintAPI()
	info := testChatInfo("evie", 0, "snapchat-bitmoji-abc", nil)
	state := sa.effectiveChatInfoState(info)
	sa.appliedChatInfoState["chat-1"] = state
	portal := testPortal("evie", "snapchat-bitmoji-abc", 86400, "fi.mau.snapchat.capabilities.test")
	if state.matchesPortal(portal) {
		t.Fatal("removed disappearing timer must not match the stored portal row")
	}
}

func TestChatResyncQueuedWhenCapabilitiesChanged(t *testing.T) {
	sa := testFingerprintAPI()
	info := testChatInfo("evie", 86400, "snapchat-bitmoji-abc", nil)
	state := sa.effectiveChatInfoState(info)
	state.CapID = "fi.mau.snapchat.capabilities.new"
	sa.appliedChatInfoState["chat-1"] = state
	portal := testPortal("evie", "snapchat-bitmoji-abc", 86400, "fi.mau.snapchat.capabilities.old")
	if state.matchesPortal(portal) {
		t.Fatal("changed capability version must not match the stored portal row")
	}
}

func TestChatResyncQueuedWhenParticipantChanged(t *testing.T) {
	sa := testFingerprintAPI()
	info := testChatInfo("group", 0, "", map[string]string{
		"user-1": "alice",
		"user-2": "bob",
	})
	state := sa.effectiveChatInfoState(info)
	sa.appliedChatInfoState["chat-1"] = state

	changed := testChatInfo("group", 0, "", map[string]string{
		"user-1": "alice",
		"user-2": "bob",
		"user-3": "carol",
	})
	changedState := sa.effectiveChatInfoState(changed)
	if changedState == state {
		t.Fatal("added group participant must produce a different chat info state")
	}
}

func TestChatResyncNotSkippedWithoutRecordedState(t *testing.T) {
	sa := testFingerprintAPI()
	info := testChatInfo("evie", 86400, "snapchat-bitmoji-abc", nil)
	if sa.chatResyncUnchanged(context.Background(), "chat-1", info) {
		t.Fatal("without a recorded state the resync must not be skipped")
	}
}

func TestChatResyncNotSkippedForNilInfo(t *testing.T) {
	sa := testFingerprintAPI()
	sa.appliedChatInfoState["chat-1"] = chatInfoState{}
	if sa.chatResyncUnchanged(context.Background(), "chat-1", nil) {
		t.Fatal("nil ChatInfo must never be treated as unchanged")
	}
}

func TestMessagePathUnaffectedByResyncSkip(t *testing.T) {
	sa := testFingerprintAPI()
	info := testChatInfo("evie", 86400, "snapchat-bitmoji-abc", nil)
	state := sa.effectiveChatInfoState(info)
	sa.appliedChatInfoState["chat-1"] = state

	if sa.isSeen("chat-1", "9001") {
		t.Fatal("fresh message must not be marked seen by the resync fingerprint path")
	}
}
