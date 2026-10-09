package monitoring

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestHistoryIsBoundedOrderedAndCopiedUnderConcurrency(t *testing.T) {
	var h eventHistory
	var wg sync.WaitGroup
	for n := 0; n < 1000; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.record(RequestEvent{At: time.Now(), Method: "GET", Route: "/loads/:id<int>", Status: 500})
			_ = h.snapshot(time.Now())
		}()
	}
	wg.Wait()
	s := h.snapshot(time.Now())
	if len(s.Requests) != 500 || len(s.Errors) != 200 || len(s.Trend) != 60 {
		t.Fatalf("retention: requests=%d errors=%d trend=%d", len(s.Requests), len(s.Errors), len(s.Trend))
	}
	for n := 1; n < len(s.Requests); n++ {
		if s.Requests[n-1].ID <= s.Requests[n].ID {
			t.Fatal("history out of order")
		}
	}
	s.Requests[0].Route = "changed"
	if h.snapshot(time.Now()).Requests[0].Route == "changed" {
		t.Fatal("snapshot aliases internal storage")
	}
}

func TestMonitoringRecordsFinalErrorsWithoutSensitiveRequestData(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware())
	app.Use(func(c fiber.Ctx) error {
		if err := c.Next(); err != nil {
			return c.Status(503).SendString("unavailable")
		}
		return nil
	})
	app.Post("/audit/:id", func(c fiber.Ctx) error { return fiber.ErrServiceUnavailable })
	req := httptest.NewRequest("POST", "/audit/secret-account?token=secret-token", strings.NewReader("password=secret-password"))
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	s := RecentHistory()
	event := s.Requests[0]
	if event.Status != 503 || event.Route != "/audit/:id" {
		t.Fatalf("wrong final event: %+v", event)
	}
	encoded, _ := json.Marshal(event)
	if strings.Contains(string(encoded), "secret-") {
		t.Fatalf("private request data leaked: %s", encoded)
	}
}
