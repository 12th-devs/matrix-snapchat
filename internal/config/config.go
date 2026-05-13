package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Bridge    BridgeConfig    `yaml:"bridge"`
	Connector ConnectorConfig `yaml:"connector"`
	Database  DatabaseConfig  `yaml:"database"`
	Network   NetworkConfig   `yaml:"network"`
}

type BridgeConfig struct {
	Listen   string `yaml:"listen"`
	LogLevel string `yaml:"log_level"`
	DataDir  string `yaml:"data_dir"`
}

type ConnectorConfig struct {
	BaseURL               string `yaml:"base_url"`
	SharedSecret          string `yaml:"shared_secret"`
	RequestTimeoutSeconds int    `yaml:"request_timeout_seconds"`
}

type DatabaseConfig struct {
	URI string `yaml:"uri"`
}

type NetworkConfig struct {
	BrowserProfileDir   string `yaml:"browser_profile_dir"`
	Headless            bool   `yaml:"headless"`
	PollIntervalSeconds int    `yaml:"poll_interval_seconds"`
	MessageFetchLimit   int    `yaml:"message_fetch_limit"`
	APIMode             string `yaml:"api_mode"`
	DOMFallbackEnabled  *bool  `yaml:"dom_fallback_enabled"`
	AutoFetchMessages   bool   `yaml:"auto_fetch_messages"`
	ReadReceiptsEnabled bool   `yaml:"read_receipts_enabled"`
	SnapMediaEnabled    bool   `yaml:"snap_media_enabled"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if cfg.Bridge.Listen == "" {
		cfg.Bridge.Listen = "127.0.0.1:29344"
	}
	if cfg.Connector.RequestTimeoutSeconds <= 0 {
		cfg.Connector.RequestTimeoutSeconds = 20
	}
	if cfg.Database.URI == "" {
		cfg.Database.URI = "./data/bridge-state.sqlite"
	}
	if cfg.Network.PollIntervalSeconds <= 0 {
		cfg.Network.PollIntervalSeconds = 2
	}
	if cfg.Network.MessageFetchLimit <= 0 {
		cfg.Network.MessageFetchLimit = 40
	}
	if cfg.Network.APIMode == "" {
		cfg.Network.APIMode = "api_only"
	}
	if cfg.Network.DOMFallbackEnabled == nil {
		enabled := false
		cfg.Network.DOMFallbackEnabled = &enabled
	}

	return &cfg, nil
}
