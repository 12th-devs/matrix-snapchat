package connector

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/colej/mautrix-snapchat/internal/snapapi"
)

func unauthorizedErr() error {
	return fmt.Errorf("%w: POST .../SyncConversations returned 403 Forbidden: unauthorized", snapapi.ErrUnauthorized)
}

func unrelatedErr() error {
	return errors.New("POST .../SyncConversations returned 500 internal server error")
}

// TestSingleUnauthorizedDoesNotMarkDead verifies a single/transient messaging
// auth failure counts but does not mark the session dead or suppress polling.
func TestSingleUnauthorizedDoesNotMarkDead(t *testing.T) {
	sa := &SnapchatAPI{}
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

	sa.recordSyncFailure(unauthorizedErr(), now)
	if sa.reLoginRequired {
		t.Fatal("a single unauthorized failure must not mark the session dead")
	}
	if sa.consecutiveUnauthorized != 1 {
		t.Fatalf("consecutiveUnauthorized = %d, want 1", sa.consecutiveUnauthorized)
	}
	if sa.syncSuppressed(now) {
		t.Fatal("polling must not be suppressed by a single auth failure")
	}
}

// TestUnrelatedErrorsAreNeutral verifies non-auth errors neither increment nor
// reset the consecutive unauthorized count.
func TestUnrelatedErrorsAreNeutral(t *testing.T) {
	sa := &SnapchatAPI{}
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

	sa.recordSyncFailure(unauthorizedErr(), now)
	sa.recordSyncFailure(unauthorizedErr(), now)
	if got := sa.consecutiveUnauthorized; got != 2 {
		t.Fatalf("consecutiveUnauthorized = %d, want 2", got)
	}

	sa.recordSyncFailure(unrelatedErr(), now)
	if got := sa.consecutiveUnauthorized; got != 2 {
		t.Fatalf("unrelated error changed the count: got %d, want 2 (must not increment)", got)
	}
	if sa.reLoginRequired {
		t.Fatal("unrelated errors must not mark the session dead")
	}

	// The count continues from where it was (not reset by the unrelated error).
	sa.recordSyncFailure(unauthorizedErr(), now)
	if got := sa.consecutiveUnauthorized; got != 3 {
		t.Fatalf("consecutiveUnauthorized = %d, want 3 (unrelated error must not reset)", got)
	}

	// Interleaving many unrelated errors past the threshold never marks dead.
	for i := 0; i < reLoginUnauthorizedThreshold+2; i++ {
		sa.recordSyncFailure(unrelatedErr(), now)
	}
	if sa.reLoginRequired {
		t.Fatal("unrelated errors must never mark the session dead")
	}
	if got := sa.consecutiveUnauthorized; got != 3 {
		t.Fatalf("consecutiveUnauthorized = %d, want 3 after unrelated errors", got)
	}
	if sa.syncSuppressed(now) {
		t.Fatal("unrelated errors must not suppress polling")
	}
}

// TestThresholdCrossingMarksDead verifies consecutive messaging-auth failures
// cross the configured threshold exactly at the boundary.
func TestThresholdCrossingMarksDead(t *testing.T) {
	sa := &SnapchatAPI{}
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

	for i := 1; i <= reLoginUnauthorizedThreshold; i++ {
		sa.recordSyncFailure(unauthorizedErr(), now)
		wantDead := i == reLoginUnauthorizedThreshold
		if sa.reLoginRequired != wantDead {
			t.Fatalf("after %d consecutive failures: dead = %v, want %v", i, sa.reLoginRequired, wantDead)
		}
	}
	if !sa.nextRecoveryCheckAt.After(now) {
		t.Fatalf("expected recovery check scheduled after %v, got %v", now, sa.nextRecoveryCheckAt)
	}
}

// TestRecoveryGateLifecycle walks the full deterministic gate lifecycle with a
// synthetic clock: threshold -> suppressed -> one bounded attempt -> failed
// recovery re-closes the gate -> success clears everything and resumes polling.
func TestRecoveryGateLifecycle(t *testing.T) {
	sa := &SnapchatAPI{}
	start := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

	// Threshold reached -> dead session.
	for i := 0; i < reLoginUnauthorizedThreshold; i++ {
		sa.recordSyncFailure(unauthorizedErr(), start)
	}
	if !sa.reLoginRequired {
		t.Fatal("threshold crossing must mark the session dead")
	}

	// Normal polling suppressed throughout the recovery window.
	if !sa.syncSuppressed(start) {
		t.Fatal("polling must be suppressed immediately after the session is marked dead")
	}
	if !sa.syncSuppressed(start.Add(reLoginRecoveryInterval - time.Second)) {
		t.Fatal("polling must stay suppressed just before the recovery interval elapses")
	}

	// One bounded recovery attempt is permitted at the interval boundary.
	retryAt := start.Add(reLoginRecoveryInterval)
	if sa.syncSuppressed(retryAt) {
		t.Fatal("a bounded recovery attempt must be permitted once the interval elapses")
	}

	// Failed recovery: dead state remains and the gate closes again.
	sa.recordSyncFailure(unauthorizedErr(), retryAt)
	if !sa.reLoginRequired {
		t.Fatal("a failed recovery attempt must keep the session dead")
	}
	if !sa.syncSuppressed(retryAt) {
		t.Fatal("gate must close again immediately after a failed recovery attempt")
	}
	if !sa.syncSuppressed(retryAt.Add(reLoginRecoveryInterval - time.Second)) {
		t.Fatal("gate must stay closed for a fresh recovery interval after a failed attempt")
	}

	// Exactly one bounded attempt per window (no hot-looping at poll rate).
	nextRetryAt := retryAt.Add(reLoginRecoveryInterval)
	if sa.syncSuppressed(nextRetryAt) {
		t.Fatal("the next bounded recovery attempt must be permitted after the fresh interval")
	}

	// Successful recovery: dead state clears, counter resets, polling resumes.
	sa.recordSyncSuccess(nextRetryAt)
	if sa.reLoginRequired {
		t.Fatal("successful sync must clear the dead state")
	}
	if got := sa.consecutiveUnauthorized; got != 0 {
		t.Fatalf("successful sync must reset the counter, got %d", got)
	}
	if !sa.nextRecoveryCheckAt.IsZero() {
		t.Fatalf("successful sync must clear the recovery gate, got %v", sa.nextRecoveryCheckAt)
	}
	if sa.syncSuppressed(nextRetryAt) || sa.syncSuppressed(nextRetryAt.Add(time.Hour)) {
		t.Fatal("polling must resume normally after a successful recovery")
	}
}

// TestSyncSuccessResetsCounterMidSequence verifies a successful sync resets the
// consecutive-auth-failure counter so the threshold only counts *consecutive*
// failures.
func TestSyncSuccessResetsCounterMidSequence(t *testing.T) {
	sa := &SnapchatAPI{}
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

	sa.recordSyncFailure(unauthorizedErr(), now)
	sa.recordSyncFailure(unauthorizedErr(), now)
	sa.recordSyncSuccess(now)
	if got := sa.consecutiveUnauthorized; got != 0 {
		t.Fatalf("successful sync must reset the counter, got %d", got)
	}

	// A single failure after a success must not mark dead (threshold restarted).
	sa.recordSyncFailure(unauthorizedErr(), now)
	if sa.reLoginRequired {
		t.Fatal("counter must restart after a successful sync")
	}
	if got := sa.consecutiveUnauthorized; got != 1 {
		t.Fatalf("consecutiveUnauthorized = %d, want 1 after reset", got)
	}
}
