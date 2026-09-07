package connector

import (
	"testing"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/store"
	"maunium.net/go/mautrix/bridgev2/database"
)

func recoveredTestMessage() sidecar.Message {
	return sidecar.Message{
		ID:   "4114",
		Text: "Media",
		Media: []sidecar.MediaAttachment{{
			ID:       "4114-ref-0",
			MimeType: "image/jpeg",
		}},
	}
}

func TestDecideMediaHydrationRecoveryRule(t *testing.T) {
	now := time.Now()
	stale := &store.MessageState{PortalKey: "chat-1", RemoteID: "4114", Kind: "media", HasMedia: true, HydratedAt: &now}
	fresh := &store.MessageState{PortalKey: "chat-1", RemoteID: "4115", Kind: "media", HasMedia: true}

	cases := []struct {
		name          string
		existing      *store.MessageState
		confirmed     bool
		isSnap        bool
		snapKeysReady bool
		want          mediaHydrationDecision
	}{
		{"stale hydration without delivery proof recovers", stale, false, false, false, mediaHydrationRecover},
		{"stale hydration with delivery proof skips", stale, true, false, false, mediaHydrationSkip},
		{"no hydration with media renders normally", fresh, false, false, false, mediaHydrationRender},
		{"unknown message with media renders normally", nil, false, false, false, mediaHydrationRender},
		{"hydrated snap without keys stays skipped", stale, false, true, false, mediaHydrationSkip},
		{"unhydrated snap without keys stays skipped", fresh, false, true, false, mediaHydrationSkip},
		{"delivered media skips without redownload", stale, true, false, false, mediaHydrationSkip},
		{"snap with recovered keys renders", nil, false, true, true, mediaHydrationRender},
		{"delivered snap with keys skips", stale, true, true, true, mediaHydrationSkip},
	}
	for _, tc := range cases {
		if got := decideMediaHydration(tc.existing, tc.confirmed, tc.isSnap, tc.snapKeysReady); got != tc.want {
			t.Fatalf("%s: decision = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestClassifyMessageSyncEditsRecoveredMedia(t *testing.T) {
	// After a recovery demotion (hydrated_at cleared) the ordinary hydration
	// architecture applies: the existing Matrix event is edited in place
	// instead of creating a second Snapchat message.
	existing := &store.MessageState{PortalKey: "chat-1", RemoteID: "4114", Kind: "media", HasMedia: true, Text: "Media"}
	if got := classifyMessageSync(existing, recoveredTestMessage()); got != messageSyncEdit {
		t.Fatalf("recovered media classify = %v, want messageSyncEdit", got)
	}
}

func TestMediaDeliveryConfirmedRequiresBridgeState(t *testing.T) {
	// Without bridge mapping state there is no delivery proof, so the confirm
	// helper must report false regardless of connector hydration state.
	sa := &SnapchatAPI{}
	if sa.mediaDeliveryConfirmed(nil, "chat-1", "4114", recoveredTestMessage(), 1) {
		t.Fatal("delivery confirmation requires bridge mapping state")
	}
}

func TestDeliveredMediaPartsRequireFullProof(t *testing.T) {
	base := &MediaDeliveryMetadata{MediaDelivered: true, AttachmentCount: 1}
	good := []*database.Message{{MXID: "$real", Metadata: base}}
	if !deliveredMediaParts(good, 1) {
		t.Fatal("fully delivered mapping must count as proof")
	}
	for name, parts := range map[string][]*database.Message{
		"no-metadata":          {{MXID: "$real"}},
		"not-delivered":        {{MXID: "$real", Metadata: &MediaDeliveryMetadata{}}},
		"count-mismatch":       {{MXID: "$real", Metadata: &MediaDeliveryMetadata{MediaDelivered: true, AttachmentCount: 2}}},
		"missing-mxid":         {{Metadata: base}},
		"too-few-parts":        {},
		"stale-hydrated-shape": {{MXID: "$real", Metadata: &MediaDeliveryMetadata{MediaDelivered: false, AttachmentCount: 1}}},
	} {
		if deliveredMediaParts(parts, 1) {
			t.Fatalf("%s must not count as delivery proof", name)
		}
	}
}
