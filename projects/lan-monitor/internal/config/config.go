package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Scan     ScanConfig     `yaml:"scan"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Alert    AlertConfig    `yaml:"alert"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
	Mode string `yaml:"mode"`
}

type ScanConfig struct {
	Enabled          bool     `yaml:"enabled"`
	IntervalSeconds  int      `yaml:"interval_seconds"`
	Subnets          []string `yaml:"subnets"`
	Methods          []string `yaml:"methods"`
	WorkerCount      int      `yaml:"worker_count"`
	TimeoutMs        int      `yaml:"timeout_ms"`
	OfflineThreshold int      `yaml:"offline_threshold"`
	TCPPorts         []int    `yaml:"tcp_ports"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type JWTConfig struct {
	Secret                   string `yaml:"secret"`
	AccessTokenExpireMinutes int    `yaml:"access_token_expire_minutes"`
	RefreshTokenExpireHours  int    `yaml:"refresh_token_expire_hours"`
}

type AlertConfig struct {
	NewDeviceEnabled        bool    `yaml:"new_device_enabled"`
	OfflineEnabled          bool    `yaml:"offline_enabled"`
	HighTrafficEnabled      bool    `yaml:"high_traffic_enabled"`
	HighTrafficThresholdMbps float64 `yaml:"high_traffic_threshold_mbps"`
}

var AppConfig *Config

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Scan.WorkerCount == 0 {
		cfg.Scan.WorkerCount = 100
	}
	if cfg.Scan.TimeoutMs == 0 {
		cfg.Scan.TimeoutMs = 800
	}
	if cfg.Scan.OfflineThreshold == 0 {
		cfg.Scan.OfflineThreshold = 3
	}
	if cfg.Scan.IntervalSeconds == 0 {
		cfg.Scan.IntervalSeconds = 60
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./data/lan_monitor.db"
	}
	if cfg.JWT.AccessTokenExpireMinutes == 0 {
		cfg.JWT.AccessTokenExpireMinutes = 60
	}
	if cfg.JWT.RefreshTokenExpireHours == 0 {
		cfg.JWT.RefreshTokenExpireHours = 168
	}

	AppConfig = cfg
	return cfg, nil
}
