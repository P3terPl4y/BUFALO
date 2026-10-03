package controllers_test

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v3"
	"goravel/app/facades"
	"goravel/app/http/controllers"
	"goravel/app/models"
	"goravel/app/viewhelpers"
	"goravel/tests"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFacturaHTTPAuthorizationAndLifecycle(t *testing.T) {
	tests.ResetDB(t)
	emisor := tests.SeedEmpresa(t, "broker", "Broker")
	receptor := tests.SeedEmpresa(t, "carrier", "Carrier")
	owner := tests.NewUserPublicador("invoice-owner@test.local")
	owner.EmpresaID = &emisor.ID
	owner.IsActive = true
	other := tests.NewUserPublicador("invoice-other@test.local")
	other.EmpresaID = &receptor.ID
	other.IsActive = true
	for _, u := range []*models.User{owner, other} {
		if err := facades.Orm().Query().Create(u); err != nil {
			t.Fatal(err)
		}
	}
	f := &models.Factura{CargaID: 1, EmisorID: emisor.ID, ReceptorID: receptor.ID, NumeroFactura: "HTTP-001", FechaEmision: time.Now(), Subtotal: 100, Total: 100, Moneda: models.MonedaUSD, Estado: models.FacturaBorrador}
	if err := facades.Orm().Query().Create(f); err != nil {
		t.Fatal(err)
	}
	engine := html.New(findProjectRoot()+"/app/views", ".html")
	viewhelpers.Register(engine)
	app := fiber.New(fiber.Config{Views: engine})
	app.Use(session.New())
	active := other
	app.Use(func(c fiber.Ctx) error {
		s := session.FromContext(c)
		s.Set("user_id", active.ID)
		s.Set("role", active.Role)
		return c.Next()
	})
	ctrl := controllers.NewFacturaController()
	app.Get("/facturas/:id", ctrl.Show)
	app.Get("/facturas/:id/edit", ctrl.Edit)
	app.Post("/facturas/:id", ctrl.Update)
	app.Post("/facturas/:id/emitir", ctrl.Emitir)
	app.Post("/facturas/:id/pagar", ctrl.MarcarPagada)
	app.Post("/facturas/:id/delete", ctrl.Delete)
	path := fmt.Sprintf("/facturas/%d", f.ID)
	request := func(method, suffix, body string) int {
		t.Helper()
		req := httptest.NewRequest(method, path+suffix, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		return res.StatusCode
	}
	for _, route := range []struct{ method, suffix string }{{"GET", ""}, {"GET", "/edit"}, {"POST", ""}, {"POST", "/emitir"}, {"POST", "/pagar"}, {"POST", "/delete"}} {
		if code := request(route.method, route.suffix, ""); code != 403 {
			t.Errorf("foreign %s %s: %d", route.method, route.suffix, code)
		}
	}
	active = owner
	if code := request("GET", "", ""); code != 200 {
		t.Fatalf("owner show: %d", code)
	}
	if code := request("GET", "/edit", ""); code != 200 {
		t.Fatalf("owner edit: %d", code)
	}
	if code := request("POST", "/emitir", ""); code < 300 || code >= 400 {
		t.Fatalf("issue: %d", code)
	}
	if code := request("POST", "/emitir", ""); code != 409 {
		t.Fatalf("duplicate issue: %d", code)
	}
	if code := request("POST", "/pagar", "metodo_pago=transferencia"); code < 300 || code >= 400 {
		t.Fatalf("pay: %d", code)
	}
	var saved models.Factura
	if err := facades.Orm().Query().Where("id = ?", f.ID).First(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Estado != models.FacturaPagada || saved.FechaPago == nil {
		t.Fatalf("payment not persisted: %+v", saved)
	}
}
