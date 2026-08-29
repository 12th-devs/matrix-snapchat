package connector

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/networkid"
)

func (sa *SnapchatAPI) avatarForGhost(ctx context.Context, ghostID networkid.UserID) *bridgev2.Avatar {
	snapUserID := sa.snapUserIDForGhost(ghostID)
	if snapUserID == "" {
		return nil
	}
	if avatar := sa.cachedAvatarForSnapUserID(snapUserID); avatar != nil {
		return avatar
	}
	if !sa.shouldFetchGhostAvatar(snapUserID) {
		return nil
	}
	client, err := sa.ensureSnapClient(ctx)
	sa.markGhostAvatarChecked(snapUserID)
	if err != nil {
		log.Printf("bridgev2 avatar: API client unavailable for ghost=%s snap_user_id=%s: %v", ghostID, snapUserID, err)
		return nil
	}
	profile, err := client.GetPublicProfile(ctx, snapUserID)
	if err != nil {
		log.Printf("bridgev2 avatar: profile lookup failed for ghost=%s snap_user_id=%s: %v", ghostID, snapUserID, err)
		return nil
	}
	if profile.Name != "" {
		sa.rememberGhostName(ghostID, profile.Name)
	}
	if profile.Username != "" {
		sa.rememberGhostUsername(ghostID, profile.Username)
	}
	sa.rememberSnapAvatarURL(snapUserID, profile.AvatarURL)
	return sa.avatarFromURL(snapUserID, profile.AvatarURL)
}

func (sa *SnapchatAPI) cachedAvatarForGhost(ghostID networkid.UserID) *bridgev2.Avatar {
	snapUserID := sa.snapUserIDForGhost(ghostID)
	if snapUserID == "" {
		return nil
	}
	return sa.cachedAvatarForSnapUserID(snapUserID)
}

func (sa *SnapchatAPI) cachedAvatarForSnapUserID(snapUserID string) *bridgev2.Avatar {
	snapUserID = strings.TrimSpace(snapUserID)
	if snapUserID == "" {
		return nil
	}
	if avatarURL := sa.lookupSnapAvatarURL(snapUserID); avatarURL != "" {
		return sa.avatarFromURL(snapUserID, avatarURL)
	}
	sa.apiMu.Lock()
	client := sa.apiClient
	sa.apiMu.Unlock()
	if client == nil {
		return nil
	}
	return sa.avatarFromURL(snapUserID, client.CachedAvatarURL(snapUserID))
}

func (sa *SnapchatAPI) rememberSnapAvatarURL(snapUserID, avatarURL string) {
	snapUserID = strings.ToLower(strings.TrimSpace(snapUserID))
	avatarURL = strings.TrimSpace(avatarURL)
	if snapUserID == "" || avatarURL == "" {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.avatarURLBySnapID[snapUserID] = avatarURL
}

func (sa *SnapchatAPI) lookupSnapAvatarURL(snapUserID string) string {
	snapUserID = strings.ToLower(strings.TrimSpace(snapUserID))
	if snapUserID == "" {
		return ""
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	return sa.avatarURLBySnapID[snapUserID]
}

func (sa *SnapchatAPI) avatarFromURL(snapUserID, avatarURL string) *bridgev2.Avatar {
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		return nil
	}
	sum := sha1.Sum([]byte(avatarURL))
	return &bridgev2.Avatar{
		ID: networkid.AvatarID("snapchat-bitmoji-" + hex.EncodeToString(sum[:])[:16]),
		Get: func(ctx context.Context) ([]byte, error) {
			client, err := sa.ensureSnapClient(ctx)
			if err != nil {
				return nil, err
			}
			return client.DownloadAvatar(ctx, avatarURL)
		},
	}
}

func (sa *SnapchatAPI) shouldFetchGhostAvatar(snapUserID string) bool {
	snapUserID = strings.TrimSpace(snapUserID)
	if snapUserID == "" {
		return false
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	checkedAt := sa.ghostAvatarCheckedAt[snapUserID]
	return checkedAt.IsZero() || time.Since(checkedAt) >= minGhostAvatarRefreshInterval
}

func (sa *SnapchatAPI) markGhostAvatarChecked(snapUserID string) {
	snapUserID = strings.TrimSpace(snapUserID)
	if snapUserID == "" {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.ghostAvatarCheckedAt[snapUserID] = time.Now()
}

func (sa *SnapchatAPI) snapUserIDForGhost(ghostID networkid.UserID) string {
	id := strings.TrimSpace(string(ghostID))
	if id == "" || strings.EqualFold(id, normalizeIDPart(sa.Label)) {
		return ""
	}
	if sa.Connector == nil || sa.Connector.store == nil {
		if snapUUIDPattern.MatchString(id) {
			return strings.ToLower(id)
		}
		return ""
	}
	portals, err := sa.Connector.store.ListPortals()
	if err != nil {
		if snapUUIDPattern.MatchString(id) {
			return strings.ToLower(id)
		}
		return ""
	}
	for _, portal := range portals {
		otherUserID := strings.TrimSpace(portal.OtherUserID)
		if otherUserID == "" || !snapUUIDPattern.MatchString(otherUserID) {
			continue
		}
		if strings.EqualFold(normalizeIDPart(otherUserID), id) ||
			strings.EqualFold(normalizeIDPart(portal.RemoteName), id) ||
			strings.EqualFold(normalizeIDPart(portal.RemoteID), id) ||
			strings.EqualFold(normalizeIDPart(portal.PortalKey), id) {
			return strings.ToLower(otherUserID)
		}
	}
	if snapUUIDPattern.MatchString(id) {
		return strings.ToLower(id)
	}
	return ""
}
