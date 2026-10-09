package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"goravel/app/requests"
	"goravel/app/services"
	"strconv"
)

func (c *AdminController) DireccionesIndex(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	page, perPage := services.NormalizePagination(page, 100)
	list, err := c.direccionService.GetAll(page, perPage)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}

	return ctx.Render("admin/direcciones/index", fiber.Map{
		"title": "Direcciones", "page": page, "nextPage": page + 1, "previousPage": page - 1, "hasNext": len(list) == perPage,
		"direcciones": list,
		"csrfToken":   csrf.TokenFromContext(ctx),
		"role":        ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) DireccionesEdit(ctx fiber.Ctx) error {
	d, err := c.direccionService.GetByID(ctx.Params("id"))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/admin/direcciones?flash_error=Dirección no encontrada")
	}

	return ctx.Render("admin/direcciones/edit", fiber.Map{
		"title":     "Editar Dirección",
		"direccion": d,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) DireccionesUpdate(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	var req requests.DireccionUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/admin/direcciones/" + id + "/edit?flash_error=Datos inválidos")
	}

	updates := map[string]interface{}{
		"calle":            strPtr(req.Calle),
		"ciudad":           req.Ciudad,
		"estado_provincia": req.EstadoProvincia,
		"codigo_postal":    strPtr(req.CodigoPostal),
		"pais":             req.Pais,
		"latitud":          req.Latitud,
		"longitud":         req.Longitud,
	}
	if err := c.direccionService.Update(id, updates); err != nil {
		return ctx.Redirect().To("/admin/direcciones/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To("/admin/direcciones?flash_success=Dirección actualizada")
}

func (c *AdminController) DireccionesDelete(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.direccionService.Delete(id); err != nil {
		return ctx.Redirect().To("/admin/direcciones?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/direcciones?flash_success=Dirección eliminada")
}

// ═══════════════════════════════════════════════════════════════
// FACTURAS
// ═══════════════════════════════════════════════════════════════
