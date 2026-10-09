package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"strconv"
)

func (c *AdminController) FacturasIndex(ctx fiber.Ctx) error {
	filters := map[string]string{
		"estado":      ctx.Query("estado"),
		"emisor_id":   ctx.Query("emisor_id"),
		"receptor_id": ctx.Query("receptor_id"),
		"fecha_desde": ctx.Query("fecha_desde"),
		"fecha_hasta": ctx.Query("fecha_hasta"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))
	page, perPage = services.NormalizePagination(page, perPage)

	list, total, err := c.facturaService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando facturas: %v", err)
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

	return ctx.Render("admin/facturas/index", fiber.Map{
		"title":      "Facturas",
		"facturas":   list,
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

func (c *AdminController) FacturasShow(ctx fiber.Ctx) error {
	f, err := c.facturaService.GetByID(ctx.Params("id"))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/admin/facturas?flash_error=Factura no encontrada")
	}
	return ctx.Render("admin/facturas/show", fiber.Map{
		"title":     "Detalle de Factura",
		"factura":   f,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) FacturasEdit(ctx fiber.Ctx) error {
	f, err := c.facturaService.GetByID(ctx.Params("id"))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).Render("admin/dashboard/404", fiber.Map{"title": "Factura no encontrada", "role": ctx.Locals("role")}, "layouts/base")
	}
	return ctx.Render("admin/facturas/edit", fiber.Map{
		"title": "Editar factura", "factura": f,
		"csrfToken": csrf.TokenFromContext(ctx), "role": ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) FacturasUpdate(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	var req requests.FacturaUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/admin/facturas/" + id + "/edit?flash_error=Datos inválidos")
	}
	updates := map[string]interface{}{
		"distancia_km":  req.DistanciaKm,
		"tarifa_por_km": req.TarifaPorKm,
		"subtotal":      req.Subtotal,
		"impuestos":     req.Impuestos,
		"total":         req.Total,
		"moneda":        req.Moneda,
		"metodo_pago":   strPtr(req.MetodoPago),
	}
	if err := c.facturaService.Update(id, updates); err != nil {
		return ctx.Redirect().To("/admin/facturas/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To("/admin/facturas?flash_success=Factura actualizada")
}

func (c *AdminController) FacturasDelete(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.facturaService.Delete(id); err != nil {
		return ctx.Redirect().To("/admin/facturas?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/facturas?flash_success=Factura eliminada")
}

func (c *AdminController) FacturasPagar(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	metodo := ctx.FormValue("metodo_pago")

	if err := c.facturaService.MarcarPagada(id, metodo); err != nil {
		return ctx.Redirect().To("/admin/facturas/" + id + "?flash_error=Error al marcar pagada")
	}
	return ctx.Redirect().To("/admin/facturas/" + id + "?flash_success=Factura marcada como pagada")
}
