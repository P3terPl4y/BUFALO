package controllers_test

import (
	"goravel/app/http/controllers"
	"goravel/app/models"
	"goravel/app/facades"
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

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v3"
)

var testApp *fiber.App

func TestMain(m *testing.M) {
	bootstrap.Boot()
	if err := tests.RunMigrations(); err != nil {
		panic(err)
	}

	// Localizar la raíz del proyecto buscando go.mod
	root := findProjectRoot()
	viewsPath := filepath.Join(root, "app", "views")

	engine := html.New(viewsPath, ".html")
	testApp = fiber.New(fiber.Config{Views: engine})

	// Registrar SOLO las rutas que necesitamos, sin rate limiter
	authCtrl := controllers.NewAuthController()
	testApp.Get("/register", authCtrl.ShowRegister)
	testApp.Post("/register", authCtrl.HandleRegister)
	testApp.Get("/login", authCtrl.ShowLogin)

	os.Exit(m.Run())
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
	resp, err := testApp.Test(req)
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
		"empresa_mode": {"new"},
		"empresa_nombre_legal": {"Transportes Juan S.A."},
		"chofer_numero_licencia": {"LIC-JUAN-001"},
		"chofer_tipo_licencia": {"A"},
		"chofer_anios_experiencia": {"5"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d. Body: %s", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "Registro exitoso") {
		t.Errorf("flash_success ausente. Body: %s", string(body))
	}
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
		"empresa_mode": {"existing"},
		"empresa_id": {strconv.FormatUint(uint64(emp.ID), 10)},
		"chofer_numero_licencia": {"LIC-PEDRO"},
		"chofer_tipo_licencia": {"B"},
		"chofer_anios_experiencia": {"3"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Registro exitoso") {
		t.Fatalf("registro falló: %s", string(body))
	}
	if n := tests.CountEmpresas(t); n != 1 {
		t.Errorf("empresas: esperado 1, got %d", n)
	}

	var user models.User
	facades.Orm().Query().Where("email = ?", "pedro@test.com").First(&user)
	if user.EmpresaID == nil || *user.EmpresaID != emp.ID {
		t.Errorf("EmpresaID: esperado %d, got %v", emp.ID, user.EmpresaID)
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
		"empresa_mode": {"none"},
		"chofer_numero_licencia": {"LIC-SIN"},
		"chofer_tipo_licencia": {"A"},
		"chofer_anios_experiencia": {"1"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Registro exitoso") {
		t.Fatalf("registro falló: %s", string(body))
	}

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
		"empresa_mode": {"new"},
		"empresa_nombre_legal": {"Brokers Ana S.A."},
		"publicador_numero_licencia_broker": {"BRK-ANA-001"},
		"publicador_anios_experiencia": {"4"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Registro exitoso") {
		t.Fatalf("registro falló: %s", string(body))
	}
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
		"empresa_mode": {"none"},
		"publicador_numero_licencia_broker": {"BRK-SOLO"},
		"publicador_anios_experiencia": {"2"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Registro exitoso") {
		t.Fatalf("registro falló: %s", string(body))
	}

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
		"empresa_mode": {"none"},
		"chofer_numero_licencia": {"LIC-001"},
		"chofer_tipo_licencia": {"A"},
		"chofer_anios_experiencia": {"1"},
	}
	postForm(t, "/register", base)

	// Segundo intento con el mismo email
	form2 := url.Values{
		"name": {"Otro"}, "email": {"dup@test.com"},
		"password": {"password456"}, "role": {"chofer"},
		"phone": {"+5355555555"}, "whatsapp": {"+5355555555"},
		"city": {"La Habana"}, "state": {"La Habana"},
		"country": {"Cuba"}, "radius": {"100"},
		"empresa_mode": {"none"},
		"chofer_numero_licencia": {"LIC-002"},
		"chofer_tipo_licencia": {"B"},
		"chofer_anios_experiencia": {"2"},
	}
	resp := postForm(t, "/register", form2)
	body, _ := io.ReadAll(resp.Body)

	if n := tests.CountUsers(t); n != 1 {
		t.Errorf("users: esperado 1, got %d", n)
	}
	if !strings.Contains(string(body), "ya está registrado") {
		t.Errorf("esperaba mensaje de duplicado. Body: %s", string(body))
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

	postForm(t, "/register", form)
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
		"empresa_mode": {"existing"},
		"empresa_id": {strconv.FormatUint(uint64(empBroker.ID), 10)},
		"chofer_numero_licencia": {"LIC-X"},
		"chofer_tipo_licencia": {"A"},
		"chofer_anios_experiencia": {"1"},
	}

	resp := postForm(t, "/register", form)
	body, _ := io.ReadAll(resp.Body)

	if n := tests.CountUsers(t); n != 0 {
		t.Errorf("no debe crear usuario, got %d", n)
	}
	if !strings.Contains(string(body), "no es válida") {
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
		"empresa_mode": {"none"},
		"chofer_numero_licencia": {"LIC-001"},
		"chofer_tipo_licencia": {"A"},
		"chofer_anios_experiencia": {"1"},
	}
	postForm(t, "/register", form)

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
