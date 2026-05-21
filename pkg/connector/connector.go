package connector

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/util/configupgrade"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/colej/mautrix-snapchat/internal/config"
	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/store"
)

type UserLoginMetadata struct {
	Label         string `json:"label"`
	Authenticated bool   `json:"authenticated"`
	LastURL       string `json:"last_url"`
}

type SnapchatConnector struct {
	br     *bridgev2.Bridge
	Config ConnectorConfig
	store  *store.Store

	networkIconMXC id.ContentURIString
}

var _ bridgev2.NetworkConnector = (*SnapchatConnector)(nil)

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
	sc.ensureNetworkIcon(ctx)
	return nil
}

func (sc *SnapchatConnector) GetCapabilities() *bridgev2.NetworkGeneralCapabilities {
	log.Printf("bridgev2 metadata: registering Snapchat general capabilities disappearing=true contact_list=true media=%t outbound_edits=false inbound_edit_sync=true", sc != nil && sc.Config.SendMediaEnabled)
	return &bridgev2.NetworkGeneralCapabilities{
		DisappearingMessages: true,
		AggressiveUpdateInfo: false,
		Provisioning: bridgev2.ProvisioningCapabilities{
			ResolveIdentifier: bridgev2.ResolveIdentifierCapabilities{
				CreateDM:    true,
				ContactList: true,
			},
		},
	}
}

func (sc *SnapchatConnector) GetBridgeInfoVersion() (info, capabilities int) {
	return 3, 3
}

func (sc *SnapchatConnector) GetName() bridgev2.BridgeName {
	icon := id.ContentURIString("")
	if sc != nil {
		icon = sc.networkIconMXC
		if icon == "" {
			icon = id.ContentURIString(strings.TrimSpace(sc.Config.NetworkIconMXC))
		}
	}
	log.Printf("bridgev2 metadata: name=Snapchat network_id=snapchat bridge_type=snapchat icon_configured=%t", icon != "")
	return bridgev2.BridgeName{
		DisplayName:          "Snapchat",
		NetworkURL:           "https://www.snapchat.com",
		NetworkIcon:          icon,
		NetworkID:            "snapchat",
		BeeperBridgeType:     "snapchat",
		DefaultPort:          29344,
		DefaultCommandPrefix: "!snapchat",
	}
}

func (sc *SnapchatConnector) ensureNetworkIcon(ctx context.Context) {
	if sc == nil {
		return
	}
	if configured := strings.TrimSpace(sc.Config.NetworkIconMXC); configured != "" {
		sc.networkIconMXC = id.ContentURIString(configured)
		return
	}
	if sc.br == nil || sc.br.Bot == nil {
		log.Printf("bridgev2 metadata: cannot upload Snapchat icon before Matrix bot is available")
		return
	}
	mxc, _, err := sc.br.Bot.UploadMedia(ctx, "", []byte(snapchatNetworkIconSVG), "snapchat.svg", "image/svg+xml")
	if err != nil {
		log.Printf("bridgev2 metadata: failed to upload Snapchat icon: %v", err)
		return
	}
	sc.networkIconMXC = mxc
	sc.Config.NetworkIconMXC = string(mxc)
	if err = sc.br.Bot.SetAvatarURL(ctx, mxc); err != nil {
		log.Printf("bridgev2 metadata: failed to set Snapchat bot avatar: %v", err)
	}
	log.Printf("bridgev2 metadata: uploaded Snapchat icon mxc=%s", mxc)
}

func (sc *SnapchatConnector) GetConfig() (example string, data any, upgrader configupgrade.Upgrader) {
	return ExampleConfig, &sc.Config, configupgrade.SimpleUpgrader(upgradeConfig)
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
	return []bridgev2.LoginFlow{
		{
			Name:        "Snapchat Web Login",
			Description: "Log in on this device and submit Snapchat Web session cookies",
			ID:          "snapchat-web",
		},
		{
			Name:        "Server Browser Session",
			Description: "Fallback/admin login using the bridge server browser sidecar",
			ID:          "browser-session",
		},
	}
}

func (sc *SnapchatConnector) CreateLogin(ctx context.Context, user *bridgev2.User, flowID string) (bridgev2.LoginProcess, error) {
	if flowID != "browser-session" && flowID != "snapchat-web" {
		return nil, fmt.Errorf("unknown login flow: %s", flowID)
	}
	return &SnapchatLogin{
		User:      user,
		Connector: sc,
		FlowID:    flowID,
	}, nil
}

func (sc *SnapchatConnector) newClient() *sidecar.Client {
	cfg := sc.Config
	if cfg.RequestTimeoutSeconds <= 0 {
		cfg.RequestTimeoutSeconds = 20
	}
	if cfg.PollIntervalSeconds <= 0 {
		cfg.PollIntervalSeconds = 10
	}
	if cfg.MessageFetchLimit <= 0 {
		cfg.MessageFetchLimit = 5
	}
	return sidecar.New(config.ConnectorConfig{
		BaseURL:               cfg.BaseURL,
		SharedSecret:          cfg.SharedSecret,
		RequestTimeoutSeconds: cfg.RequestTimeoutSeconds,
	})
}

func (sc *SnapchatConnector) pollInterval() time.Duration {
	seconds := sc.Config.PollIntervalSeconds
	if seconds <= 0 {
		seconds = 10
	}
	return time.Duration(seconds) * time.Second
}

func (sc *SnapchatConnector) apiMode() string {
	mode := strings.ToLower(strings.TrimSpace(sc.Config.APIMode))
	switch mode {
	case "api_only", "dom_only":
		return mode
	default:
		return "auto"
	}
}

func (sc *SnapchatConnector) domFallbackEnabled() bool {
	if sc == nil {
		return false
	}
	if sc.apiMode() == "api_only" {
		return false
	}
	if sc.Config.DOMFallbackEnabled == nil {
		return false
	}
	return *sc.Config.DOMFallbackEnabled
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
	features := &event.RoomFeatures{
		ID:                  "fi.mau.snapchat.capabilities.2026_05_21",
		MaxTextLength:       5000,
		Edit:                event.CapLevelRejected,
		Delete:              event.CapLevelRejected,
		ReadReceipts:        sc != nil && sc.Config.ReadReceiptsEnabled,
		TypingNotifications: sc != nil && sc.Config.TypingIndicators,
		DisappearingTimer: &event.DisappearingTimerCapability{
			Types:          []event.DisappearingType{event.DisappearingTypeAfterSend},
			OmitEmptyTimer: true,
		},
	}
	if sc != nil && sc.Config.SendMediaEnabled {
		mediaTypes := map[string]event.CapabilitySupportLevel{
			"image/jpeg": event.CapLevelFullySupported,
			"image/png":  event.CapLevelFullySupported,
			"image/gif":  event.CapLevelFullySupported,
			"image/webp": event.CapLevelPartialSupport,
			"image/*":    event.CapLevelPartialSupport,
			"video/mp4":  event.CapLevelPartialSupport,
			"video/*":    event.CapLevelPartialSupport,
		}
		mediaFeatures := &event.FileFeatures{
			MimeTypes:        mediaTypes,
			Caption:          event.CapLevelDropped,
			MaxCaptionLength: 5000,
			MaxSize:          16 * 1024 * 1024,
			ViewOnce:         true,
		}
		features.File = event.FileFeatureMap{
			event.MsgImage: mediaFeatures,
			event.MsgVideo: mediaFeatures,
			event.MsgFile:  mediaFeatures,
		}
	}
	return features
}
