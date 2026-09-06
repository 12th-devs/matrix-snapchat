package connector

import (
	"testing"
	"time"
)

func TestHasExactRecentOutgoingRemoteID(t *testing.T) {
	sa := &SnapchatAPI{recentOutgoing: map[string][]pendingOutgoing{}}
	sa.recordRecentOutgoing("chat-1", "", "8675309")
	sa.recordRecentOutgoing("chat-1", "", "client-fallback")

	if !sa.hasExactRecentOutgoingRemoteID("chat-1", "8675309") {
		t.Fatal("exact recent outgoing remote ID must match")
	}
	// Media-scoped echo IDs must reduce to the base remote ID.
	if !sa.hasExactRecentOutgoingRemoteID("chat-1", "8675309-media-abc") {
		t.Fatal("media-scoped echo ID must match its base remote ID")
	}
	if sa.hasExactRecentOutgoingRemoteID("chat-1", "11111") {
		t.Fatal("different remote ID must not match")
	}
	if sa.hasExactRecentOutgoingRemoteID("chat-2", "8675309") {
		t.Fatal("other chat must not match")
	}
	if sa.hasExactRecentOutgoingRemoteID("chat-1", "") {
		t.Fatal("empty remote ID must not match")
	}

	// Entries older than the suppression window must not match.
	sa.mu.Lock()
	sa.recentOutgoing["chat-3"] = []pendingOutgoing{{RemoteID: "4242", SentAt: time.Now().Add(-20 * time.Minute)}}
	sa.mu.Unlock()
	if sa.hasExactRecentOutgoingRemoteID("chat-3", "4242") {
		t.Fatal("expired outgoing entry must not match")
	}
}
