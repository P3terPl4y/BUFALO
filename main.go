package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"goravel/app/models"
	"goravel/bootstrap"
	"goravel/routes"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/storage/redis/v3"
	"github.com/gofiber/template/html/v3"
	"github.com/goravel/framework/facades"
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
	adminEmail := "admin@example.com"
	adminPassword := "Admin123!"
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
	log.Println("🚀 Iniciando DAT Clone...")

	// ── Bootstrap de Goravel ──
	appGoravel := bootstrap.Boot()
	_ = appGoravel

	// ── Comandos artisan ──
	if len(os.Args) > 1 && os.Args[1] == "artisan" {
		log.Printf("📦 Ejecutando comando artisan: %v", os.Args[2:])
		if err := facades.Artisan().Run(os.Args[2:], false); err != nil {
			log.Fatal(err)
		}
		return
	}

	ensureAdminUser()
	ensureDemoData()

	// ── Environment ──
	isProd := env("APP_ENV", "local") == "production"

	// ── Redis storage para sesiones ──
	redisStore := redis.New(redis.Config{
		Host:     env("REDIS_HOST", "127.0.0.1"),
		Port:     envInt("REDIS_PORT", 6379),
		Username: env("REDIS_USERNAME", ""),
		Password: env("REDIS_PASSWORD", ""),
		Database: envInt("REDIS_DB", 0),
		PoolSize: envInt("REDIS_POOL_SIZE", 10),
	})

	// ── Template engine ──
	log.Println("🖼️  Configurando motor de plantillas HTML...")
	engine := html.New("./app/views", ".html")
	engine.Reload(!isProd)
	engine.Delims("{{", "}}")
	engine.AddFunc("deref", func(p *uint) uint {
		if p == nil {
			return 0
		}
		return *p
	})
	engine.AddFunc("add", func(a, b int) int { return a + b })
	engine.AddFunc("sub", func(a, b int) int { return a - b })
	// ── Fiber app ──
	appFiber := fiber.New(fiber.Config{
		Views:      engine,
		TrustProxy: true,
		// 2. Leer la IP real desde la cabecera que envía Ngrok
		ProxyHeader: fiber.HeaderXForwardedFor,
		// 3. Confiar solo en proxies locales (Ngrok se conecta desde 127.0.0.1)
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback: true, // Confía en 127.0.0.0/8 y ::1/128
		},
	})
	log.Println("✅ App Fiber creada")

	// ════════════════════════════════════════════════════════════
	// MIDDLEWARES GLOBALES
	// ════════════════════════════════════════════════════════════
	log.Println("🔒 Configurando middlewares...")

	// 1) Logger + Recover
	appFiber.Use(logger.New())
	appFiber.Use(recover.New())

	// 2) Helmet — cabeceras de seguridad
	appFiber.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
			"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
			"img-src 'self' data: blob: https://*.tile.openstreetmap.org https://tile.openstreetmap.org; " +
			"font-src 'self' https://cdn.jsdelivr.net; " +
			"connect-src 'self' https://cdn.jsdelivr.net https://nominatim.openstreetmap.org; " +
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
	appFiber.Use(sessionMiddleware)
	log.Println("✅ Middleware de sesiones configurado")

	// 4) CSRF — vinculado a la sesión
	csrfCookieName := "csrf_"
	if isProd {
		csrfCookieName = "__Host-csrf_"
	}

	appFiber.Use(csrf.New(csrf.Config{
		CookieName:     csrfCookieName,
		CookieSecure:   isProd,
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		Extractor:      extractors.FromForm("_csrf"),
		IdleTimeout:    30 * time.Minute,
		Session:        sessionStore, // ← vincula el token a la sesión
		TrustedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
			"https://mariana-flagless-inaudibly.ngrok-free.dev",
		},
	}))

	// ── Archivos estáticos ──
	log.Println("📁 Configurando archivos estáticos...")
	appFiber.Get("/*", static.New("./public", static.Config{
		Browse: false,
	}))
	log.Println("✅ Archivos estáticos configurados")

	// ── Rutas ──
	log.Println("🛤️  Registrando rutas...")
	routes.SetupWebRoutes(appFiber)
	log.Println("✅ Rutas registradas")

	// ── Arrancar ──
	log.Println("🌐 Servidor iniciado en http://localhost:3000")
	port := env("APP_PORT", "3000")
	log.Printf("🚀 BUFALO escuchando en :%s", port)
	log.Fatal(appFiber.Listen(":" + port))
}
