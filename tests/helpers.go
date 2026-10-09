package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"goravel/app/dbresilience"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/bootstrap"
)

// ResetDB trunca todas las tablas respetando FK.
func ResetDB(t *testing.T) {
	t.Helper()
	if err := RequireTestDatabase(); err != nil {
		t.Fatal(err)
	}
	_, err := facades.Orm().Query().Exec(`
		TRUNCATE TABLE
			notification_email_outbox,
			user_notifications,
			empresa_chat_mensajes,
			company_membership_requests,
notification_outbox,
			load_interests,
            chofer_calificaciones,
            red_choferes,
            pending_registrations,
			facturas,
			cargas,
			chofers,
			publicadors,
			users,
			empresas,
			direccions
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("ResetDB falló: %v", err)
	}
}

// RunMigrations creates the registered schema in an isolated test database.
func RunMigrations() error {
	if err := RequireTestDatabase(); err != nil {
		return err
	}
	for _, migration := range bootstrap.Migrations() {
		if err := migration.Up(); err != nil {
			return fmt.Errorf("migration %s: %w", migration.Signature(), err)
		}
	}
	return nil
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

// RequireTestDatabase refuses destructive tests unless explicitly isolated.
func RequireTestDatabase() error {
	name := os.Getenv("DB_DATABASE")
	return ValidateTestDatabase(
		os.Getenv("APP_ENV"),
		name,
		facades.Config().GetString("database.connections.postgres.database"),
		facades.Config().GetString("database.connections.postgres.dsn"),
	)
}

// ValidateTestDatabase refuses destructive tests unless both the configured
// ORM database and any overriding DSN resolve to the explicit *_test target.
func ValidateTestDatabase(appEnv, envDatabase, configDatabase, dsn string) error {
	if appEnv != "testing" || !strings.HasSuffix(envDatabase, "_test") || configDatabase != envDatabase {
		return fmt.Errorf("tests require APP_ENV=testing and an explicit DB_DATABASE ending in _test matching active configuration")
	}
	if strings.TrimSpace(dsn) != "" {
		dsnDatabase, err := dbresilience.DatabaseNameFromDSN(dsn)
		if err != nil || dsnDatabase != envDatabase || !strings.HasSuffix(dsnDatabase, "_test") {
			return fmt.Errorf("tests require DB_DSN to target the same explicit *_test database as DB_DATABASE")
		}
	}
	return nil
}
