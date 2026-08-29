package connector

import (
	_ "embed"

	"go.mau.fi/util/configupgrade"
)

type ConnectorConfig struct {
	BaseURL               string `yaml:"base_url"`
	SharedSecret          string `yaml:"shared_secret"`
	RequestTimeoutSeconds int    `yaml:"request_timeout_seconds"`
	PollIntervalSeconds   int    `yaml:"poll_interval_seconds"`
	MessageFetchLimit     int    `yaml:"message_fetch_limit"`
	APIMode               string `yaml:"api_mode"`
	DOMFallbackEnabled    *bool  `yaml:"dom_fallback_enabled"`
	AutoFetchMessages     bool   `yaml:"auto_fetch_messages"`
	ReadReceiptsEnabled   bool   `yaml:"read_receipts_enabled"`
	TypingEnabled         bool   `yaml:"typing_enabled"`
	SnapMediaEnabled      bool   `yaml:"snap_media_enabled"`
	SnapMediaOnRead       bool   `yaml:"snap_media_on_read"`
	SendMediaEnabled      bool   `yaml:"send_media_enabled"`
	NetworkIconMXC        string `yaml:"network_icon_mxc"`
	StateDBPath           string `yaml:"state_db_path"`
	LoginWaitSeconds      int    `yaml:"login_wait_seconds"`
}

//go:embed example-config.yaml
var ExampleConfig string

func upgradeConfig(helper configupgrade.Helper) {
	helper.Copy(configupgrade.Str, "base_url")
	helper.Copy(configupgrade.Str, "shared_secret")
	helper.Copy(configupgrade.Int, "request_timeout_seconds")
	helper.Copy(configupgrade.Int, "poll_interval_seconds")
	helper.Copy(configupgrade.Int, "message_fetch_limit")
	helper.Copy(configupgrade.Str, "api_mode")
	helper.Copy(configupgrade.Bool, "dom_fallback_enabled")
	helper.Copy(configupgrade.Bool, "auto_fetch_messages")
	helper.Copy(configupgrade.Bool, "read_receipts_enabled")
	helper.Copy(configupgrade.Bool, "typing_enabled")
	helper.Copy(configupgrade.Bool, "snap_media_enabled")
	helper.Copy(configupgrade.Bool, "snap_media_on_read")
	helper.Copy(configupgrade.Bool, "send_media_enabled")
	helper.Copy(configupgrade.Str, "network_icon_mxc")
	helper.Copy(configupgrade.Str, "state_db_path")
	helper.Copy(configupgrade.Int, "login_wait_seconds")
}
