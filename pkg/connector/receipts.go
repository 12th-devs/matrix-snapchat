package connector

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/colej/mautrix-snapchat/internal/store"
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/bridgev2/simplevent"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
)

// watermarkAdvance is a single participant read-position increase.
type watermarkAdvance struct {
	Participant string
	Watermark   int64
}

// diffReadWatermarks compares the read watermarks reported by a freshly synced
// conversation against the stored per-chat state. It returns every non-self
// participant whose read position strictly advanced and updates the stored
// state. Self is excluded: the user's own Snapchat reads are mirrored by their
// own Matrix client, and echoing them back would cause loops.
func (sa *SnapchatAPI) diffReadWatermarks(chatID string, incoming map[string]int64, selfID string) []watermarkAdvance {
	advances := sa.pendingReadWatermarkAdvances(chatID, incoming, selfID)
	for _, advance := range advances {
		sa.markReadWatermarkSynced(chatID, advance.Participant, advance.Watermark)
	}
	return advances
}

func (sa *SnapchatAPI) pendingReadWatermarkAdvances(chatID string, incoming map[string]int64, selfID string) []watermarkAdvance {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	previous := sa.readWatermarks[chatID]
	var advances []watermarkAdvance
	for participant, watermark := range incoming {
		if participant == "" || participant == selfID || watermark <= 0 {
			continue
		}
		if previous[participant] >= watermark {
			continue
		}
		advances = append(advances, watermarkAdvance{Participant: participant, Watermark: watermark})
	}
	return advances
}

func (sa *SnapchatAPI) markReadWatermarkSynced(chatID, participant string, watermark int64) {
	if chatID == "" || participant == "" || watermark <= 0 {
		return
	}
	sa.mu.Lock()
	defer sa.mu.Unlock()
	current := sa.readWatermarks[chatID]
	if current == nil {
		current = make(map[string]int64)
		sa.readWatermarks[chatID] = current
	}
	if current[participant] < watermark {
		current[participant] = watermark
	}
}

// syncReadWatermarks emits bridge read receipts for remote participants whose
// read position advanced since the last sync. Text read state only: this never
// opens snaps or other media (safeNoOpen stays intact; snap/media handling is
// unaffected).
func (sa *SnapchatAPI) syncReadWatermarks(ctx context.Context, chat sidecar.Chat) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" || len(chat.ReadWatermarks) == 0 {
		return
	}
	sa.mu.Lock()
	selfID := sa.apiSelfUserID
	sa.mu.Unlock()
	for _, advance := range sa.pendingReadWatermarkAdvances(chat.ID, chat.ReadWatermarks, selfID) {
		adv := advance
		targetRemoteID := strconv.FormatInt(adv.Watermark, 10)
		targetState := sa.lookupStoredMessage(chat.ID, targetRemoteID)
		if targetState == nil {
			log.Printf("bridgev2 event_class: kind=read_receipt source=snapchat chat_id=%s participant=%s watermark=%d action=skip reason=target_unmapped", chat.ID, adv.Participant, adv.Watermark)
			continue
		}
		if shouldSkipRemoteReadReceiptTarget(chat, adv, targetState) {
			log.Printf("bridgev2 event_class: kind=read_receipt source=snapchat chat_id=%s participant=%s watermark=%d action=skip reason=target_is_receipt_sender_own_message", chat.ID, adv.Participant, adv.Watermark)
			sa.markReadWatermarkSynced(chat.ID, adv.Participant, adv.Watermark)
			continue
		}
		target := makeMessageID(scopedSnapchatMessageID(chat.ID, targetRemoteID))
		bridgeTarget, err := sa.UserLogin.Bridge.DB.Message.GetLastPartByID(ctx, sa.UserLogin.ID, target)
		if err != nil {
			log.Printf("bridgev2 event_class: kind=read_receipt source=snapchat chat_id=%s participant=%s watermark=%d action=skip reason=bridge_target_lookup_failed err=%v", chat.ID, adv.Participant, adv.Watermark, err)
			continue
		}
		if bridgeTarget == nil || bridgeTarget.HasFakeMXID() {
			log.Printf("bridgev2 event_class: kind=read_receipt source=snapchat chat_id=%s participant=%s watermark=%d action=skip reason=bridge_target_unmapped", chat.ID, adv.Participant, adv.Watermark)
			continue
		}
		sa.markReadWatermarkSynced(chat.ID, adv.Participant, adv.Watermark)
		receiptSender := makeUserID(adv.Participant)
		log.Printf("bridgev2 event_class: kind=read_receipt source=snapchat chat_id=%s participant=%s watermark=%d target_remote_id=%s target_mxid=%s target_sender=%s target_double_puppeted=%t receipt_sender=%s action=queue", chat.ID, adv.Participant, adv.Watermark, targetRemoteID, bridgeTarget.MXID, bridgeTarget.SenderID, bridgeTarget.IsDoublePuppeted, receiptSender)
		sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.Receipt{
			EventMeta: simplevent.EventMeta{
				Type: bridgev2.RemoteEventReadReceipt,
				LogContext: func(c zerolog.Context) zerolog.Context {
					return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("participant", adv.Participant).Int64("watermark", adv.Watermark).Stringer("target_mxid", bridgeTarget.MXID).Str("target_remote_id", targetRemoteID).Str("target_sender", string(bridgeTarget.SenderID)).Bool("target_double_puppeted", bridgeTarget.IsDoublePuppeted).Str("receipt_sender", string(receiptSender))
				},
				PortalKey: networkid.PortalKey{
					ID:       networkid.PortalID(chat.ID),
					Receiver: sa.UserLogin.ID,
				},
				Sender: bridgev2.EventSender{
					IsFromMe:    false,
					Sender:      receiptSender,
					ForceDMUser: true,
				},
				Timestamp: time.Now(),
			},
			LastTarget: target,
			Targets:    []networkid.MessageID{target},
			ReadUpTo:   time.Now(),
		})
	}
}

func shouldSkipRemoteReadReceiptTarget(chat sidecar.Chat, advance watermarkAdvance, target *store.MessageState) bool {
	if target == nil || target.Outgoing {
		return false
	}
	otherUserID := strings.TrimSpace(chat.OtherUserID)
	if otherUserID == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(advance.Participant), otherUserID)
}
