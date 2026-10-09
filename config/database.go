package config

import (
	"fmt"
	"os"
	"path/filepath"

	"goravel/app/dbresilience"
	"goravel/app/facades"

	"github.com/goravel/framework/contracts/database/driver"
	postgresfacades "github.com/goravel/postgres/facades"
)

func init() {
	config := facades.Config()
	pool, err := dbresilience.ParsePoolSettings(func(key string) string {
		value := config.Env(key, "")
		if value == nil {
			return ""
		}
		return fmt.Sprint(value)
	})
	if err != nil {
		// Production startup rejects invalid values. Keep local boot predictable
		// by falling back to bounded defaults if the environment is malformed.
		pool = dbresilience.DefaultPoolSettings()
	}
	migrationWorkload := filepath.Base(os.Args[0]) == "migrate" || (len(os.Args) > 1 && (os.Args[1] == "migrate" || os.Args[1] == "artisan"))
	dsn, dsnErr := dbresilience.BoundedDSN(func(key string) string {
		v := config.Env(key, "")
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}, migrationWorkload)
	if dsnErr != nil {
		panic(dsnErr)
	}
	config.Add("database", map[string]any{
		// Default database connection name
		"default": config.Env("DB_CONNECTION"),
		// Database connections
		"connections": map[string]any{
			"postgres": map[string]any{
				"dsn":      dsn,
				"host":     config.Env("DB_HOST"),
				"port":     config.Env("DB_PORT"),
				"database": config.Env("DB_DATABASE"),
				"username": config.Env("DB_USERNAME"),
				"password": config.Env("DB_PASSWORD"),
				"sslmode":  config.Env("DB_SSLMODE", "disable"),
				"singular": false,
				"prefix":   "",
				"schema":   config.Env("DB_SCHEMA", "public"),
				"via": func() (driver.Driver, error) {
					return postgresfacades.Postgres("postgres")
				},
			},
		},
		// database/sql discards broken connections and dials a replacement on a
		// later operation. Bounded lifetimes also recycle stale provider-side
		// connections; writes are never replayed by application middleware.
		// Pool configuration
		"pool": map[string]any{
			// Sets the maximum number of connections in the idle
			// connection pool.
			//
			// If MaxOpenConns is greater than 0 but less than the new MaxIdleConns,
			// then the new MaxIdleConns will be reduced to match the MaxOpenConns limit.
			//
			// If n <= 0, no idle connections are retained.
			"max_idle_conns": pool.MaxIdleConns,
			// Sets the maximum number of open connections to the database.
			//
			// If MaxIdleConns is greater than 0 and the new MaxOpenConns is less than
			// MaxIdleConns, then MaxIdleConns will be reduced to match the new
			// MaxOpenConns limit.
			//
			// If n <= 0, then there is no limit on the number of open connections.
			"max_open_conns": pool.MaxOpenConns,
			// Sets the maximum amount of time a connection may be idle.
			//
			// Expired connections may be closed lazily before reuse.
			//
			// If d <= 0, connections are not closed due to a connection's idle time.
			// Unit: Second
			"conn_max_idletime": pool.ConnMaxIdleSeconds,
			// Sets the maximum amount of time a connection may be reused.
			//
			// Expired connections may be closed lazily before reuse.
			//
			// If d <= 0, connections are not closed due to a connection's age.
			// Unit: Second
			"conn_max_lifetime": pool.ConnMaxLifeSeconds,
		},
		// Sets the threshold for slow queries in milliseconds, the slow query will be logged.
		// Unit: Millisecond
		"slow_threshold": 200,
		// Migration Repository Table
		//
		// This table keeps track of all the migrations that have already run for
		// your application. Using this information, we can determine which of
		// the migrations on disk haven't actually been run in the database.
		"migrations": map[string]any{
			"table": "migrations",
		},
	})
}
