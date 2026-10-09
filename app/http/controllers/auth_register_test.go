package controllers_test

import (
	"errors"
	"goravel/app/facades"
	"goravel/app/http/controllers"
	"goravel/app/models"
	"goravel/app/viewhelpers"
	"goravel/bootstrap"
	"goravel/tests"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v3"
)

var testApp *fiber.App
var lastVerificationURL string
var verificationSendError error

func TestMain(m *testing.M) {
	bootstrap.Boot()
	if err := tests.RunMigrations(); err != nil {
		panic(err)
	}

	// Localizar la raíz del proyecto buscando go.mod
	root := findProjectRoot()
	viewsPath := filepath.Join(root, "app", "views")

	engine := html.New(viewsPath, ".html")
	viewhelpers.Register(engine)
	testApp = fiber.New(fiber.Config{Views: engine})

	// Registrar SOLO las rutas que necesitamos, sin rate limiter
	lastVerificationURL = ""
	verificationSendError = nil
	authCtrl := controllers.NewAuthControllerWithVerificationSender(func(_ string, confirmationURL string) error {
		if verificationSendError != nil {
			return verificationSendError
		}
		lastVerificationURL = confirmationURL
		return nil
	})
	testApp.Get("/register", authCtrl.ShowRegister)
	testApp.Post("/register", authCtrl.HandleRegister)
	testApp.Get("/register/confirm", authCtrl.ShowEmailConfirmation)
	testApp.Post("/register/confirm", authCtrl.ConfirmEmail)
	testApp.Get("/login", authCtrl.ShowLogin)

	os.Exit(m.Run())
}

func confirmPendingRegistration(t *testing.T) *http.Response {
	t.Helper()
	return postForm(t, "/register/confirm", url.Values{"token": {confirmationToken(t)}})
}

func confirmationToken(t *testing.T) string {
	t.Helper()
	confirmationURL, err := url.Parse(lastVerificationURL)
	if err != nil || confirmationURL.Query().Get("token") == "" {
		t.Fatalf("no se capturó un enlace de confirmación válido: %q", lastVerificationURL)
	}
	return confirmationURL.Query().Get("token")
}

func findProjectRoot() string {
	wd, _ := os.Getwd()
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return wd
}

func postForm(t *testing.T, path string, form url.Values) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// bcrypt under the race detector can exceed Fiber's 1 s default.
	resp, err := testApp.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// ═══════════════════════════════════════════════════════════════════
// CHOFER
// ═══════════════════════════════════════════════════════════════════

func TestHandleRegister_Chofer_NewEmpresa(t *testing.T) {
	tests.ResetDB(t)

	form := url.Values{
		"name": {"Juan Chofer"}, "email": {"juan.chofer@test.com"},
		"password": {"password123"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"100"},
		"empresa_mode":             {"new"},
		"empresa_nombre_legal":     {"Transportes Juan S.A."},
		"chofer_numero_licencia":   {"LIC-JUAN-001"},
		"chofer_tipo_licencia":     {"A"},
		"chofer_anios_experiencia": {"5"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d. Body: %s", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "Confirma tu correo") {
		t.Errorf("no se indicó que falta confirmar el correo. Body: %s", string(body))
	}
	if n := tests.CountUsers(t); n != 0 {
		t.Fatalf("el usuario se creó antes de confirmar: %d", n)
	}
	if n := tests.CountChoferes(t); n != 0 {
		t.Fatalf("el perfil se creó antes de confirmar: %d", n)
	}
	if n := tests.CountEmpresas(t); n != 0 {
		t.Fatalf("la empresa se creó antes de confirmar: %d", n)
	}
	var pending models.PendingRegistration
	if err := facades.Orm().Query().Where("email = ?", "juan.chofer@test.com").First(&pending); err != nil {
		t.Fatalf("no se guardó la solicitud temporal: %v", err)
	}
	token := confirmationToken(t)
	if strings.Contains(pending.Payload, "password123") || strings.Contains(pending.Payload, "Juan Chofer") {
		t.Fatal("el payload pendiente debe estar cifrado")
	}
	getReq := httptest.NewRequest(http.MethodGet, "/register/confirm?token="+url.QueryEscape(token), nil)
	getResp, err := testApp.Test(getReq)
	if err != nil {
		t.Fatal(err)
	}
	getResp.Body.Close()
	if n := tests.CountUsers(t); n != 0 {
		t.Fatalf("GET de escáner creó el usuario antes del POST: %d", n)
	}
	confirmed := confirmPendingRegistration(t)
	if confirmed.StatusCode != http.StatusOK {
		t.Fatalf("confirmación: status=%d", confirmed.StatusCode)
	}
	confirmed.Body.Close()
	if n := tests.CountUsers(t); n != 1 {
		t.Errorf("users: esperado 1, got %d", n)
	}
	if n := tests.CountChoferes(t); n != 1 {
		t.Errorf("choferes: esperado 1, got %d", n)
	}
	if n := tests.CountEmpresas(t); n != 1 {
		t.Errorf("empresas: esperado 1, got %d", n)
	}

	var user models.User
	facades.Orm().Query().Where("email = ?", "juan.chofer@test.com").First(&user)
	if user.Role != "chofer" {
		t.Errorf("role: esperado chofer, got %s", user.Role)
	}
	if user.ChoferID == nil {
		t.Error("ChoferID debe estar seteado")
	}
	if user.PublicadorID != nil {
		t.Error("PublicadorID debe ser nil")
	}
	if user.Phone == nil || *user.Phone != "+5355555555" {
		t.Error("Phone no se guardó")
	}
	replay := postForm(t, "/register/confirm", url.Values{"token": {token}})
	replayBody, _ := io.ReadAll(replay.Body)
	replay.Body.Close()
	if !strings.Contains(string(replayBody), "no es válido o venció") || tests.CountUsers(t) != 1 {
		t.Fatalf("el token reutilizado no se rechazó correctamente: %s", string(replayBody))
	}
}

func TestHandleRegister_Chofer_ExistingEmpresa(t *testing.T) {
	tests.ResetDB(t)
	emp := tests.SeedEmpresa(t, "carrier", "Empresa Existente")

	form := url.Values{
		"name": {"Pedro Chofer"}, "email": {"pedro@test.com"},
		"password": {"password123"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"100"},
		"empresa_mode":             {"existing"},
		"empresa_id":               {strconv.FormatUint(uint64(emp.ID), 10)},
		"chofer_numero_licencia":   {"LIC-PEDRO"},
		"chofer_tipo_licencia":     {"B"},
		"chofer_anios_experiencia": {"3"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "requiere aprobación") {
		t.Fatalf("unexpected response: %s", body)
	}
	if tests.CountUsers(t) != 0 {
		t.Fatal("self registration joined an existing company")
	}
	count, err := facades.Orm().Query().Model(&models.PendingRegistration{}).Count()
	if err != nil || count != 0 {
		t.Fatalf("unauthorized pending registration: count=%d err=%v", count, err)
	}
}

func TestHandleRegister_Chofer_NoEmpresa(t *testing.T) {
	tests.ResetDB(t)

	form := url.Values{
		"name": {"Sin Empresa"}, "email": {"sinempresa@test.com"},
		"password": {"password123"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"100"},
		"empresa_mode":             {"none"},
		"chofer_numero_licencia":   {"LIC-SIN"},
		"chofer_tipo_licencia":     {"A"},
		"chofer_anios_experiencia": {"1"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Confirma tu correo") {
		t.Fatalf("registro falló: %s", string(body))
	}
	confirmPendingRegistration(t).Body.Close()

	var user models.User
	facades.Orm().Query().Where("email = ?", "sinempresa@test.com").First(&user)
	if user.EmpresaID != nil {
		t.Errorf("EmpresaID debe ser nil, got %v", user.EmpresaID)
	}
	if n := tests.CountEmpresas(t); n != 0 {
		t.Errorf("empresas: esperado 0, got %d", n)
	}
}

// ═══════════════════════════════════════════════════════════════════
// PUBLICADOR
// ═══════════════════════════════════════════════════════════════════

func TestHandleRegister_Publicador_NewEmpresa(t *testing.T) {
	tests.ResetDB(t)

	form := url.Values{
		"name": {"Ana Broker"}, "email": {"ana@test.com"},
		"password": {"password123"}, "role": {"publicador"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"50"},
		"empresa_mode":                      {"new"},
		"empresa_nombre_legal":              {"Brokers Ana S.A."},
		"publicador_numero_licencia_broker": {"BRK-ANA-001"},
		"publicador_anios_experiencia":      {"4"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Confirma tu correo") {
		t.Fatalf("registro falló: %s", string(body))
	}
	confirmPendingRegistration(t).Body.Close()
	if n := tests.CountPublicadores(t); n != 1 {
		t.Errorf("publicadores: esperado 1, got %d", n)
	}

	var user models.User
	facades.Orm().Query().Where("email = ?", "ana@test.com").First(&user)
	if user.PublicadorID == nil {
		t.Error("PublicadorID debe estar seteado")
	}
	if user.ChoferID != nil {
		t.Error("ChoferID debe ser nil")
	}
}

func TestHandleRegister_Publicador_NoEmpresa(t *testing.T) {
	tests.ResetDB(t)

	form := url.Values{
		"name": {"Broker Solo"}, "email": {"broker.solo@test.com"},
		"password": {"password123"}, "role": {"publicador"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"50"},
		"empresa_mode":                      {"none"},
		"publicador_numero_licencia_broker": {"BRK-SOLO"},
		"publicador_anios_experiencia":      {"2"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Confirma tu correo") {
		t.Fatalf("registro falló: %s", string(body))
	}
	confirmPendingRegistration(t).Body.Close()

	var user models.User
	facades.Orm().Query().Where("email = ?", "broker.solo@test.com").First(&user)
	if user.EmpresaID != nil {
		t.Errorf("EmpresaID debe ser nil, got %v", user.EmpresaID)
	}
}

// ═══════════════════════════════════════════════════════════════════
// VALIDACIONES
// ═══════════════════════════════════════════════════════════════════

func TestHandleRegister_MissingRequiredFields(t *testing.T) {
	tests.ResetDB(t)

	form := url.Values{"role": {"chofer"}}
	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)

	if n := tests.CountUsers(t); n != 0 {
		t.Errorf("users: esperado 0, got %d", n)
	}
	if !strings.Contains(string(body), "Error") && !strings.Contains(string(body), "Revisa") {
		t.Errorf("esperaba mensaje de error")
	}
}

func TestHandleRegister_InvalidRole(t *testing.T) {
	tests.ResetDB(t)

	form := url.Values{
		"name": {"Test"}, "email": {"test@test.com"},
		"password": {"password123"}, "role": {"admin"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)

	if n := tests.CountUsers(t); n != 0 {
		t.Errorf("no debe crear usuario, got %d", n)
	}
	if !strings.Contains(string(body), "válido") {
		t.Errorf("esperaba mensaje de rol inválido")
	}
}

func TestHandleRegister_DuplicateEmail(t *testing.T) {
	tests.ResetDB(t)

	base := url.Values{
		"name": {"Test"}, "email": {"dup@test.com"},
		"password": {"password123"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"100"},
		"empresa_mode":             {"none"},
		"chofer_numero_licencia":   {"LIC-001"},
		"chofer_tipo_licencia":     {"A"},
		"chofer_anios_experiencia": {"1"},
	}
	postForm(t, "/register", base).Body.Close()
	confirmPendingRegistration(t).Body.Close()

	// Segundo intento con el mismo email
	form2 := url.Values{
		"name": {"Otro"}, "email": {"DUP@TEST.COM"},
		"password": {"password456"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"100"},
		"empresa_mode":             {"none"},
		"chofer_numero_licencia":   {"LIC-002"},
		"chofer_tipo_licencia":     {"B"},
		"chofer_anios_experiencia": {"2"},
	}
	resp := postForm(t, "/register", form2)
	body, _ := io.ReadAll(resp.Body)

	if n := tests.CountUsers(t); n != 1 {
		t.Errorf("users: esperado 1, got %d", n)
	}
	if !strings.Contains(string(body), "Confirma tu correo") {
		t.Errorf("esperaba mensaje de duplicado. Body: %s", string(body))
	}
}

func TestEmailConfirmationRejectsExpiredToken(t *testing.T) {
	tests.ResetDB(t)
	form := url.Values{
		"name": {"Chofer Expirado"}, "email": {"expired@test.com"},
		"password": {"password123"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"}, "country": {"Cuba"}, "radius": {"100"},
		"empresa_mode": {"none"}, "chofer_numero_licencia": {"LIC-EXP"},
		"chofer_tipo_licencia": {"A"}, "chofer_anios_experiencia": {"1"},
	}
	postForm(t, "/register", form).Body.Close()
	token := confirmationToken(t)
	if _, err := facades.Orm().Query().Model(&models.PendingRegistration{}).
		Where("email = ?", "expired@test.com").Update("expires_at", time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	response := postForm(t, "/register/confirm", url.Values{"token": {token}})
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if !strings.Contains(string(body), "no es válido o venció") {
		t.Fatalf("se esperaba rechazo de token vencido: %s", string(body))
	}
	if got := tests.CountUsers(t); got != 0 {
		t.Fatalf("token vencido creó un usuario: %d", got)
	}
}

func TestRegistrationEmailFailureCreatesNoAccountAndRemovesPendingData(t *testing.T) {
	tests.ResetDB(t)
	verificationSendError = errors.New("smtp-provider-secret-should-not-leak-91a7")
	defer func() { verificationSendError = nil }()
	form := url.Values{
		"name": {"Chofer Sin Correo"}, "email": {"mail-error@test.com"},
		"password": {"password123"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"}, "country": {"Cuba"}, "radius": {"100"},
		"empresa_mode": {"none"}, "chofer_numero_licencia": {"LIC-MAIL"},
		"chofer_tipo_licencia": {"A"}, "chofer_anios_experiencia": {"1"},
	}
	response := postForm(t, "/register", form)
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if !strings.Contains(string(body), "No pudimos enviar el correo") || strings.Contains(string(body), "smtp-provider-secret-should-not-leak-91a7") {
		t.Fatalf("unexpected SMTP failure response: %s", string(body))
	}
	if count := tests.CountUsers(t); count != 0 {
		t.Fatalf("mailer failure created a user: %d", count)
	}
	if count, err := facades.Orm().Query().Model(&models.PendingRegistration{}).Count(); err != nil || count != 0 {
		t.Fatalf("pending payload remained after mail failure: count=%d err=%v", count, err)
	}
}

func TestHandleRegister_WeakPassword(t *testing.T) {
	tests.ResetDB(t)

	form := url.Values{
		"name": {"Test"}, "email": {"weak@test.com"},
		"password": {"123"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"100"},
	}

	postForm(t, "/register", form).Body.Close()
	if n := tests.CountUsers(t); n != 0 {
		t.Errorf("no debe crear usuario, got %d", n)
	}
}

func TestHandleRegister_ExistingEmpresaDeOtroTipo(t *testing.T) {
	tests.ResetDB(t)
	empBroker := tests.SeedEmpresa(t, "broker", "Broker Existente")

	form := url.Values{
		"name": {"Chofer Mal"}, "email": {"mal@test.com"},
		"password": {"password123"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"100"},
		"empresa_mode":             {"existing"},
		"empresa_id":               {strconv.FormatUint(uint64(empBroker.ID), 10)},
		"chofer_numero_licencia":   {"LIC-X"},
		"chofer_tipo_licencia":     {"A"},
		"chofer_anios_experiencia": {"1"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)

	if n := tests.CountUsers(t); n != 0 {
		t.Errorf("no debe crear usuario, got %d", n)
	}
	if !strings.Contains(string(body), "requiere aprobación") {
		t.Errorf("esperaba mensaje de empresa inválida. Body: %s", string(body))
	}
}

func TestHandleRegister_PasswordIsHashed(t *testing.T) {
	tests.ResetDB(t)

	plain := "password123"
	form := url.Values{
		"name": {"Hash Test"}, "email": {"hash@test.com"},
		"password": {plain}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"100"},
		"empresa_mode":             {"none"},
		"chofer_numero_licencia":   {"LIC-001"},
		"chofer_tipo_licencia":     {"A"},
		"chofer_anios_experiencia": {"1"},
	}
	postForm(t, "/register", form).Body.Close()
	confirmPendingRegistration(t).Body.Close()

	var user models.User
	facades.Orm().Query().Where("email = ?", "hash@test.com").First(&user)

	if user.Password == plain {
		t.Error("password en texto plano")
	}
	if len(user.Password) < 40 {
		t.Errorf("hash muy corto: %d", len(user.Password))
	}
	if !user.CheckPassword(plain) {
		t.Error("CheckPassword falló")
	}
}

func TestRegistrationSelectablePreferencesPersistAfterConfirmation(t *testing.T) {
	tests.ResetDB(t)
	form := url.Values{"name": {"Selections Test"}, "email": {"selections@test.invalid"}, "password": {"Password!123"}, "role": {"chofer"}, "phone": {"+5355555555"}, "whatsapp": {"+5355555555"}, "city": {"La Habana"}, "state": {"La Habana"}, "country": {"Cuba"}, "radius": {"100"}, "empresa_mode": {"none"}, "chofer_numero_licencia": {"LIC-SELECT"}, "chofer_tipo_licencia": {"A"}, "chofer_anios_experiencia": {"1"}, "preferred_equipment_types_present": {"1"}, "preferred_equipment_types_choice": {"dry_van", "reefer"}, "preferred_cargo_types_present": {"1"}, "preferred_cargo_types_choice": {"FTL", "LTL"}, "chofer_tipos_equipo_permitidos_present": {"1"}, "chofer_tipos_equipo_permitidos_choice": {"flatbed", "reefer"}}
	form.Set("preferred_cargo_types_choice", "invented")
	form.Del("preferred_cargo_types_present") // Modified body omits the UI marker.
	response := postForm(t, "/register", form)
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if !strings.Contains(string(body), "preferencias") || tests.CountUsers(t) != 0 {
		t.Fatal("invalid selection accepted")
	}
	form["preferred_cargo_types_choice"] = []string{"FTL", "LTL"}
	response = postForm(t, "/register", form)
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if !strings.Contains(string(body), "Confirma tu correo") {
		t.Fatalf("registration rejected: %s", body)
	}
	confirmPendingRegistration(t).Body.Close()
	var user models.User
	if err := facades.Orm().Query().Where("email = ?", "selections@test.invalid").First(&user); err != nil {
		t.Fatal(err)
	}
	if user.PreferredEquipmentTypes != "dry_van,reefer" || user.PreferredCargoTypes != "FTL,LTL" {
		t.Fatal("selections not preserved")
	}
	var driver models.Chofer
	if err := facades.Orm().Query().Where("user_id = ?", user.ID).First(&driver); err != nil {
		t.Fatal(err)
	}
	if driver.TiposEquipoPermitidos != "flatbed,reefer" {
		t.Fatal("permitted equipment not preserved")
	}
}

func TestConfirmationPolicyHidesTokenAndAllowsHTTPSCSRF(t *testing.T) {
	resp, err := testApp.Test(httptest.NewRequest(http.MethodGet, "/register/confirm?token=sensitive-token", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("Referrer-Policy"); got != "origin" {
		t.Fatalf("unsafe or incompatible referrer policy: %q", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `name="referrer" content="origin"`) {
		t.Fatal("HTML overrides the safe origin-only policy")
	}
}
