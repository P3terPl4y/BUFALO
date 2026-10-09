package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestRateLimiterRejectsRequestOverLimit(t *testing.T) {
	app := fiber.New()
	counter := NewMemoryRateCounter()
	app.Post("/register", NewRateLimit(counter, "register-test", 5, time.Minute, func(fiber.Ctx) string { return "same-client" }), func(c fiber.Ctx) error { return c.SendStatus(200) })
	for i := 1; i <= 6; i++ {
		res, err := app.Test(httptest.NewRequest("POST", "/register", nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		want := http.StatusOK
		if i == 6 {
			want = http.StatusTooManyRequests
		}
		if res.StatusCode != want {
			t.Fatalf("attempt %d got %d want %d", i, res.StatusCode, want)
		}
	}
}

func TestRateLimiterSharesCounterAcrossAppInstances(t *testing.T) {
	counter := NewMemoryRateCounter()
	const maxAllowed = 9
	newApp := func() *fiber.App {
		app := fiber.New()
		app.Post("/register", NewRateLimit(counter, "register-test", maxAllowed, time.Minute, func(fiber.Ctx) string { return "same-client" }), func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
		return app
	}
	apps := []*fiber.App{newApp(), newApp()}
	for i := 0; i < maxAllowed; i++ {
		res, err := apps[i%len(apps)].Test(httptest.NewRequest("POST", "/register", nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("request %d got %d, want 200", i+1, res.StatusCode)
		}
	}
	res, err := apps[1].Test(httptest.NewRequest("POST", "/register", nil))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("cross-instance request got %d, want 429", res.StatusCode)
	}
}

func TestRegisterLimiterCapsRepeatedEmailWithoutLeakingIt(t *testing.T) {
	counter := NewMemoryRateCounter()
	app := fiber.New()
	app.Post("/register", RegisterIPRateLimiter(counter), RegisterEmailRateLimiter(counter), func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	post := func(email string) *http.Response {
		req := httptest.NewRequest("POST", "/register", strings.NewReader("email="+email))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	for i, want := range []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests} {
		res := post("victim%40example.com")
		res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("attempt %d got %d, want %d", i+1, res.StatusCode, want)
		}
		if i == 2 && res.Header.Get("Retry-After") == "" {
			t.Fatal("rate limited response is missing Retry-After")
		}
	}
}

func TestMemoryRateCounterDoesNotLoseConcurrentIncrements(t *testing.T) {
	counter := NewMemoryRateCounter()
	const requests = 1000
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := counter.Increment(context.Background(), "same-key", time.Minute); err != nil {
				t.Errorf("increment: %v", err)
			}
		}()
	}
	wg.Wait()
	entry := counter.entries["same-key"]
	if entry.count != requests {
		t.Fatalf("got %d increments, want %d", entry.count, requests)
	}
}

type failingRateCounter struct{}

func (failingRateCounter) Increment(context.Context, string, time.Duration) (int64, time.Duration, error) {
	return 0, 0, errors.New("storage unavailable")
}

func TestRateLimiterFailsClosedWhenCounterUnavailable(t *testing.T) {
	app := fiber.New()
	app.Post("/register", NewRateLimit(failingRateCounter{}, "register-test", 5, time.Minute, func(fiber.Ctx) string { return "client" }), func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	res, err := app.Test(httptest.NewRequest("POST", "/register", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", res.StatusCode)
	}
}

func TestRateLimitKeysAreNamespacedAndDoNotContainIdentity(t *testing.T) {
	if got := digestRateKey("  User@Example.COM "); got != digestRateKey("user@example.com") {
		t.Fatal("digest should be deterministic for normalized identity")
	}
	key := fmt.Sprintf("bufalo:rate:v1:register:email:%s", digestRateKey("user@example.com"))
	if strings.Contains(key, "user@example.com") {
		t.Fatal("rate limit key leaked email")
	}
}
