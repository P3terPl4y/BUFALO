package middleware

import (
	"github.com/gofiber/fiber/v3"
	"net/http/httptest"
	"testing"
)

func TestLoginLimiterRejectsSixthAttempt(t *testing.T) {
	app := fiber.New()
	app.Post("/login", LoginRateLimiter(), func(c fiber.Ctx) error { return c.SendStatus(200) })
	for i := 1; i <= 6; i++ {
		res, err := app.Test(httptest.NewRequest("POST", "/login", nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		want := 200
		if i == 6 {
			want = 429
		}
		if res.StatusCode != want {
			t.Fatalf("attempt %d got %d want %d", i, res.StatusCode, want)
		}
	}
}
