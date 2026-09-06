package connector

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/matrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func resyncHTTPError(status int, code string) error {
	return mautrix.HTTPError{Response: &http.Response{StatusCode: status}, RespError: &mautrix.RespError{ErrCode: code}}
}

func TestResyncConclusiveMatrixMiss(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"not found", resyncHTTPError(404, "M_NOT_FOUND"), true},
		{"forbidden", resyncHTTPError(403, "M_FORBIDDEN"), true},
		{"wrapped", fmt.Errorf("probe: %w", resyncHTTPError(404, "M_NOT_FOUND")), true},
		{"token", resyncHTTPError(403, "M_UNKNOWN_TOKEN"), false},
		{"missing token", resyncHTTPError(403, "M_MISSING_TOKEN"), false},
		{"unauthorized", resyncHTTPError(401, "M_FORBIDDEN"), false},
		{"rate limit", resyncHTTPError(429, "M_LIMIT_EXCEEDED"), false},
		{"server", resyncHTTPError(503, "M_NOT_FOUND"), false},
		{"plain HTTP", mautrix.HTTPError{Response: &http.Response{StatusCode: 404}}, false},
		{"bare Matrix error", mautrix.MNotFound, false},
		{"transport", errors.New("connection reset"), false},
		{"cancelled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
		{"success", nil, false},
		{"retryable", mautrix.HTTPError{Response: &http.Response{StatusCode: 404}, RespError: &mautrix.RespError{ErrCode: "M_NOT_FOUND", CanRetry: true}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := conclusiveMatrixMiss(tc.err); got != tc.want {
				t.Fatalf("conclusiveMatrixMiss = %t, want %t", got, tc.want)
			}
		})
	}
}

type resyncProber struct {
	reads                []error
	join                 error
	readCount, joinCount int
}

func (p *resyncProber) Members(context.Context, id.RoomID, ...mautrix.ReqMembers) (*mautrix.RespMembers, error) {
	err := p.reads[p.readCount]
	p.readCount++
	return &mautrix.RespMembers{}, err
}

func (p *resyncProber) JoinRoomByID(context.Context, id.RoomID) (*mautrix.RespJoinRoom, error) {
	p.joinCount++
	return &mautrix.RespJoinRoom{}, p.join
}

func TestResyncLiveRoomDead(t *testing.T) {
	miss := resyncHTTPError(404, "M_NOT_FOUND")
	for _, tc := range []struct {
		name            string
		identity, phase int
		result          error
		dead, wantErr   bool
	}{
		{"all conclusively dead", -1, -1, nil, true, false},
		{"bot readable", 0, 0, nil, false, false},
		{"ghost readable", 1, 0, nil, false, false},
		{"dp readable", 2, 0, nil, false, false},
		{"bot rejoins", 0, 1, nil, false, false},
		{"ghost rejoins", 1, 1, nil, false, false},
		{"dp rejoins", 2, 1, nil, false, false},
		{"second probe readable", 2, 2, nil, false, false},
		{"initial auth failure", 0, 0, resyncHTTPError(403, "M_UNKNOWN_TOKEN"), false, true},
		{"join transient", 1, 1, resyncHTTPError(503, "M_UNKNOWN"), false, true},
		{"join auth failure", 2, 1, resyncHTTPError(401, "M_UNKNOWN_TOKEN"), false, true},
		{"second probe timeout", 2, 2, context.DeadlineExceeded, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probers := make([]liveRoomProber, 3)
			for i := range probers {
				p := &resyncProber{reads: []error{miss, miss}, join: miss}
				if i == tc.identity {
					if tc.phase == 1 {
						p.join = tc.result
					} else {
						p.reads[tc.phase/2] = tc.result
					}
				}
				probers[i] = p
			}
			dead, err := liveRoomDead(context.Background(), "!room:test", probers)
			if dead != tc.dead || (err != nil) != tc.wantErr {
				t.Fatalf("dead=%t err=%v, want dead=%t error=%t", dead, err, tc.dead, tc.wantErr)
			}
			if dead {
				for _, prober := range probers {
					p := prober.(*resyncProber)
					if p.readCount != 2 || p.joinCount != 1 {
						t.Fatal("dead classification skipped an identity probe or recovery join")
					}
				}
			}
		})
	}
	if dead, err := liveRoomDead(context.Background(), "!room:test", nil); dead || err == nil {
		t.Fatal("no live identities must preserve the room")
	}
}

func TestResyncLiveSpaceProbes(t *testing.T) {
	requests := 0
	status, body := http.StatusOK, `{"chunk":[]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected mutation: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	}))
	defer server.Close()
	client, err := mautrix.NewClient(server.URL, "@bot:test", "test")
	if err != nil {
		t.Fatal(err)
	}
	client.DefaultHTTPRetries = 0
	client.StateStore = mautrix.NewMemoryStateStore()
	if err := client.StateStore.SetMembership(context.Background(), "!space:test", client.UserID, event.MembershipJoin); err != nil {
		t.Fatal(err)
	}
	sa := &SnapchatAPI{UserLogin: &bridgev2.UserLogin{
		UserLogin: &database.UserLogin{SpaceRoom: "!space:test"},
		Bridge:    &bridgev2.Bridge{Matrix: &matrix.Connector{Bot: &appservice.IntentAPI{Client: client}}},
	}}
	if _, err := sa.existingSpaceRoom(context.Background()); err != nil || requests != 1 {
		t.Fatalf("space must be validated live: requests=%d err=%v", requests, err)
	}
	status, body = 404, `{"errcode":"M_NOT_FOUND","error":"gone"}`
	if _, err := sa.existingSpaceRoom(context.Background()); err == nil || requests != 2 {
		t.Fatal("cached membership masked dead PFS")
	}
	sa.UserLogin.SpaceRoom = ""
	if _, err := sa.existingSpaceRoom(context.Background()); err == nil || requests != 2 {
		t.Fatal("missing PFS must abort without creating a space")
	}
	for _, tc := range []struct {
		body            string
		status          int
		linked, wantErr bool
	}{
		{`{"via":["test"]}`, 200, true, false},
		{`{"via":[]}`, 200, false, false},
		{`{}`, 200, false, false},
		{`{"errcode":"M_NOT_FOUND"}`, 404, false, false},
		{`{"errcode":"M_FORBIDDEN"}`, 403, false, true},
		{`{"errcode":"M_UNKNOWN_TOKEN"}`, 403, false, true},
	} {
		body, status = tc.body, tc.status
		linked, err := sa.spaceChildLinked(context.Background(), "!space:test", "!room:test")
		if linked != tc.linked || (err != nil) != tc.wantErr {
			t.Fatalf("child %s: linked=%t err=%v", body, linked, err)
		}
	}
}
