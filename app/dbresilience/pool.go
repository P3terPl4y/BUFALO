// Package dbresilience contains conservative PostgreSQL pool settings and
// validation shared by configuration and the production startup guard.
package dbresilience

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type PoolSettings struct {
	MaxIdleConns       int
	MaxOpenConns       int
	ConnMaxIdleSeconds int
	ConnMaxLifeSeconds int
}

func DefaultPoolSettings() PoolSettings {
	return PoolSettings{
		MaxIdleConns:       10,
		MaxOpenConns:       25,
		ConnMaxIdleSeconds: 300,
		ConnMaxLifeSeconds: 1800,
	}
}

// ParsePoolSettings validates environment values instead of allowing an
// invalid max-open value (zero means unlimited in database/sql) to exhaust PG.
func ParsePoolSettings(getenv func(string) string) (PoolSettings, error) {
	settings := DefaultPoolSettings()
	values := []struct {
		name string
		dest *int
	}{
		{"DB_POOL_MAX_IDLE_CONNS", &settings.MaxIdleConns},
		{"DB_POOL_MAX_OPEN_CONNS", &settings.MaxOpenConns},
		{"DB_POOL_CONN_MAX_IDLE_SECONDS", &settings.ConnMaxIdleSeconds},
		{"DB_POOL_CONN_MAX_LIFETIME_SECONDS", &settings.ConnMaxLifeSeconds},
	}
	for _, item := range values {
		raw := strings.TrimSpace(getenv(item.name))
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil {
			return DefaultPoolSettings(), fmt.Errorf("%s must be an integer", item.name)
		}
		*item.dest = value
	}
	if settings.MaxOpenConns < 1 || settings.MaxOpenConns > 500 {
		return DefaultPoolSettings(), fmt.Errorf("DB_POOL_MAX_OPEN_CONNS must be between 1 and 500")
	}
	if settings.MaxIdleConns < 0 || settings.MaxIdleConns > settings.MaxOpenConns {
		return DefaultPoolSettings(), fmt.Errorf("DB_POOL_MAX_IDLE_CONNS must be between 0 and DB_POOL_MAX_OPEN_CONNS")
	}
	if settings.ConnMaxIdleSeconds < 0 || settings.ConnMaxIdleSeconds > 86400 {
		return DefaultPoolSettings(), fmt.Errorf("DB_POOL_CONN_MAX_IDLE_SECONDS must be between 0 and 86400")
	}
	if settings.ConnMaxLifeSeconds < 60 || settings.ConnMaxLifeSeconds > 86400 {
		return DefaultPoolSettings(), fmt.Errorf("DB_POOL_CONN_MAX_LIFETIME_SECONDS must be between 60 and 86400")
	}
	return settings, nil
}

// DatabaseNameFromDSN extracts the effective database name using the same
// PostgreSQL DSN parser as the driver. Callers must compare it to an explicit
// allowlisted name before running destructive test operations.
func DatabaseNameFromDSN(dsn string) (string, error) {
	config, err := pgconn.ParseConfig(dsn)
	if err != nil {
		return "", fmt.Errorf("invalid PostgreSQL DSN")
	}
	return config.Database, nil
}
