package routes

import (
	"goravel/app/http/controllers"
	"goravel/app/http/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupWebRoutes(app *fiber.App) {
	// ------------------------------------------------------------------
	// Controladores
	// ------------------------------------------------------------------
	authCtrl         := controllers.NewAuthController()
	cargaCtrl        := controllers.NewCargaController()
	direccionCtrl    := controllers.NewDireccionController()
	empresaCtrl      := controllers.NewEmpresaController()
	choferCtrl       := controllers.NewChoferController()
	publicadorCtrl   := controllers.NewPublicadorController()
	facturaCtrl      := controllers.NewFacturaController()
	loadInterestCtrl := controllers.NewLoadInterestController()
	userCtrl         := controllers.NewUserController()
	adminCtrl        := controllers.NewAdminController()

	// ============================================================
	// PÚBLICAS
	// ============================================================
	app.Get("/login", authCtrl.ShowLogin)
	app.Post("/login", middleware.LoginRateLimiter(), authCtrl.HandleLogin)
	app.Get("/register", authCtrl.ShowRegister)
	app.Post("/register", middleware.LoginRateLimiter(), authCtrl.HandleRegister)

	// ============================================================
	// PROTEGIDAS — todas las rutas van planas con su middleware
	// ============================================================
	auth := middleware.SessionAuth()
	pub  := middleware.PublicadorAuth()
	chof := middleware.ChoferAuth()
	both := middleware.RoleAuth("publicador", "chofer", "admin")
	adm  := middleware.AdminAuth()
	is_active:=middleware.IsActiveUserHandler()
	// Atajo para no repetir: todas las rutas cuelgan de un solo Use
	protected := app.Group("", auth,is_active)

	// ── Generales ──
	protected.Get("/home", authCtrl.ShowHome)
	protected.Get("/logout", authCtrl.Logout)
	protected.Get("/profile", userCtrl.Show)
	protected.Get("/profile/edit", userCtrl.Edit)
	protected.Post("/profile/update", userCtrl.Update)

	// ═══════════════════════════════════════════════════════════
	// LECTURA (cualquier autenticado)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/loads", cargaCtrl.Index)
	protected.Get("/loads/:id<int>", cargaCtrl.Show)

	protected.Get("/direcciones", direccionCtrl.Index)
	protected.Get("/direcciones/:id<int>", direccionCtrl.Show)

	protected.Get("/empresas", empresaCtrl.Index)
	protected.Get("/empresas/:id<int>", empresaCtrl.Show)

	protected.Get("/choferes", choferCtrl.Index)
	protected.Get("/choferes/:id<int>", choferCtrl.Show)

	protected.Get("/publicadores", publicadorCtrl.Index)
	protected.Get("/publicadores/:id<int>", publicadorCtrl.Show)

	protected.Get("/facturas", facturaCtrl.Index)
	protected.Get("/facturas/:id<int>", facturaCtrl.Show)

	// ═══════════════════════════════════════════════════════════
	// CARGAS — escritura (publicador + admin)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/loads/create", pub, cargaCtrl.Create)
	protected.Post("/loads", pub, is_active,cargaCtrl.Store)
	protected.Get("/loads/:id<int>/edit", pub, cargaCtrl.Edit)
	protected.Put("/loads/:id<int>", pub, cargaCtrl.Update)
	protected.Delete("/loads/:id<int>", pub, cargaCtrl.Delete)
	protected.Post("/loads/:id<int>", pub, cargaCtrl.Update)
	protected.Post("/loads/:id<int>/delete", pub, cargaCtrl.Delete)
	protected.Post("/loads/:id<int>/assign-chofer", pub, cargaCtrl.AssignChofer)

	// ═══════════════════════════════════════════════════════════
	// CARGAS — aceptar / interés (chofer + admin)
	// ═══════════════════════════════════════════════════════════
	protected.Post("/loads/:id<int>/accept", chof, cargaCtrl.AcceptLoad)
	protected.Post("/loads/:id<int>/interest", chof, loadInterestCtrl.SendInterest)

	// ═══════════════════════════════════════════════════════════
	// DIRECCIONES — escritura (publicador + chofer + admin)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/direcciones/create", both, direccionCtrl.Create)
	protected.Post("/direcciones", both, direccionCtrl.Store)
	protected.Get("/direcciones/:id<int>/edit", both, direccionCtrl.Edit)
	protected.Put("/direcciones/:id<int>", both, direccionCtrl.Update)
	protected.Delete("/direcciones/:id<int>", both, direccionCtrl.Delete)
	protected.Post("/direcciones/:id<int>/update", both, direccionCtrl.Update)
	protected.Post("/direcciones/:id<int>/delete", both, direccionCtrl.Delete)

	// ═══════════════════════════════════════════════════════════
	// EMPRESAS — escritura (publicador + chofer + admin)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/empresas/create", both, empresaCtrl.Create)
	protected.Post("/empresas", both, empresaCtrl.Store)
	protected.Get("/empresas/:id<int>/edit", both, empresaCtrl.Edit)
	protected.Put("/empresas/:id<int>", both, empresaCtrl.Update)
	protected.Delete("/empresas/:id<int>", both, empresaCtrl.Delete)
	protected.Post("/empresas/:id<int>", both, empresaCtrl.Update)
	protected.Post("/empresas/:id<int>/delete", both, empresaCtrl.Delete)

	// ═══════════════════════════════════════════════════════════
	// CHOFERES — editar propio perfil (chofer + admin)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/choferes/:id<int>/edit", chof, choferCtrl.Edit)
	protected.Put("/choferes/:id<int>", chof, choferCtrl.Update)
	protected.Delete("/choferes/:id<int>", chof, choferCtrl.Delete)
	protected.Post("/choferes/:id<int>", chof, choferCtrl.Update)
	protected.Post("/choferes/:id<int>/delete", chof, choferCtrl.Delete)

	// ═══════════════════════════════════════════════════════════
	// PUBLICADORES — editar propio perfil (publicador + admin)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/publicadores/:id<int>/edit", pub, publicadorCtrl.Edit)
	protected.Put("/publicadores/:id<int>", pub, publicadorCtrl.Update)
	protected.Delete("/publicadores/:id<int>", pub, publicadorCtrl.Delete)
	protected.Post("/publicadores/:id<int>", pub, publicadorCtrl.Update)
	protected.Post("/publicadores/:id<int>/delete", pub, publicadorCtrl.Delete)

	// ═══════════════════════════════════════════════════════════
	// FACTURAS — escritura (publicador + admin)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/facturas/create", pub, facturaCtrl.Create)
	protected.Post("/facturas", pub, facturaCtrl.Store)
	protected.Get("/facturas/:id<int>/edit", pub, facturaCtrl.Edit)
	protected.Put("/facturas/:id<int>", pub, facturaCtrl.Update)
	protected.Delete("/facturas/:id<int>", pub, facturaCtrl.Delete)
	protected.Post("/facturas/:id<int>", pub, facturaCtrl.Update)
	protected.Post("/facturas/:id<int>/delete", pub, facturaCtrl.Delete)
	protected.Post("/facturas/:id<int>/pagar", pub, facturaCtrl.MarcarPagada)

		// ═══════════════════════════════════════════════════════════
	// ADMIN — todo el panel
	// ═══════════════════════════════════════════════════════════
	admin := app.Group("/admin", auth, adm)

	// Dashboard
	admin.Get("/", adminCtrl.Dashboard)

	// ── Users ──
	admin.Get("/users", adminCtrl.UsersIndex)
	admin.Get("/users/create", adminCtrl.UsersCreate)
	admin.Post("/users", adminCtrl.UsersStore)
	admin.Get("/users/:id<int>/edit", adminCtrl.UsersEdit)
	admin.Post("/users/:id<int>", adminCtrl.UsersUpdate)
	admin.Post("/users/:id<int>/delete", adminCtrl.UsersDelete)
	admin.Post("/users/:id<int>/toggle", adminCtrl.UsersToggleActive)

	// ── Empresas ──
	admin.Get("/empresas", adminCtrl.EmpresasIndex)
	admin.Get("/empresas/:id<int>/edit", adminCtrl.EmpresasEdit)
	admin.Post("/empresas/:id<int>", adminCtrl.EmpresasUpdate)
	admin.Post("/empresas/:id<int>/delete", adminCtrl.EmpresasDelete)

	// ── Choferes ──
	admin.Get("/choferes", adminCtrl.ChoferesIndex)
	admin.Get("/choferes/:id<int>/edit", adminCtrl.ChoferesEdit)
	admin.Post("/choferes/:id<int>", adminCtrl.ChoferesUpdate)
	admin.Post("/choferes/:id<int>/delete", adminCtrl.ChoferesDelete)

	// ── Publicadores ──
	admin.Get("/publicadores", adminCtrl.PublicadoresIndex)
	admin.Get("/publicadores/:id<int>/edit", adminCtrl.PublicadoresEdit)
	admin.Post("/publicadores/:id<int>", adminCtrl.PublicadoresUpdate)
	admin.Post("/publicadores/:id<int>/delete", adminCtrl.PublicadoresDelete)

	// ── Cargas ──
	admin.Get("/cargas", adminCtrl.CargasIndex)
	admin.Get("/cargas/:id<int>", adminCtrl.CargasShow)
	admin.Post("/cargas/:id<int>/delete", adminCtrl.CargasDelete)

	// ── Direcciones ──
	admin.Get("/direcciones", adminCtrl.DireccionesIndex)
	admin.Get("/direcciones/:id<int>/edit", adminCtrl.DireccionesEdit)
	admin.Post("/direcciones/:id<int>", adminCtrl.DireccionesUpdate)
	admin.Post("/direcciones/:id<int>/delete", adminCtrl.DireccionesDelete)

	// ── Facturas ──
	admin.Get("/facturas", adminCtrl.FacturasIndex)
	admin.Get("/facturas/:id<int>", adminCtrl.FacturasShow)
	admin.Post("/facturas/:id<int>", adminCtrl.FacturasUpdate)
	admin.Post("/facturas/:id<int>/delete", adminCtrl.FacturasDelete)
	admin.Post("/facturas/:id<int>/pagar", adminCtrl.FacturasPagar)
}
