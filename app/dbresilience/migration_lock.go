package dbresilience

import (
	"context"
	"database/sql"
	"time"
)

const migrationLock = int64(0x425546414c4f)

func WithMigrationLock(ctx context.Context, pool *sql.DB, run func() error) error {
	conn, err := pool.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		var acquired bool
		if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", migrationLock).Scan(&acquired); err != nil {
			return err
		}
		if acquired {
			break
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(cleanup, "SELECT pg_advisory_unlock($1)", migrationLock)
	}()
	return run()
}
