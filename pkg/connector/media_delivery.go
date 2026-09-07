package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/snapapi"
	"github.com/colej/mautrix-snapchat/internal/store"
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/bridgev2/simplevent"
	"maunium.net/go/mautrix/event"
)

// MediaDeliveryMetadata is persisted by bridgev2 only after the Matrix send.
// Upload completion alone must never make a placeholder permanently hydrated.
type MediaDeliveryMetadata struct {
	MediaDelivered      bool `json:"media_delivered,omitempty"`
	AttachmentCount     int  `json:"attachment_count,omitempty"`
	PresentationVersion int  `json:"presentation_version,omitempty"`
}

// Repair only proven deliveries during resync, using their existing Matrix media.
func (sa *SnapchatAPI) repairMediaPresentation(ctx context.Context, chat sidecar.Chat, baseID string, message sidecar.Message) error {
	br := sa.UserLogin.Bridge
	for _, mappingID := range []string{mediaMessageID(baseID, message.Media), baseID} {
		targetID := makeMessageID(scopedSnapchatMessageID(chat.ID, mappingID))
		parts, err := br.DB.Message.GetAllPartsByID(ctx, sa.UserLogin.ID, targetID)
		if err != nil {
			return err
		}
		if !deliveredMediaParts(parts, len(message.Media)) {
			continue
		}
		needsRepair := false
		for _, part := range parts {
			needsRepair = needsRepair || part.Metadata.(*MediaDeliveryMetadata).PresentationVersion == 0
		}
		if !needsRepair {
			return nil
		}
		portal, err := br.GetExistingPortalByKey(ctx, parts[0].Room)
		if err != nil || portal == nil {
			return fmt.Errorf("load presentation portal: %v", err)
		}
		// Key-ratcheting installations intentionally disable historical reads.
		// A cosmetic caption repair must never interrupt message resync or weaken
		// that policy. Preflight before queueing to avoid a visible error notice.
		for _, part := range parts {
			if part.Metadata.(*MediaDeliveryMetadata).PresentationVersion != 0 {
				continue
			}
			if _, err := br.Bot.GetEvent(ctx, portal.MXID, part.MXID); err != nil {
				zerolog.Ctx(ctx).Warn().Str("message_id", baseID).Msg("media: retaining historical caption; event decryption unavailable")
				return nil
			}
		}
		message = sa.normalizeRemoteMessageDirection(chat.ID, message)
		sender := bridgev2.EventSender{IsFromMe: message.Outgoing, Sender: sa.messageSenderID(chat, message)}
		if message.Outgoing {
			sender.Sender = makeUserID(sa.Label)
			sender.SenderLogin = makeUserLoginID(sa.Label)
		}
		done := make(chan struct{})
		result := br.QueueRemoteEvent(sa.UserLogin, &simplevent.Message[*event.MessageEventContent]{
			EventMeta: simplevent.EventMeta{
				Type:           bridgev2.RemoteEventEdit,
				PortalKey:      networkid.PortalKey{ID: networkid.PortalID(chat.ID), Receiver: sa.UserLogin.ID},
				Sender:         sender,
				PostHandleFunc: func(context.Context, *bridgev2.Portal) { close(done) },
			},
			ID:              networkid.MessageID(string(targetID) + "-presentation-1"),
			TargetMessage:   targetID,
			ConvertEditFunc: sa.convertMediaPresentationEdit,
		})
		if err := waitRemoteMessage(result, done, []context.Context{ctx}); err != nil {
			return err
		}
		// Queue completion is not send success. Check persisted metadata, not an
		// immediate GetEvent of the edit (which may not be readable yet).
		for _, original := range parts {
			updated, err := br.DB.Message.GetPartByID(ctx, sa.UserLogin.ID, targetID, original.PartID)
			if err != nil {
				return err
			}
			if updated == nil || updated.MXID != original.MXID {
				return fmt.Errorf("media presentation mapping changed for %s", targetID)
			}
			meta, ok := updated.Metadata.(*MediaDeliveryMetadata)
			if !ok || meta.PresentationVersion == 0 {
				return fmt.Errorf("media presentation repair did not persist for %s part %s", targetID, original.PartID)
			}
		}
		return nil
	}
	return fmt.Errorf("media presentation delivery mapping missing for %s", baseID)
}

func (sa *SnapchatAPI) convertMediaPresentationEdit(ctx context.Context, portal *bridgev2.Portal, _ bridgev2.MatrixAPI, existing []*database.Message, _ *event.MessageEventContent) (*bridgev2.ConvertedEdit, error) {
	edit := &bridgev2.ConvertedEdit{}
	for _, target := range existing {
		meta, ok := target.Metadata.(*MediaDeliveryMetadata)
		if !ok || !meta.MediaDelivered || meta.PresentationVersion != 0 {
			continue
		}
		if target.Room != portal.PortalKey || target.MXID == "" || target.HasFakeMXID() {
			return nil, fmt.Errorf("invalid media presentation target %s", target.ID)
		}
		// Bot.GetEvent uses the bridge's normal event decryption path.
		evt, err := portal.Bridge.Bot.GetEvent(ctx, portal.MXID, target.MXID)
		if err != nil {
			return nil, err
		}
		if evt == nil || evt.Type != event.EventMessage || !evt.Content.AsMessage().MsgType.IsMedia() {
			return nil, fmt.Errorf("media presentation target %s is not a decrypted media event", target.MXID)
		}
		content := evt.Content.AsMessage()
		updatedMeta := *meta
		updatedMeta.PresentationVersion = 1
		updated := *target
		updated.Metadata = &updatedMeta
		if content.Body != "Media" {
			if err := portal.Bridge.DB.Message.Update(ctx, &updated); err != nil {
				return nil, err
			}
			continue
		}
		// Clone nested file/encryption/relation fields before bridgev2 sets the edit.
		raw, err := json.Marshal(content)
		if err != nil {
			return nil, err
		}
		var cloned event.MessageEventContent
		if err := json.Unmarshal(raw, &cloned); err != nil {
			return nil, err
		}
		mimeType := ""
		if cloned.Info != nil {
			mimeType = cloned.Info.MimeType
		}
		cloned.Body = cloned.GetFileName()
		if cloned.Body == "" || cloned.Body == "Media" {
			cloned.Body = mediaFileNameForMIME("", cloned.MsgType, mimeType)
		}
		part := &bridgev2.ConvertedMessagePart{Type: event.EventMessage, Content: &cloned, DBMetadata: &updatedMeta}
		edit.ModifiedParts = append(edit.ModifiedParts, part.ToEditPart(&updated))
	}
	return edit, nil
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

// mediaDeliveryConfirmed reports whether bridge-side Matrix mapping state
// proves the message's media attachments were successfully delivered, under
// either the media-scoped or the base remote mapping. HydratedAt alone is not
// authoritative for historical rows.
func (sa *SnapchatAPI) mediaDeliveryConfirmed(ctx context.Context, chatID, baseID string, message sidecar.Message, count int) bool {
	if count == 0 {
		return false
	}
	for _, mappingID := range []string{mediaMessageID(baseID, message.Media), baseID} {
		parts, err := sa.currentMessageParts(ctx, chatID, mappingID)
		if err == nil && deliveredMediaParts(parts, count) {
			return sa.reconcileMediaDelivery(ctx, chatID, mappingID, baseID, count)
		}
	}
	return false
}

// Historical proof must belong to the current room. Keep mappings on uncertain
// reads; deleting only conclusively missing parts lets bridgev2 retry delivery.
func (sa *SnapchatAPI) currentMessageParts(ctx context.Context, chatID, mappingID string) ([]*database.Message, error) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || sa.UserLogin.Bridge.DB == nil {
		return nil, fmt.Errorf("bridge database unavailable")
	}
	br := sa.UserLogin.Bridge
	parts, err := br.DB.Message.GetAllPartsByID(ctx, sa.UserLogin.ID, makeMessageID(scopedSnapchatMessageID(chatID, mappingID)))
	if err != nil || len(parts) == 0 {
		return parts, err
	}
	portal, err := br.GetExistingPortalByKey(ctx, networkid.PortalKey{ID: networkid.PortalID(chatID), Receiver: sa.UserLogin.ID})
	if err != nil || portal == nil || portal.MXID == "" {
		return parts, fmt.Errorf("current portal unavailable: %v", err)
	}
	client, err := sa.liveBotClient()
	if err != nil {
		return parts, err
	}
	kept := make([]*database.Message, 0, len(parts))
	for _, part := range parts {
		_, err = client.GetEvent(ctx, portal.MXID, part.MXID)
		if !conclusiveMatrixMiss(err) {
			kept = append(kept, part)
			continue
		}
		if err := br.DB.Message.Delete(ctx, part.RowID); err != nil {
			return nil, err
		}
		log.Printf("bridgev2 resync: stale mapping removed chat_id=%s message_id=%s event_id=%s current_room=%s", chatID, mappingID, part.MXID, portal.MXID)
	}
	return kept, nil
}

// beginMediaRehydration performs the one-time demotion of a stale hydrated
// media row to the ordinary retryable hydration path. It reports whether this
// call won the transition.
func (sa *SnapchatAPI) beginMediaRehydration(chatID, remoteID string) bool {
	if sa.Connector == nil || sa.Connector.store == nil {
		return false
	}
	started, err := sa.Connector.store.BeginMessageRehydration(chatID, remoteID)
	if err != nil {
		log.Printf("bridgev2 media: rehydration transition failed chat_id=%s message_id=%s: %v", chatID, remoteID, err)
		return false
	}
	return started
}

// mediaHydrationDecision is what the API sync does with a media-bearing row.
type mediaHydrationDecision int

const (
	// mediaHydrationSkip keeps the row as messageSyncUnchanged: delivery is
	// proven (or the row is a view-once snap placeholder).
	mediaHydrationSkip mediaHydrationDecision = iota
	// mediaHydrationRecover demotes stale hydrated state without delivery
	// proof to the ordinary retryable hydration path.
	mediaHydrationRecover
	// mediaHydrationRender is the ordinary unhydrated media flow.
	mediaHydrationRender
)

// decideMediaHydration encodes the authoritative hydration rule: HydratedAt
// alone never proves delivery; successful MediaDelivered evidence does.
// Snaps stay on the placeholder path unless their media keys were recovered
// from the decrypted contents (the keys are required to download and decrypt
// the media; snaps without keys are never auto-opened).
func decideMediaHydration(existing *store.MessageState, deliveryConfirmed, isSnap, snapKeysReady bool) mediaHydrationDecision {
	if deliveryConfirmed {
		return mediaHydrationSkip
	}
	if isSnap && !snapKeysReady {
		return mediaHydrationSkip
	}
	if existing != nil && existing.HydratedAt != nil {
		return mediaHydrationRecover
	}
	return mediaHydrationRender
}

// snapMediaKeysReady reports whether the snap's media key was recovered from
// its decrypted contents, making the media downloadable and decryptable.
func snapMediaKeysReady(message snapapi.Message) bool {
	return len(message.Media) > 0 && len(message.Media[0].Key) > 0
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
