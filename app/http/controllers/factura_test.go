package controllers_test

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v3"
	"goravel/app/facades"
	"goravel/app/http/controllers"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/viewhelpers"
	"goravel/tests"
	"io"
	"net/http/httptest"
	"net/url"
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
	other.EmpresaID = &emisor.ID
	other.IsActive = true
	driverUser := tests.NewUserChofer("invoice-driver@test.local")
	driverUser.EmpresaID = &receptor.ID
	peerDriverUser := tests.NewUserChofer("invoice-peer@test.local")
	peerDriverUser.EmpresaID = &receptor.ID
	for _, u := range []*models.User{owner, other, driverUser, peerDriverUser} {
		if err := facades.Orm().Query().Create(u); err != nil {
			t.Fatal(err)
		}
	}
	ownerProfile := tests.NewPublicadorProfile()
	ownerProfile.UserID, ownerProfile.EmpresaID = owner.ID, emisor.ID
	otherProfile := tests.NewPublicadorProfile()
	otherProfile.NumeroLicenciaBroker = "BRK-OTHER"
	otherProfile.UserID, otherProfile.EmpresaID = other.ID, emisor.ID
	for _, profile := range []*models.Publicador{ownerProfile, otherProfile} {
		if err := facades.Orm().Query().Create(profile); err != nil {
			t.Fatal(err)
		}
	}
	driverProfile := tests.NewChoferProfile()
	driverProfile.UserID, driverProfile.EmpresaID = driverUser.ID, receptor.ID
	peerDriverProfile := tests.NewChoferProfile()
	peerDriverProfile.NumeroLicencia = "LIC-PEER"
	peerDriverProfile.UserID, peerDriverProfile.EmpresaID = peerDriverUser.ID, receptor.ID
	for _, profile := range []*models.Chofer{driverProfile, peerDriverProfile} {
		if err := facades.Orm().Query().Create(profile); err != nil {
			t.Fatal(err)
		}
	}
	f := &models.Factura{CargaID: 1, EmisorID: &emisor.ID, ReceptorID: &receptor.ID, PublicadorID: &ownerProfile.ID, ChoferID: &driverProfile.ID, EmisorTipo: models.EmisorFacturaPublicador, NumeroFactura: "HTTP-001", FechaEmision: time.Now(), Subtotal: 100, Total: 100, Moneda: models.MonedaUSD, Estado: models.FacturaBorrador}
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
	active = driverUser
	if code := request("GET", "", ""); code != 200 {
		t.Fatalf("associated driver show: %d", code)
	}
	if code := request("GET", "/edit", ""); code != 403 {
		t.Fatalf("non-issuer driver edit: %d", code)
	}
	active = peerDriverUser
	if code := request("GET", "", ""); code != 403 {
		t.Fatalf("peer driver in same carrier company: %d", code)
	}
	var saved models.Factura
	if err := facades.Orm().Query().Where("id = ?", f.ID).First(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Estado != models.FacturaPagada || saved.FechaPago == nil {
		t.Fatalf("payment not persisted: %+v", saved)
	}
}

func TestFacturaHTTPStoreCurrencyAndDueDateMatrix(t *testing.T) {
	tests.ResetDB(t)
	emisor := tests.SeedEmpresa(t, "broker", "Emisor HTTP matriz")
	user := tests.NewUserPublicador("invoice-create-matrix@test.local")
	user.EmpresaID = &emisor.ID
	user.IsActive = true
	if err := facades.Orm().Query().Create(user); err != nil {
		t.Fatal(err)
	}
	profile := tests.NewPublicadorProfile()
	profile.UserID, profile.EmpresaID = user.ID, emisor.ID
	if err := facades.Orm().Query().Create(profile); err != nil {
		t.Fatal(err)
	}
	driverUser := tests.NewUserChofer("invoice-create-matrix-driver@test.local")
	driverUser.IsActive = true
	if err := facades.Orm().Query().Create(driverUser); err != nil {
		t.Fatal(err)
	}
	driver := tests.NewChoferProfile()
	driver.UserID = driverUser.ID
	if err := facades.Orm().Query().Create(driver); err != nil {
		t.Fatal(err)
	}
	templateService := services.NewFacturaPlantillaService()
	defaultTemplate, err := templateService.Save(profile.ID, services.FacturaPlantillaInput{
		Nombre: "Factura broker", Preset: "classic", Formato: "a4", Color: "#253746",
		Bloques: []string{"brand", "parties", "details", "items", "totals"}, Predeterminada: true,
	})
	if err != nil {
		t.Fatalf("save default invoice template: %v", err)
	}
	selectedTemplate, err := templateService.Save(profile.ID, services.FacturaPlantillaInput{
		Nombre: "Recibo broker", Preset: "receipt", Formato: "receipt", Color: "#123456",
		Bloques: []string{"parties", "details", "items", "totals"},
	})
	if err != nil {
		t.Fatalf("save selected invoice template: %v", err)
	}

	engine := html.New(findProjectRoot()+"/app/views", ".html")
	viewhelpers.Register(engine)
	app := fiber.New(fiber.Config{Views: engine})
	app.Use(session.New())
	activeUser := user
	app.Use(func(c fiber.Ctx) error {
		s := session.FromContext(c)
		s.Set("user_id", activeUser.ID)
		s.Set("role", activeUser.Role)
		return c.Next()
	})
	app.Post("/facturas", controllers.NewFacturaController().Store)
	app.Get("/facturas/:id<int>", controllers.NewFacturaController().Show)
	app.Get("/facturas/plantillas", controllers.NewFacturaController().ListTemplates)
	app.Post("/facturas/plantillas", controllers.NewFacturaController().SaveTemplate)
	app.Post("/facturas/:id<int>/plantilla", controllers.NewFacturaController().ChooseTemplate)

	caseNumber := 0
	for _, issuer := range []models.TipoEmisorFactura{models.EmisorFacturaPublicador, models.EmisorFacturaChofer} {
		activeUser = user
		issuerProfileID := profile.ID
		if issuer == models.EmisorFacturaChofer {
			activeUser, issuerProfileID = driverUser, driver.ID
		}
		for _, currency := range []models.Moneda{models.MonedaCUP, models.MonedaMLC, models.MonedaUSD, models.MonedaEUR} {
			for _, withDueDate := range []bool{false, true} {
				caseNumber++
				load := &models.Carga{
					NumeroReferencia: fmt.Sprintf("HTTP-MATRIX-%02d", caseNumber),
					PublicadorID:     profile.ID, EmpresaID: emisor.ID, ChoferID: &driver.ID,
					OrigenDireccionID: 1, DestinoDireccionID: 2,
					FechaRecogida: time.Now(), TipoCarga: models.CargaFTL,
					TipoEquipo: models.EquipoDryVan, Estado: models.CargaEntregada, Moneda: currency,
				}
				if err := facades.Orm().Query().Create(load); err != nil {
					t.Fatalf("case %d create load: %v", caseNumber, err)
				}
				form := url.Values{
					"carga_id":       {fmt.Sprint(load.ID)},
					"numero_factura": {fmt.Sprintf("HTTP-CREATE-%02d", caseNumber)},
					"fecha_emision":  {"2026-01-01"},
					"distancia_km":   {"10.25"}, "tarifa_por_km": {"2.5"},
					"subtotal": {"123.45"}, "impuestos": {"6.55"}, "total": {"999"},
					"moneda": {string(currency)}, "metodo_pago": {"transferencia"},
				}
				templateID := defaultTemplate.ID
				if caseNumber%2 == 0 {
					templateID = selectedTemplate.ID
				}
				form.Set("plantilla_id", fmt.Sprint(templateID))
				if withDueDate {
					form.Set("fecha_vencimiento", "2026-02-01")
				}
				req := httptest.NewRequest("POST", "/facturas", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				res, err := app.Test(req)
				if err != nil {
					t.Fatalf("case %d request: %v", caseNumber, err)
				}
				res.Body.Close()
				if res.StatusCode < 300 || res.StatusCode >= 400 {
					t.Fatalf("case %d (%s, %s, due=%v) returned %d", caseNumber, issuer, currency, withDueDate, res.StatusCode)
				}
				var saved models.Factura
				if err := facades.Orm().Query().Where("numero_factura = ?", form.Get("numero_factura")).First(&saved); err != nil {
					t.Fatalf("case %d invoice not persisted: %v", caseNumber, err)
				}
				if saved.Moneda != currency || saved.Estado != models.FacturaBorrador || saved.Total != 130 || saved.EmisorTipo != issuer || saved.MetodoPago != nil {
					t.Errorf("case %d saved incorrect invoice: %+v", caseNumber, saved)
				}
				if saved.PlantillaID == nil || *saved.PlantillaID != templateID {
					t.Errorf("case %d selected template id=%v, want %d", caseNumber, saved.PlantillaID, templateID)
				}
				if issuer == models.EmisorFacturaPublicador && (saved.PublicadorID == nil || *saved.PublicadorID != issuerProfileID) {
					t.Errorf("case %d publisher issuer profile missing: %+v", caseNumber, saved)
				}
				if issuer == models.EmisorFacturaChofer && (saved.ChoferID == nil || *saved.ChoferID != issuerProfileID) {
					t.Errorf("case %d driver issuer profile missing: %+v", caseNumber, saved)
				}
				if (saved.FechaVencimiento != nil) != withDueDate {
					t.Errorf("case %d due date present=%v, want %v", caseNumber, saved.FechaVencimiento != nil, withDueDate)
				}
			}
		}
	}
	activeUser = driverUser
	var driverInvoice models.Factura
	if err := facades.Orm().Query().Where("numero_factura = ?", "HTTP-CREATE-16").First(&driverInvoice); err != nil {
		t.Fatal(err)
	}
	response, err := app.Test(httptest.NewRequest("GET", fmt.Sprintf("/facturas/%d", driverInvoice.ID), nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || !strings.Contains(string(body), "Formato que ve el chofer") || !strings.Contains(string(body), "HTTP-CREATE-16") || !strings.Contains(string(body), "Recibo broker") {
		t.Fatalf("driver did not receive selected invoice design: status=%d body_error=%v", response.StatusCode, err)
	}
	if response, err = app.Test(httptest.NewRequest("GET", "/facturas/plantillas", nil)); err != nil {
		t.Fatal(err)
	} else {
		response.Body.Close()
		if response.StatusCode != 403 {
			t.Fatalf("driver accessed broker template API: %d", response.StatusCode)
		}
	}
	activeUser = user
	form := url.Values{"config": {`{"name":"Desde API","preset":"classic","format":"a4","color":"#345678","blocks":["parties","details","totals"],"default":false}`}}
	request := httptest.NewRequest("POST", "/facturas/plantillas", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	apiBody, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 201 || !strings.Contains(string(apiBody), `"name":"Desde API"`) {
		t.Fatalf("broker template API did not save a design: status=%d body_error=%v", response.StatusCode, err)
	}
	var apiTemplate models.FacturaPlantilla
	if err := facades.Orm().Query().Where("nombre = ? AND publicador_id = ?", "Desde API", profile.ID).First(&apiTemplate); err != nil {
		t.Fatal(err)
	}
	var selectedInvoice models.Factura
	if err := facades.Orm().Query().Where("numero_factura = ?", "HTTP-CREATE-16").First(&selectedInvoice); err != nil {
		t.Fatal(err)
	}
	activeUser = user
	chooseForm := url.Values{"plantilla_id": {fmt.Sprint(apiTemplate.ID)}}
	chooseRequest := httptest.NewRequest("POST", fmt.Sprintf("/facturas/%d/plantilla", selectedInvoice.ID), strings.NewReader(chooseForm.Encode()))
	chooseRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err = app.Test(chooseRequest)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode < 300 || response.StatusCode >= 400 {
		t.Fatalf("broker could not change invoice design: %d", response.StatusCode)
	}
	activeUser = driverUser
	response, err = app.Test(httptest.NewRequest("GET", fmt.Sprintf("/facturas/%d", selectedInvoice.ID), nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || !strings.Contains(string(body), "Desde API") {
		t.Fatalf("driver did not see broker's changed design: status=%d body_error=%v", response.StatusCode, err)
	}
}
