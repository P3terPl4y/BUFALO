package services_test

import (
	"goravel/app/models"
	"goravel/app/services"
	"goravel/bootstrap"
	"goravel/tests"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	bootstrap.Boot()
	if err := tests.RunMigrations(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// ═══════════════════════════════════════════════════════════════════
// CHOFER
// ═══════════════════════════════════════════════════════════════════

func TestCreateWithRole_Chofer_NewEmpresa(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	user := tests.NewUserChofer("chofer1@test.com")
	empresa := &models.Empresa{
		Tipo:        models.TipoCarrier,
		NombreLegal: "Transportes Nuevos S.A.",
		Estado:      models.EmpresaActiva,
	}
	chofer := tests.NewChoferProfile()

	if err := svc.CreateWithRole(user, empresa, chofer, nil); err != nil {
		t.Fatalf("CreateWithRole falló: %v", err)
	}

	if user.ID == 0 {
		t.Error("User.ID no fue asignado")
	}
	if user.EmpresaID == nil || *user.EmpresaID != empresa.ID {
		t.Errorf("User.EmpresaID esperado %d", empresa.ID)
	}
	if user.ChoferID == nil {
		t.Error("User.ChoferID no fue asignado")
	}
	if user.PublicadorID != nil {
		t.Error("User.PublicadorID debe ser nil")
	}
	if chofer.UserID != user.ID {
		t.Error("Chofer.UserID mal asignado")
	}
	if chofer.EmpresaID != empresa.ID {
		t.Errorf("Chofer.EmpresaID: esperado %d, got %d", empresa.ID, chofer.EmpresaID)
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
}

func TestCreateWithRole_Chofer_ExistingEmpresa(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	emp := tests.SeedEmpresa(t, "carrier", "Transportes Existentes S.A.")

	user := tests.NewUserChofer("chofer2@test.com")
	chofer := tests.NewChoferProfile()

	if err := svc.CreateWithRole(user, emp, chofer, nil); err != nil {
		t.Fatalf("CreateWithRole falló: %v", err)
	}

	if user.EmpresaID == nil || *user.EmpresaID != emp.ID {
		t.Error("User.EmpresaID no apunta a la empresa")
	}
	if chofer.EmpresaID != emp.ID {
		t.Errorf("Chofer.EmpresaID: esperado %d, got %d", emp.ID, chofer.EmpresaID)
	}
	if n := tests.CountEmpresas(t); n != 1 {
		t.Errorf("empresas: esperado 1, got %d", n)
	}
}

func TestCreateWithRole_Chofer_NoEmpresa(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	user := tests.NewUserChofer("chofer3@test.com")
	chofer := tests.NewChoferProfile()

	if err := svc.CreateWithRole(user, nil, chofer, nil); err != nil {
		t.Fatalf("CreateWithRole falló: %v", err)
	}

	if user.EmpresaID != nil {
		t.Error("User.EmpresaID debe ser nil")
	}
	if chofer.EmpresaID != 0 {
		t.Errorf("Chofer.EmpresaID debe ser 0, got %d", chofer.EmpresaID)
	}
	if n := tests.CountEmpresas(t); n != 0 {
		t.Errorf("empresas: esperado 0, got %d", n)
	}
	if n := tests.CountChoferes(t); n != 1 {
		t.Errorf("choferes: esperado 1, got %d", n)
	}
}

// ═══════════════════════════════════════════════════════════════════
// PUBLICADOR
// ═══════════════════════════════════════════════════════════════════

func TestCreateWithRole_Publicador_NewEmpresa(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	user := tests.NewUserPublicador("pub1@test.com")
	empresa := &models.Empresa{
		Tipo:        models.TipoBroker,
		NombreLegal: "Brokers Nuevos S.A.",
		Estado:      models.EmpresaActiva,
	}
	publicador := tests.NewPublicadorProfile()

	if err := svc.CreateWithRole(user, empresa, nil, publicador); err != nil {
		t.Fatalf("CreateWithRole falló: %v", err)
	}

	if user.PublicadorID == nil {
		t.Error("User.PublicadorID no fue asignado")
	}
	if user.ChoferID != nil {
		t.Error("User.ChoferID debe ser nil")
	}
	if publicador.UserID != user.ID {
		t.Error("Publicador.UserID mal asignado")
	}
	if publicador.EmpresaID != empresa.ID {
		t.Errorf("Publicador.EmpresaID: esperado %d, got %d", empresa.ID, publicador.EmpresaID)
	}

	if n := tests.CountPublicadores(t); n != 1 {
		t.Errorf("publicadores: esperado 1, got %d", n)
	}
}

func TestCreateWithRole_Publicador_ExistingEmpresa(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	emp := tests.SeedEmpresa(t, "broker", "Brokers Existentes S.A.")

	user := tests.NewUserPublicador("pub2@test.com")
	publicador := tests.NewPublicadorProfile()

	if err := svc.CreateWithRole(user, emp, nil, publicador); err != nil {
		t.Fatalf("CreateWithRole falló: %v", err)
	}

	if user.EmpresaID == nil || *user.EmpresaID != emp.ID {
		t.Error("User.EmpresaID no apunta a la empresa")
	}
	if publicador.EmpresaID != emp.ID {
		t.Errorf("Publicador.EmpresaID: esperado %d, got %d", emp.ID, publicador.EmpresaID)
	}
}

func TestCreateWithRole_Publicador_NoEmpresa(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	user := tests.NewUserPublicador("pub3@test.com")
	publicador := tests.NewPublicadorProfile()

	if err := svc.CreateWithRole(user, nil, nil, publicador); err != nil {
		t.Fatalf("CreateWithRole falló: %v", err)
	}

	if user.EmpresaID != nil {
		t.Error("User.EmpresaID debe ser nil")
	}
	if publicador.EmpresaID != 0 {
		t.Errorf("Publicador.EmpresaID debe ser 0, got %d", publicador.EmpresaID)
	}
}

// ═══════════════════════════════════════════════════════════════════
// ROLLBACK
// ═══════════════════════════════════════════════════════════════════

func TestCreateWithRole_Rollback_EmailDuplicado(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	u1 := tests.NewUserChofer("dup@test.com")
	if err := svc.CreateWithRole(u1, nil, tests.NewChoferProfile(), nil); err != nil {
		t.Fatalf("primer registro falló: %v", err)
	}

	u2 := tests.NewUserChofer("dup@test.com")
	empresa2 := &models.Empresa{
		Tipo:        models.TipoCarrier,
		NombreLegal: "Empresa Que No Debe Quedar",
		Estado:      models.EmpresaActiva,
	}
	err := svc.CreateWithRole(u2, empresa2, tests.NewChoferProfile(), nil)
	if err == nil {
		t.Fatal("se esperaba error por email duplicado")
	}

	if n := tests.CountEmpresas(t); n != 0 {
		t.Errorf("empresas tras rollback: esperado 0, got %d", n)
	}
	if n := tests.CountUsers(t); n != 1 {
		t.Errorf("users tras rollback: esperado 1, got %d", n)
	}
}

func TestCreateWithRole_Rollback_ChoferNil(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	user := tests.NewUserChofer("nilchofer@test.com")
	empresa := &models.Empresa{
		Tipo:        models.TipoCarrier,
		NombreLegal: "Empresa Fantasma",
		Estado:      models.EmpresaActiva,
	}

	err := svc.CreateWithRole(user, empresa, nil, nil)
	if err == nil {
		t.Fatal("se esperaba error por chofer nil")
	}

	if n := tests.CountUsers(t); n != 0 {
		t.Errorf("users tras rollback: esperado 0, got %d", n)
	}
	if n := tests.CountEmpresas(t); n != 0 {
		t.Errorf("empresas tras rollback: esperado 0, got %d", n)
	}
}

func TestCreateWithRole_Rollback_PublicadorNil(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	user := tests.NewUserPublicador("nilpub@test.com")

	err := svc.CreateWithRole(user, nil, nil, nil)
	if err == nil {
		t.Fatal("se esperaba error por publicador nil")
	}

	if n := tests.CountUsers(t); n != 0 {
		t.Errorf("users tras rollback: esperado 0, got %d", n)
	}
}

// ═══════════════════════════════════════════════════════════════════
// VALIDACIONES
// ═══════════════════════════════════════════════════════════════════

func TestValidateUserRole(t *testing.T) {
	cases := []struct {
		name    string
		user    models.User
		wantErr bool
	}{
		{"admin sin perfiles: ok", models.User{Role: "admin"}, false},
		{"admin con ChoferID: error", models.User{Role: "admin", ChoferID: tests.PtrUint(1)}, true},
		{"publicador ok", models.User{Role: "publicador", PublicadorID: tests.PtrUint(1)}, false},
		{"publicador sin perfil: error", models.User{Role: "publicador"}, true},
		{"publicador con ambos: error",
			models.User{Role: "publicador", PublicadorID: tests.PtrUint(1), ChoferID: tests.PtrUint(2)}, true},
		{"chofer ok", models.User{Role: "chofer", ChoferID: tests.PtrUint(1)}, false},
		{"chofer sin perfil: error", models.User{Role: "chofer"}, true},
		{"chofer con ambos: error",
			models.User{Role: "chofer", ChoferID: tests.PtrUint(1), PublicadorID: tests.PtrUint(2)}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := services.ValidateUserRole(&tc.user)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestEmailExists(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewUserService()

	exists, _ := svc.EmailExists("nuevo@test.com", 0)
	if exists {
		t.Error("EmailExists devolvió true en BD vacía")
	}

	u := tests.NewUserChofer("existente@test.com")
	if err := svc.CreateWithRole(u, nil, tests.NewChoferProfile(), nil); err != nil {
		t.Fatalf("registro falló: %v", err)
	}

	exists, _ = svc.EmailExists("existente@test.com", 0)
	if !exists {
		t.Error("EmailExists devolvió false para existente")
	}

	exists, _ = svc.EmailExists("existente@test.com", u.ID)
	if exists {
		t.Error("EmailExists con excludeID no excluyó")
	}
}
