package monitoring

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

// ReadinessCheck reports whether one dependency needed to serve normal traffic
// can answer a lightweight probe.
type ReadinessCheck func(context.Context) error

// ReadinessHandler is intended for orchestrator probes. Keep /healthz as a
// liveness check; /readyz returns 503 when any required dependency is down.
// Error details are deliberately omitted from the public response.
func ReadinessHandler(checks ...ReadinessCheck) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		for _, check := range checks {
			if check == nil || check(ctx) != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not_ready"})
			}
		}
		return c.JSON(fiber.Map{"status": "ready"})
	}
}

// SkipForHealthProbes prevents session and CSRF stores from hiding liveness or
// readiness results when Redis itself is unavailable.
func SkipForHealthProbes(next fiber.Handler) fiber.Handler {
	return func(c fiber.Ctx) error {
		switch c.Path() {
		case "/healthz", "/readyz":
			return c.Next()
		default:
			return next(c)
		}
	}
}

// InternalHealth runs on a separate loopback listener, outside client quotas.
func InternalHealth(checks ...ReadinessCheck) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(200)
		case "/readyz":
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			for _, check := range checks {
				if check == nil || check(ctx) != nil {
					w.WriteHeader(503)
					return
				}
			}
			w.WriteHeader(200)
		default:
			w.WriteHeader(404)
		}
	})
}

// CachedReadiness admits only one dependency probe at a time. Public probes
// cannot turn the quota-free health endpoint into unbounded SQL/Redis work.
// Successful and failed results remain valid for one second.
func CachedReadiness(checks ...ReadinessCheck) ReadinessCheck {
	var mu sync.Mutex
	var expires time.Time
	var result error
	running := false
	return func(parent context.Context) error {
		mu.Lock()
		if time.Now().Before(expires) {
			err := result
			mu.Unlock()
			return err
		}
		if running {
			mu.Unlock()
			return errors.New("readiness probe in progress")
		}
		running = true
		mu.Unlock()
		ctx, cancel := context.WithTimeout(parent, 2*time.Second)
		defer cancel()
		var err error
		for _, check := range checks {
			if check == nil {
				err = errors.New("missing readiness dependency")
				break
			}
			if err = check(ctx); err != nil {
				break
			}
		}
		mu.Lock()
		result = err
		expires = time.Now().Add(time.Second)
		running = false
		mu.Unlock()
		return err
	}
}
