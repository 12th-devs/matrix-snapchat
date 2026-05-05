package bridgev2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/util/configupgrade"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/event"

	"github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/store"
)

type ConnectorConfig struct {
	BaseURL               string `yaml:"base_url"`
	SharedSecret          string `yaml:"shared_secret"`
	RequestTimeoutSeconds int    `yaml:"request_timeout_seconds"`
	PollIntervalSeconds   int    `yaml:"poll_interval_seconds"`
	MessageFetchLimit     int    `yaml:"message_fetch_limit"`
	StateDBPath           string `yaml:"state_db_path"`
	LoginWaitSeconds      int    `yaml:"login_wait_seconds"`
}

type UserLoginMetadata struct {
	Label         string `json:"label"`
	Authenticated bool   `json:"authenticated"`
	LastURL       string `json:"last_url"`
}

type SnapchatConnector struct {
	br     *bridgev2.Bridge
	Config ConnectorConfig
	store  *store.Store
}

var _ bridgev2.NetworkConnector = (*SnapchatConnector)(nil)

const ExampleConfig = `# Snapchat Web sidecar connector
base_url: "http://127.0.0.1:3101"
shared_secret: "change-me"
request_timeout_seconds: 20
poll_interval_seconds: 8
message_fetch_limit: 40
state_db_path: "./data/bridgev2-state.sqlite"
login_wait_seconds: 90
`

func NewConnector() *SnapchatConnector {
	return &SnapchatConnector{}
}

func (sc *SnapchatConnector) Init(bridge *bridgev2.Bridge) {
	sc.br = bridge
}

func (sc *SnapchatConnector) Start(ctx context.Context) error {
	if sc.Config.StateDBPath == "" {
		sc.Config.StateDBPath = "./data/bridgev2-state.sqlite"
	}
	if err := os.MkdirAll(filepath.Dir(sc.Config.StateDBPath), 0o755); err != nil {
		return fmt.Errorf("create bridgev2 state db directory: %w", err)
	}
	stateStore, err := store.New(sc.Config.StateDBPath)
	if err != nil {
		return fmt.Errorf("open bridgev2 state db: %w", err)
	}
	sc.store = stateStore
	return nil
}

func (sc *SnapchatConnector) GetCapabilities() *bridgev2.NetworkGeneralCapabilities {
	return &bridgev2.NetworkGeneralCapabilities{
		DisappearingMessages: true,
	}
}

func (sc *SnapchatConnector) GetBridgeInfoVersion() (info, capabilities int) {
	return 1, 1
}

func (sc *SnapchatConnector) GetName() bridgev2.BridgeName {
	return bridgev2.BridgeName{
		DisplayName:      "Snapchat",
		NetworkURL:       "https://www.snapchat.com",
		NetworkID:        "snapchat",
		BeeperBridgeType: "github.com/colej/mautrix-snapchat",
		DefaultPort:      29344,
	}
}

func (sc *SnapchatConnector) GetConfig() (example string, data any, upgrader configupgrade.Upgrader) {
	return ExampleConfig, &sc.Config, configupgrade.SimpleUpgrader(func(helper configupgrade.Helper) {
		helper.Copy(configupgrade.Str, "base_url")
		helper.Copy(configupgrade.Str, "shared_secret")
		helper.Copy(configupgrade.Int, "request_timeout_seconds")
		helper.Copy(configupgrade.Int, "poll_interval_seconds")
		helper.Copy(configupgrade.Int, "message_fetch_limit")
		helper.Copy(configupgrade.Str, "state_db_path")
		helper.Copy(configupgrade.Int, "login_wait_seconds")
	})
}

func (sc *SnapchatConnector) GetDBMetaTypes() database.MetaTypes {
	return database.MetaTypes{
		UserLogin: func() any {
			return &UserLoginMetadata{}
		},
	}
}

func (sc *SnapchatConnector) LoadUserLogin(ctx context.Context, login *bridgev2.UserLogin) error {
	meta, _ := login.Metadata.(*UserLoginMetadata)
	label := "browser-session"
	if meta != nil && meta.Label != "" {
		label = meta.Label
	}
	if meta != nil && sc.store != nil {
		state, err := sc.store.GetLogin(string(login.ID))
		if err == nil && state != nil {
			meta.Authenticated = meta.Authenticated || strings.EqualFold(state.LastSeenState, "ready")
			if meta.LastURL == "" {
				meta.LastURL = state.RemoteID
			}
		}
	}
	api := NewSnapchatAPI(sc, login, label)
	if sc.store != nil {
		api.restoreChatMappings()
	}
	login.Client = api
	return nil
}

func (sc *SnapchatConnector) GetLoginFlows() []bridgev2.LoginFlow {
	return []bridgev2.LoginFlow{{
		Name:        "Browser Session",
		Description: "Open Snapchat Web login in the bridge browser sidecar",
		ID:          "browser-session",
	}}
}

func (sc *SnapchatConnector) CreateLogin(ctx context.Context, user *bridgev2.User, flowID string) (bridgev2.LoginProcess, error) {
	if flowID != "browser-session" {
		return nil, fmt.Errorf("unknown login flow: %s", flowID)
	}
	return &SnapchatLogin{
		User:      user,
		Connector: sc,
	}, nil
}

func (sc *SnapchatConnector) newClient() *connector.Client {
	cfg := sc.Config
	if cfg.RequestTimeoutSeconds <= 0 {
		cfg.RequestTimeoutSeconds = 20
	}
	if cfg.PollIntervalSeconds <= 0 {
		cfg.PollIntervalSeconds = 8
	}
	if cfg.MessageFetchLimit <= 0 {
		cfg.MessageFetchLimit = 40
	}
	return connector.New(struct {
		BaseURL               string `yaml:"base_url"`
		SharedSecret          string `yaml:"shared_secret"`
		RequestTimeoutSeconds int    `yaml:"request_timeout_seconds"`
	}{
		BaseURL:               cfg.BaseURL,
		SharedSecret:          cfg.SharedSecret,
		RequestTimeoutSeconds: cfg.RequestTimeoutSeconds,
	})
}

func (sc *SnapchatConnector) pollInterval() time.Duration {
	seconds := sc.Config.PollIntervalSeconds
	if seconds <= 0 {
		seconds = 8
	}
	return time.Duration(seconds) * time.Second
}

func (sc *SnapchatConnector) loginWaitDuration() time.Duration {
	seconds := sc.Config.LoginWaitSeconds
	if seconds <= 0 {
		seconds = 90
	}
	return time.Duration(seconds) * time.Second
}

func (sc *SnapchatConnector) resetSyncState() error {
	if sc == nil || sc.store == nil {
		return nil
	}
	return sc.store.ResetSyncState()
}

func (sc *SnapchatConnector) chatCapabilities() *event.RoomFeatures {
	return &event.RoomFeatures{MaxTextLength: 5000}
}
