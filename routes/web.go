package routes

import (
	"time"

	"goravel/app/http/controllers"
	"goravel/app/http/middleware"
	"goravel/app/monitoring"

	"github.com/gofiber/fiber/v3"
)

// SetupWebRoutes accepts a limiter override for isolated HTTP test harnesses.
func SetupWebRoutes(app *fiber.App, limiterOverride ...fiber.Handler) {
	SetupWebRoutesWithRateCounter(app, middleware.NewMemoryRateCounter(), limiterOverride...)
}

// SetupWebRoutesWithRateCounter wires atomic shared counters into authentication
// routes. Production must pass a Redis-backed counter.
func SetupWebRoutesWithRateCounter(app *fiber.App, counter middleware.RateCounter, limiterOverride ...fiber.Handler) {
	setupWebRoutes(app, counter, nil, limiterOverride...)
}

// SetupWebRoutesWithEmailSender installs an injectable sender for isolated
// integration tests; production callers should use SetupWebRoutesWithRateCounter.
func SetupWebRoutesWithEmailSender(app *fiber.App, counter middleware.RateCounter, sender func(string, string) error, limiterOverride ...fiber.Handler) {
	setupWebRoutes(app, counter, sender, limiterOverride...)
}

func setupWebRoutes(app *fiber.App, counter middleware.RateCounter, sender func(string, string) error, limiterOverride ...fiber.Handler) {
	loginIPLimiter := middleware.LoginIPRateLimiter(counter)
	loginEmailLimiter := middleware.LoginEmailRateLimiter(counter)
	confirmationLimiter := middleware.ConfirmationIPRateLimiter(counter)
	registerIPLimiter := middleware.RegisterIPRateLimiter(counter)
	registerEmailLimiter := middleware.RegisterEmailRateLimiter(counter)
	profileLimiter := middleware.ProfileAccountRateLimiter(counter)
	interestLimiter := middleware.InterestAccountRateLimiter(counter)
	loginCapacity := middleware.NewConcurrencyLimit(4)
	registerCapacity := middleware.NewConcurrencyLimit(2)
	if len(limiterOverride) > 0 {
		loginIPLimiter = limiterOverride[0]
		loginEmailLimiter = limiterOverride[0]
		confirmationLimiter = limiterOverride[0]
		registerIPLimiter = limiterOverride[0]
		registerEmailLimiter = limiterOverride[0]
		profileLimiter = limiterOverride[0]
		interestLimiter = limiterOverride[0]
	}
	// ------------------------------------------------------------------
	// Controladores
	// ------------------------------------------------------------------
	var authCtrl *controllers.AuthController
	if sender == nil {
		authCtrl = controllers.NewAuthController()
	} else {
		authCtrl = controllers.NewAuthControllerWithVerificationSender(sender)
	}
	cargaCtrl := controllers.NewCargaController()
	direccionCtrl := controllers.NewDireccionController()
	empresaCtrl := controllers.NewEmpresaController()
	choferCtrl := controllers.NewChoferController()
	publicadorCtrl := controllers.NewPublicadorController()
	facturaCtrl := controllers.NewFacturaController()
	loadInterestCtrl := controllers.NewLoadInterestController()
	userCtrl := controllers.NewUserController()
	if sender != nil {
		userCtrl = controllers.NewUserControllerWithVerificationSender(sender)
	}
	adminCtrl := controllers.NewAdminController()
	communityCtrl := controllers.NewDriverCommunityController()

	// ============================================================
	// PÚBLICAS
	// ============================================================
	app.Get("/login", authCtrl.ShowLogin)
	app.Post("/login", middleware.AuthFormOnly, loginIPLimiter, loginEmailLimiter, loginCapacity, authCtrl.HandleLogin)
	app.Get("/register", authCtrl.ShowRegister)
	app.Post("/register", middleware.AuthFormOnly, registerIPLimiter, registerEmailLimiter, registerCapacity, authCtrl.HandleRegister)
	app.Post("/register/resend", middleware.AuthFormOnly, confirmationLimiter, loginCapacity, authCtrl.ResendVerification)
	app.Get("/register/confirm", authCtrl.ShowEmailConfirmation)
	app.Post("/register/confirm", middleware.AuthFormOnly, confirmationLimiter, loginCapacity, authCtrl.ConfirmEmail)

	// ============================================================
	// PROTEGIDAS — todas las rutas van planas con su middleware
	// ============================================================
	auth := middleware.SessionAuth()
	pub := middleware.PublicadorAuth()
	chof := middleware.ChoferAuth()
	both := middleware.RoleAuth("publicador", "chofer", "admin")
	pubAdmin := middleware.RoleAuth("publicador", "admin")
	adm := middleware.AdminAuth()
	is_active := middleware.IsActiveUserHandler()
	// Atajo para no repetir: todas las rutas cuelgan de un solo Use
	protected := app.Group("", auth, is_active)

	// ── Generales ──
	protected.Get("/home", authCtrl.ShowHome)
	protected.Get("/logout", authCtrl.ShowLogout)
	protected.Post("/logout", authCtrl.Logout)
	protected.Get("/presence/heartbeat", func(ctx fiber.Ctx) error {
		if userID, ok := ctx.Locals("user_id").(uint); ok {
			monitoring.RecordUserActivity(userID, time.Now())
		}
		ctx.Set("Cache-Control", "no-store, private")
		return ctx.SendStatus(fiber.StatusNoContent)
	})
	protected.Get("/profile", userCtrl.Show)
	protected.Get("/profile/edit", userCtrl.Edit)
	protected.Post("/profile/update", profileLimiter, middleware.NewConcurrencyLimit(2), userCtrl.Update)
	protected.Post("/profile/photo", userCtrl.UploadPhoto)
	protected.Get("/red-choferes", pub, communityCtrl.Index)
	protected.Post("/red-choferes/:id<int>", pub, communityCtrl.Add)
	protected.Post("/red-choferes/:id<int>/remove", pub, communityCtrl.Remove)

	// ═══════════════════════════════════════════════════════════
	// LECTURA (cualquier autenticado)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/loads", cargaCtrl.Index)
	protected.Get("/loads/:id<int>", cargaCtrl.Show)

	protected.Get("/direcciones", direccionCtrl.Index)
	protected.Get("/direcciones/options", both, direccionCtrl.All)
	protected.Get("/direcciones/:id<int>", direccionCtrl.Show)

	protected.Post("/empresas/:id<int>/membership", profileLimiter, empresaCtrl.RequestMembership)
	protected.Get("/empresas", empresaCtrl.Index)
	protected.Get("/empresas/:id<int>", empresaCtrl.Show)

	protected.Get("/choferes", choferCtrl.Index)
	protected.Get("/choferes/:id<int>", choferCtrl.Show)

	protected.Get("/publicadores", publicadorCtrl.Index)
	protected.Get("/publicadores/:id<int>", publicadorCtrl.Show)

	protected.Get("/facturas", facturaCtrl.Index)
	protected.Get("/facturas/export", both, facturaCtrl.Export)
	protected.Get("/facturas/disenador", both, facturaCtrl.Studio)
	protected.Get("/facturas/:id<int>", facturaCtrl.Show)

	// ═══════════════════════════════════════════════════════════
	// CARGAS — escritura (publicador + admin)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/loads/create", pubAdmin, cargaCtrl.Create)
	protected.Post("/loads", pubAdmin, is_active, cargaCtrl.Store)
	protected.Get("/loads/:id<int>/edit", pubAdmin, cargaCtrl.Edit)
	protected.Put("/loads/:id<int>", pubAdmin, cargaCtrl.Update)
	protected.Delete("/loads/:id<int>", pubAdmin, cargaCtrl.Delete)
	protected.Post("/loads/:id<int>", pubAdmin, cargaCtrl.Update)
	protected.Post("/loads/:id<int>/delete", pubAdmin, cargaCtrl.Delete)
	protected.Post("/loads/:id<int>/assign-chofer", pub, cargaCtrl.AssignChofer)
	protected.Post("/loads/:id<int>/rate-driver", pub, cargaCtrl.RateDriver)

	// ═══════════════════════════════════════════════════════════
	// CARGAS — aceptar / interés (chofer + admin)
	// ═══════════════════════════════════════════════════════════
	protected.Post("/loads/:id<int>/accept", chof, cargaCtrl.AcceptLoad)
	protected.Post("/loads/:id<int>/start-transit", chof, cargaCtrl.StartTransit)
	protected.Post("/loads/:id<int>/deliver", chof, cargaCtrl.MarkDelivered)
	protected.Post("/loads/:id<int>/interest", chof, interestLimiter, middleware.NewConcurrencyLimit(2), loadInterestCtrl.SendInterest)

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
	// FACTURAS — escritura por el perfil emisor (publicador o chofer)
	// ═══════════════════════════════════════════════════════════
	protected.Get("/facturas/create", both, facturaCtrl.Create)
	protected.Post("/facturas", both, facturaCtrl.Store)
	protected.Get("/facturas/:id<int>/edit", both, facturaCtrl.Edit)
	protected.Put("/facturas/:id<int>", both, facturaCtrl.Update)
	protected.Delete("/facturas/:id<int>", both, facturaCtrl.Delete)
	protected.Post("/facturas/:id<int>", both, facturaCtrl.Update)
	protected.Post("/facturas/:id<int>/delete", both, facturaCtrl.Delete)
	protected.Post("/facturas/:id<int>/emitir", both, facturaCtrl.Emitir)
	protected.Post("/facturas/:id<int>/pagar", both, facturaCtrl.MarcarPagada)

	// ═══════════════════════════════════════════════════════════
	// ADMIN — todo el panel
	// ═══════════════════════════════════════════════════════════
	admin := app.Group("/admin", auth, adm)

	// Dashboard
	admin.Get("/", adminCtrl.Dashboard)
	admin.Get("/metrics", adminCtrl.Metrics)
	admin.Get("/metrics/export", adminCtrl.MetricsExport)
	admin.Get("/health", adminCtrl.HealthMetrics)
	admin.Get("/health/data", adminCtrl.HealthMetrics)
	admin.Get("/health/export", adminCtrl.HealthExport)

	// ── Users ──
	admin.Get("/users", adminCtrl.UsersIndex)
	admin.Get("/users/create", adminCtrl.UsersCreate)
	admin.Post("/users", adminCtrl.UsersStore)
	admin.Get("/users/:id<int>/edit", adminCtrl.UsersEdit)
	admin.Post("/users/:id<int>", adminCtrl.UsersUpdate)
	admin.Post("/users/:id<int>/delete", adminCtrl.UsersDelete)
	admin.Post("/users/:id<int>/toggle", adminCtrl.UsersToggleActive)

	// ── Empresas ──
	admin.Get("/memberships", adminCtrl.MembershipRequests)
	admin.Post("/memberships/:id<int>", adminCtrl.DecideMembership)
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
	admin.Get("/cargas/create", adminCtrl.CargasCreate)
	admin.Post("/cargas", cargaCtrl.Store)
	admin.Get("/cargas/:id<int>/edit", cargaCtrl.Edit)
	admin.Get("/cargas/:id<int>", adminCtrl.CargasShow)
	admin.Post("/cargas/:id<int>/delete", adminCtrl.CargasDelete)

	// ── Direcciones ──
	admin.Get("/direcciones", adminCtrl.DireccionesIndex)
	admin.Get("/direcciones/:id<int>/edit", adminCtrl.DireccionesEdit)
	admin.Post("/direcciones/:id<int>", adminCtrl.DireccionesUpdate)
	admin.Post("/direcciones/:id<int>/delete", adminCtrl.DireccionesDelete)

	// ── Facturas ──
	admin.Get("/facturas", adminCtrl.FacturasIndex)
	admin.Get("/facturas/disenador", facturaCtrl.Studio)
	admin.Get("/facturas/export", facturaCtrl.Export)
	admin.Get("/facturas/:id<int>", adminCtrl.FacturasShow)
	admin.Get("/facturas/:id<int>/edit", adminCtrl.FacturasEdit)
	admin.Post("/facturas/:id<int>", adminCtrl.FacturasUpdate)
	admin.Post("/facturas/:id<int>/delete", adminCtrl.FacturasDelete)
	admin.Post("/facturas/:id<int>/pagar", adminCtrl.FacturasPagar)
}
