package monitoring

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestReadinessHandlerReportsHealthyOnlyWhenEveryCheckPasses(t *testing.T) {
	app := fiber.New()
	checks := 0
	app.Get("/readyz", ReadinessHandler(
		func(ctx context.Context) error {
			checks++
			if _, ok := ctx.Deadline(); !ok {
				t.Error("readiness check has no deadline")
			}
			return nil
		},
		func(context.Context) error { checks++; return nil },
	))

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || checks != 2 {
		t.Fatalf("status=%d checks=%d, want 200 and two checks", response.StatusCode, checks)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q, want no-store", got)
	}
}

func TestReadinessHandlerFailsClosedWithoutLeakingDependencyErrors(t *testing.T) {
	app := fiber.New()
	app.Get("/readyz", ReadinessHandler(func(context.Context) error {
		return errors.New("database credentials must not be exposed")
	}))

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"status":"not_ready"`) || strings.Contains(string(body), "credentials") {
		t.Fatalf("unexpected readiness body: %s", body)
	}
}

func TestSkipForHealthProbesBypassesSessionAndCSRFMiddleware(t *testing.T) {
	app := fiber.New()
	calls := 0
	app.Use(SkipForHealthProbes(func(c fiber.Ctx) error {
		calls++
		return c.Next()
	}))
	app.Get("/healthz", func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	app.Get("/readyz", func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	app.Get("/private", func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	for _, path := range []string{"/healthz", "/readyz", "/private"} {
		response, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status=%d", path, response.StatusCode)
		}
	}
	if calls != 1 {
		t.Fatalf("store middleware called %d times, want only once for non-probe route", calls)
	}
}

func TestReadinessConcurrentRequestsDoNotMultiplyDependencyWork(t *testing.T) {
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	check := CachedReadiness(func(context.Context) error { calls.Add(1); close(started); <-release; return nil })
	done := make(chan error, 1)
	go func() { done <- check(context.Background()) }()
	<-started
	for n := 0; n < 100; n++ {
		if err := check(context.Background()); err == nil {
			t.Fatal("unfinished readiness reported ready")
		}
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 100; n++ {
		if err := check(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("unbounded readiness work", calls.Load())
	}
}
