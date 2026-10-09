package viewhelpers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v3"
	"goravel/app/models"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllTemplatesLoad(t *testing.T) {
	engine := html.New(filepath.Join("..", "views"), ".html")
	Register(engine)
	if err := engine.Load(); err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{Views: engine})
	app.Get("/admin-dashboard-preview", func(c fiber.Ctx) error {
		return c.Render("admin/dashboard", fiber.Map{
			"title": "Panel de Administración", "role": "admin", "isAdmin": true,
			"user":       fiber.Map{"Name": "Admin"},
			"totalUsers": 0, "totalEmpresas": 0, "totalChoferes": 0, "totalPublicadores": 0,
			"totalDirecciones": 0, "totalCargas": 0, "cargasPublicadas": 0, "cargasAsignadas": 0,
			"cargasEntregadas": 0, "cargasCanceladas": 0, "totalFacturas": 0,
			"facturasPagadas": 0, "facturasPendientes": 0,
			"ultimosUsers": []models.User{}, "ultimasCargas": []models.Carga{},
		}, "layouts/base")
	})
	app.Get("/admin-metrics-preview", func(c fiber.Ctx) error {
		return c.Render("admin/metrics", fiber.Map{
			"title": "Métricas del sistema", "role": "admin", "metricsAt": "05/10/2026 12:00:00",
			"totalUsers": 4, "activeUsers": 3, "inactiveUsers": 1, "connectedUsers": 2,
			"disconnectedUsers": 1, "onlineWindow": "2m0s", "roles": []map[string]any{{"Rol": "admin", "Total": int64(1)}},
			"totalCompanies": 1, "totalDrivers": 1, "totalPublishers": 1, "totalAddresses": 2,
			"totalNetworkMemberships": 1, "totalRatings": 2,
			"loadStates": []map[string]any{{"Estado": "publicada", "Total": int64(1)}}, "totalLoads": 1,
			"invoiceStates": []map[string]any{{"Estado": "pagada", "Total": int64(1)}}, "totalInvoices": 1,
			"invoicesByCompany": []map[string]any{{"Nombre": "Empresa demo", "Total": int64(1)}},
			"invoicesByDriver":  []map[string]any{}, "invoicesByPublisher": []map[string]any{},
		}, "layouts/base")
	})
	app.Get("/admin-users-create-preview", func(c fiber.Ctx) error {
		return c.Render("admin/users/create", fiber.Map{
			"title": "Crear Usuario", "role": "admin", "csrfToken": "csrf-test",
			"old":             fiber.Map{"Role": "chofer", "Country": "Cuba", "Radius": 100},
			"empresasCarrier": []models.Empresa{{ID: 1, NombreLegal: "Transportes Demo"}},
			"empresasBroker":  []models.Empresa{{ID: 2, NombreLegal: "Broker Demo"}},
		}, "layouts/base")
	})
	response, err := app.Test(httptest.NewRequest("GET", "/admin-dashboard-preview", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Salud y rendimiento", "/admin/health", "healthPostgres", "healthRedis", "healthP95", "href=\"/admin/metrics\""} {
		if !strings.Contains(string(body), expected) {
			t.Errorf("rendered admin dashboard does not contain %q", expected)
		}
	}
	response, err = app.Test(httptest.NewRequest("GET", "/admin-metrics-preview", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Métricas del sistema", "Conectados recientemente", "Activos sin actividad reciente", "Facturas por chofer asociado", "Facturas por publicador asociado", "Salud y rendimiento", "/js/admin-metrics.js", "/admin/metrics/export?format=csv", "/admin/metrics/export?format=excel", "/admin/metrics/export?format=pdf"} {
		if !strings.Contains(string(body), expected) {
			t.Errorf("rendered admin metrics page does not contain %q", expected)
		}
	}
	if !strings.Contains(string(body), `href="/admin/metrics"`) {
		t.Error("admin sidebar metrics link is missing")
	}
	response, err = app.Test(httptest.NewRequest("GET", "/admin-users-create-preview", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"Crear Nuevo Usuario", `name="city"`, `name="chofer_numero_licencia"`,
		`name="publicador_numero_licencia_broker"`, `name="empresa_mode"`, "Transportes Demo", "Broker Demo",
	} {
		if !strings.Contains(string(body), expected) {
			t.Errorf("rendered admin user form does not contain %q", expected)
		}
	}
}

func TestDriverDoesNotSeePublisherActions(t *testing.T) {
	engine := html.New(filepath.Join("..", "views"), ".html")
	Register(engine)
	app := fiber.New(fiber.Config{Views: engine})
	app.Get("/home/:role", func(c fiber.Ctx) error {
		role := c.Params("role")
		return c.Render("home", fiber.Map{
			"title": "Inicio", "role": role, "user": fiber.Map{"Name": "Usuario"},
			"isAdmin": role == "admin", "isPublicador": role == "publicador", "isChofer": role == "chofer",
			"cargasDisponibles": []models.Carga{}, "cargasAsignadas": []models.Carga{}, "cargasEntregadas": []models.Carga{},
		}, "layouts/base")
	})
	app.Get("/facturas/:role", func(c fiber.Ctx) error {
		return c.Render("facturas/index", fiber.Map{
			"title": "Facturas", "role": c.Params("role"), "facturas": []models.Factura{},
			"total": 0, "page": 1, "perPage": 10, "hasNext": false,
			"filters": fiber.Map{"estado": "", "fecha_desde": "", "fecha_hasta": ""},
		}, "layouts/base")
	})

	for _, role := range []string{"chofer", "publicador", "admin"} {
		t.Run(role, func(t *testing.T) {
			for _, path := range []string{"/home/" + role, "/facturas/" + role} {
				response, err := app.Test(httptest.NewRequest("GET", path, nil))
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(response.Body)
				response.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != fiber.StatusOK {
					t.Fatalf("%s returned %d: %s", path, response.StatusCode, string(body))
				}
				content := string(body)
				wantsPublisherActions := role == "publicador" || role == "admin"
				if strings.Contains(path, "/home/") {
					if got := strings.Contains(content, `href="/loads/create"`); got != wantsPublisherActions {
						t.Errorf("%s publisher load link visible=%v", path, got)
					}
					if got := strings.Contains(content, ">Publicar carga<"); got != wantsPublisherActions {
						t.Errorf("%s publish action visible=%v", path, got)
					}
				} else {
					if strings.Contains(content, `href="/facturas/create"`) != (role == "publicador" || role == "chofer" || role == "admin") {
						t.Errorf("%s invoice create link role mismatch", path)
					}
				}
				if !strings.Contains(content, `href="/facturas"`) {
					t.Errorf("%s should retain invoice access", path)
				}
				if strings.Contains(path, "/facturas/") {
					for _, format := range []string{"csv", "excel", "pdf"} {
						if !strings.Contains(content, "/facturas/export?format="+format) {
							t.Errorf("%s missing %s invoice export action", role, format)
						}
					}
				}
			}
		})
	}
}
