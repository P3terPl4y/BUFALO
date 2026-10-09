package viewhelpers_test

import (
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v3"
	"goravel/app/models"
	"goravel/app/viewhelpers"
)

func TestEntityDirectoryTemplatesRenderCardAndTableModes(t *testing.T) {
	engine := html.New(filepath.Join("..", "views"), ".html")
	viewhelpers.Register(engine)
	app := fiber.New(fiber.Config{Views: engine})
	user := &models.User{Name: "Chofer de prueba", Email: "driver@example.test", ProfilePhoto: "/uploads/avatars/driver.jpg", Role: "chofer", IsActive: true}
	driver := models.Chofer{ID: 9, User: user, Estado: models.ChoferDisponible, RatingAverage: 4.8, RatingCount: 3, NumeroLicencia: "L-9", AniosExperiencia: 5}
	publisher := models.Publicador{ID: 11, User: &models.User{Name: "Broker de prueba", ProfilePhoto: "/uploads/avatars/broker.jpg"}, Estado: models.PublicadorActivo, NumeroLicenciaBroker: "B-11", AniosExperiencia: 4}
	company := models.Empresa{ID: 13, NombreLegal: "Empresa de prueba", Tipo: models.TipoCarrier, Estado: models.EmpresaActiva, ProfilePhoto: "/uploads/company-logos/company.png", Owner: &models.User{Name: "Dueño de prueba", ProfilePhoto: "/uploads/avatars/owner.jpg"}}
	member := models.User{Name: "Miembro de prueba", ProfilePhoto: "/uploads/avatars/member.jpg", Role: "chofer", IsActive: true, Chofer: &driver}
	load := models.Carga{ID: 17, NumeroReferencia: "REF-17", Estado: models.EstadoCarga("publicada"), FechaRecogida: time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC), Publicador: &publisher, Chofer: &driver, OrigenDireccion: &models.Direccion{Ciudad: "Origen", EstadoProvincia: "Estado A"}, DestinoDireccion: &models.Direccion{Ciudad: "Destino", EstadoProvincia: "Estado B"}}

	views := []struct {
		name string
		data fiber.Map
		want []string
	}{
		{"admin/users/index", fiber.Map{"users": []models.User{}, "total": 0, "page": 1, "totalPages": 1, "filters": map[string]string{}, "csrfToken": "csrf"}, []string{"data-entity-view", "data-view-cards", "data-view-table", "/js/entity-view-toggle.js"}},
		{"choferes/index", fiber.Map{"choferes": []models.Chofer{driver}, "total": 1, "filters": map[string]string{}, "role": "admin"}, []string{"Chofer de prueba", "/uploads/avatars/driver.jpg", "data-view-cards", "data-view-table"}},
		{"admin/choferes/index", fiber.Map{"choferes": []models.Chofer{driver}, "total": 1, "csrfToken": "csrf"}, []string{"Chofer de prueba", "/uploads/avatars/driver.jpg", "data-view-cards", "data-view-table"}},
		{"publicadores/index", fiber.Map{"publicadores": []models.Publicador{publisher}, "total": 1, "filters": map[string]string{}, "role": "admin"}, []string{"Broker de prueba", "/uploads/avatars/broker.jpg", "data-view-cards", "data-view-table"}},
		{"admin/publicadores/index", fiber.Map{"publicadores": []models.Publicador{publisher}, "total": 1, "csrfToken": "csrf"}, []string{"Broker de prueba", "/uploads/avatars/broker.jpg", "data-view-cards", "data-view-table"}},
		{"empresas/index", fiber.Map{"empresas": []models.Empresa{company}, "total": 1, "page": 1, "perPage": 10, "filters": map[string]string{}, "role": "admin"}, []string{"Empresa de prueba", "/uploads/company-logos/company.png", "data-view-cards", "data-view-table"}},
		{"admin/empresas/index", fiber.Map{"empresas": []models.Empresa{company}, "total": 1, "filters": map[string]string{}, "csrfToken": "csrf"}, []string{"Empresa de prueba", "Dueño de prueba", "/uploads/avatars/owner.jpg", "data-view-cards", "data-view-table"}},
		{"empresas/show", fiber.Map{"empresa": company, "members": []models.User{member}, "role": "admin", "userID": uint(1), "csrfToken": "csrf", "driverPrevious": 0, "driverNext": 1, "driverHasNext": false}, []string{"Usuarios de la empresa", "Miembro de prueba", "/uploads/avatars/member.jpg", "data-view-cards", "data-view-table"}},
		{"empresas/chat", fiber.Map{"empresa": company, "messages": []fiber.Map{{"ID": uint(7), "UserID": uint(1), "UserName": "Miembro", "Message": "Mensaje de prueba", "Moderated": false, "CreatedAt": time.Now()}}, "moderator": true, "userID": uint(1), "csrfToken": "csrf"}, []string{"Chat de la empresa", "Mensaje de prueba", "Moderador", "/js/company-chat.js"}},
		{"dashboard/index", fiber.Map{"loads": []models.Carga{load}, "total": 1, "page": 1, "perPage": 10, "filters": map[string]string{}, "role": "admin", "csrfToken": "csrf"}, []string{"REF-17", "Broker de prueba", "Chofer de prueba", "data-view-cards", "data-view-table", "/js/entity-view-toggle.js"}},
		{"admin/dashboard/index", fiber.Map{"loads": []models.Carga{load}, "total": 1, "page": 1, "perPage": 10, "filters": map[string]string{}, "role": "admin", "csrfToken": "csrf"}, []string{"REF-17", "Broker de prueba", "Chofer de prueba", "data-view-cards", "data-view-table", "/js/entity-view-toggle.js"}},
		{"admin/cargas/index", fiber.Map{"loads": []models.Carga{load}, "total": 1, "csrfToken": "csrf"}, []string{"REF-17", "Broker de prueba", "Chofer de prueba", "data-view-cards", "data-view-table", "/js/entity-view-toggle.js"}},
	}
	for _, item := range views {
		path := "/render/" + strings.ReplaceAll(item.name, "/", "-")
		view := item.name
		data := item.data
		app.Get(path, func(c fiber.Ctx) error { return c.Render(view, data) })
		response, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatalf("render %s: %v", view, err)
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil || response.StatusCode != fiber.StatusOK {
			t.Fatalf("render %s: status=%d err=%v body=%s", view, response.StatusCode, readErr, body)
		}
		for _, expected := range item.want {
			if !strings.Contains(string(body), expected) {
				t.Errorf("%s missing %q", view, expected)
			}
		}
	}
}

func TestCompanyWorkspaceLinksAndOwnerAffiliationControls(t *testing.T) {
	engine := html.New(filepath.Join("..", "views"), ".html")
	viewhelpers.Register(engine)
	app := fiber.New(fiber.Config{Views: engine})
	company := models.Empresa{ID: 17, NombreLegal: "Empresa asociada", ProfilePhoto: "/uploads/company-logos/logo.png", CanChat: true, CanManage: true}
	app.Get("/company-workspace", func(c fiber.Ctx) error {
		return c.Render("empresas/index", fiber.Map{"empresas": []models.Empresa{company}, "total": 1, "filters": map[string]string{}, "role": "publicador", "csrfToken": "csrf"})
	})
	app.Get("/company-owner", func(c fiber.Ctx) error {
		return c.Render("empresas/show", fiber.Map{"empresa": company, "members": []models.User{}, "role": "publicador", "userID": uint(1), "csrfToken": "csrf", "canChat": true, "canManage": true, "isOwner": true, "driverPrevious": 0, "driverNext": 1, "driverHasNext": false})
	})
	app.Get("/company-directory-empty", func(c fiber.Ctx) error {
		return c.Render("empresas/index", fiber.Map{"empresas": []models.Empresa{}, "total": 0, "filters": map[string]string{}, "role": "chofer", "browse": true, "hasSearch": false, "csrfToken": "csrf"})
	})
	candidate := models.Empresa{ID: 18, NombreLegal: "Carrier disponible", Tipo: models.TipoCarrier, Estado: models.EmpresaActiva, CanRequestMembership: true}
	app.Get("/company-directory-results", func(c fiber.Ctx) error {
		return c.Render("empresas/index", fiber.Map{"empresas": []models.Empresa{candidate}, "total": 1, "filters": map[string]string{"q": "Carrier"}, "role": "chofer", "browse": true, "hasSearch": true, "csrfToken": "csrf"})
	})
	for _, tc := range []struct {
		path   string
		want   []string
		reject []string
	}{
		{path: "/company-workspace", want: []string{`href="/empresas/17/chat"`, `href="/empresas/17"`, "/uploads/company-logos/logo.png", "Abrir chat"}, reject: []string{"Solicitar afiliación"}},
		{path: "/company-owner", want: []string{"Ya perteneces a esta empresa", "/empresas/17/chat", "/uploads/company-logos/logo.png"}, reject: []string{`action="/empresas/17/membership"`}},
		{path: "/company-directory-empty", want: []string{"Busca una empresa", "Escribe el nombre, número MC o DOT"}, reject: []string{`action="/empresas/17/membership"`}},
		{path: "/company-directory-results", want: []string{"Carrier disponible", "Solicitar afiliación", `action="/empresas/18/membership"`}, reject: []string{}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response, err := app.Test(httptest.NewRequest("GET", tc.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			body, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr != nil || response.StatusCode != fiber.StatusOK {
				t.Fatalf("render status=%d err=%v body=%s", response.StatusCode, readErr, body)
			}
			for _, value := range tc.want {
				if !strings.Contains(string(body), value) {
					t.Errorf("missing %q", value)
				}
			}
			for _, value := range tc.reject {
				if strings.Contains(string(body), value) {
					t.Errorf("unexpected %q", value)
				}
			}
		})
	}
}

func TestCompanyEditLinkIsControlledByOwnerPermission(t *testing.T) {
	engine := html.New(filepath.Join("..", "views"), ".html")
	viewhelpers.Register(engine)
	app := fiber.New(fiber.Config{Views: engine})
	company := models.Empresa{ID: 13, NombreLegal: "Empresa privada", Tipo: models.TipoCarrier, Estado: models.EmpresaActiva}
	for _, test := range []struct {
		name      string
		canManage bool
		wantEdit  bool
	}{
		{name: "non-owner", canManage: false, wantEdit: false},
		{name: "owner", canManage: true, wantEdit: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			company.CanManage = test.canManage
			data := fiber.Map{"empresas": []models.Empresa{company}, "total": 1, "page": 1, "perPage": 10, "filters": map[string]string{}, "role": "publicador", "userID": uint(42)}
			app.Get("/company-"+test.name, func(c fiber.Ctx) error { return c.Render("empresas/index", data) })
			response, err := app.Test(httptest.NewRequest("GET", "/company-"+test.name, nil))
			if err != nil {
				t.Fatal(err)
			}
			body, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr != nil || response.StatusCode != fiber.StatusOK {
				t.Fatalf("render company index: status=%d err=%v", response.StatusCode, readErr)
			}
			hasEdit := strings.Contains(string(body), `href="/empresas/13/edit"`)
			if hasEdit != test.wantEdit {
				t.Fatalf("edit link present=%t for %s, want %t", hasEdit, test.name, test.wantEdit)
			}
		})
	}
}
