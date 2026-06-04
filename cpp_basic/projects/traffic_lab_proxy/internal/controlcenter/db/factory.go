package db

import (
	"fmt"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// NewDB creates a database connection based on the driver name.
// Supported drivers: "sqlite" (default), "postgres".
func NewDB(driver, dsn string) (DB, error) {
	switch driver {
	case "sqlite":
		return NewSQLiteDB(dsn)
	case "postgres":
		return NewPostgresDB(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s (supported: sqlite, postgres)", driver)
	}
}

// NewDBFromConfig creates a database connection from a ControlCenterConfig.
func NewDBFromConfig(cfg *common.ControlCenterConfig) (DB, error) {
	return NewDB(cfg.DatabaseDriver, cfg.DatabaseDSN)
}
