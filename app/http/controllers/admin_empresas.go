package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"strconv"
)

func (c *AdminController) EmpresasIndex(ctx fiber.Ctx) error {
	filters := map[string]string{
		"tipo":   ctx.Query("tipo"),
		"estado": ctx.Query("estado"),
		"q":      ctx.Query("q"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))
	page, perPage = services.NormalizePagination(page, perPage)

	list, total, err := c.empresaService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando empresas: %v", err)
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

	return ctx.Render("admin/empresas/index", fiber.Map{
		"title":      "Empresas",
		"empresas":   list,
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

func (c *AdminController) EmpresasEdit(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/empresas?flash_error=ID inválido")
	}
	e, err := c.empresaService.GetByID(strconv.FormatUint(uint64(id), 10))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/admin/empresas?flash_error=Empresa no encontrada")
	}
	dests, _ := c.direccionService.GetAll()

	return ctx.Render("admin/empresas/edit", fiber.Map{
		"title":        "Editar Empresa",
		"empresa":      e,
		"destinations": dests,
		"csrfToken":    csrf.TokenFromContext(ctx),
		"role":         ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) EmpresasUpdate(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	var req requests.EmpresaUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/admin/empresas/" + id + "/edit?flash_error=Datos inválidos")
	}
	updates := map[string]interface{}{
		"tipo":             req.Tipo,
		"nombre_legal":     req.NombreLegal,
		"nombre_comercial": strPtr(req.NombreComercial),
		"tax_id":           strPtr(req.TaxID),
		"mc_number":        strPtr(req.MCNumber),
		"dot_number":       strPtr(req.DOTNumber),
		"direccion_id":     req.DireccionID,
		"telefono":         strPtr(req.Telefono),
		"email":            strPtr(req.Email),
		"sitio_web":        strPtr(req.SitioWeb),
		"credit_score":     req.CreditScore,
		"days_to_pay":      req.DaysToPay,
		"estado":           req.Estado,
	}
	if err := c.empresaService.Update(id, updates); err != nil {
		return ctx.Redirect().To("/admin/empresas/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To("/admin/empresas?flash_success=Empresa actualizada")
}

func (c *AdminController) EmpresasDelete(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.empresaService.Delete(id); err != nil {
		return ctx.Redirect().To("/admin/empresas?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/empresas?flash_success=Empresa eliminada")
}

// ═══════════════════════════════════════════════════════════════
// CHOFERES
// ═══════════════════════════════════════════════════════════════
