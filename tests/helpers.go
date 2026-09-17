package tests

import (
	"testing"

	"goravel/app/facades"
	"goravel/app/models"
)

// ResetDB trunca todas las tablas respetando FK.
func ResetDB(t *testing.T) {
	t.Helper()
	_, err := facades.Orm().Query().Exec(`
		TRUNCATE TABLE
			facturas,
			cargas,
			choferes,
			publicadores,
			users,
			empresas,
			direcciones
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("ResetDB falló: %v", err)
	}
}

// RunMigrations corre migrate:fresh una sola vez. Llamar desde TestMain.
func RunMigrations() error {
	return facades.Artisan().Run([]string{"migrate:fresh"}, false)
}

// SeedEmpresa crea una empresa de prueba y la devuelve.
func SeedEmpresa(t *testing.T, tipo, nombre string) *models.Empresa {
	t.Helper()
	emp := &models.Empresa{
		Tipo:        models.TipoEmpresa(tipo),
		NombreLegal: nombre,
		Estado:      models.EmpresaActiva,
	}
	if err := facades.Orm().Query().Create(emp); err != nil {
		t.Fatalf("seed empresa: %v", err)
	}
	return emp
}

func CountUsers(t *testing.T) int64 {
	t.Helper()
	n, err := facades.Orm().Query().Model(&models.User{}).Count()
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	return n
}
func CountChoferes(t *testing.T) int64 {
	t.Helper()
	n, _ := facades.Orm().Query().Model(&models.Chofer{}).Count()
	return n
}
func CountPublicadores(t *testing.T) int64 {
	t.Helper()
	n, _ := facades.Orm().Query().Model(&models.Publicador{}).Count()
	return n
}
func CountEmpresas(t *testing.T) int64 {
	t.Helper()
	n, _ := facades.Orm().Query().Model(&models.Empresa{}).Count()
	return n
}

func NewUserChofer(email string) *models.User {
	return &models.User{
		Name:     "Chofer Test",
		Email:    email,
		Password: "$2a$10$fakehashfakehashfakehashfakehashfakehashfakehas",
		Role:     "chofer",
		IsActive: true,
		Phone:    StrPtr("+5355555555"),
		WhatsApp: StrPtr("+5355555555"),
		City:     "La Habana",
		State:    "La Habana",
		Country:  "Cuba",
		Radius:   100,
	}
}

func NewUserPublicador(email string) *models.User {
	return &models.User{
		Name:     "Publicador Test",
		Email:    email,
		Password: "$2a$10$fakehashfakehashfakehashfakehashfakehashfakehas",
		Role:     "publicador",
		IsActive: true,
		Phone:    StrPtr("+5355555555"),
		WhatsApp: StrPtr("+5355555555"),
		City:     "La Habana",
		State:    "La Habana",
		Country:  "Cuba",
		Radius:   100,
	}
}

func NewChoferProfile() *models.Chofer {
	return &models.Chofer{
		NumeroLicencia:      "LIC-001",
		TipoLicencia:        "A",
		PaisEmisionLicencia: "Cuba",
		AniosExperiencia:    5,
		Estado:              models.ChoferDisponible,
	}
}

func NewPublicadorProfile() *models.Publicador {
	return &models.Publicador{
		NumeroLicenciaBroker: "BRK-001",
		PaisEmisionLicencia:  "Cuba",
		AniosExperiencia:     3,
		Estado:               models.PublicadorActivo,
	}
}

func StrPtr(s string) *string { return &s }
func PtrUint(v uint) *uint    { return &v }
