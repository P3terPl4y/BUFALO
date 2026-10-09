package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"goravel/app/http/ingress"
	"goravel/app/http/middleware"
	"goravel/app/models"
	"goravel/app/monitoring"
	"goravel/app/services"
	"goravel/app/sessionstore"
	"goravel/app/viewhelpers"
	"goravel/bootstrap"
	"goravel/routes"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v3"
	"github.com/goravel/framework/facades"
	goredis "github.com/redis/go-redis/v9"
)

// ── Helpers para variables de entorno ──

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

// ensureAdminUser crea un usuario administrador si no existe en la base de datos.
func ensureAdminUser() {
	adminEmail := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	adminPassword := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if adminEmail == "" || adminPassword == "" {
		return
	}
	if len(adminPassword) < 12 {
		log.Println("BOOTSTRAP_ADMIN_PASSWORD requiere al menos 12 caracteres")
		return
	}
	adminName := "Administrador"
	adminRole := "admin"

	log.Println("🔍 Verificando existencia de usuario administrador...")

	var user models.User
	err := facades.Orm().Query().Where("email = ?", adminEmail).First(&user)

	if err == nil && user.ID > 0 {
		log.Printf("✅ Usuario administrador ya existe (ID: %d, Email: %s)", user.ID, user.Email)
		return
	}

	log.Printf("⚠️  Usuario administrador no encontrado. Creando...")

	hashed, err := facades.Hash().Make(adminPassword)
	if err != nil {
		log.Printf("❌ Error al hashear contraseña del admin: %v", err)
		return
	}

	admin := models.User{
		Name:     adminName,
		Email:    adminEmail,
		Password: hashed,
		Role:     adminRole,
		IsActive: true,
	}

	if err := facades.Orm().Query().Create(&admin); err != nil {
		log.Printf("❌ Error al crear usuario administrador: %v", err)
		return
	}

	log.Printf("✅ Usuario administrador creado con éxito (ID: %d, Email: %s)", admin.ID, admin.Email)
}

func main() {
	if value := os.Getenv("APP_ENV"); value != "" {
		normalized, err := normalizeEnvironment(value)
		if err != nil {
			log.Fatal(err)
		}
		if err := os.Setenv("APP_ENV", normalized); err != nil {
			log.Fatal(err)
		}
	}
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(min(2, runtime.NumCPU()))
	}
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(128 << 20)
	}
	log.Println("🚀 Iniciando DAT Clone...")

	// ── Bootstrap de Goravel ──
	appGoravel := bootstrap.Boot()

	if len(os.Args) > 1 && (os.Args[1] == "migrate" || (len(os.Args) == 3 && os.Args[1] == "artisan" && os.Args[2] == "migrate")) {
		if err := bootstrap.RunMigrations(context.Background()); err != nil {
			log.Fatal(err)
		}
		return
	}
	// ── Comandos artisan ──
	if len(os.Args) > 1 && os.Args[1] == "artisan" {
		log.Printf("📦 Ejecutando comando artisan: %v", os.Args[2:])
		if err := facades.Artisan().Run(os.Args[2:], false); err != nil {
			log.Fatal(err)
		}
		return
	}

	// ── Environment ──
	isProd := productionEnvironment(env("APP_ENV", "local"))
	if err := validateProductionConfig(os.Getenv); err != nil {
		log.Fatal(err)
	}
	rateLimitKeySecret := env("RATE_LIMIT_KEY_SECRET", env("APP_KEY", ""))
	if rateLimitKeySecret != "" {
		if err := os.Setenv("RATE_LIMIT_KEY_SECRET", rateLimitKeySecret); err != nil {
			log.Fatal("could not configure rate limit key secret")
		}
	}

	// Validate critical production settings before an optional bootstrap user
	// write can occur.
	trafficPolicy := middleware.DefaultTrafficPolicy()
	switch env("DDOS_MODE", "observe") {
	case "observe":
		trafficPolicy.Observe = true
	case "enforce":
	default:
		log.Fatal("DDOS_MODE must be observe or enforce")
	}
	monitoring.ConfigureTraffic(monitoring.TrafficSettings{Mode: env("DDOS_MODE", "observe"), Rate: trafficPolicy.Rate, Burst: trafficPolicy.Burst, StaticRate: trafficPolicy.StaticRate, ShortBanSeconds: int64(trafficPolicy.ShortBan.Seconds()), LongBanSeconds: int64(trafficPolicy.LongBan.Seconds()), RequestCapacity: ingress.RequestCapacity, AuthCapacity: 4})
	ensureAdminUser()

	// ── Redis storage para sesiones ──
	sessionRedis := goredis.NewClient(&goredis.Options{Addr: net.JoinHostPort(env("REDIS_HOST", "127.0.0.1"), strconv.Itoa(envInt("REDIS_PORT", 6379))), Username: env("REDIS_USERNAME", ""), Password: env("REDIS_PASSWORD", ""), DB: envInt("REDIS_DB", 0), PoolSize: 10, MaxRetries: -1, DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second, PoolTimeout: time.Second, ContextTimeoutEnabled: true})
	defer sessionRedis.Close()
	redisStore := sessionstore.New(sessionRedis, 10000)

	rateLimitRedis := goredis.NewClient(&goredis.Options{
		Addr:                  net.JoinHostPort(env("RATE_REDIS_HOST", env("REDIS_HOST", "127.0.0.1")), strconv.Itoa(envInt("RATE_REDIS_PORT", envInt("REDIS_PORT", 6379)))),
		Username:              env("RATE_REDIS_USERNAME", env("REDIS_USERNAME", "")),
		Password:              env("RATE_REDIS_PASSWORD", env("REDIS_PASSWORD", "")),
		DB:                    envInt("RATE_REDIS_DB", envInt("REDIS_DB", 0)),
		PoolSize:              envInt("REDIS_POOL_SIZE", 10),
		DialTimeout:           2 * time.Second,
		ReadTimeout:           1 * time.Second,
		WriteTimeout:          1 * time.Second,
		PoolTimeout:           2 * time.Second,
		MaxRetries:            -1,
		ContextTimeoutEnabled: true,
	})
	trafficGuard, err := middleware.NewTrafficGuard(rateLimitRedis, trafficPolicy)
	if err != nil {
		log.Fatal(err)
	}

	// ── Template engine ──
	log.Println("🖼️  Configurando motor de plantillas HTML...")
	engine := html.New("./app/views", ".html")
	// Reloading reparses every template on each render. Keep it opt-in when
	// debug is disabled, including local performance/acceptance runs.
	engine.Reload(envBool("VIEWS_RELOAD", envBool("APP_DEBUG", !isProd)))
	engine.Delims("{{", "}}")
	viewhelpers.Register(engine)
	// ── Fiber app ──
	appFiber := fiber.New(fiber.Config{
		Views:             engine,
		BodyLimit:         6 << 20, // 5 MB profile images plus multipart headers.
		Concurrency:       64,
		ReduceMemoryUsage: true,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       30 * time.Second,
		ErrorHandler:      middleware.RenderError,
		TrustProxy:        true,
		// cloudflared supplies this header. Trust it only from the local tunnel
		// or a loopback reverse proxy that overwrites it; never arbitrary XFF.
		ProxyHeader: "CF-Connecting-IP",
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback: true, // Confía en 127.0.0.0/8 y ::1/128
			Proxies:  trustedProxyIPs(),
		},
	})
	log.Println("✅ App Fiber creada")
	appFiber.Server().HeaderReceived = middleware.AuthBodyConfig

	// ════════════════════════════════════════════════════════════
	// MIDDLEWARES GLOBALES
	// ════════════════════════════════════════════════════════════
	log.Println("🔒 Configurando middlewares...")

	// 1) Logger + Recover
	appFiber.Use(monitoring.Middleware())
	appFiber.Use(middleware.ErrorHandler())
	appFiber.Use(recover.New())

	// 2) Helmet — cabeceras de seguridad
	appFiber.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
			"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
			"img-src 'self' data: blob: https://*.tile.openstreetmap.org https://tile.openstreetmap.org; " +
			"font-src 'self' https://cdn.jsdelivr.net; " +
			"connect-src 'self' https://nominatim.openstreetmap.org https://router.project-osrm.org; " +
			"frame-ancestors 'none';",
		HSTSMaxAge:                31536000,
		HSTSPreloadEnabled:        true,
		XFrameOptions:             "DENY",
		XSSProtection:             "1; mode=block",
		ContentTypeNosniff:        "nosniff",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginEmbedderPolicy: "", // ← sin esto, bloquea los tiles
		CrossOriginResourcePolicy: "", // ← también mejor vacío en dev
	}))

	// 3) Sesiones con Redis
	log.Println("🍪 Configurando middleware de sesiones...")
	appFiber.Use(func(c fiber.Ctx) error {
		c.Response().Header.Del("Cross-Origin-Embedder-Policy")
		c.Response().Header.Del("Cross-Origin-Resource-Policy")
		return c.Next()
	})

	// Rejections are counted in traffic metrics instead of producing one disk
	// log line per abusive request.
	appFiber.Use(logger.New(logger.Config{Next: func(c fiber.Ctx) bool { return c.Path() == "/healthz" }}))
	// Public files do not need a session or a CSRF token. Missing files fall
	// through to the normal middleware and routes, preserving authentication.
	appFiber.Get("/*", static.New("./public", static.Config{Browse: false}))
	// En HTTPS usar __Host- (más seguro). En HTTP usar nombre normal.
	sessionCookieName := "session_id"
	if isProd {
		sessionCookieName = "__Host-session"
	}

	sessionMiddleware, sessionStore := session.NewWithStore(session.Config{
		Storage:           redisStore,
		Extractor:         extractors.FromCookie(sessionCookieName),
		CookieSecure:      isProd, // true solo en prod con HTTPS
		CookieHTTPOnly:    true,
		CookieSameSite:    "Lax",
		CookieSessionOnly: true,
		IdleTimeout:       30 * time.Minute,
		AbsoluteTimeout:   24 * time.Hour,
	})
	appFiber.Use(monitoring.SkipForHealthProbes(sessionMiddleware))
	log.Println("✅ Middleware de sesiones configurado")

	// 4) CSRF — vinculado a la sesión
	csrfCookieName := "csrf_"
	if isProd {
		csrfCookieName = "__Host-csrf_"
	}

	csrfMiddleware := csrf.New(csrf.Config{
		CookieName:     csrfCookieName,
		CookieSecure:   isProd,
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		Extractor:      extractors.FromForm("_csrf"),
		IdleTimeout:    30 * time.Minute,
		Session:        sessionStore, // ← vincula el token a la sesión
		TrustedOrigins: trustedOrigins(),
		ErrorHandler:   middleware.RenderError,
	})
	appFiber.Use(monitoring.SkipForHealthProbes(csrfMiddleware))

	appFiber.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	postgresCheck := func(ctx context.Context) error {
		var result []int
		return facades.DB().WithContext(ctx).Select(&result, "SELECT 1")
	}
	redisCheck := func(ctx context.Context) error {
		if err := sessionstore.WriteCheck(sessionRedis, "bufalo:probe:sessions")(ctx); err != nil {
			return err
		}
		return sessionstore.WriteCheck(rateLimitRedis, "bufalo:probe:traffic")(ctx)
	}
	readinessCheck := monitoring.CachedReadiness(postgresCheck, redisCheck)
	monitoring.SetDependencyChecks(postgresCheck, redisCheck)
	appFiber.Get("/readyz", monitoring.ReadinessHandler(readinessCheck))

	// ── Rutas ──
	log.Println("🛤️  Registrando rutas...")
	routes.SetupWebRoutesWithRateCounter(appFiber, middleware.NewRedisRateCounter(rateLimitRedis))
	log.Println("✅ Rutas registradas")

	// Start cleanup independently of request handling, in bounded batches.
	go services.RunPendingCleanup(appGoravel.Context())
	go services.RunInterestOutbox(appGoravel.Context())
	host := env("APP_HOST", "127.0.0.1")
	port := envInt("APP_PORT", 3000)
	server := &http.Server{Addr: net.JoinHostPort(host, strconv.Itoa(port)), Handler: ingress.New(adaptor.FiberApp(appFiber), trafficGuard, trustedProxyIPs()), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 25 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 15 * time.Second, MaxHeaderBytes: 16 << 10, BaseContext: func(net.Listener) context.Context { return appGoravel.Context() }}
	health := &http.Server{Addr: net.JoinHostPort("127.0.0.1", strconv.Itoa(envInt("APP_HEALTH_PORT", port+1))), Handler: monitoring.InternalHealth(readinessCheck), ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: 3 * time.Second, MaxHeaderBytes: 4096}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Fatal(err)
	}
	healthListener, err := net.Listen("tcp", health.Addr)
	if err != nil {
		listener.Close()
		log.Fatal(err)
	}
	failures := make(chan error, 2)
	go func() { failures <- server.Serve(ingress.LimitListener(listener, 256)) }()
	go func() { failures <- health.Serve(ingress.LimitListener(healthListener, 16)) }()
	log.Printf("BUFALO listening on %s; internal health on %s", server.Addr, health.Addr)
	select {
	case <-appGoravel.Context().Done():
	case err = <-failures:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server stopped: %v", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if shutdownErr := server.Shutdown(ctx); shutdownErr != nil {
		log.Printf("HTTP shutdown: %v", shutdownErr)
		_ = server.Close()
	}
	_ = health.Shutdown(ctx)
	_ = rateLimitRedis.Close()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func trustedProxyIPs() []string {
	proxies, err := parseTrustedProxyIPs(os.Getenv("TRUSTED_PROXY_IPS"))
	if err != nil {
		log.Fatal(err)
	}
	return proxies
}

func trustedOrigins() []string {
	raw := env("CSRF_TRUSTED_ORIGINS", "")
	var origins []string
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			origins = append(origins, value)
		}
	}
	return origins
}
