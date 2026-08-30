package connector

import (
	"testing"
)

// TestDiffReadWatermarksFirstSyncEmitsReceipts verifies that on the first sync
// after bridge start, every non-self participant with a watermark produces a
// receipt (so freshly recreated Beeper rooms get the correct seen state).
func TestDiffReadWatermarksFirstSyncEmitsReceipts(t *testing.T) {
	sa := &SnapchatAPI{readWatermarks: make(map[string]map[string]int64)}
	incoming := map[string]int64{
		"self-participant": 50,
		"other-1":          42,
		"other-2":          7,
	}
	advances := sa.diffReadWatermarks("chat-1", incoming, "self-participant")
	if len(advances) != 2 {
		t.Fatalf("advances = %d, want 2 (self must be excluded)", len(advances))
	}
	seen := map[string]int64{}
	for _, adv := range advances {
		seen[adv.Participant] = adv.Watermark
	}
	if seen["other-1"] != 42 || seen["other-2"] != 7 {
		t.Fatalf("unexpected advances: %v", advances)
	}
}

// TestDiffReadWatermarksNoReemitForUnchanged verifies unchanged or lower
// watermarks do not re-emit receipts, and that an advance re-emits once.
func TestDiffReadWatermarksNoReemitForUnchanged(t *testing.T) {
	sa := &SnapchatAPI{readWatermarks: make(map[string]map[string]int64)}
	incoming := map[string]int64{"other-1": 42}
	if advances := sa.diffReadWatermarks("chat-1", incoming, "self"); len(advances) != 1 {
		t.Fatalf("first diff advances = %d, want 1", len(advances))
	}
	// Same watermark: no re-emission.
	if advances := sa.diffReadWatermarks("chat-1", incoming, "self"); len(advances) != 0 {
		t.Fatalf("unchanged watermark emitted %d receipts, want 0", len(advances))
	}
	// Lower watermark (shouldn't happen, but must not emit or regress state).
	if advances := sa.diffReadWatermarks("chat-1", map[string]int64{"other-1": 10}, "self"); len(advances) != 0 {
		t.Fatalf("lower watermark emitted %d receipts, want 0", len(advances))
	}
	// Advance: emits.
	if advances := sa.diffReadWatermarks("chat-1", map[string]int64{"other-1": 50}, "self"); len(advances) != 1 {
		t.Fatalf("advance emitted %d receipts, want 1", len(advances))
	}
	// State must have moved to 50.
	if got := sa.readWatermarks["chat-1"]["other-1"]; got != 50 {
		t.Fatalf("stored watermark = %d, want 50", got)
	}
}

// TestDiffReadWatermarksSkipsZeroAndEmpty verifies zero/empty watermarks never
// emit receipts.
func TestDiffReadWatermarksSkipsZeroAndEmpty(t *testing.T) {
	sa := &SnapchatAPI{readWatermarks: make(map[string]map[string]int64)}
	if advances := sa.diffReadWatermarks("chat-1", map[string]int64{"other-1": 0}, "self"); len(advances) != 0 {
		t.Fatalf("zero watermark emitted %d receipts, want 0", len(advances))
	}
	if advances := sa.diffReadWatermarks("chat-1", nil, "self"); len(advances) != 0 {
		t.Fatalf("empty map emitted %d receipts, want 0", len(advances))
	}
}
