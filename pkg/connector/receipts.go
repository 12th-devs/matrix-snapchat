package connector

import (
	"log"
	"strconv"
	"time"

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
	sa.mu.Lock()
	defer sa.mu.Unlock()
	previous := sa.readWatermarks[chatID]
	current := make(map[string]int64, len(previous))
	for participant, watermark := range previous {
		current[participant] = watermark
	}
	var advances []watermarkAdvance
	for participant, watermark := range incoming {
		if participant == "" || participant == selfID || watermark <= 0 {
			continue
		}
		if previous[participant] >= watermark {
			current[participant] = previous[participant]
			continue
		}
		advances = append(advances, watermarkAdvance{Participant: participant, Watermark: watermark})
		current[participant] = watermark
	}
	if len(current) > 0 {
		sa.readWatermarks[chatID] = current
	}
	return advances
}

// syncReadWatermarks emits bridge read receipts for remote participants whose
// read position advanced since the last sync. Text read state only: this never
// opens snaps or other media (safeNoOpen stays intact; snap/media handling is
// unaffected).
func (sa *SnapchatAPI) syncReadWatermarks(chat sidecar.Chat) {
	if sa.UserLogin == nil || sa.UserLogin.Bridge == nil || chat.ID == "" || len(chat.ReadWatermarks) == 0 {
		return
	}
	sa.mu.Lock()
	selfID := sa.apiSelfUserID
	sa.mu.Unlock()
	for _, advance := range sa.diffReadWatermarks(chat.ID, chat.ReadWatermarks, selfID) {
		adv := advance
		target := makeMessageID(scopedSnapchatMessageID(chat.ID, strconv.FormatInt(adv.Watermark, 10)))
		log.Printf("bridgev2 event_class: kind=read_receipt source=snapchat chat_id=%s participant=%s watermark=%d action=queue", chat.ID, adv.Participant, adv.Watermark)
		sa.UserLogin.Bridge.QueueRemoteEvent(sa.UserLogin, &simplevent.Receipt{
			EventMeta: simplevent.EventMeta{
				Type: bridgev2.RemoteEventReadReceipt,
				LogContext: func(c zerolog.Context) zerolog.Context {
					return c.Str("chat", chat.Name).Str("chat_id", chat.ID).Str("participant", adv.Participant).Int64("watermark", adv.Watermark)
				},
				PortalKey: networkid.PortalKey{
					ID:       networkid.PortalID(chat.ID),
					Receiver: sa.UserLogin.ID,
				},
				Sender: bridgev2.EventSender{
					IsFromMe: false,
					Sender:   makeUserID(adv.Participant),
				},
				Timestamp: time.Now(),
			},
			LastTarget: target,
			Targets:    []networkid.MessageID{target},
			ReadUpTo:   time.Now(),
		})
	}
}
