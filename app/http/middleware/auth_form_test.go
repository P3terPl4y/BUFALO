package middleware

import (
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestAuthFormRejectsAmbiguousAndOversizedInputs(t *testing.T) {
	app := fiber.New()
	app.Post("/register", AuthFormOnly, func(c fiber.Ctx) error { return c.SendStatus(204) })
	for _, tc := range []struct {
		name, path, kind, body string
		want                   int
	}{
		{"valid", "/register", "application/x-www-form-urlencoded", "email=a%40test.com", 204},
		{"json", "/register", "application/json", `{"email":"a@test.com"}`, 415},
		{"duplicate", "/register", "application/x-www-form-urlencoded", "email=a%40test.com&email=b%40test.com", 400},
		{"query override", "/register?email=b%40test.com", "application/x-www-form-urlencoded", "email=a%40test.com", 400},
		{"oversized", "/register", "application/x-www-form-urlencoded", "notes=" + strings.Repeat("x", 17<<10), 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.kind)
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != tc.want {
				t.Fatalf("got %d want %d", res.StatusCode, tc.want)
			}
		})
	}
}

func TestLoginAccountLimitIsAtomicAcrossApplicationsAndIPs(t *testing.T) {
	counter := NewMemoryRateCounter()
	newApp := func() *fiber.App {
		a := fiber.New()
		a.Post("/login", AuthFormOnly, LoginEmailRateLimiter(counter), func(c fiber.Ctx) error { return c.SendStatus(204) })
		return a
	}
	apps := []*fiber.App{newApp(), newApp()}
	var successes, limited atomic.Int32
	var wg sync.WaitGroup
	for n := 0; n < 64; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			email := "USER%40test.com"
			if n%2 == 0 {
				email = "%20user%40test.com%20"
			}
			req := httptest.NewRequest("POST", "/login", strings.NewReader("email="+email))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			res, err := apps[n%2].Test(req)
			if err != nil {
				t.Error(err)
				return
			}
			defer res.Body.Close()
			if res.StatusCode == 204 {
				successes.Add(1)
			} else if res.StatusCode == 429 {
				limited.Add(1)
			} else {
				t.Errorf("status=%d", res.StatusCode)
			}
		}(n)
	}
	wg.Wait()
	if successes.Load() != 20 || limited.Load() != 44 {
		t.Fatalf("success=%d limited=%d", successes.Load(), limited.Load())
	}
}
