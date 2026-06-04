package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// APIGatewayConfig holds all configuration for the API Gateway service.
type APIGatewayConfig struct {
	Server    ServerConfig    `yaml:"server"`
	JWT       JWTConfig       `yaml:"jwt"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Log       LogConfig       `yaml:"log"`
}

// TrafficNodeConfig holds all configuration for a Traffic Node.
type TrafficNodeConfig struct {
	Node    NodeConfig    `yaml:"node"`
	Server  ServerConfig  `yaml:"server"`
	Control ControlConfig `yaml:"control"`
	Limits  LimitsConfig  `yaml:"limits"`
	Log     LogConfig     `yaml:"log"`
}

type ServerConfig struct {
	Listen     string `yaml:"listen"`
	TLSEnabled bool   `yaml:"tls_enabled"`
	TLSCert    string `yaml:"tls_cert"`
	TLSKey     string `yaml:"tls_key"`
}

type JWTConfig struct {
	Issuer           string `yaml:"issuer"`
	AccessTokenTTL   string `yaml:"access_token_ttl"`
	RefreshTokenTTL  string `yaml:"refresh_token_ttl"`
	SigningKey       string `yaml:"signing_key"`
	RefreshSigningKey string `yaml:"refresh_signing_key"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type RedisConfig struct {
	Addr string `yaml:"addr"`
	DB   int    `yaml:"db"`
}

type RateLimitConfig struct {
	Enabled          bool `yaml:"enabled"`
	RequestsPerMinute int `yaml:"requests_per_minute"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

type NodeConfig struct {
	NodeID       string `yaml:"node_id"`
	Name         string `yaml:"name"`
	Region       string `yaml:"region"`
	PublicAddr   string `yaml:"public_addr"`
	Secret       string `yaml:"secret"`
	MaxConnections int  `yaml:"max_connections"`
}

type ControlConfig struct {
	Endpoint         string `yaml:"endpoint"`
	HeartbeatInterval string `yaml:"heartbeat_interval"`
	RegistrationKey  string `yaml:"registration_key"`
}

type LimitsConfig struct {
	MaxConnections int    `yaml:"max_connections"`
	IdleTimeout    string `yaml:"idle_timeout"`
	ReadTimeout    string `yaml:"read_timeout"`
	WriteTimeout   string `yaml:"write_timeout"`
}

// LoadAPIGatewayConfig reads and parses the API Gateway YAML config file.
func LoadAPIGatewayConfig(path string) (*APIGatewayConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &APIGatewayConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	setAPIGatewayDefaults(cfg)
	return cfg, nil
}

// LoadTrafficNodeConfig reads and parses the Traffic Node YAML config file.
func LoadTrafficNodeConfig(path string) (*TrafficNodeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &TrafficNodeConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	setTrafficNodeDefaults(cfg)
	return cfg, nil
}

func setAPIGatewayDefaults(cfg *APIGatewayConfig) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}
	if cfg.JWT.Issuer == "" {
		cfg.JWT.Issuer = "secure-gateway"
	}
	if cfg.JWT.AccessTokenTTL == "" {
		cfg.JWT.AccessTokenTTL = "15m"
	}
	if cfg.JWT.RefreshTokenTTL == "" {
		cfg.JWT.RefreshTokenTTL = "168h"
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "postgres"
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "redis:6379"
	}
	if cfg.RateLimit.RequestsPerMinute == 0 {
		cfg.RateLimit.RequestsPerMinute = 120
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	// Fall back to env vars for secrets
	if cfg.JWT.SigningKey == "" {
		cfg.JWT.SigningKey = os.Getenv("JWT_SIGNING_KEY")
		if cfg.JWT.SigningKey == "" {
			cfg.JWT.SigningKey = "change-me-in-production"
		}
	}
	if cfg.JWT.RefreshSigningKey == "" {
		cfg.JWT.RefreshSigningKey = os.Getenv("JWT_REFRESH_SIGNING_KEY")
		if cfg.JWT.RefreshSigningKey == "" {
			cfg.JWT.RefreshSigningKey = "change-me-in-production-refresh"
		}
	}
}

func setTrafficNodeDefaults(cfg *TrafficNodeConfig) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":9443"
	}
	if cfg.Node.NodeID == "" {
		cfg.Node.NodeID = "edge-node-01"
	}
	if cfg.Node.Region == "" {
		cfg.Node.Region = "default"
	}
	if cfg.Control.Endpoint == "" {
		cfg.Control.Endpoint = "https://api-gateway:8080"
	}
	if cfg.Control.HeartbeatInterval == "" {
		cfg.Control.HeartbeatInterval = "5s"
	}
	if cfg.Limits.MaxConnections == 0 {
		cfg.Limits.MaxConnections = 10000
	}
	if cfg.Limits.IdleTimeout == "" {
		cfg.Limits.IdleTimeout = "120s"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Node.Secret == "" {
		cfg.Node.Secret = os.Getenv("NODE_SECRET")
		if cfg.Node.Secret == "" {
			cfg.Node.Secret = "node-secret-change-me"
		}
	}
}