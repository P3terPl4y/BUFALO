package services

import (
	"context"
	"fmt"
	"goravel/app/facades"
	"log"
	"time"
)

// A single SQL statement locks only its bounded batch; SKIP LOCKED permits
// multiple instances and never waits on a confirmation holding the same row.
func CleanupPending(parent context.Context, batch int) error {
	if batch < 1 || batch > 1000 {
		return fmt.Errorf("invalid cleanup batch")
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	return facades.DB().WithContext(ctx).Statement(`DELETE FROM pending_registrations WHERE id IN (SELECT id FROM pending_registrations WHERE expires_at < CURRENT_TIMESTAMP ORDER BY expires_at LIMIT $1 FOR UPDATE SKIP LOCKED)`, batch)
}
func RunPendingCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := CleanupPending(ctx, 256); err != nil && ctx.Err() == nil {
				log.Printf("Pending registration cleanup failed: %v", err)
			}
		}
	}
}
