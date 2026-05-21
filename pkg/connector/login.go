package connector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/snapapi"
	"github.com/colej/mautrix-snapchat/internal/store"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
)

type SnapchatLogin struct {
	User      *bridgev2.User
	Connector *SnapchatConnector
	FlowID    string
}

var _ bridgev2.LoginProcess = (*SnapchatLogin)(nil)
var _ bridgev2.LoginProcessCookies = (*SnapchatLogin)(nil)

var snapchatCookieLoginStep = &bridgev2.LoginStep{
	Type:         bridgev2.LoginStepTypeCookies,
	StepID:       "com.colej.snapchat.cookies",
	Instructions: "Log into Snapchat Web on this device. When login finishes, submit the captured Snapchat cookies. The bridge only stores the browser session cookies required for Snapchat Web/API access.",
	CookiesParams: &bridgev2.LoginCookiesParams{
		URL:               "https://www.snapchat.com/web",
		WaitForURLPattern: `^https://(www|web)\.snapchat\.com/web`,
		Fields: []bridgev2.LoginCookieField{
			snapchatCookieField("__Host-sc-a-nonce", true),
			snapchatCookieField("sc-a-nonce", false),
			snapchatCookieField("__Host-sc-a-session", false),
			snapchatCookieField("__Host-sc-a-auth-session", false),
			snapchatCookieField("__Host-X-Snap-Client-Cookie", true),
		},
	},
}

func snapchatCookieField(name string, required bool) bridgev2.LoginCookieField {
	return bridgev2.LoginCookieField{
		ID:       name,
		Required: required,
		Sources: []bridgev2.LoginCookieFieldSource{{
			Type:         bridgev2.LoginCookieTypeCookie,
			Name:         name,
			CookieDomain: ".snapchat.com",
		}},
	}
}

func (sl *SnapchatLogin) Start(ctx context.Context) (*bridgev2.LoginStep, error) {
	if sl.FlowID == "snapchat-web" {
		log.Printf("bridgev2 login: starting Snapchat Web cookie login flow")
		return snapchatCookieLoginStep, nil
	}
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
	return sl.finishLogin(ctx, status, "Snapchat Web")
}

func (sl *SnapchatLogin) SubmitCookies(ctx context.Context, cookies map[string]string) (*bridgev2.LoginStep, error) {
	cookieString, err := snapchatCookieString(cookies)
	if err != nil {
		return nil, err
	}
	apiClient, err := snapapi.New(snapapi.Config{
		CookieString: cookieString,
		Timeout:      sl.Connector.loginWaitDuration(),
	})
	if err != nil {
		return nil, fmt.Errorf("invalid Snapchat Web cookies: %w", err)
	}
	if err = apiClient.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("Snapchat Web cookie authentication failed: %w", err)
	}

	client := sl.Connector.newClient()
	auth, err := client.ImportAuth(ctx, sidecar.ImportAuthRequest{
		CookieString: cookieString,
		SelfUserID:   apiClient.SelfUserID(),
	})
	if err != nil {
		return nil, fmt.Errorf("import Snapchat Web session into connector: %w", err)
	}
	if !auth.Authenticated {
		return nil, fmt.Errorf("connector rejected Snapchat Web session (state=%s)", auth.State)
	}
	status := &sidecar.SessionStatus{
		State:         auth.State,
		Authenticated: true,
		URL:           auth.URL,
	}
	remoteName := strings.TrimSpace(auth.DisplayName)
	if remoteName == "" {
		remoteName = strings.TrimSpace(auth.Username)
	}
	if remoteName == "" {
		remoteName = "Snapchat Web"
	}
	log.Printf("bridgev2 login: Snapchat Web cookie flow authenticated self_user_id=%s identity_source=%s", apiClient.SelfUserID(), auth.IdentitySource)
	return sl.finishLogin(ctx, status, remoteName)
}

func (sl *SnapchatLogin) finishLogin(ctx context.Context, status *sidecar.SessionStatus, remoteName string) (*bridgev2.LoginStep, error) {
	meta := &UserLoginMetadata{
		Label:         "browser-session",
		Authenticated: true,
		LastURL:       status.URL,
	}

	if sl.Connector != nil && sl.Connector.store != nil {
		_ = sl.Connector.store.UpsertLogin(store.LoginState{
			UserID:        string(makeUserLoginID(meta.Label)),
			RemoteID:      status.URL,
			RemoteName:    remoteName,
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
		RemoteName: remoteName,
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

func snapchatCookieString(cookies map[string]string) (string, error) {
	get := func(name string) string {
		return strings.TrimSpace(cookies[name])
	}
	nonce := get("__Host-sc-a-nonce")
	if nonce == "" {
		nonce = get("sc-a-nonce")
	}
	session := get("__Host-sc-a-session")
	sessionName := "__Host-sc-a-session"
	if session == "" {
		session = get("__Host-sc-a-auth-session")
		sessionName = "__Host-sc-a-auth-session"
	}
	client := get("__Host-X-Snap-Client-Cookie")
	var missing []string
	if nonce == "" {
		missing = append(missing, "__Host-sc-a-nonce")
	}
	if session == "" {
		missing = append(missing, "__Host-sc-a-session or __Host-sc-a-auth-session")
	}
	if client == "" {
		missing = append(missing, "__Host-X-Snap-Client-Cookie")
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("missing required Snapchat cookies: %s", strings.Join(missing, ", "))
	}
	return strings.Join([]string{
		"__Host-sc-a-nonce=" + nonce,
		sessionName + "=" + session,
		"__Host-X-Snap-Client-Cookie=" + client,
	}, "; "), nil
}

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
