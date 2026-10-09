package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/monitoring"
	"goravel/app/services"
	"math"
	"os"
	"strconv"
	"time"
)

func (c *AdminController) Dashboard(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)

	// Conteos globales
	totalUsers, err := facades.Orm().Query().Model(&models.User{}).Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	totalEmpresas, err := facades.Orm().Query().Model(&models.Empresa{}).Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	totalChoferes, err := facades.Orm().Query().Model(&models.Chofer{}).Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	totalPublicadores, err := facades.Orm().Query().Model(&models.Publicador{}).Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	totalDirecciones, err := facades.Orm().Query().Model(&models.Direccion{}).Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}

	totalCargas, err := facades.Orm().Query().Model(&models.Carga{}).Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	cargasPublicadas, err := facades.Orm().Query().Model(&models.Carga{}).
		Where("estado = ?", "publicada").Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	cargasAsignadas, err := facades.Orm().Query().Model(&models.Carga{}).
		Where("estado = ?", "asignada").Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	cargasEntregadas, err := facades.Orm().Query().Model(&models.Carga{}).
		Where("estado = ?", "entregada").Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	cargasCanceladas, err := facades.Orm().Query().Model(&models.Carga{}).
		Where("estado = ?", "cancelada").Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}

	totalFacturas, err := facades.Orm().Query().Model(&models.Factura{}).Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	facturasPagadas, err := facades.Orm().Query().Model(&models.Factura{}).
		Where("estado = ?", "pagada").Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	facturasPendientes, err := facades.Orm().Query().Model(&models.Factura{}).
		Where("estado IN ?", []string{"emitida", "vencida"}).Count()
	if err != nil {
		return fiber.ErrServiceUnavailable
	}

	// Últimos usuarios y cargas
	var ultimosUsers []models.User
	var ultimasCargas []models.Carga

	err = facades.Orm().Query().Model(&models.User{}).
		Order("created_at desc").Limit(5).Find(&ultimosUsers)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	err = facades.Orm().Query().Model(&models.Carga{}).
		With("Publicador").With("Empresa").
		Order("created_at desc").Limit(5).Find(&ultimasCargas)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}

	return ctx.Render("admin/dashboard", fiber.Map{
		"title": "Panel de Administración",
		"role":  sess.Get("role"),

		"totalUsers":        totalUsers,
		"totalEmpresas":     totalEmpresas,
		"totalChoferes":     totalChoferes,
		"totalPublicadores": totalPublicadores,
		"totalDirecciones":  totalDirecciones,

		"totalCargas":      totalCargas,
		"cargasPublicadas": cargasPublicadas,
		"cargasAsignadas":  cargasAsignadas,
		"cargasEntregadas": cargasEntregadas,
		"cargasCanceladas": cargasCanceladas,

		"totalFacturas":      totalFacturas,
		"facturasPagadas":    facturasPagadas,
		"facturasPendientes": facturasPendientes,

		"ultimosUsers":  ultimosUsers,
		"ultimasCargas": ultimasCargas,
	}, "layouts/base")
}

// HealthMetrics exposes aggregate process metrics and dependency reachability
// only to the administrator route group. It never returns connection details.
func (c *AdminController) HealthMetrics(ctx fiber.Ctx) error {
	ctx.Set("Cache-Control", "no-store, private")
	ctx.Vary("Accept")
	if ctx.Path() == "/admin/health" && ctx.Accepts("json", "html") == "html" {
		return ctx.Render("admin/health", fiber.Map{"title": "Salud del sistema", "role": ctx.Locals("role")}, "layouts/base")
	}
	return ctx.JSON(healthSnapshot(ctx))
}

func healthSnapshot(ctx fiber.Ctx) fiber.Map {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "127.0.0.1"
	}
	dbPort, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	if dbPort <= 0 {
		dbPort = 5432
	}
	redisPort, _ := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if redisPort <= 0 {
		redisPort = 6379
	}
	dbOK, dbLatency := monitoring.CheckTCP(dbHost, dbPort)
	redisOK, redisLatency := monitoring.CheckTCP(redisHost, redisPort)

	status := "ok"
	if !dbOK || !redisOK {
		status = "degraded"
	}
	checks := monitoring.CheckDependencies(ctx.Context())
	if checks["postgres"].Status != "available" || checks["redis"].Status != "available" {
		status = "degraded"
	}
	ctx.Set("Cache-Control", "no-store, private")
	outbox, outboxErr := services.InterestOutboxStatus(ctx.Context())
	if outboxErr != nil || outbox["status"] != "ok" {
		status = "degraded"
	}
	return fiber.Map{
		"notifications": outbox,
		"status":        status,
		"checked_at":    time.Now().UTC().Format(time.RFC3339),
		"dependencies": fiber.Map{
			"postgres_tcp": fiber.Map{"status": componentStatus(dbOK), "latency_ms": roundMetric(dbLatency)},
			"redis_tcp":    fiber.Map{"status": componentStatus(redisOK), "latency_ms": roundMetric(redisLatency)},
		},
		"runtime":            monitoring.Snapshot(),
		"traffic_protection": monitoring.Traffic(),
		"checks":             checks,
		"history":            monitoring.RecentHistory(),
	}
}

func componentStatus(available bool) string {
	if available {
		return "available"
	}
	return "unavailable"
}

func roundMetric(value float64) float64 {
	return math.Round(value*100) / 100
}

// ═══════════════════════════════════════════════════════════════
// USERS
// ═══════════════════════════════════════════════════════════════
