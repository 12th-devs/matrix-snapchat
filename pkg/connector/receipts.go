package connector

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/colej/mautrix-snapchat/internal/store"
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/bridgev2/simplevent"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

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
		receiptSender := makeUserID(adv.Participant)
		if err := sa.emitRemoteReadReceipt(ctx, chat, adv, target, bridgeTarget, receiptSender); err != nil {
			log.Printf("bridgev2 event_class: kind=read_receipt source=snapchat chat_id=%s participant=%s watermark=%d target_remote_id=%s target_mxid=%s target_sender=%s target_double_puppeted=%t receipt_sender=%s action=direct_receipt_failed endpoint=/receipt/m.read err=%v", chat.ID, adv.Participant, adv.Watermark, targetRemoteID, bridgeTarget.MXID, bridgeTarget.SenderID, bridgeTarget.IsDoublePuppeted, receiptSender, err)
			continue
		}
		sa.markReadWatermarkSynced(chat.ID, adv.Participant, adv.Watermark)
	}
}

func (sa *SnapchatAPI) emitRemoteReadReceipt(ctx context.Context, chat sidecar.Chat, advance watermarkAdvance, target networkid.MessageID, bridgeTarget *database.Message, receiptSender networkid.UserID) error {
	targetRemoteID := strconv.FormatInt(advance.Watermark, 10)
	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(chat.ID),
		Receiver: sa.UserLogin.ID,
	}
	portal, err := sa.UserLogin.Bridge.GetPortalByKey(ctx, portalKey)
	if err != nil || portal == nil || portal.MXID == "" {
		if err != nil {
			return err
		}
		return bridgev2.ErrNoPortal
	}
	ghost, err := sa.UserLogin.Bridge.GetGhostByID(ctx, receiptSender)
	if err != nil {
		return err
	}
	if !intentSupportsDirectReceipt(ghost.Intent) {
		log.Printf("bridgev2 event_class: kind=read_receipt source=snapchat chat_id=%s participant=%s watermark=%d action=queue reason=non_as_intent", chat.ID, advance.Participant, advance.Watermark)
		sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.Receipt{
			EventMeta: simplevent.EventMeta{
				Type: bridgev2.RemoteEventReadReceipt,
				LogContext: func(c zerolog.Context) zerolog.Context {
					return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("participant", advance.Participant).Int64("watermark", advance.Watermark).Str("target_remote_id", targetRemoteID).Str("receipt_sender", string(receiptSender))
				},
				PortalKey: portalKey,
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
		return nil
	}
	receiptTS := time.Now().UnixMilli()
	content := map[string]any{"ts": receiptTS}
	if err := sendDirectReceipt(ctx, ghost.Intent, portal.MXID, bridgeTarget.MXID, content); err != nil {
		return err
	}
	log.Printf("bridgev2 event_class: kind=read_receipt source=snapchat chat_id=%s participant=%s watermark=%d target_remote_id=%s target_mxid=%s receipt_sender=%s ghost_mxid=%s endpoint=/receipt/m.read action=direct_receipt_success ts=%d", chat.ID, advance.Participant, advance.Watermark, targetRemoteID, bridgeTarget.MXID, receiptSender, ghost.Intent.GetMXID(), receiptTS)
	return nil
}

func intentSupportsDirectReceipt(intent bridgev2.MatrixAPI) bool {
	if intent == nil {
		return false
	}
	client, ok := reflectedIntentClient(intent)
	if !ok {
		return false
	}
	return client.MethodByName("SendReceipt").IsValid()
}

func sendDirectReceipt(ctx context.Context, intent bridgev2.MatrixAPI, roomID id.RoomID, eventID id.EventID, content any) error {
	if err := intent.EnsureJoined(ctx, roomID); err != nil {
		return err
	}
	client, ok := reflectedIntentClient(intent)
	if !ok {
		return fmt.Errorf("intent %T does not expose a direct Matrix client", intent)
	}
	method := client.MethodByName("SendReceipt")
	if !method.IsValid() {
		return fmt.Errorf("intent %T direct Matrix client does not expose SendReceipt", intent)
	}
	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(roomID),
		reflect.ValueOf(eventID),
		reflect.ValueOf(event.ReceiptTypeRead),
		reflect.ValueOf(content),
	})
	if len(results) != 1 || results[0].IsNil() {
		return nil
	}
	err, ok := results[0].Interface().(error)
	if !ok {
		return fmt.Errorf("SendReceipt returned non-error %T", results[0].Interface())
	}
	return err
}

func reflectedIntentClient(intent bridgev2.MatrixAPI) (reflect.Value, bool) {
	value := reflect.ValueOf(intent)
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	matrixField := value.FieldByName("Matrix")
	if !matrixField.IsValid() || matrixField.IsNil() {
		return reflect.Value{}, false
	}
	matrixValue := matrixField
	if matrixValue.Kind() == reflect.Pointer {
		matrixValue = matrixValue.Elem()
	}
	if !matrixValue.IsValid() || matrixValue.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	clientField := matrixValue.FieldByName("Client")
	if !clientField.IsValid() || clientField.IsNil() {
		return reflect.Value{}, false
	}
	return clientField, true
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
