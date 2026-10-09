package redteam

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/viewhelpers"
	"goravel/bootstrap"
	"goravel/routes"
	"goravel/tests"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v3"
)

const password = "RedTeam!Test#2026"

var (
	server       *httptest.Server
	tokenPattern = regexp.MustCompile(`name="_csrf" value="([^"]+)"`)
)

type principal struct {
	user   *models.User
	role   string
	client *http.Client
	csrf   string
}

type fixture struct {
	people     []principal
	brokerIDs  []uint
	carrierID  uint
	invoiceID  uint
	addressID  uint
	ownerID    uint
	raceLoadID uint
}

func TestMain(m *testing.M) {
	os.Setenv("APP_ENV", "testing")
	os.Setenv("MAIL_MAILER", "log")
	os.Setenv("SESSION_DRIVER", "memory")
	os.Setenv("APP_KEY", "base64:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	app := bootstrap.Boot()
	app.Boot()
	if err := tests.RunMigrations(); err != nil {
		panic(err)
	}
	root, err := filepath.Abs(filepath.Join(projectRoot(), "app", "views"))
	if err != nil {
		panic(err)
	}
	engine := html.New(root, ".html")
	viewhelpers.Register(engine)
	appFiber := fiber.New(fiber.Config{Views: engine})
	appFiber.Use(recover.New())
	sessionMiddleware, sessionStore := session.NewWithStore(session.Config{Extractor: extractors.FromCookie("redteam_session"), CookieSecure: false, CookieHTTPOnly: true, CookieSameSite: "Lax", CookieSessionOnly: true, IdleTimeout: 30 * time.Minute, AbsoluteTimeout: 24 * time.Hour})
	appFiber.Use(sessionMiddleware)
	appFiber.Use(csrf.New(csrf.Config{CookieName: "redteam_csrf", CookieSecure: false, CookieHTTPOnly: true, CookieSameSite: "Lax", CookieSessionOnly: true, IdleTimeout: 30 * time.Minute, Session: sessionStore, Extractor: extractors.FromForm("_csrf")}))
	routes.SetupWebRoutes(appFiber, func(c fiber.Ctx) error { return c.Next() })
	server = httptest.NewServer(adaptor.FiberApp(appFiber))
	code := m.Run()
	server.Close()
	os.Exit(code)
}

func projectRoot() string {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		panic(err)
	}
	return root
}

func TestSecurityRedTeam100Users(t *testing.T) {
	tests.ResetDB(t)
	fx := seedFixture(t, 100)
	var missingProfileUserID uint
	if len(fx.people) != 100 {
		t.Fatalf("seeded %d principals; expected exactly 100", len(fx.people))
	}
	// A valid role with a missing profile must fail closed rather than receive
	// an unfiltered load list. This models stale/corrupt account data.
	for _, p := range fx.people {
		if p.role != "chofer" {
			continue
		}
		if _, err := facades.Orm().Query().Where("user_id = ?", p.user.ID).Delete(&models.Chofer{}); err != nil {
			t.Fatalf("remove test driver profile: %v", err)
		}
		missingProfileUserID = p.user.ID
		res := request(t, p.client, http.MethodGet, "/loads", nil, "", "")
		if res.StatusCode != http.StatusForbidden {
			t.Errorf("driver without profile got status %d, want 403", res.StatusCode)
		}
		res.Body.Close()
		break
	}
	for _, p := range fx.people {
		if p.role != "publicador" || p.user.ID == fx.ownerID {
			continue
		}
		res := request(t, p.client, http.MethodGet, "/direcciones/"+u(fx.addressID), nil, "", "")
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("foreign address detail returned %d, want 404", res.StatusCode)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if strings.Contains(string(body), "Ciudad de Prueba") {
			t.Error("foreign address detail leaked its city")
		}
		res = request(t, p.client, http.MethodGet, "/direcciones", nil, "", "")
		body, _ = io.ReadAll(res.Body)
		res.Body.Close()
		if strings.Contains(string(body), "Ciudad de Prueba") {
			t.Error("foreign address listing leaked its city")
		}
		break
	}
	admin := fx.people[2]
	if res := request(t, admin.client, http.MethodGet, "/admin/cargas", nil, "", ""); res.StatusCode != http.StatusOK {
		t.Errorf("admin loads page returned %d, want 200", res.StatusCode)
		res.Body.Close()
	} else {
		res.Body.Close()
	}
	var owner principal
	for _, p := range fx.people {
		if p.role == "publicador" && p.user.EmpresaID != nil && *p.user.EmpresaID == fx.brokerIDs[0] {
			owner = p
			break
		}
	}
	if owner.user == nil {
		t.Fatal("fixture lacks invoice-owner principal")
	}
	for _, path := range []string{"/publicadores", "/publicadores/1"} {
		res := request(t, owner.client, http.MethodGet, path, nil, "", "")
		if res.StatusCode != http.StatusOK {
			t.Errorf("publicador page %s returned %d, want 200", path, res.StatusCode)
		}
		res.Body.Close()
	}
	var ownerProfile models.Publicador
	if err := facades.Orm().Query().Where("user_id = ?", owner.user.ID).First(&ownerProfile); err != nil {
		t.Fatalf("load owner publisher profile: %v", err)
	}
	editRes := request(t, owner.client, http.MethodGet, "/publicadores/"+u(ownerProfile.ID)+"/edit", nil, "", "")
	if editRes.StatusCode != http.StatusOK {
		t.Errorf("publisher owner edit returned %d, want 200", editRes.StatusCode)
	}
	editRes.Body.Close()

	// Anonymous requests must not reach protected pages or mutations.
	anon := newClient(t)
	for _, path := range []string{"/home", "/loads", "/facturas", "/admin"} {
		res := request(t, anon, http.MethodGet, path, nil, "", "")
		if res.StatusCode != http.StatusSeeOther {
			t.Errorf("anonymous GET %s returned %d", path, res.StatusCode)
		}
		res.Body.Close()
	}
	if res := request(t, anon, http.MethodPost, "/facturas/1/pagar", url.Values{"metodo_pago": {"attacker"}}, "", ""); res.StatusCode != http.StatusForbidden {
		t.Errorf("anonymous state change returned %d, want 403", res.StatusCode)
		res.Body.Close()
	} else {
		res.Body.Close()
	}

	// Invalid and cross-origin writes are rejected even for authenticated sessions.
	res := request(t, owner.client, http.MethodPost, "/facturas/"+u(fx.invoiceID)+"/pagar", url.Values{"metodo_pago": {"transferencia"}}, "", "")
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("missing CSRF token returned %d, want 403", res.StatusCode)
	}
	res.Body.Close()
	res = request(t, owner.client, http.MethodPost, "/facturas/"+u(fx.invoiceID)+"/emitir", url.Values{"_csrf": {owner.csrf}}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Errorf("authorized invoice issue returned %d, want redirect", res.StatusCode)
	}
	res.Body.Close()
	var issued models.Factura
	if err := facades.Orm().Query().Where("id = ?", fx.invoiceID).First(&issued); err != nil {
		t.Fatalf("reload issued invoice: %v", err)
	}
	if issued.Estado != models.FacturaEmitida {
		t.Errorf("invoice state after issue=%s, want emitida", issued.Estado)
	}
	res = request(t, owner.client, http.MethodPost, "/facturas/"+u(fx.invoiceID)+"/pagar", url.Values{"_csrf": {owner.csrf}, "metodo_pago": {"transferencia"}}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Errorf("authorized invoice payment returned %d, want redirect", res.StatusCode)
	}
	res.Body.Close()
	res = request(t, owner.client, http.MethodPost, "/facturas/"+u(fx.invoiceID)+"/pagar", url.Values{"_csrf": {owner.csrf}, "metodo_pago": {"transferencia"}}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Errorf("duplicate invoice payment returned %d, want controlled redirect", res.StatusCode)
	}
	res.Body.Close()
	res = request(t, owner.client, http.MethodPost, "/facturas/"+u(fx.invoiceID)+"/pagar", url.Values{"_csrf": {owner.csrf}, "metodo_pago": {"transferencia"}}, owner.csrf, "https://evil.invalid")
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("cross-origin state change returned %d, want 403", res.StatusCode)
	}
	res.Body.Close()

	var mu sync.Mutex
	var failures []string
	record := func(msg string) { mu.Lock(); failures = append(failures, msg); mu.Unlock() }
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20)
	for i := range fx.people {
		p := fx.people[i]
		wg.Add(1)
		sem <- struct{}{}
		go func(index int, p principal) {
			defer wg.Done()
			defer func() { <-sem }()
			probes := []struct {
				method, path string
				form         url.Values
				want         int
				label        string
			}{
				{http.MethodGet, "/home", nil, http.StatusOK, "home"},
				{http.MethodGet, "/facturas/" + u(fx.invoiceID), nil, 0, "invoice object access"},
				{http.MethodGet, "/admin", nil, 0, "admin role boundary"},
				{http.MethodPost, "/facturas/" + u(fx.invoiceID) + "/pagar", url.Values{"_csrf": {p.csrf}, "metodo_pago": {"transferencia"}}, 0, "cross-company invoice payment"},
				{http.MethodPost, "/direcciones/" + u(fx.addressID) + "/update", url.Values{"_csrf": {p.csrf}, "ciudad": {"Attacker City"}, "estado_provincia": {"Test Province"}, "pais": {"Cuba"}}, 0, "address ownership"},
				{http.MethodPost, "/loads/" + u(fx.raceLoadID) + "/accept", url.Values{"_csrf": {p.csrf}}, 0, "role-restricted load accept"},
				{http.MethodPost, "/facturas", url.Values{"_csrf": {p.csrf}, "carga_id": {"NaN"}, "emisor_id": {"-1"}, "receptor_id": {"0"}, "numero_factura": {"x"}, "fecha_emision": {"nonsense"}, "subtotal": {"NaN"}, "total": {"999999999999999999999"}, "moneda": {"XYZ"}}, 0, "invalid invoice payload"},
			}
			for _, q := range probes {
				if q.label == "invoice object access" {
					// Invoice visibility follows the associated profile, not the
					// user's company. The fixture invoice belongs to the issuing
					// publisher and deliberately has no ChoferID.
					if p.role == "admin" || p.user.ID == fx.ownerID {
						q.want = http.StatusOK
					} else {
						q.want = http.StatusForbidden
					}
				}
				if q.label == "admin role boundary" {
					if p.role == "admin" {
						q.want = http.StatusOK
					} else {
						q.want = http.StatusSeeOther
					}
				}
				if q.label == "cross-company invoice payment" {
					// Only the issuer profile (or admin) may mutate this invoice;
					// sharing a company does not grant payment authority.
					if p.role == "admin" || p.user.ID == fx.ownerID {
						q.want = http.StatusSeeOther
					} else {
						q.want = http.StatusForbidden
					}
				}
				if q.label == "address ownership" {
					if p.role == "admin" || p.user.ID == fx.ownerID {
						q.want = http.StatusSeeOther
					} else {
						q.want = http.StatusForbidden
					}
				}
				if q.label == "role-restricted load accept" {
					if p.role == "chofer" || p.role == "admin" {
						q.want = http.StatusSeeOther
					} else {
						q.want = http.StatusSeeOther
					}
				}
				if q.label == "invalid invoice payload" {
					if p.role == "publicador" || p.role == "admin" {
						q.want = http.StatusSeeOther
					} else {
						q.want = http.StatusSeeOther
					}
				}
				if q.path == "/home" && p.user.ID == missingProfileUserID {
					q.want = http.StatusForbidden
				}
				res := request(nil, p.client, q.method, q.path, q.form, p.csrf, "")
				if res.StatusCode != q.want {
					record(fmt.Sprintf("user=%d role=%s %s %s [%s] got=%d want=%d", index, p.role, q.method, q.path, q.label, res.StatusCode, q.want))
				}
				res.Body.Close()
			}
			// An authenticated foreign origin with a valid token must still fail.
			res := request(nil, p.client, http.MethodPost, "/facturas/"+u(fx.invoiceID)+"/pagar", url.Values{"_csrf": {p.csrf}, "metodo_pago": {"transferencia"}}, p.csrf, "https://evil.invalid")
			if res.StatusCode != http.StatusForbidden {
				record(fmt.Sprintf("user=%d role=%s cross-origin POST returned %d", index, p.role, res.StatusCode))
			}
			res.Body.Close()
		}(i, p)
	}
	wg.Wait()
	if len(failures) > 0 {
		for _, f := range failures {
			t.Error(f)
		}
	}
	var contested models.Carga
	if err := facades.Orm().Query().Where("id = ?", fx.raceLoadID).First(&contested); err != nil {
		t.Fatalf("read contested load: %v", err)
	}
	if contested.Estado != models.CargaAsignada || contested.ChoferID == nil {
		t.Errorf("concurrent accept left load in invalid state: estado=%s chofer_id=%v", contested.Estado, contested.ChoferID)
	}
	winner := uint(0)
	if contested.ChoferID != nil {
		winner = *contested.ChoferID
	}
	t.Logf("security run: users=%d, role boundary/object/CSRF/invalid-input probes=%d, concurrent accept winner driver_id=%d", len(fx.people), len(fx.people)*8, winner)
}

// TestProductionJourneys exercises complete browser-style mutations with CSRF
// enabled and verifies persisted state, rather than accepting redirects alone.
func TestProductionJourneys(t *testing.T) {
	tests.ResetDB(t)
	fx := seedFixture(t, 4)
	var owner, driver, admin principal
	for _, p := range fx.people {
		switch p.role {
		case "publicador":
			if p.user.ID == fx.ownerID {
				owner = p
			}
		case "chofer":
			driver = p
		case "admin":
			admin = p
		}
	}
	if owner.user == nil || driver.user == nil || admin.user == nil {
		t.Fatal("fixtures must include each role")
	}

	// Create, edit, assign and delete a load through the publisher routes.
	now := time.Now().Add(48 * time.Hour).Format("2006-01-02T15:04")
	loadForm := url.Values{
		"numero_referencia": {"FLOW-LOAD-001"}, "origen_direccion_id": {u(fx.addressID)},
		"destino_direccion_id": {u(fx.addressID + 1)}, "fecha_recogida": {now},
		"tipo_carga": {"FTL"}, "tipo_equipo": {"dry_van"}, "peso_kg": {"1200"},
		"distancia_km": {"20"}, "tarifa_total": {"500"}, "moneda": {"USD"}, "audiencia": {"load_board"},
	}
	res := request(t, owner.client, http.MethodPost, "/loads", loadForm, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("load create status=%d", res.StatusCode)
	}
	res.Body.Close()
	var load models.Carga
	if err := facades.Orm().Query().Where("numero_referencia = ?", "FLOW-LOAD-001").First(&load); err != nil {
		t.Fatalf("load create was not persisted: %v", err)
	}
	if load.Estado != models.CargaPublicada || load.PublicadorID == 0 {
		t.Fatalf("new load has invalid ownership/state: %+v", load)
	}
	loadForm.Set("numero_referencia", "FLOW-LOAD-001-EDIT")
	res = request(t, owner.client, http.MethodPost, "/loads/"+u(load.ID), loadForm, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("load update status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", load.ID).First(&load); err != nil || load.NumeroReferencia != "FLOW-LOAD-001-EDIT" {
		t.Fatalf("load update not persisted: load=%+v err=%v", load, err)
	}
	res = request(t, owner.client, http.MethodPost, "/loads/"+u(load.ID)+"/assign-chofer", url.Values{"chofer_id": {"99999999"}}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("invalid driver assignment should be rejected with controlled redirect, status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", load.ID).First(&load); err != nil || load.ChoferID != nil || load.Estado != models.CargaPublicada {
		t.Fatalf("invalid driver ID changed load: load=%+v err=%v", load, err)
	}
	if _, err := facades.Orm().Query().Model(&models.Chofer{}).Where("id = ?", *driver.user.ChoferID).Update("estado", models.ChoferEnViaje); err != nil {
		t.Fatalf("mark driver unavailable: %v", err)
	}
	res = request(t, owner.client, http.MethodPost, "/loads/"+u(load.ID)+"/assign-chofer", url.Values{"chofer_id": {u(*driver.user.ChoferID)}}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("unavailable driver assignment should be rejected with controlled redirect, status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", load.ID).First(&load); err != nil || load.ChoferID != nil || load.Estado != models.CargaPublicada {
		t.Fatalf("unavailable driver changed load: load=%+v err=%v", load, err)
	}
	if _, err := facades.Orm().Query().Model(&models.Chofer{}).Where("id = ?", *driver.user.ChoferID).Update("estado", models.ChoferDisponible); err != nil {
		t.Fatalf("restore available driver fixture: %v", err)
	}
	res = request(t, owner.client, http.MethodPost, "/loads/"+u(load.ID)+"/assign-chofer", url.Values{"chofer_id": {u(*driver.user.ChoferID)}}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("load assignment status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", load.ID).First(&load); err != nil || load.ChoferID == nil || *load.ChoferID != *driver.user.ChoferID || load.Estado != models.CargaAsignada {
		t.Fatalf("assignment not persisted: load=%+v err=%v", load, err)
	}
	if err := services.NewCargaService().AssignChofer(fmt.Sprint(load.ID), *driver.user.ChoferID); err == nil {
		t.Fatal("already assigned load accepted another assignment")
	}
	res = request(t, driver.client, http.MethodPost, "/loads/"+u(load.ID)+"/accept", url.Values{}, driver.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("already-assigned load accept status=%d", res.StatusCode)
	}
	res.Body.Close()
	res = request(t, driver.client, http.MethodGet, "/loads/"+u(load.ID), nil, "", "")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "Iniciar tránsito") {
		t.Fatalf("assigned driver does not see start-transit action: status=%d", res.StatusCode)
	}
	res = request(t, driver.client, http.MethodPost, "/loads/"+u(load.ID)+"/start-transit", url.Values{}, driver.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("start transit status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", load.ID).First(&load); err != nil || load.Estado != models.CargaEnTransito {
		t.Fatalf("transit transition not persisted: load=%+v err=%v", load, err)
	}
	res = request(t, driver.client, http.MethodGet, "/loads/"+u(load.ID), nil, "", "")
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "Confirmar entrega") {
		t.Fatalf("assigned driver does not see delivery action: status=%d", res.StatusCode)
	}
	res = request(t, driver.client, http.MethodPost, "/loads/"+u(load.ID)+"/deliver", url.Values{}, driver.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("mark delivered status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", load.ID).First(&load); err != nil || load.Estado != models.CargaEntregada || load.FechaEntrega == nil {
		t.Fatalf("delivery transition not persisted: load=%+v err=%v", load, err)
	}
	res = request(t, owner.client, http.MethodPost, "/loads/"+u(load.ID)+"/delete", url.Values{}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("load delete status=%d", res.StatusCode)
	}
	res.Body.Close()
	loadCount, err := facades.Orm().Query().Model(&models.Carga{}).Where("id = ?", load.ID).Count()
	if err != nil || loadCount != 1 {
		t.Fatalf("completed load was removed: count=%d err=%v", loadCount, err)
	}

	// Address CRUD enforces ownership while allowing the owner to mutate it.
	addrForm := url.Values{"ciudad": {"Flow City"}, "estado_provincia": {"Flow Province"}, "pais": {"Cuba"}, "calle": {"1 Test Road"}, "latitud": {"23.113592"}, "longitud": {"-82.366592"}}
	res = request(t, owner.client, http.MethodPost, "/direcciones", addrForm, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("address create status=%d", res.StatusCode)
	}
	res.Body.Close()
	var address models.Direccion
	if err := facades.Orm().Query().Where("ciudad = ?", "Flow City").First(&address); err != nil {
		t.Fatalf("address create was not persisted: %v", err)
	}
	if address.Latitud == nil || address.Longitud == nil || math.Abs(*address.Latitud-23.113592) > 0.0000001 || math.Abs(*address.Longitud-(-82.366592)) > 0.0000001 {
		t.Fatalf("address coordinates were lost or rounded: lat=%v lng=%v", address.Latitud, address.Longitud)
	}
	other := fx.people[1]
	res = request(t, other.client, http.MethodPost, "/direcciones/"+u(address.ID)+"/update", url.Values{"ciudad": {"Hijacked City"}, "estado_provincia": {"Flow Province"}, "pais": {"Cuba"}}, other.csrf, "")
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign address mutation returned %d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", address.ID).First(&address); err != nil || address.Ciudad != "Flow City" {
		t.Fatalf("foreign address write changed the row: %+v err=%v", address, err)
	}
	res = request(t, owner.client, http.MethodPost, "/direcciones/"+u(address.ID)+"/update", url.Values{"ciudad": {"Flow City Updated"}, "estado_provincia": {"Flow Province"}, "pais": {"Cuba"}}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("address update status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", address.ID).First(&address); err != nil || address.Ciudad != "Flow City Updated" {
		t.Fatalf("address update failed: %+v err=%v", address, err)
	}
	res = request(t, owner.client, http.MethodPost, "/direcciones/"+u(address.ID)+"/delete", url.Values{}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("address delete status=%d", res.StatusCode)
	}
	res.Body.Close()
	addressCount, err := facades.Orm().Query().Model(&models.Direccion{}).Where("id = ?", address.ID).Count()
	if err != nil || addressCount != 0 {
		t.Fatalf("address delete left a visible row: count=%d err=%v", addressCount, err)
	}

	// Company writes are role-scoped and the owner can update/delete its own row.
	companyForm := url.Values{"tipo": {"broker"}, "nombre_legal": {"Flow Broker Company"}, "estado": {"activo"}, "tax_id": {"FLOW-TAX-001"}}
	res = request(t, owner.client, http.MethodPost, "/empresas", companyForm, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("company create status=%d", res.StatusCode)
	}
	res.Body.Close()
	var company models.Empresa
	if err := facades.Orm().Query().Where("nombre_legal = ?", "Flow Broker Company").First(&company); err != nil {
		t.Fatalf("company create was not persisted: %v", err)
	}
	companyForm.Set("nombre_legal", "Flow Broker Updated")
	res = request(t, owner.client, http.MethodPost, "/empresas/"+u(company.ID), companyForm, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("company update status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", company.ID).First(&company); err != nil || company.NombreLegal != "Flow Broker Updated" {
		t.Fatalf("company update failed: %+v err=%v", company, err)
	}
	carrierForm := url.Values{"tipo": {"carrier"}, "nombre_legal": {"Role Violation Inc"}, "estado": {"activo"}}
	res = request(t, owner.client, http.MethodPost, "/empresas", carrierForm, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("role violation should be safely redirected, got %d", res.StatusCode)
	}
	res.Body.Close()
	forbiddenCompanyCount, err := facades.Orm().Query().Model(&models.Empresa{}).Where("nombre_legal = ?", "Role Violation Inc").Count()
	if err != nil || forbiddenCompanyCount != 0 {
		t.Fatalf("publisher created carrier company: count=%d err=%v", forbiddenCompanyCount, err)
	}
	res = request(t, owner.client, http.MethodPost, "/empresas/"+u(company.ID)+"/delete", url.Values{}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("company delete status=%d", res.StatusCode)
	}
	res.Body.Close()
	companyCount, err := facades.Orm().Query().Model(&models.Empresa{}).Where("id = ?", company.ID).Count()
	if err != nil || companyCount != 0 {
		t.Fatalf("company delete left a visible row: count=%d err=%v", companyCount, err)
	}

	// Invoice draft -> update -> issue -> pay, with every state read back.
	var invoice models.Factura
	if err := facades.Orm().Query().Where("id = ?", fx.invoiceID).First(&invoice); err != nil {
		t.Fatal(err)
	}
	invoiceForm := url.Values{"distancia_km": {"25"}, "tarifa_por_km": {"4"}, "subtotal": {"100"}, "impuestos": {"10"}, "total": {"110"}, "moneda": {"USD"}, "metodo_pago": {"transferencia"}}
	res = request(t, owner.client, http.MethodPost, "/facturas/"+u(invoice.ID), invoiceForm, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("invoice update status=%d", res.StatusCode)
	}
	res.Body.Close()
	res = request(t, owner.client, http.MethodPost, "/facturas/"+u(invoice.ID)+"/emitir", url.Values{}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("invoice issue status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", invoice.ID).First(&invoice); err != nil || invoice.Estado != models.FacturaEmitida {
		t.Fatalf("invoice issue failed: estado=%s err=%v", invoice.Estado, err)
	}
	res = request(t, owner.client, http.MethodPost, "/facturas/"+u(invoice.ID)+"/pagar", url.Values{"metodo_pago": {"transferencia"}}, owner.csrf, "")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("invoice payment status=%d", res.StatusCode)
	}
	res.Body.Close()
	if err := facades.Orm().Query().Where("id = ?", invoice.ID).First(&invoice); err != nil || invoice.Estado != models.FacturaPagada || invoice.FechaPago == nil {
		t.Fatalf("invoice payment failed: %+v err=%v", invoice, err)
	}

	// Admin pages and cross-role view pages must render without nil-profile crashes.
	for _, path := range []string{"/admin/", "/admin/users", "/admin/cargas", "/publicadores", "/choferes", "/empresas", "/facturas"} {
		client := owner.client
		if strings.HasPrefix(path, "/admin") {
			client = admin.client
		}
		res = request(t, client, http.MethodGet, path, nil, "", "")
		if res.StatusCode != http.StatusOK {
			t.Errorf("flow page %s status=%d", path, res.StatusCode)
		}
		res.Body.Close()
	}
}

func seedFixture(t *testing.T, n int) fixture {
	t.Helper()
	brokerA := tests.SeedEmpresa(t, "broker", "RedTeam Broker A")
	brokerB := tests.SeedEmpresa(t, "broker", "RedTeam Broker B")
	carrier := tests.SeedEmpresa(t, "carrier", "RedTeam Carrier")
	passwordHash, err := facades.Hash().Make(password)
	if err != nil {
		t.Fatal(err)
	}
	var out fixture
	out.brokerIDs = []uint{brokerA.ID, brokerB.ID}
	out.carrierID = carrier.ID
	out.people = make([]principal, 0, n)
	var publisherProfiles []*models.Publicador
	var driverProfiles []*models.Chofer
	for i := 0; i < n; i++ {
		role := "publicador"
		company := brokerA.ID
		if i%3 == 1 {
			role = "chofer"
			company = carrier.ID
		} else if i%3 == 2 {
			role = "admin"
			company = brokerB.ID
		}
		if role == "publicador" && i%2 == 0 {
			company = brokerB.ID
		}
		user := &models.User{Name: fmt.Sprintf("RedTeam %03d", i), Email: fmt.Sprintf("redteam-%03d@test.invalid", i), Password: passwordHash, Role: role, EmpresaID: &company, IsActive: true}
		if err := facades.Orm().Query().Create(user); err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
		if role == "publicador" {
			profile := tests.NewPublicadorProfile()
			profile.UserID = user.ID
			profile.EmpresaID = company
			profile.NumeroLicenciaBroker = fmt.Sprintf("RT-BRK-%03d", i)
			if err := facades.Orm().Query().Create(profile); err != nil {
				t.Fatalf("create publisher profile %d: %v", i, err)
			}
			publisherProfiles = append(publisherProfiles, profile)
			user.PublicadorID = &profile.ID
		} else if role == "chofer" {
			profile := tests.NewChoferProfile()
			profile.UserID = user.ID
			profile.EmpresaID = company
			profile.NumeroLicencia = fmt.Sprintf("RT-LIC-%03d", i)
			if err := facades.Orm().Query().Create(profile); err != nil {
				t.Fatalf("create driver profile %d: %v", i, err)
			}
			driverProfiles = append(driverProfiles, profile)
			user.ChoferID = &profile.ID
		}
		if user.PublicadorID != nil {
			if _, err := facades.Orm().Query().Model(&models.User{}).Where("id = ?", user.ID).Update("publicador_id", user.PublicadorID); err != nil {
				t.Fatal(err)
			}
		}
		if user.ChoferID != nil {
			if _, err := facades.Orm().Query().Model(&models.User{}).Where("id = ?", user.ID).Update("chofer_id", user.ChoferID); err != nil {
				t.Fatal(err)
			}
		}
		out.people = append(out.people, principal{user: user, role: role})
	}
	if len(publisherProfiles) < 2 || len(driverProfiles) < 1 {
		t.Fatal("fixture lacks principals for each role")
	}
	ownerProfile := publisherProfiles[0]
	if ownerProfile.EmpresaID != brokerA.ID {
		ownerProfile = publisherProfiles[1]
	}
	for _, p := range publisherProfiles {
		if p.EmpresaID == brokerA.ID {
			ownerProfile = p
			break
		}
	}
	driver := driverProfiles[0]
	owner := out.people[0].user.ID
	for _, p := range out.people {
		if p.role == "publicador" {
			var profile models.Publicador
			_ = facades.Orm().Query().Where("user_id = ?", p.user.ID).First(&profile)
			if profile.ID == ownerProfile.ID {
				owner = p.user.ID
				break
			}
		}
	}
	addr := &models.Direccion{OwnerID: &owner, Ciudad: "Ciudad de Prueba", EstadoProvincia: "Provincia", Pais: "Cuba"}
	if err := facades.Orm().Query().Create(addr); err != nil {
		t.Fatal(err)
	}
	out.addressID = addr.ID
	out.ownerID = owner
	destination := &models.Direccion{OwnerID: &owner, Ciudad: "Destino", EstadoProvincia: "Provincia", Pais: "Cuba"}
	if err := facades.Orm().Query().Create(destination); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	load := &models.Carga{NumeroReferencia: "RT-LOAD-0001", PublicadorID: ownerProfile.ID, EmpresaID: brokerA.ID, ChoferID: &driver.ID, OrigenDireccionID: addr.ID, DestinoDireccionID: destination.ID, FechaRecogida: now, FechaEntrega: &now, TipoCarga: models.CargaFTL, TipoEquipo: models.EquipoDryVan, DistanciaKm: 10, Moneda: models.MonedaUSD, Estado: models.CargaEntregada}
	if err := facades.Orm().Query().Create(load); err != nil {
		t.Fatal(err)
	}
	invoice := &models.Factura{CargaID: load.ID, EmisorID: &brokerA.ID, ReceptorID: &carrier.ID, PublicadorID: &ownerProfile.ID, EmisorTipo: models.EmisorFacturaPublicador, NumeroFactura: "RT-INV-0001", FechaEmision: now, Subtotal: 100, Impuestos: 10, Moneda: models.MonedaUSD}
	if err := services.NewFacturaService().Create(invoice); err != nil {
		t.Fatalf("create red-team invoice: %v", err)
	}
	out.invoiceID = invoice.ID
	contested := &models.Carga{NumeroReferencia: "RT-RACE-0001", PublicadorID: ownerProfile.ID, EmpresaID: brokerA.ID, OrigenDireccionID: addr.ID, DestinoDireccionID: destination.ID, FechaRecogida: now, TipoCarga: models.CargaFTL, TipoEquipo: models.EquipoDryVan, DistanciaKm: 10, Moneda: models.MonedaUSD, Estado: models.CargaPublicada}
	if err := facades.Orm().Query().Create(contested); err != nil {
		t.Fatalf("create contested load: %v", err)
	}
	out.raceLoadID = contested.ID
	for i := range out.people {
		out.people[i].client = newClient(t)
		out.people[i].csrf = login(t, &out.people[i])
	}
	return out
}

func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Timeout: 15 * time.Second}
}
func login(t *testing.T, p *principal) string {
	t.Helper()
	res := request(t, p.client, http.MethodGet, "/login", nil, "", "")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("login form status %d", res.StatusCode)
	}
	matches := tokenPattern.FindStringSubmatch(string(body))
	if len(matches) != 2 {
		t.Fatal("could not extract CSRF token from login form")
	}
	form := url.Values{"email": {p.user.Email}, "password": {password}, "_csrf": {matches[1]}}
	res = request(t, p.client, http.MethodPost, "/login", form, matches[1], "")
	if res.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("login %s returned %d: %s", p.user.Email, res.StatusCode, string(body))
	}
	res.Body.Close()
	p.csrf = matches[1]
	return matches[1]
}
func request(t *testing.T, c *http.Client, method, path string, form url.Values, token, origin string) *http.Response {
	if t != nil {
		t.Helper()
	}
	var body io.Reader
	if form != nil {
		if token != "" && form.Get("_csrf") == "" {
			form.Set("_csrf", token)
		}
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, server.URL+path, body)
	if err != nil {
		if t != nil {
			t.Fatal(err)
		}
		panic(err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	res, err := c.Do(req)
	if err != nil {
		if t != nil {
			t.Fatal(err)
		}
		panic(err)
	}
	return res
}
func u(n uint) string { return fmt.Sprintf("%d", n) }
