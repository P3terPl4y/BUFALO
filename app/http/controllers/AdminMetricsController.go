package controllers

import (
	"fmt"
	"goravel/app/exports"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/monitoring"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

type metricStateRow struct {
	Estado string `db:"estado"`
	Total  int64  `db:"total"`
}

type metricRoleRow struct {
	Rol   string `db:"rol"`
	Total int64  `db:"total"`
}

type invoiceEntityMetric struct {
	ID     uint   `db:"id"`
	Nombre string `db:"nombre"`
	Total  int64  `db:"total"`
}

func countMetric(model any, condition string, args ...any) (int64, error) {
	query := facades.Orm().Query().Model(model)
	if condition != "" {
		query = query.Where(condition, args...)
	}
	return query.Count()
}

func loadMetricStates(table string, expectedStates []string) ([]metricStateRow, error) {
	rows := make([]metricStateRow, 0)
	query := "SELECT estado::text AS estado, COUNT(*) AS total FROM " + table + " WHERE deleted_at IS NULL GROUP BY estado::text ORDER BY estado::text"
	if err := facades.DB().Select(&rows, query); err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.Estado] = row.Total
	}
	complete := make([]metricStateRow, 0, len(rows)+len(expectedStates))
	for _, state := range expectedStates {
		complete = append(complete, metricStateRow{Estado: state, Total: counts[state]})
		delete(counts, state)
	}
	unknown := make([]string, 0, len(counts))
	for state := range counts {
		unknown = append(unknown, state)
	}
	sort.Strings(unknown)
	for _, state := range unknown {
		complete = append(complete, metricStateRow{Estado: state, Total: counts[state]})
	}
	return complete, nil
}

func (c *AdminController) Metrics(ctx fiber.Ctx) error {
	data, err := c.collectMetrics()
	if err != nil {
		return metricsUnavailable(err)
	}
	data["title"] = "Métricas del sistema"
	data["role"] = session.FromContext(ctx).Get("role")
	ctx.Set("Cache-Control", "no-store, private")
	return ctx.Render("admin/metrics", data, "layouts/base")
}

func (c *AdminController) collectMetrics() (fiber.Map, error) {
	userTotal, err := countMetric(&models.User{}, "")
	if err != nil {
		return nil, err
	}
	activeUsers, err := countMetric(&models.User{}, "is_active = ?", true)
	if err != nil {
		return nil, err
	}
	var activeUserIDs []uint
	if err := facades.Orm().Query().Model(&models.User{}).Where("is_active = ?", true).Pluck("id", &activeUserIDs); err != nil {
		return nil, err
	}
	inactiveUsers := userTotal - activeUsers
	connectedUsers := monitoring.ActiveOnlineUserCount(activeUserIDs, time.Now())
	if connectedUsers > activeUsers {
		connectedUsers = activeUsers
	}
	disconnectedUsers := activeUsers - connectedUsers

	companyTotal, err := countMetric(&models.Empresa{}, "")
	if err != nil {
		return nil, err
	}
	driverTotal, err := countMetric(&models.Chofer{}, "")
	if err != nil {
		return nil, err
	}
	publisherTotal, err := countMetric(&models.Publicador{}, "")
	if err != nil {
		return nil, err
	}
	addressTotal, err := countMetric(&models.Direccion{}, "")
	if err != nil {
		return nil, err
	}
	networkTotal, err := countMetric(&models.RedChofer{}, "")
	if err != nil {
		return nil, err
	}
	ratingTotal, err := countMetric(&models.ChoferCalificacion{}, "")
	if err != nil {
		return nil, err
	}

	var roleRows []metricRoleRow
	if err := facades.DB().Select(&roleRows, "SELECT role AS rol, COUNT(*) AS total FROM users GROUP BY role ORDER BY role"); err != nil {
		return nil, err
	}
	loadStates, err := loadMetricStates("cargas", []string{"publicada", "negociando", "asignada", "en_transito", "entregada", "cancelada"})
	if err != nil {
		return nil, err
	}
	facturaStates, err := loadMetricStates("facturas", []string{"borrador", "emitida", "pagada", "vencida", "cancelada"})
	if err != nil {
		return nil, err
	}
	totalLoads := sumMetricStates(loadStates)
	totalInvoices := sumMetricStates(facturaStates)

	companies := make([]invoiceEntityMetric, 0)
	if err := facades.DB().Select(&companies, `
		SELECT e.id, COALESCE(NULLIF(e.nombre_comercial, ''), e.nombre_legal) AS nombre,
		       COUNT(DISTINCT f.id) AS total
		FROM empresas e
		LEFT JOIN facturas f ON f.deleted_at IS NULL AND (f.emisor_id = e.id OR f.receptor_id = e.id)
		WHERE e.deleted_at IS NULL
		GROUP BY e.id, e.nombre_comercial, e.nombre_legal
		ORDER BY total DESC, nombre ASC`); err != nil {
		return nil, err
	}
	drivers := make([]invoiceEntityMetric, 0)
	if err := facades.DB().Select(&drivers, `
		SELECT ch.id, COALESCE(u.name, 'Chofer sin usuario') AS nombre, COUNT(f.id) AS total
		FROM chofers ch
		LEFT JOIN users u ON u.id = ch.user_id
		LEFT JOIN facturas f ON f.chofer_id = ch.id AND f.deleted_at IS NULL
		WHERE ch.deleted_at IS NULL
		GROUP BY ch.id, u.name
		ORDER BY total DESC, nombre ASC`); err != nil {
		return nil, err
	}
	publishers := make([]invoiceEntityMetric, 0)
	if err := facades.DB().Select(&publishers, `
		SELECT p.id, COALESCE(u.name, 'Publicador sin usuario') AS nombre, COUNT(f.id) AS total
		FROM publicadors p
		LEFT JOIN users u ON u.id = p.user_id
		LEFT JOIN facturas f ON f.publicador_id = p.id AND f.deleted_at IS NULL
		WHERE p.deleted_at IS NULL
		GROUP BY p.id, u.name
		ORDER BY total DESC, nombre ASC`); err != nil {
		return nil, err
	}

	return fiber.Map{
		"metricsAt":  time.Now().Format("02/01/2006 15:04:05"),
		"totalUsers": userTotal, "activeUsers": activeUsers, "inactiveUsers": inactiveUsers,
		"connectedUsers": connectedUsers, "disconnectedUsers": disconnectedUsers,
		"onlineWindow": monitoring.OnlineWindow.String(), "roles": roleRows,
		"totalCompanies": companyTotal, "totalDrivers": driverTotal, "totalPublishers": publisherTotal,
		"totalAddresses": addressTotal, "totalNetworkMemberships": networkTotal, "totalRatings": ratingTotal,
		"loadStates": loadStates, "totalLoads": totalLoads,
		"invoiceStates": facturaStates, "totalInvoices": totalInvoices,
		"invoicesByCompany": companies, "invoicesByDriver": drivers, "invoicesByPublisher": publishers,
	}, nil
}

func (c *AdminController) MetricsExport(ctx fiber.Ctx) error {
	data, err := c.collectMetrics()
	if err != nil {
		return metricsUnavailable(err)
	}
	rows := make([][]exports.Cell, 0)
	add := func(section, name, value string, numeric bool) {
		rows = append(rows, []exports.Cell{{Value: section}, {Value: name}, {Value: value, Numeric: numeric}})
	}
	add("Reporte", "Generado", time.Now().Format(time.RFC3339), false)
	add("Usuarios", "Total", fmt.Sprint(data["totalUsers"]), true)
	add("Usuarios", "Habilitados", fmt.Sprint(data["activeUsers"]), true)
	add("Usuarios", "Deshabilitados", fmt.Sprint(data["inactiveUsers"]), true)
	add("Usuarios", "Conectados recientemente", fmt.Sprint(data["connectedUsers"]), true)
	add("Usuarios", "Activos sin actividad reciente", fmt.Sprint(data["disconnectedUsers"]), true)
	for _, row := range data["roles"].([]metricRoleRow) {
		add("Usuarios por rol", row.Rol, fmt.Sprint(row.Total), true)
	}
	for _, item := range []struct {
		name string
		key  string
	}{
		{"Empresas", "totalCompanies"}, {"Perfiles de chofer", "totalDrivers"},
		{"Perfiles de publicador", "totalPublishers"}, {"Direcciones", "totalAddresses"},
		{"Vínculos en redes", "totalNetworkMemberships"}, {"Calificaciones", "totalRatings"},
		{"Cargas totales", "totalLoads"}, {"Facturas totales", "totalInvoices"},
	} {
		add("Entidades", item.name, fmt.Sprint(data[item.key]), true)
	}
	for _, row := range data["loadStates"].([]metricStateRow) {
		add("Cargas por estado", row.Estado, fmt.Sprint(row.Total), true)
	}
	for _, row := range data["invoiceStates"].([]metricStateRow) {
		add("Facturas por estado", row.Estado, fmt.Sprint(row.Total), true)
	}
	for _, entity := range []struct {
		section string
		key     string
	}{
		{"Facturas por empresa", "invoicesByCompany"}, {"Facturas por chofer", "invoicesByDriver"},
		{"Facturas por publicador", "invoicesByPublisher"},
	} {
		for _, row := range data[entity.key].([]invoiceEntityMetric) {
			add(entity.section, fmt.Sprintf("%s (#%d)", row.Nombre, row.ID), fmt.Sprint(row.Total), true)
		}
	}
	addHealthExportRows(&rows)
	body, contentType, extension, err := exports.Render(exports.Report{
		Title: "Métricas del sistema BUFALO", Headers: []string{"Sección", "Métrica", "Valor"}, Rows: rows,
	}, ctx.Query("format"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	filename := "bufalo-metricas-" + time.Now().Format("20060102-150405") + "." + extension
	ctx.Set("Content-Type", contentType)
	ctx.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	ctx.Set("Cache-Control", "no-store, private")
	return ctx.Send(body)
}

func addHealthExportRows(rows *[][]exports.Cell) {
	add := func(name, value string, numeric bool) {
		*rows = append(*rows, []exports.Cell{{Value: "Salud del proceso"}, {Value: name}, {Value: value, Numeric: numeric}})
	}
	runtime := monitoring.Snapshot()
	dbHost, dbPort := os.Getenv("DB_HOST"), metricPort("DB_PORT", 5432)
	if dbHost == "" {
		dbHost = "localhost"
	}
	redisHost, redisPort := os.Getenv("REDIS_HOST"), metricPort("REDIS_PORT", 6379)
	if redisHost == "" {
		redisHost = "127.0.0.1"
	}
	dbOK, dbLatency := monitoring.CheckTCP(dbHost, dbPort)
	redisOK, redisLatency := monitoring.CheckTCP(redisHost, redisPort)
	add("PostgreSQL TCP", fmt.Sprintf("%s · %.2f ms", componentStatus(dbOK), roundMetric(dbLatency)), false)
	add("Redis TCP", fmt.Sprintf("%s · %.2f ms", componentStatus(redisOK), roundMetric(redisLatency)), false)
	add("Tiempo activo (segundos)", fmt.Sprint(runtime.UptimeSeconds), true)
	add("Solicitudes", fmt.Sprint(runtime.Requests), true)
	add("Solicitudes por minuto promedio", fmt.Sprintf("%.2f", runtime.RequestsPerMinute), true)
	add("Errores HTTP 4xx", fmt.Sprint(runtime.ClientErrors), true)
	add("Errores HTTP 5xx", fmt.Sprint(runtime.ServerErrors), true)
	add("Tasa de errores 5xx (%)", fmt.Sprintf("%.2f", runtime.ErrorRatePercent), true)
	add("Latencia media (ms)", fmt.Sprintf("%.2f", runtime.AverageLatencyMS), true)
	add("Latencia p95 aproximada", runtime.P95Latency, false)
	add("Heap en uso (MB)", fmt.Sprintf("%.2f", runtime.HeapInUseMB), true)
	add("Heap asignado (MB)", fmt.Sprintf("%.2f", runtime.HeapAllocMB), true)
	add("Memoria acumulada (MB)", fmt.Sprintf("%.2f", runtime.TotalAllocMB), true)
	add("Goroutines", fmt.Sprint(runtime.Goroutines), true)
	add("GOMAXPROCS", fmt.Sprint(runtime.GOMAXPROCS), true)
	add("Ciclos GC", fmt.Sprint(runtime.GCCycles), true)
}

func metricPort(envName string, fallback int) int {
	port, err := strconv.Atoi(os.Getenv(envName))
	if err != nil || port <= 0 {
		return fallback
	}
	return port
}

func sumMetricStates(rows []metricStateRow) int64 {
	var total int64
	for _, row := range rows {
		total += row.Total
	}
	return total
}

func metricsUnavailable(err error) error {
	log.Printf("No se pudieron cargar las métricas administrativas: %v", err)
	return fiber.ErrServiceUnavailable
}
