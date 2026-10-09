package controllers

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"strconv"
	"time"
)

func (c *AdminController) PublicadoresIndex(ctx fiber.Ctx) error {
	filters := map[string]string{
		"estado":     ctx.Query("estado"),
		"empresa_id": ctx.Query("empresa_id"),
		"q":          ctx.Query("q"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))
	page, perPage = services.NormalizePagination(page, perPage)

	list, total, err := c.publicadorService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando publicadores: %v", err)
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

	return ctx.Render("admin/publicadores/index", fiber.Map{
		"title":        "Publicadores",
		"publicadores": list,
		"total":        total,
		"page":         page,
		"perPage":      perPage,
		"prevPage":     prevPage,
		"nextPage":     nextPage,
		"totalPages":   totalPages,
		"filters":      filters,
		"role":         ctx.Locals("role"),
		"csrfToken":    csrf.TokenFromContext(ctx),
	}, "layouts/base")
}

func (c *AdminController) PublicadoresEdit(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/publicadores?flash_error=ID inválido")
	}
	pub, err := c.publicadorService.GetByID(uint(id))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/admin/publicadores?flash_error=Publicador no encontrado")
	}

	return ctx.Render("admin/publicadores/edit", fiber.Map{
		"title":      "Editar Publicador",
		"publicador": pub,
		"csrfToken":  csrf.TokenFromContext(ctx),
		"role":       ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) PublicadoresUpdate(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/publicadores?flash_error=ID inválido")
	}
	var req requests.PublicadorUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To(fmt.Sprintf("/admin/publicadores/%d/edit?flash_error=Datos inválidos", id))
	}

	for _, date := range []string{req.FechaVencimientoLicencia} {
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return fiber.ErrBadRequest
			}
		}
	}
	parseDate := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return &t
		}
		return nil
	}

	updates := map[string]interface{}{
		"numero_licencia_broker":     req.NumeroLicenciaBroker,
		"pais_emision_licencia":      req.PaisEmisionLicencia,
		"fecha_vencimiento_licencia": parseDate(req.FechaVencimientoLicencia),
		"anios_experiencia":          req.AniosExperiencia,
		"especialidad":               req.Especialidad,
		"comision":                   req.Comision,
		"credit_score":               req.CreditScore,
		"estado":                     req.Estado,
	}
	if err := c.publicadorService.Update(uint(id), updates); err != nil {
		return ctx.Redirect().To(fmt.Sprintf("/admin/publicadores/%d/edit?flash_error=Error al actualizar", id))
	}
	return ctx.Redirect().To("/admin/publicadores?flash_success=Publicador actualizado")
}

func (c *AdminController) PublicadoresDelete(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/publicadores?flash_error=ID inválido")
	}
	if err := c.publicadorService.Delete(uint(id)); err != nil {
		return ctx.Redirect().To("/admin/publicadores?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/publicadores?flash_success=Publicador eliminado")
}

// ═══════════════════════════════════════════════════════════════
// CARGAS
// ═══════════════════════════════════════════════════════════════
