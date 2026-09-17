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

	// Atajo para no repetir: todas las rutas cuelgan de un solo Use
	protected := app.Group("", auth)

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
	protected.Post("/loads", pub, cargaCtrl.Store)
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
	// ADMIN
	// ═══════════════════════════════════════════════════════════
	admin := app.Group("/admin", auth, adm)
	admin.Get("/users", adminCtrl.Index)
	admin.Get("/users/create", adminCtrl.Create)
	admin.Post("/users", adminCtrl.Store)
	admin.Get("/users/:id<int>/edit", adminCtrl.Edit)
	admin.Post("/users/:id<int>", adminCtrl.Update)
	admin.Post("/users/:id<int>/delete", adminCtrl.Delete)
	admin.Post("/users/:id<int>/toggle", adminCtrl.ToggleActive)
}
