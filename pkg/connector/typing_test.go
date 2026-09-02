package connector

import "testing"

// TestDiffTypingState verifies the typing transition diff: new typers are
// reported as started, missing typers as stopped, and unchanged typers are
// silent (no duplicate spam when the watermark/state has not advanced).
func TestDiffTypingState(t *testing.T) {
	previous := map[string]map[string]bool{
		"chat-a": {"alice": true, "bob": true},
		"chat-b": {"carol": true},
	}
	current := map[string]map[string]bool{
		"chat-a": {"alice": true, "dave": true},
		"chat-c": {"erin": true},
	}

	changes := diffTypingState(previous, current)

	stopped := map[string]bool{}
	started := map[string]bool{}
	for _, change := range changes {
		key := change.ChatID + "/" + change.Participant
		if change.Typing {
			started[key] = true
		} else {
			stopped[key] = true
		}
	}

	if !stopped["chat-a/bob"] {
		t.Fatal("expected bob to stop typing in chat-a")
	}
	if !stopped["chat-b/carol"] {
		t.Fatal("expected carol to stop typing in chat-b (chat gone)")
	}
	if !started["chat-a/dave"] {
		t.Fatal("expected dave to start typing in chat-a")
	}
	if !started["chat-c/erin"] {
		t.Fatal("expected erin to start typing in chat-c (new chat)")
	}
	if started["chat-a/alice"] || stopped["chat-a/alice"] {
		t.Fatal("alice unchanged: no transition should be emitted")
	}
	if len(changes) != 4 {
		t.Fatalf("changes = %d, want 4", len(changes))
	}
}

// TestDiffTypingStateEmpty verifies transitions when everything stops.
func TestDiffTypingStateEmpty(t *testing.T) {
	previous := map[string]map[string]bool{
		"chat-a": {"alice": true},
	}
	changes := diffTypingState(previous, map[string]map[string]bool{})
	if len(changes) != 1 || changes[0].ChatID != "chat-a" || changes[0].Participant != "alice" || changes[0].Typing {
		t.Fatalf("unexpected changes: %+v", changes)
	}
}
