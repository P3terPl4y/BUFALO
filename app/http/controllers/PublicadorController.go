package controllers

import (
	"fmt"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
)

type PublicadorController struct {
	service *services.PublicadorService
}

func NewPublicadorController() *PublicadorController {
	return &PublicadorController{
		service: services.NewPublicadorService(),
	}
}

// canEditPublicador: admin siempre; si no, solo el UserID del propio publicador.
func canEditPublicador(pub *models.Publicador, userID uint, role string) bool {
	if role == "admin" {
		return true
	}
	if pub == nil {
		return false
	}
	return pub.UserID == userID
}

// ─────────────────────────────────────────────────────────────
// Index
// ─────────────────────────────────────────────────────────────
func (c *PublicadorController) Index(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	role, _ := sess.Get("role").(string)

	filters := map[string]string{
		"estado": ctx.Query("estado"),
		"q":      ctx.Query("q"),
	}
	if eid := ctx.Query("empresa_id"); eid != "" {
		filters["empresa_id"] = eid
	}

	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "10"))

	list, total, err := c.service.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando publicadores: %v", err)
	}

	return ctx.Render("publicadores/index", fiber.Map{
		"title":        "Publicadores",
		"publicadores": list,
		"total":        total,
		"page":         page,
		"perPage":      perPage,
		"filters":      filters,
		"role":         role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Show
// ─────────────────────────────────────────────────────────────
func (c *PublicadorController) Show(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/publicadores?flash_error=ID inválido")
	}

	pub, err := c.service.GetByID(uint(id))
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{
			"title": "Publicador no encontrado",
			"role":  sess.Get("role"),
		}, "layouts/base")
	}

	return ctx.Render("publicadores/show", fiber.Map{
		"title":      "Detalle del Publicador",
		"publicador": pub,
		"csrfToken":  csrf.TokenFromContext(ctx),
		"role":       sess.Get("role"),
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Edit
// ─────────────────────────────────────────────────────────────
func (c *PublicadorController) Edit(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/publicadores?flash_error=ID inválido")
	}

	// 1. Cargar PRIMERO
	pub, err := c.service.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/publicadores?flash_error=Publicador no encontrado")
	}

	// 2. Verificar ownership
	if !canEditPublicador(pub, userID, role) {
		return ctx.Redirect().To("/publicadores?flash_error=No autorizado")
	}

	return ctx.Render("publicadores/edit", fiber.Map{
		"title":      "Editar Publicador",
		"publicador": pub,
		"csrfToken":  csrf.TokenFromContext(ctx),
		"role":       role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Update — SOLO campos de la tabla publicadores
// ─────────────────────────────────────────────────────────────
func (c *PublicadorController) Update(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/publicadores?flash_error=ID inválido")
	}

	// 1. Cargar
	existing, err := c.service.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/publicadores?flash_error=Publicador no encontrado")
	}

	// 2. Verificar ownership
	if !canEditPublicador(existing, userID, role) {
		return ctx.Redirect().To("/publicadores?flash_error=No autorizado")
	}

	// 3. Bind
	var req requests.PublicadorUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To(fmt.Sprintf("/publicadores/%d/edit?flash_error=Datos inválidos", id))
	}

	// 4. Validación
	rules := map[string]any{
		"numero_licencia_broker": "required|min:3|max:50",
		"anios_experiencia":      "required|integer|min:0",
		"comision":               "nullable|numeric|min:0|max:100",
		"credit_score":           "nullable|integer|min:0",
		"estado":                 "required|in:activo,inactivo,suspendido",
	}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil || validator.Fails() {
		errs := map[string]string{}
		if validator != nil {
			for f, msgs := range validator.Errors().All() {
				for _, m := range msgs {
					errs[f] = m
					break
				}
			}
		}
		return ctx.Render("publicadores/edit", fiber.Map{
			"title":      "Editar Publicador",
			"publicador": existing,
			"errors":     errs,
			"old":        req,
			"csrfToken":  csrf.TokenFromContext(ctx),
			"role":       role,
		}, "layouts/base")
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

	// 5. Whitelist
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

	if err := c.service.Update(uint(id), updates); err != nil {
		log.Printf("Error actualizando publicador %d: %v", id, err)
		return ctx.Redirect().To(fmt.Sprintf("/publicadores/%d/edit?flash_error=Error al actualizar", id))
	}
	return ctx.Redirect().To(fmt.Sprintf("/publicadores/%d?flash_success=Publicador actualizado", id))
}

// ─────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────
func (c *PublicadorController) Delete(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/publicadores?flash_error=ID inválido")
	}

	pub, err := c.service.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/publicadores?flash_error=Publicador no encontrado")
	}

	if !canEditPublicador(pub, userID, role) {
		return ctx.Redirect().To("/publicadores?flash_error=No autorizado")
	}

	if err := c.service.Delete(uint(id)); err != nil {
		return ctx.Redirect().To("/publicadores?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/publicadores?flash_success=Publicador eliminado")
}
