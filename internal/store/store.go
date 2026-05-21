package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type LoginState struct {
	UserID        string
	RemoteID      string
	RemoteName    string
	SessionJSON   string
	LastSeenState string
	UpdatedAt     time.Time
}

type PortalState struct {
	PortalKey    string
	RemoteID     string
	RemoteName   string
	OtherUserID  string
	Preview      string
	LastMessage  string
	Unread       bool
	LastSyncedAt time.Time
}

type MessageState struct {
	PortalKey    string
	RemoteID     string
	Author       string
	Text         string
	Outgoing     bool
	TimestampRaw string
	Kind         string
	HasMedia     bool
	HydratedAt   *time.Time
	LastSeenAt   time.Time
}

type APISyncState struct {
	UserID            string
	SyncToken         []byte
	ConversationState map[string]int64
	SelfUserID        string
	UpdatedAt         time.Time
}

type ReadWatermark struct {
	PortalKey string
	MessageID int64
	Version   int64
	UpdatedAt time.Time
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	s := &Store{db: db}
	if err = s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) init() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS login_state (
			user_id TEXT PRIMARY KEY,
			remote_id TEXT,
			remote_name TEXT,
			session_json TEXT,
			last_seen_state TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS portal_state (
			portal_key TEXT PRIMARY KEY,
			remote_id TEXT,
			remote_name TEXT,
			other_user_id TEXT,
			preview TEXT,
			last_message TEXT,
			unread BOOLEAN DEFAULT FALSE,
			last_synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS message_state (
			portal_key TEXT NOT NULL,
			remote_id TEXT NOT NULL,
			author TEXT,
			text TEXT,
			outgoing BOOLEAN DEFAULT FALSE,
			timestamp_raw TEXT,
			kind TEXT,
			has_media BOOLEAN DEFAULT FALSE,
			hydrated_at DATETIME,
			last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (portal_key, remote_id)
		)`,
		`CREATE TABLE IF NOT EXISTS api_sync_state (
			user_id TEXT PRIMARY KEY,
			sync_token BLOB,
			conversation_state_json TEXT,
			self_user_id TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS read_watermark (
			portal_key TEXT PRIMARY KEY,
			message_id INTEGER DEFAULT 0,
			version INTEGER DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_message_state_portal_key ON message_state(portal_key)`,
		`CREATE INDEX IF NOT EXISTS idx_portal_state_remote_name ON portal_state(remote_name)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("init sqlite schema: %w", err)
		}
	}
	if err := s.ensureColumn("message_state", "kind", "kind TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("message_state", "has_media", "has_media BOOLEAN DEFAULT FALSE"); err != nil {
		return err
	}
	if err := s.ensureColumn("message_state", "hydrated_at", "hydrated_at DATETIME"); err != nil {
		return err
	}
	if err := s.ensureColumn("portal_state", "other_user_id", "other_user_id TEXT"); err != nil {
		return err
	}

	return nil
}

func (s *Store) ensureColumn(table, column, definition string) error {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("inspect sqlite table %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue any
		if err = rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if _, err = s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, definition)); err != nil {
		return fmt.Errorf("migrate sqlite table %s add %s: %w", table, column, err)
	}
	return nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) UpsertLogin(state LoginState) error {
	_, err := s.db.Exec(`
		INSERT INTO login_state (user_id, remote_id, remote_name, session_json, last_seen_state, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			remote_id=excluded.remote_id,
			remote_name=excluded.remote_name,
			session_json=excluded.session_json,
			last_seen_state=excluded.last_seen_state,
			updated_at=CURRENT_TIMESTAMP
	`, state.UserID, state.RemoteID, state.RemoteName, state.SessionJSON, state.LastSeenState)
	return err
}

func (s *Store) GetLogin(userID string) (*LoginState, error) {
	row := s.db.QueryRow(`
		SELECT user_id, remote_id, remote_name, session_json, last_seen_state, updated_at
		FROM login_state
		WHERE user_id = ?
	`, userID)

	var state LoginState
	err := row.Scan(
		&state.UserID,
		&state.RemoteID,
		&state.RemoteName,
		&state.SessionJSON,
		&state.LastSeenState,
		&state.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Store) GetReadWatermark(portalKey string) (*ReadWatermark, error) {
	row := s.db.QueryRow(`
		SELECT portal_key, message_id, version, updated_at
		FROM read_watermark
		WHERE portal_key = ?
	`, portalKey)

	var state ReadWatermark
	err := row.Scan(&state.PortalKey, &state.MessageID, &state.Version, &state.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Store) UpsertReadWatermark(state ReadWatermark) error {
	_, err := s.db.Exec(`
		INSERT INTO read_watermark (portal_key, message_id, version, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(portal_key) DO UPDATE SET
			message_id=excluded.message_id,
			version=excluded.version,
			updated_at=CURRENT_TIMESTAMP
	`, state.PortalKey, state.MessageID, state.Version)
	return err
}

func (s *Store) UpsertAPISyncState(state APISyncState) error {
	encodedState, err := json.Marshal(state.ConversationState)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO api_sync_state (user_id, sync_token, conversation_state_json, self_user_id, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			sync_token=excluded.sync_token,
			conversation_state_json=excluded.conversation_state_json,
			self_user_id=excluded.self_user_id,
			updated_at=CURRENT_TIMESTAMP
	`, state.UserID, state.SyncToken, string(encodedState), state.SelfUserID)
	return err
}

func (s *Store) GetAPISyncState(userID string) (*APISyncState, error) {
	row := s.db.QueryRow(`
		SELECT user_id, sync_token, conversation_state_json, self_user_id, updated_at
		FROM api_sync_state
		WHERE user_id = ?
	`, userID)

	var state APISyncState
	var stateJSON string
	err := row.Scan(
		&state.UserID,
		&state.SyncToken,
		&stateJSON,
		&state.SelfUserID,
		&state.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if stateJSON != "" {
		if err = json.Unmarshal([]byte(stateJSON), &state.ConversationState); err != nil {
			return nil, err
		}
	}
	if state.ConversationState == nil {
		state.ConversationState = make(map[string]int64)
	}
	return &state, nil
}

func (s *Store) UpsertPortal(state PortalState) error {
	_, err := s.db.Exec(`
		INSERT INTO portal_state (portal_key, remote_id, remote_name, other_user_id, preview, last_message, unread, last_synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(portal_key) DO UPDATE SET
			remote_id=excluded.remote_id,
			remote_name=excluded.remote_name,
			other_user_id=excluded.other_user_id,
			preview=excluded.preview,
			last_message=excluded.last_message,
			unread=excluded.unread,
			last_synced_at=CURRENT_TIMESTAMP
	`, state.PortalKey, state.RemoteID, state.RemoteName, state.OtherUserID, state.Preview, state.LastMessage, state.Unread)
	return err
}

func (s *Store) GetPortalByKey(portalKey string) (*PortalState, error) {
	row := s.db.QueryRow(`
		SELECT portal_key, remote_id, remote_name, COALESCE(other_user_id, ''), preview, last_message, unread, last_synced_at
		FROM portal_state
		WHERE portal_key = ?
	`, portalKey)

	var state PortalState
	err := row.Scan(
		&state.PortalKey,
		&state.RemoteID,
		&state.RemoteName,
		&state.OtherUserID,
		&state.Preview,
		&state.LastMessage,
		&state.Unread,
		&state.LastSyncedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Store) GetPortalByRemoteID(remoteID string) (*PortalState, error) {
	row := s.db.QueryRow(`
		SELECT portal_key, remote_id, remote_name, COALESCE(other_user_id, ''), preview, last_message, unread, last_synced_at
		FROM portal_state
		WHERE remote_id = ?
	`, remoteID)

	var state PortalState
	err := row.Scan(
		&state.PortalKey,
		&state.RemoteID,
		&state.RemoteName,
		&state.OtherUserID,
		&state.Preview,
		&state.LastMessage,
		&state.Unread,
		&state.LastSyncedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Store) ListPortals() ([]PortalState, error) {
	rows, err := s.db.Query(`
		SELECT portal_key, remote_id, remote_name, COALESCE(other_user_id, ''), preview, last_message, unread, last_synced_at
		FROM portal_state
		ORDER BY last_synced_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PortalState
	for rows.Next() {
		var state PortalState
		err = rows.Scan(
			&state.PortalKey,
			&state.RemoteID,
			&state.RemoteName,
			&state.OtherUserID,
			&state.Preview,
			&state.LastMessage,
			&state.Unread,
			&state.LastSyncedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, state)
	}

	return result, rows.Err()
}

func (s *Store) UpsertMessages(states []MessageState) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT INTO message_state (portal_key, remote_id, author, text, outgoing, timestamp_raw, kind, has_media, hydrated_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(portal_key, remote_id) DO UPDATE SET
			author=excluded.author,
			text=excluded.text,
			outgoing=excluded.outgoing,
			timestamp_raw=excluded.timestamp_raw,
			kind=excluded.kind,
			has_media=excluded.has_media,
			hydrated_at=COALESCE(excluded.hydrated_at, message_state.hydrated_at),
			last_seen_at=CURRENT_TIMESTAMP
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, state := range states {
		var hydratedAt any
		if state.HydratedAt != nil {
			hydratedAt = *state.HydratedAt
		}
		if _, err = stmt.Exec(state.PortalKey, state.RemoteID, state.Author, state.Text, state.Outgoing, state.TimestampRaw, state.Kind, state.HasMedia, hydratedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetMessage(portalKey, remoteID string) (*MessageState, error) {
	row := s.db.QueryRow(`
		SELECT portal_key, remote_id, author, text, outgoing, timestamp_raw, kind, has_media, hydrated_at, last_seen_at
		FROM message_state
		WHERE portal_key = ? AND remote_id = ?
	`, portalKey, remoteID)

	var state MessageState
	var hydratedAt sql.NullTime
	var author, text, timestampRaw, kind sql.NullString
	err := row.Scan(
		&state.PortalKey,
		&state.RemoteID,
		&author,
		&text,
		&state.Outgoing,
		&timestampRaw,
		&kind,
		&state.HasMedia,
		&hydratedAt,
		&state.LastSeenAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	state.Author = author.String
	state.Text = text.String
	state.TimestampRaw = timestampRaw.String
	state.Kind = kind.String
	if hydratedAt.Valid {
		state.HydratedAt = &hydratedAt.Time
	}
	return &state, nil
}

func (s *Store) MarkMessageHydrated(portalKey, remoteID string) error {
	_, err := s.db.Exec(`
		UPDATE message_state
		SET hydrated_at=CURRENT_TIMESTAMP
		WHERE portal_key = ? AND remote_id = ?
	`, portalKey, remoteID)
	return err
}

func (s *Store) ResetSyncState() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM message_state`); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM portal_state`); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM api_sync_state`); err != nil {
		return err
	}

	return tx.Commit()
}

func MarshalJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}
