package controllers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"goravel/app/monitoring"
)

func TestHealthReportSectionsApplySelectedHistoryFilters(t *testing.T) {
	app := fiber.New()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	history := monitoring.HistorySnapshot{
		Requests: []monitoring.RequestEvent{
			{At: now, Method: "GET", Route: "/admin/health", Status: 200, DurationMS: 2},
			{At: now.Add(-time.Second), Method: "GET", Route: "/admin/metrics", Status: 503, DurationMS: 9},
			{At: now.Add(-2 * time.Second), Method: "POST", Route: "/admin/metrics", Status: 500, DurationMS: 12},
		},
		Errors: []monitoring.RequestEvent{
			{At: now.Add(-time.Second), Method: "GET", Route: "/admin/metrics", Status: 503, DurationMS: 9},
			{At: now.Add(-2 * time.Second), Method: "POST", Route: "/admin/metrics", Status: 500, DurationMS: 12},
		},
	}
	app.Get("/export", func(ctx fiber.Ctx) error {
		sections := healthReportSections(fiber.Map{"status": "ok", "checked_at": now.Format(time.RFC3339)}, history, ctx)
		return ctx.JSON(sections[len(sections)-1])
	})

	request := httptest.NewRequest("GET", "/export?tab=errors&method=GET&status=5&route=METRICS", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("export filter status = %d", response.StatusCode)
	}
	var section struct {
		Title string             `json:"Title"`
		Rows  [][]map[string]any `json:"Rows"`
	}
	if err := json.NewDecoder(response.Body).Decode(&section); err != nil {
		t.Fatal(err)
	}
	if section.Title != "Historial de errores" || len(section.Rows) != 1 {
		t.Fatalf("unexpected filtered history section: %+v", section)
	}
	if got := section.Rows[0][2]["Value"]; got != "/admin/metrics" {
		t.Fatalf("filtered route = %v", got)
	}
}

func TestHealthExportRejectsUnsupportedFormatBeforeCollectingMetrics(t *testing.T) {
	app := fiber.New()
	controller := &AdminController{}
	app.Get("/admin/health/export", controller.HealthExport)
	response, err := app.Test(httptest.NewRequest("GET", "/admin/health/export?format=pdf", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("unsupported format status = %d, want 400", response.StatusCode)
	}
}
