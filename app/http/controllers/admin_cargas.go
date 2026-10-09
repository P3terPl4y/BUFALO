package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"goravel/app/services"
	"log"
	"strconv"
)

func (c *AdminController) CargasIndex(ctx fiber.Ctx) error {
	filters := map[string]string{
		"status":        ctx.Query("status"),
		"tipo_equipo":   ctx.Query("tipo_equipo"),
		"tipo_carga":    ctx.Query("tipo_carga"),
		"publicador_id": ctx.Query("publicador_id"),
		"chofer_id":     ctx.Query("chofer_id"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))
	page, perPage = services.NormalizePagination(page, perPage)

	list, total, err := c.cargaService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando cargas: %v", err)
		return fiber.ErrServiceUnavailable
	}

	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}

	return ctx.Render("admin/cargas/index", fiber.Map{
		"title":      "Cargas",
		"loads":      list,
		"total":      total,
		"page":       page,
		"perPage":    perPage,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"filters":    filters,
		"role":       ctx.Locals("role"),
		"csrfToken":  csrf.TokenFromContext(ctx),
	}, "layouts/base")
}

func (c *AdminController) CargasCreate(ctx fiber.Ctx) error {
	publicadores, _, err := c.publicadorService.GetAllWithFilters(map[string]string{}, 1, 500)
	if err != nil {
		log.Printf("Error listando publicadores para carga admin: %v", err)
		return fiber.ErrServiceUnavailable
	}
	destinations, err := c.direccionService.GetAll()
	if err != nil {
		log.Printf("Error listando direcciones para carga admin: %v", err)
		return fiber.ErrServiceUnavailable
	}
	return ctx.Render("admin/cargas/create", fiber.Map{
		"title": "Crear carga", "publicadores": publicadores,
		"destinations": destinations, "csrfToken": csrf.TokenFromContext(ctx),
		"role": ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) CargasShow(ctx fiber.Ctx) error {
	carga, err := c.cargaService.GetByID(ctx.Params("id"))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/admin/cargas?flash_error=Carga no encontrada")
	}

	hasMap := carga.OrigenDireccion != nil && carga.DestinoDireccion != nil &&
		carga.OrigenDireccion.Latitud != nil && carga.OrigenDireccion.Longitud != nil &&
		carga.DestinoDireccion.Latitud != nil && carga.DestinoDireccion.Longitud != nil

	return ctx.Render("admin/cargas/show", fiber.Map{
		"title":     "Detalle de Carga",
		"load":      carga,
		"hasMap":    hasMap,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) CargasDelete(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.cargaService.Delete(id); err != nil {
		return ctx.Redirect().To("/admin/cargas?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/cargas?flash_success=Carga eliminada")
}

// ═══════════════════════════════════════════════════════════════
// DIRECCIONES
// ═══════════════════════════════════════════════════════════════
