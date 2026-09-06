package connector

import (
	"context"
	"errors"
	"testing"
)

type fakeRemoteNameLogin struct {
	remoteName string
	saveErr    error
	saves      int
	sends      int
}

func (f *fakeRemoteNameLogin) RemoteName() string { return f.remoteName }

func (f *fakeRemoteNameLogin) SetRemoteName(name string) { f.remoteName = name }

func (f *fakeRemoteNameLogin) Save(context.Context) error {
	f.saves++
	return f.saveErr
}

func (f *fakeRemoteNameLogin) SendConnectedState() { f.sends++ }

func TestApplyRemoteNameUpdateUnchangedDoesNothing(t *testing.T) {
	login := &fakeRemoteNameLogin{remoteName: "Cole Fekete"}
	applyRemoteNameUpdate(context.Background(), login, "Cole Fekete", "cole.fekete")
	if login.remoteName != "Cole Fekete" {
		t.Fatalf("remoteName = %q, want unchanged", login.remoteName)
	}
	if login.saves != 0 || login.sends != 0 {
		t.Fatalf("saves = %d, sends = %d, want 0 and 0", login.saves, login.sends)
	}
}

func TestApplyRemoteNameUpdateChangedSavesAndPropagatesOnce(t *testing.T) {
	login := &fakeRemoteNameLogin{remoteName: "Snapchat Web"}
	applyRemoteNameUpdate(context.Background(), login, "Cole Fekete", "cole.fekete")
	if login.remoteName != "Cole Fekete" {
		t.Fatalf("remoteName = %q, want %q", login.remoteName, "Cole Fekete")
	}
	if login.saves != 1 {
		t.Fatalf("saves = %d, want 1", login.saves)
	}
	if login.sends != 1 {
		t.Fatalf("sends = %d, want 1", login.sends)
	}
	applyRemoteNameUpdate(context.Background(), login, "Cole Fekete", "cole.fekete")
	if login.saves != 1 || login.sends != 1 {
		t.Fatalf("repeat resolution: saves = %d, sends = %d, want 1 and 1", login.saves, login.sends)
	}
}

func TestApplyRemoteNameUpdateNoPropagationWhenSaveFails(t *testing.T) {
	login := &fakeRemoteNameLogin{remoteName: "Snapchat Web", saveErr: errors.New("db down")}
	applyRemoteNameUpdate(context.Background(), login, "Cole Fekete", "")
	if login.saves != 1 {
		t.Fatalf("saves = %d, want 1", login.saves)
	}
	if login.sends != 0 {
		t.Fatalf("sends = %d, want 0 after failed save", login.sends)
	}
}

func TestApplyRemoteNameUpdateFallbacks(t *testing.T) {
	login := &fakeRemoteNameLogin{remoteName: "Snapchat Web"}
	applyRemoteNameUpdate(context.Background(), login, "", "cole.fekete")
	if login.remoteName != "cole.fekete" || login.sends != 1 {
		t.Fatalf("username fallback: remoteName = %q, sends = %d", login.remoteName, login.sends)
	}
	fallback := &fakeRemoteNameLogin{remoteName: "Snapchat Web"}
	applyRemoteNameUpdate(context.Background(), fallback, "", "")
	if fallback.remoteName != "Snapchat" || fallback.sends != 1 {
		t.Fatalf("empty fallback: remoteName = %q, sends = %d", fallback.remoteName, fallback.sends)
	}
}
