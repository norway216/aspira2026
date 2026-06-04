package config

import (
	"encoding/json"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the agent configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Agent    AgentConfig    `yaml:"agent"`
	Security SecurityConfig `yaml:"security"`
	Collector CollectorConfig `yaml:"collector"`
	Command  CommandConfig  `yaml:"command"`
}

type ServerConfig struct {
	URL           string `yaml:"url"`
	WebSocketURL  string `yaml:"websocket_url"`
}

type AgentConfig struct {
	DeviceName   string `yaml:"device_name"`
	RegisterCode string `yaml:"register_code"`
	DataDir      string `yaml:"data_dir"`
	LogFile      string `yaml:"log_file"`
}

type SecurityConfig struct {
	TLSVerify  bool   `yaml:"tls_verify"`
	TokenFile  string `yaml:"token_file"`
	DeviceCert string `yaml:"device_cert"`
	DeviceKey  string `yaml:"device_key"`
}

type CollectorConfig struct {
	HeartbeatIntervalSec int `yaml:"heartbeat_interval_sec"`
	MetricsIntervalSec   int `yaml:"metrics_interval_sec"`
	LogsIntervalSec      int `yaml:"logs_interval_sec"`
}

type CommandConfig struct {
	EnableRemoteCommand bool `yaml:"enable_remote_command"`
	AllowReboot         bool `yaml:"allow_reboot"`
	AllowShell          bool `yaml:"allow_shell"`
}

// AgentIdentity holds the agent's registered identity.
type AgentIdentity struct {
	DeviceID    string `json:"device_id" yaml:"device_id"`
	Token       string `json:"agent_token" yaml:"token"`
	ServerTime  int64  `json:"server_time" yaml:"server_time"`
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			URL:          "http://localhost:8080",
			WebSocketURL: "ws://localhost:8080/ws",
		},
		Agent: AgentConfig{
			DeviceName:   "",
			RegisterCode: "",
			DataDir:      "/var/lib/embedded-agent",
			LogFile:      "/var/log/embedded-agent.log",
		},
		Security: SecurityConfig{
			TLSVerify:  false,
			TokenFile:  "/var/lib/embedded-agent/identity.json",
			DeviceCert: "/var/lib/embedded-agent/device.crt",
			DeviceKey:  "/var/lib/embedded-agent/device.key",
		},
		Collector: CollectorConfig{
			HeartbeatIntervalSec: 5,
			MetricsIntervalSec:   10,
			LogsIntervalSec:      60,
		},
		Command: CommandConfig{
			EnableRemoteCommand: true,
			AllowReboot:         false,
			AllowShell:          false,
		},
	}
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadIdentity loads the saved agent identity from disk.
func LoadIdentity(path string) (*AgentIdentity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	identity := &AgentIdentity{}
	if err := json.Unmarshal(data, identity); err != nil {
		return nil, err
	}
	return identity, nil
}

// SaveIdentity saves the agent identity to disk as JSON.
func SaveIdentity(path string, identity *AgentIdentity) error {
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// EnsureDir ensures a directory exists.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}