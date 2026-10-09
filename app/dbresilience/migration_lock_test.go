package dbresilience

import (
	"context"
	"database/sql"
	"errors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"os"
	"testing"
	"time"
)

func TestMigrationLockSerializesAndHonorsDeadline(t *testing.T) {
	dsn := os.Getenv("BUFALO_LOCK_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated PostgreSQL")
	}
	pool, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- WithMigrationLock(context.Background(), pool, func() error { close(held); <-release; return nil })
	}()
	select {
	case <-held:
	case <-time.After(3 * time.Second):
		t.Fatal("lock not acquired")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	called := false
	err = WithMigrationLock(ctx, pool, func() error { called = true; return nil })
	close(release)
	if first := <-done; first != nil {
		t.Fatal(first)
	}
	if called || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("concurrent migration entered: %v", err)
	}
	if err := WithMigrationLock(context.Background(), pool, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}
