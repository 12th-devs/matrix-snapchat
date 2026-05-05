package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/colej/mautrix-snapchat/internal/config"
	"github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/store"
)

type App struct {
	cfg       *config.Config
	connector *connector.Client
	store     *store.Store
	server    *http.Server
	lastState string
}

type HealthResponse struct {
	OK        bool                     `json:"ok"`
	Connector *connector.SessionStatus `json:"connector,omitempty"`
}

func New(cfg *config.Config) (*App, error) {
	if cfg.Connector.BaseURL == "" {
		return nil, errors.New("connector.base_url is required")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Database.URI), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	stateStore, err := store.New(cfg.Database.URI)
	if err != nil {
		return nil, fmt.Errorf("create state store: %w", err)
	}

	app := &App{
		cfg:       cfg,
		connector: connector.New(cfg.Connector),
		store:     stateStore,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", app.handleHealth)
	mux.HandleFunc("/connector/start", app.handleStartConnector)
	mux.HandleFunc("/connector/chats", app.handleChats)
	mux.HandleFunc("/connector/messages", app.handleMessages)
	mux.HandleFunc("/connector/send", app.handleSend)
	mux.HandleFunc("/connector/debug", app.handleDebug)

	app.server = &http.Server{
		Addr:              cfg.Bridge.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		log.Printf("bridge listening on %s", a.cfg.Bridge.Listen)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ticker := time.NewTicker(time.Duration(a.cfg.Network.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := a.server.Shutdown(shutdownCtx); err != nil {
				return err
			}
			return a.store.Close()
		case err := <-errCh:
			return err
		case <-ticker.C:
			a.pollConnector(ctx)
		}
	}
}

func (a *App) pollConnector(ctx context.Context) {
	status, err := a.connector.Status(ctx)
	if err != nil {
		log.Printf("connector status error: %v", err)
		return
	}

	if status.Authenticated {
		_ = a.store.UpsertLogin(store.LoginState{
			UserID:        "default",
			RemoteID:      status.URL,
			RemoteName:    status.ActiveChatName,
			SessionJSON:   store.MarshalJSON(status),
			LastSeenState: status.State,
		})

		chats, chatErr := a.connector.ListChats(ctx)
		if chatErr == nil {
			for _, chat := range chats {
				_ = a.store.UpsertPortal(store.PortalState{
					PortalKey:    chat.Name,
					RemoteID:     chat.ID,
					RemoteName:   chat.Name,
					Preview:      chat.Preview,
					LastMessage:  chat.LastMessage,
					Unread:       chat.Unread,
					LastSyncedAt: time.Now(),
				})
			}
		}

		if a.lastState != status.State {
			log.Printf("snapchat connector online; active chat=%q visibleChats=%d", status.ActiveChatName, status.VisibleChatCount)
		}
		a.lastState = status.State
		return
	}

	if a.lastState != status.State {
		log.Printf("snapchat connector awaiting login; state=%s", status.State)
	}
	_ = a.store.UpsertLogin(store.LoginState{
		UserID:        "default",
		RemoteID:      status.URL,
		RemoteName:    status.ActiveChatName,
		SessionJSON:   store.MarshalJSON(status),
		LastSeenState: status.State,
	})
	a.lastState = status.State
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	status, err := a.connector.Status(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		OK:        true,
		Connector: status,
	})
}

func (a *App) handleStartConnector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := a.connector.StartSession(r.Context()); err != nil {
		http.Error(w, fmt.Sprintf("start connector: %v", err), http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"started": true})
}

func (a *App) handleChats(w http.ResponseWriter, r *http.Request) {
	chats, err := a.connector.ListChats(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("list chats: %v", err), http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, chats)
}

func (a *App) handleMessages(w http.ResponseWriter, r *http.Request) {
	chatName := r.URL.Query().Get("chatName")
	if chatName == "" {
		http.Error(w, "chatName is required", http.StatusBadRequest)
		return
	}

	messages, err := a.connector.Messages(r.Context(), "", chatName)
	if err != nil {
		http.Error(w, fmt.Sprintf("get messages: %v", err), http.StatusBadGateway)
		return
	}

	states := make([]store.MessageState, 0, len(messages))
	for _, message := range messages {
		states = append(states, store.MessageState{
			PortalKey:    chatName,
			RemoteID:     message.ID,
			Author:       message.Author,
			Text:         message.Text,
			Outgoing:     message.Outgoing,
			TimestampRaw: message.Timestamp,
			LastSeenAt:   time.Now(),
		})
	}
	_ = a.store.UpsertMessages(states)

	writeJSON(w, http.StatusOK, messages)
}

func (a *App) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload connector.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("decode request: %v", err), http.StatusBadRequest)
		return
	}

	if payload.ChatName == "" || payload.Text == "" {
		http.Error(w, "chatName and text are required", http.StatusBadRequest)
		return
	}

	if err := a.connector.SendMessage(r.Context(), payload.ChatID, payload.ChatName, payload.Text); err != nil {
		http.Error(w, fmt.Sprintf("send message: %v", err), http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"queued": true})
}

func (a *App) handleDebug(w http.ResponseWriter, r *http.Request) {
	diagnostics, err := a.connector.Diagnostics(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("connector diagnostics: %v", err), http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, diagnostics)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
