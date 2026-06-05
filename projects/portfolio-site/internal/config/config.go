package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Session  SessionConfig
	Upload   UploadConfig
	Site     SiteConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxHeaderBytes  int
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Driver          string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// SessionConfig holds session/auth configuration.
type SessionConfig struct {
	Secret     string
	Name       string
	MaxAge     int
	Secure     bool
	HTTPOnly   bool
	SameSite   string
}

// UploadConfig holds file upload configuration.
type UploadConfig struct {
	Dir       string
	MaxSize   int64 // bytes
	AllowedTypes []string
}

// SiteConfig holds site-level configuration.
type SiteConfig struct {
	Title       string
	Subtitle    string
	Description string
	Author      string
	BaseURL     string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Addr:            envStr("SERVER_ADDR", ":8080"),
			ReadTimeout:     envDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    envDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     envDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: envDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
			MaxHeaderBytes:  envInt("SERVER_MAX_HEADER_BYTES", 1<<20), // 1MB
		},
		Database: DatabaseConfig{
			Driver:          envStr("DB_DRIVER", "sqlite3"),
			DSN:             envStr("DB_DSN", "data/portfolio.db"),
			MaxOpenConns:    envInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: envDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Session: SessionConfig{
			Secret:   envStr("SESSION_SECRET", "change-me-in-production-32-bytes!"),
			Name:     envStr("SESSION_NAME", "portfolio_session"),
			MaxAge:   envInt("SESSION_MAX_AGE", 86400), // 24 hours
			Secure:   envBool("SESSION_SECURE", false),
			HTTPOnly: envBool("SESSION_HTTP_ONLY", true),
			SameSite: envStr("SESSION_SAME_SITE", "Lax"),
		},
		Upload: UploadConfig{
			Dir:     envStr("UPLOAD_DIR", "web/static/uploads"),
			MaxSize: int64(envInt("UPLOAD_MAX_SIZE", 10<<20)), // 10MB
			AllowedTypes: []string{
				"image/jpeg", "image/png", "image/webp",
				"image/svg+xml", "application/pdf",
			},
		},
		Site: SiteConfig{
			Title:       envStr("SITE_TITLE", "Aspira Studio"),
			Subtitle:    envStr("SITE_SUBTITLE", "Embedded Linux · Medical Imaging · C++ · Go"),
			Description: envStr("SITE_DESCRIPTION", "Building quiet, reliable engineering systems."),
			Author:      envStr("SITE_AUTHOR", "Meng Yan"),
			BaseURL:     envStr("SITE_BASE_URL", "http://localhost:8080"),
		},
	}
}

func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func envBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}

// Addr returns the formatted listen address.
func (s ServerConfig) AddrStr() string {
	return s.Addr
}

// String returns a safe representation of the config (no secrets).
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{Server:%+v, DB:{Driver:%s, DSN:%s, MaxOpen:%d}, Site:{Title:%s}}",
		c.Server, c.Database.Driver, c.Database.DSN, c.Database.MaxOpenConns, c.Site.Title,
	)
}
