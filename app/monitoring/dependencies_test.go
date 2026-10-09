package monitoring

import (
	"context"
	"errors"
	"testing"
)

func TestDependencyChecksReportFailureWithoutInternalDetails(t *testing.T) {
	SetDependencyChecks(func(context.Context) error { return nil }, func(context.Context) error { return errors.New("secret-redis-password") })
	t.Cleanup(func() { SetDependencyChecks(nil, nil) })
	r := CheckDependencies(context.Background())
	if r["postgres"].Status != "available" || r["redis"].Status != "unavailable" {
		t.Fatalf("unexpected dependency result: %+v", r)
	}
}

func TestDependencyChecksRespectCancelledDeadline(t *testing.T) {
	SetDependencyChecks(func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }, func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
	t.Cleanup(func() { SetDependencyChecks(nil, nil) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := CheckDependencies(ctx)
	if r["postgres"].Status != "unavailable" || r["redis"].Status != "unavailable" {
		t.Fatal("cancelled checks reported ready")
	}
}
