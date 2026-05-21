package snapapi

import "github.com/0xzer/snapper/protos"

func ExtractConversationStatusFromEntry(entry *protos.ConversationEntry) ConversationStatus {
	if entry == nil {
		return ConversationStatus{}
	}
	status := ConversationStatus{
		ConversationID: uuidToString(entry.GetVersionInfo().GetConversationId()),
	}
	for _, participant := range entry.GetParticipants() {
		status.Participants = append(status.Participants, ParticipantStatus{UserID: uuidToString(participant)})
	}
	return status
}

func ExtractConversationStatusFromDelta(resp *protos.DeltaSyncResponse) ConversationStatus {
	if resp == nil {
		return ConversationStatus{}
	}
	status := conversationStatusFromConversation(resp.GetConversation())
	if status.ConversationID == "" {
		status = conversationStatusFromLightweight(resp.GetLightweightConversation())
	}
	if status.ConversationID == "" {
		status = ExtractConversationStatusFromEntry(resp.GetFeedEntry())
	}
	status.FeedOpenedMessageDisplayTimestamp = resp.GetFeedOpenedMessageDisplayTimestamp()
	return status
}

func ExtractMessageStatus(msg *protos.ContentMessage) MessageStatus {
	if msg == nil || msg.GetMetaData() == nil {
		return MessageStatus{}
	}
	meta := msg.GetMetaData()
	return MessageStatus{
		ReadTimestamp:       meta.GetReadTimestamp(),
		ReadBy:              uuidListToStrings(meta.GetReadBy()),
		ReleasedBy:          uuidListToStrings(meta.GetReleasedBy()),
		SavedBy:             uuidListToStrings(meta.GetSavedBy()),
		ScreenshottedBy:     uuidListToStrings(meta.GetScreenshottedBy()),
		ScreenRecordedBy:    uuidListToStrings(meta.GetScreenRecordedBy()),
		ReplayedBy:          uuidListToStrings(meta.GetReplayedBy()),
		ConversationVersion: meta.GetConversationVersion(),
	}
}

func ExtractReadUpdateStatus(resp *protos.UpdateContentMessageResponse) (messageID int64, readTimestamp int64, ok bool) {
	if resp == nil || !resp.GetSuccess() {
		return 0, 0, false
	}
	if read := resp.GetRead(); read != nil {
		return 0, read.GetReadTimestamp(), true
	}
	if statusMessage := resp.GetStatusMessage(); statusMessage != nil {
		return statusMessage.GetMessageId(), statusMessage.GetMetaData().GetReadTimestamp(), true
	}
	return 0, 0, false
}

func conversationStatusFromConversation(conv *protos.Conversation) ConversationStatus {
	if conv == nil {
		return ConversationStatus{}
	}
	status := ConversationStatus{
		ConversationID: uuidToString(conv.GetConversationId()),
	}
	for _, participant := range conv.GetParticipants() {
		status.Participants = append(status.Participants, participantStatusFromParticipant(participant))
	}
	return status
}

func conversationStatusFromLightweight(conv *protos.LightweightConversation) ConversationStatus {
	if conv == nil {
		return ConversationStatus{}
	}
	status := ConversationStatus{
		ConversationID: uuidToString(conv.GetConversationId()),
	}
	for _, participant := range conv.GetActiveParticipants() {
		status.Participants = append(status.Participants, participantStatusFromActive(participant))
	}
	return status
}

func participantStatusFromParticipant(participant *protos.Participant) ParticipantStatus {
	if participant == nil {
		return ParticipantStatus{}
	}
	return ParticipantStatus{
		UserID:                    uuidToString(participant.GetUserId()),
		ReadHighWatermark:         participant.GetReadHighWatermark(),
		ReleaseHighWatermark:      participant.GetReleaseHighWatermark(),
		SnapReleaseHighWatermark:  participant.GetSnapReleaseHighWatermark(),
		ReactionReadHighWatermark: participant.GetReactionReadHighWatermark(),
	}
}

func participantStatusFromActive(participant *protos.ActiveParticipantData) ParticipantStatus {
	if participant == nil {
		return ParticipantStatus{}
	}
	return ParticipantStatus{
		UserID:                    uuidToString(participant.GetParticipantId()),
		ReadHighWatermark:         participant.GetReadHighWatermark(),
		SnapReleaseHighWatermark:  participant.GetSnapReleaseHighWatermark(),
		ReactionReadHighWatermark: participant.GetReactionReadHighWatermark(),
		ReleaseWatermark:          participant.GetReleaseWatermark(),
	}
}

func uuidListToStrings(items []*protos.UUID) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if id := uuidToString(item); id != "" {
			out = append(out, id)
		}
	}
	return out
}
