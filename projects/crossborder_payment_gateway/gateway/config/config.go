package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Engine   EngineConfig   `yaml:"engine"`
	Auth     AuthConfig     `yaml:"auth"`
	Redis    RedisConfig    `yaml:"redis"`
	Exchange ExchangeConfig `yaml:"exchange"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	Mode         string        `yaml:"mode"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type EngineConfig struct {
	Addr    string        `yaml:"addr"`
	Enabled bool          `yaml:"enabled"`
	Timeout time.Duration `yaml:"timeout"`
}

type AuthConfig struct {
	JWTSecret     string        `yaml:"jwt_secret"`
	TokenExpiry   time.Duration `yaml:"token_expiry"`
	RefreshExpiry time.Duration `yaml:"refresh_expiry"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type ExchangeConfig struct {
	APIURL          string        `yaml:"api_url"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	Enabled         bool          `yaml:"enabled"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			Mode:         "debug",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "data/payment_gateway.db",
		},
		Engine: EngineConfig{
			Addr:    "localhost:9100",
			Enabled: false,
			Timeout: 5 * time.Second,
		},
		Auth: AuthConfig{
			JWTSecret:     "change-me-in-production-please",
			TokenExpiry:   24 * time.Hour,
			RefreshExpiry: 168 * time.Hour,
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
			DB:   0,
		},
		Exchange: ExchangeConfig{
			APIURL:          "https://open.er-api.com/v6/latest/USD",
			RefreshInterval: 1 * time.Hour,
			Enabled:         true,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
