package controllers

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"goravel/app/exports"
	"goravel/app/monitoring"
)

// HealthExport serializes the same bounded snapshot consumed by the health UI.
// PDF is printed by that UI itself so its visual layout is identical on screen
// and on paper; CSV/XLSX use the same sections, filters and displayed values.
func (c *AdminController) HealthExport(ctx fiber.Ctx) error {
	format := strings.ToLower(strings.TrimSpace(ctx.Query("format")))
	if format != "csv" && format != "excel" && format != "xlsx" {
		return fiber.ErrBadRequest
	}
	snapshot := healthSnapshot(ctx)
	history, ok := snapshot["history"].(monitoring.HistorySnapshot)
	if !ok {
		return fiber.ErrServiceUnavailable
	}
	sections := healthReportSections(snapshot, history, ctx)
	body, contentType, extension, err := exports.Render(exports.Report{
		Title: "Salud del sistema BUFALO", Sections: sections,
	}, format)
	if err != nil {
		return fiber.ErrBadRequest
	}
	filename := "bufalo-salud-" + time.Now().Format("20060102-150405") + "." + extension
	ctx.Set("Content-Type", contentType)
	ctx.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	ctx.Set("Cache-Control", "no-store, private")
	return ctx.Send(body)
}

func healthReportSections(snapshot fiber.Map, history monitoring.HistorySnapshot, ctx fiber.Ctx) []exports.Section {
	sections := make([]exports.Section, 0, 8)
	addRows := func(title string, headers []string, rows [][]exports.Cell) {
		sections = append(sections, exports.Section{Title: title, Headers: headers, Rows: rows})
	}
	cells := func(values ...string) []exports.Cell {
		row := make([]exports.Cell, len(values))
		for i, value := range values {
			row[i] = exports.Cell{Value: value}
		}
		return row
	}
	statusRows := [][]exports.Cell{cells("Estado general", fmt.Sprint(snapshot["status"])), cells("Generado", fmt.Sprint(snapshot["checked_at"]))}
	if dependencies, ok := snapshot["dependencies"].(fiber.Map); ok {
		for _, name := range []string{"postgres_tcp", "redis_tcp"} {
			if item, exists := dependencies[name].(fiber.Map); exists {
				statusRows = append(statusRows, cells(name, fmt.Sprint(item["status"]), fmt.Sprint(item["latency_ms"])+" ms"))
			}
		}
	}
	if checks, ok := snapshot["checks"].(map[string]monitoring.DependencyResult); ok {
		for _, name := range []string{"postgres", "redis"} {
			if item, exists := checks[name]; exists {
				statusRows = append(statusRows, cells(name+" autenticado", item.Status, fmt.Sprintf("%.2f ms", item.LatencyMS)))
			}
		}
	}
	addRows("Estado y dependencias", []string{"Indicador", "Estado", "Latencia"}, statusRows)

	if runtime, ok := snapshot["runtime"].(monitoring.RuntimeSnapshot); ok {
		addRows("Rendimiento del proceso", []string{"Indicador", "Valor"}, [][]exports.Cell{
			cells("Tiempo activo", (time.Duration(runtime.UptimeSeconds) * time.Second).String()),
			cells("Peticiones desde el inicio", fmt.Sprint(runtime.Requests)),
			cells("Peticiones por minuto", fmt.Sprintf("%.2f", runtime.RequestsPerMinute)),
			cells("Errores 4xx", fmt.Sprint(runtime.ClientErrors)), cells("Errores 5xx", fmt.Sprint(runtime.ServerErrors)),
			cells("Tasa de errores 5xx", fmt.Sprintf("%.2f%%", runtime.ErrorRatePercent)),
			cells("Latencia media", fmt.Sprintf("%.2f ms", runtime.AverageLatencyMS)), cells("Latencia p95 aproximada", runtime.P95Latency),
			cells("Memoria residente", optionalMB(runtime.ResidentMemoryMB)), cells("Heap en uso", fmt.Sprintf("%.2f MB", runtime.HeapInUseMB)),
			cells("Heap asignado", fmt.Sprintf("%.2f MB", runtime.HeapAllocMB)), cells("Goroutines", fmt.Sprint(runtime.Goroutines)),
			cells("GOMAXPROCS", fmt.Sprint(runtime.GOMAXPROCS)), cells("Ciclos GC", fmt.Sprint(runtime.GCCycles)),
		})
	}
	if notifications, ok := snapshot["notifications"].(map[string]any); ok {
		rows := make([][]exports.Cell, 0, len(notifications))
		for _, name := range []string{"status", "pending", "retrying", "oldest_seconds"} {
			if value, exists := notifications[name]; exists {
				rows = append(rows, cells(name, fmt.Sprint(value)))
			}
		}
		addRows("Notificaciones pendientes", []string{"Indicador", "Valor"}, rows)
	}

	if traffic, ok := snapshot["traffic_protection"].(monitoring.TrafficSnapshot); ok {
		policy := traffic.Policy
		addRows("Protección de tráfico", []string{"Indicador", "Valor"}, [][]exports.Cell{
			cells("Modo", policy.Mode), cells("Límite dinámico por IP", fmt.Sprint(policy.Rate)+" req/s"),
			cells("Ráfaga por IP", fmt.Sprint(policy.Burst)), cells("Rechazadas por límite", fmt.Sprint(traffic.Limited)),
			cells("Bloqueos activos", fmt.Sprint(traffic.Banned)), cells("Rechazadas por capacidad", fmt.Sprint(traffic.CapacityRejected)),
			cells("Rechazadas por transporte", fmt.Sprint(traffic.TransportRejected)), cells("Cuerpo excedido", fmt.Sprint(traffic.BodyRejected)),
			cells("Tiempo agotado al leer", fmt.Sprint(traffic.ReadTimeouts)),
		})
		rows := make([][]exports.Cell, 0, len(traffic.RecentRejections))
		for _, event := range traffic.RecentRejections {
			rows = append(rows, cells(event.At.Format(time.RFC3339), event.Outcome, fmt.Sprint(event.Status), event.Identity, fmt.Sprint(event.Count)))
		}
		addRows("Rechazos de seguridad", []string{"Hora UTC", "Motivo", "HTTP", "Identificador seudónimo", "Cantidad"}, rows)
		scan := traffic.SecurityScan
		addRows("Análisis de seguridad", []string{"Indicador", "Valor"}, [][]exports.Cell{
			cells("Modo", scan.Mode), cells("Estado", scan.Status), cells("Iniciado", scan.StartedAt),
			cells("Finalizado", scan.FinishedAt), cells("Memoria máxima", fmt.Sprintf("%.2f MiB", float64(scan.MemoryPeakBytes)/(1024*1024))),
			cells("Límite de memoria", fmt.Sprintf("%.2f MiB", float64(scan.MemoryLimitBytes)/(1024*1024))),
			cells("CPU (s)", fmt.Sprintf("%.2f", scan.CPUSeconds)), cells("Duración (s)", fmt.Sprintf("%.2f", scan.DurationSeconds)), cells("Resultado", scan.Result),
		})
	}

	trendRows := make([][]exports.Cell, 0, len(history.Trend))
	for _, point := range history.Trend {
		trendRows = append(trendRows, []exports.Cell{
			{Value: point.At.Format("2006-01-02 15:04 UTC")}, exports.Number(int64(point.Requests)),
			exports.Number(int64(point.ClientErrors)), exports.Number(int64(point.ServerErrors)),
		})
	}
	addRows("Pulso de actividad · última hora", []string{"Minuto", "Peticiones", "Errores 4xx", "Errores 5xx"}, trendRows)

	tab := ctx.Query("tab", "requests")
	rows := history.Requests
	if tab == "errors" {
		rows = history.Errors
		tab = "errors"
	} else {
		tab = "requests"
	}
	method := strings.ToUpper(strings.TrimSpace(ctx.Query("method")))
	if !validHealthMethod(method) {
		method = ""
	}
	status := ctx.Query("status")
	if status != "2" && status != "3" && status != "4" && status != "5" {
		status = ""
	}
	route := strings.ToLower(strings.TrimSpace(ctx.Query("route")))
	if len(route) > 100 {
		route = route[:100]
	}
	historyRows := make([][]exports.Cell, 0, len(rows))
	for _, event := range rows {
		if method != "" && event.Method != method || status != "" && fmt.Sprint(event.Status/100) != status || route != "" && !strings.Contains(strings.ToLower(event.Route), route) {
			continue
		}
		historyRows = append(historyRows, []exports.Cell{
			{Value: event.At.Format("2006-01-02 15:04:05")}, {Value: event.Method}, {Value: event.Route},
			exports.Number(int64(event.Status)), exports.Cell{Value: fmt.Sprintf("%.2f", event.DurationMS), Numeric: true},
		})
	}
	sectionName := "Historial de peticiones"
	if tab == "errors" {
		sectionName = "Historial de errores"
	}
	addRows(sectionName, []string{"Hora", "Método", "Ruta", "HTTP", "Duración (ms)"}, historyRows)
	return sections
}

func optionalMB(value *float64) string {
	if value == nil {
		return "No disponible"
	}
	return fmt.Sprintf("%.2f MB", *value)
}

func validHealthMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}
