package connector

import (
	"context"
	"fmt"
	"strings"

	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/networkid"
)

var _ bridgev2.ContactListingNetworkAPI = (*SnapchatAPI)(nil)

func (sa *SnapchatAPI) GetContactList(ctx context.Context) ([]*bridgev2.ResolveIdentifierResponse, error) {
	if sa == nil || sa.Connector == nil || sa.Connector.store == nil {
		return nil, nil
	}
	portals, err := sa.Connector.store.ListPortals()
	if err != nil {
		return nil, fmt.Errorf("list cached Snapchat contacts: %w", err)
	}
	contacts := make([]*bridgev2.ResolveIdentifierResponse, 0, len(portals))
	seen := make(map[networkid.UserID]struct{}, len(portals))
	for _, portal := range portals {
		chatID := strings.TrimSpace(portal.RemoteID)
		if chatID == "" {
			chatID = strings.TrimSpace(portal.PortalKey)
		}
		name := strings.TrimSpace(portal.RemoteName)
		if name == "" {
			name = chatID
		}
		userID := sa.remoteUserIDForChat(chatID, name)
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		contacts = append(contacts, &bridgev2.ResolveIdentifierResponse{
			UserID: userID,
			UserInfo: &bridgev2.UserInfo{
				Name:        &name,
				Avatar:      sa.cachedAvatarForGhost(userID),
				Identifiers: []string{fmt.Sprintf("snapchat:%s", userID)},
			},
			Chat: &bridgev2.CreateChatResponse{
				PortalKey: networkid.PortalKey{
					ID:       makePortalID(chatID),
					Receiver: makeUserLoginID(sa.Label),
				},
				PortalInfo: sa.chatInfoFor(chatID, name),
			},
		})
	}
	return contacts, nil
}
