package connector

import (
	"context"
	"fmt"
	"log"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/store"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
)

type SnapchatLogin struct {
	User      *bridgev2.User
	Connector *SnapchatConnector
}

var _ bridgev2.LoginProcess = (*SnapchatLogin)(nil)

func (sl *SnapchatLogin) Start(ctx context.Context) (*bridgev2.LoginStep, error) {
	client := sl.Connector.newClient()
	log.Printf("bridgev2 login: starting browser-session flow via %s", client.BaseURL())
	status, err := client.Status(ctx)
	if err != nil {
		log.Printf("bridgev2 login: connector status failed via %s: %v", client.BaseURL(), err)
	} else {
		log.Printf("bridgev2 login: connector status via %s => state=%s authenticated=%t url=%s visibleChats=%d", client.BaseURL(), status.State, status.Authenticated, status.URL, status.VisibleChatCount)
	}
	if err != nil || !status.Authenticated {
		log.Printf("bridgev2 login: requesting connector session start via %s", client.BaseURL())
		if err := client.StartSession(ctx); err != nil {
			log.Printf("bridgev2 login: connector session start failed via %s: %v", client.BaseURL(), err)
			return nil, fmt.Errorf("failed to start browser session: %w", err)
		}
		log.Printf("bridgev2 login: waiting for connector authentication via %s", client.BaseURL())

		status, err = waitForAuthenticatedSession(ctx, client, sl.Connector.loginWaitDuration())
		if err != nil {
			log.Printf("bridgev2 login: wait for connector authentication failed via %s: %v", client.BaseURL(), err)
			return nil, err
		}
	}
	log.Printf("bridgev2 login: connector authenticated via %s => url=%s", client.BaseURL(), status.URL)

	meta := &UserLoginMetadata{
		Label:         "browser-session",
		Authenticated: true,
		LastURL:       status.URL,
	}

	if sl.Connector != nil && sl.Connector.store != nil {
		_ = sl.Connector.store.UpsertLogin(store.LoginState{
			UserID:        string(makeUserLoginID(meta.Label)),
			RemoteID:      status.URL,
			RemoteName:    "Snapchat Web",
			SessionJSON:   store.MarshalJSON(status),
			LastSeenState: status.State,
		})
	}

	if err := sl.Connector.resetSyncState(); err != nil {
		log.Printf("bridgev2 login: failed to reset sync state before relogin: %v", err)
	} else {
		log.Printf("bridgev2 login: reset sync state for fresh resync")
	}

	api := NewSnapchatAPI(sl.Connector, nil, meta.Label)
	api.updateLoginState(status)
	ul, err := sl.User.NewLogin(ctx, &database.UserLogin{
		ID:         makeUserLoginID(meta.Label),
		RemoteName: "Snapchat Web",
		Metadata:   meta,
	}, &bridgev2.NewLoginParams{
		LoadUserLogin: func(ctx context.Context, login *bridgev2.UserLogin) error {
			api.UserLogin = login
			login.Client = api
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create login: %w", err)
	}
	if ul.Client != nil && ul.Bridge != nil {
		connectCtx := ul.Log.WithContext(ul.Bridge.BackgroundCtx)
		log.Printf("bridgev2 login: starting fresh user login sync id=%s", ul.ID)
		ul.Client.Connect(connectCtx)
	}

	return &bridgev2.LoginStep{
		Type:         bridgev2.LoginStepTypeComplete,
		StepID:       "com.colej.snapchat.browser.complete",
		Instructions: "Snapchat Web session is authenticated.",
		CompleteParams: &bridgev2.LoginCompleteParams{
			UserLoginID: ul.ID,
			UserLogin:   ul,
		},
	}, nil
}

func (sl *SnapchatLogin) Cancel() {}

func waitForAuthenticatedSession(ctx context.Context, client *sidecar.Client, timeout time.Duration) (*sidecar.SessionStatus, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		status, err := client.Status(waitCtx)
		if err == nil && status.Authenticated {
			return status, nil
		}
		if err != nil {
			log.Printf("bridgev2 login: polling connector status via %s failed: %v", client.BaseURL(), err)
		} else {
			log.Printf("bridgev2 login: polling connector status via %s => state=%s authenticated=%t url=%s", client.BaseURL(), status.State, status.Authenticated, status.URL)
		}

		select {
		case <-waitCtx.Done():
			return nil, fmt.Errorf("session is not authenticated yet: log into Snapchat Web, then retry the login flow")
		case <-ticker.C:
		}
	}
}
