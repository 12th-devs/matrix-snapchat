package connector

import (
	"context"
	"fmt"
	"log"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/store"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
)

// MediaDeliveryMetadata is persisted by bridgev2 only after the Matrix send.
// Upload completion alone must never make a placeholder permanently hydrated.
type MediaDeliveryMetadata struct {
	MediaDelivered  bool `json:"media_delivered,omitempty"`
	AttachmentCount int  `json:"attachment_count,omitempty"`
}

func mediaPartID(index int) networkid.PartID {
	if index == 0 {
		return ""
	}
	return networkid.PartID(fmt.Sprintf("media-%d", index))
}

func deliveredMediaParts(parts []*database.Message, count int) bool {
	if count == 0 || len(parts) != count {
		return false
	}
	for _, part := range parts {
		meta, ok := part.Metadata.(*MediaDeliveryMetadata)
		if !ok || !meta.MediaDelivered || meta.AttachmentCount != count || part.MXID == "" || part.HasFakeMXID() {
			return false
		}
	}
	return true
}

func (sa *SnapchatAPI) reconcileMediaDelivery(ctx context.Context, chatID, mappingID, baseID string, count int) bool {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || sa.UserLogin.Bridge.DB == nil {
		return false
	}
	parts, err := sa.UserLogin.Bridge.DB.Message.GetAllPartsByID(ctx, sa.UserLogin.ID, makeMessageID(scopedSnapchatMessageID(chatID, mappingID)))
	if err != nil || !deliveredMediaParts(parts, count) {
		return false
	}
	if sa.Connector != nil && sa.Connector.store != nil {
		if err := sa.Connector.store.MarkMessageHydrated(chatID, baseID); err != nil {
			log.Printf("bridgev2 media: failed to persist delivery chat_id=%s message_id=%s: %v", chatID, baseID, err)
			return false
		}
	}
	return true
}

func (sa *SnapchatAPI) mediaDeliveryPostHandle(chat sidecar.Chat, mappingID string, message sidecar.Message) func(context.Context, *bridgev2.Portal) {
	if len(message.Media) == 0 {
		return nil
	}
	return func(ctx context.Context, _ *bridgev2.Portal) {
		baseID := baseSnapchatMessageID(message.ID)
		if sa.Connector != nil && sa.Connector.store != nil && sa.lookupStoredMessage(chat.ID, baseID) == nil {
			_ = sa.Connector.store.UpsertMessages([]store.MessageState{{PortalKey: chat.ID, RemoteID: baseID, HasMedia: true, Kind: "media", Text: message.Text, LastSeenAt: time.Now()}})
		}
		sa.reconcileMediaDelivery(ctx, chat.ID, mappingID, baseID, len(message.Media))
		// A failed send must be retryable in this process as well as after restart.
		sa.mu.Lock()
		delete(sa.seenByChat[chat.ID], message.ID)
		sa.mu.Unlock()
	}
}
