package viewhelpers

import (
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v3"
)

func TestHealthPageRendersAndNavigationIsAdminOnly(t *testing.T) {
	engine := html.New(filepath.Join("..", "views"), ".html")
	Register(engine)
	app := fiber.New(fiber.Config{Views: engine})
	app.Get("/preview/:role", func(c fiber.Ctx) error {
		return c.Render("admin/health", fiber.Map{"title": "Salud del sistema", "role": c.Params("role")}, "layouts/base")
	})
	for _, role := range []string{"admin", "chofer"} {
		res, err := app.Test(httptest.NewRequest("GET", "/preview/"+role, nil))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("render health: %d %s", res.StatusCode, body)
		}
		for _, s := range []string{"Historial de actividad", "healthHistoryRows", "healthTrend", "/js/admin-health.js", "trafficLimited", "trafficBanned", "trafficMode", "/js/admin-traffic.js"} {
			if !strings.Contains(string(body), s) {
				t.Errorf("missing %q", s)
			}
		}
		if strings.Contains(string(body), `href="/admin/health"`) != (role == "admin") {
			t.Fatal("health navigation exposed to wrong role")
		}
	}
}
