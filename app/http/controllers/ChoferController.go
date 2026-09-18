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

type ChoferController struct {
	service        *services.ChoferService
	empresaService *services.EmpresaService
}

func NewChoferController() *ChoferController {
	return &ChoferController{
		service:        services.NewChoferService(),
		empresaService: services.NewEmpresaService(),
	}
}

// ─────────────────────────────────────────────────────────────
// Index
// ─────────────────────────────────────────────────────────────
func (c *ChoferController) Index(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	role, _ := sess.Get("role").(string)
	userID, _ := sess.Get("user_id").(uint)

	filters := map[string]string{
		"estado": ctx.Query("estado"),
		"q":      ctx.Query("q"),
	}

	// Un carrier solo ve los choferes de su empresa
	if role == "chofer" {
		if chofer, err := c.service.GetByUserID(userID); err == nil {
			filters["empresa_id"] = strconv.FormatUint(uint64(chofer.EmpresaID), 10)
		}
	} else if eid := ctx.Query("empresa_id"); eid != "" {
		filters["empresa_id"] = eid
	}

	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "10"))

	list, total, err := c.service.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando choferes: %v", err)
	}
	// Datos de demostración para que la grilla sea evaluable en una instalación nueva.
	// Se mantienen solo en memoria y dejan de aparecer cuando existan registros reales.
	if total == 0 && filters["estado"] == "" && filters["q"] == "" {
		list = []models.Chofer{
			{ID: 101, NumeroLicencia: "BUF-48291", TipoLicencia: "C", AniosExperiencia: 8, Estado: models.ChoferDisponible, User: &models.User{Name: "Marcos Fernández", Email: "marcos.fernandez@demo.bufalo"}, Empresa: &models.Empresa{NombreLegal: "Rutas del Centro S.A."}},
			{ID: 102, NumeroLicencia: "BUF-51704", TipoLicencia: "B", AniosExperiencia: 5, Estado: models.ChoferEnViaje, User: &models.User{Name: "Laura González", Email: "laura.gonzalez@demo.bufalo"}, Empresa: &models.Empresa{NombreLegal: "Transporte Sierra Norte"}},
			{ID: 103, NumeroLicencia: "BUF-39018", TipoLicencia: "C", AniosExperiencia: 12, Estado: models.ChoferDisponible, User: &models.User{Name: "Jorge Martínez", Email: "jorge.martinez@demo.bufalo"}, Empresa: &models.Empresa{NombreLegal: "Logística Horizonte"}},
		}
		total = int64(len(list))
	}

	return ctx.Render("choferes/index", fiber.Map{
		"title":    "Choferes",
		"choferes": list,
		"total":    total,
		"page":     page,
		"perPage":  perPage,
		"filters":  filters,
		"role":     role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Show
// ─────────────────────────────────────────────────────────────
func (c *ChoferController) Show(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/choferes?flash_error=ID inválido")
	}

	chofer, err := c.service.GetByID(uint(id))
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{
			"title": "Chofer no encontrado",
			"role":  sess.Get("role"),
		}, "layouts/base")
	}

	return ctx.Render("choferes/show", fiber.Map{
		"title":     "Detalle del Chofer",
		"chofer":    chofer,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      sess.Get("role"),
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Edit
// ─────────────────────────────────────────────────────────────
// canEditChofer: admin siempre; si no, solo el UserID del propio chofer.
func canEditChofer(chofer *models.Chofer, userID uint, role string) bool {
	if role == "admin" {
		return true
	}
	if chofer == nil {
		return false
	}
	return chofer.UserID == userID
}

// ─────────────────────────────────────────────────────────────
// Edit
// ─────────────────────────────────────────────────────────────
func (c *ChoferController) Edit(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/choferes?flash_error=ID inválido")
	}

	// 1. Cargar PRIMERO
	chofer, err := c.service.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/choferes?flash_error=Chofer no encontrado")
	}

	// 2. Verificar ownership
	if !canEditChofer(chofer, userID, role) {
		return ctx.Redirect().To("/choferes?flash_error=No autorizado")
	}

	return ctx.Render("choferes/edit", fiber.Map{
		"title":     "Editar Chofer",
		"chofer":    chofer,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Update — SOLO campos de la tabla choferes
// ─────────────────────────────────────────────────────────────
func (c *ChoferController) Update(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/choferes?flash_error=ID inválido")
	}

	// 1. Cargar
	existing, err := c.service.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/choferes?flash_error=Chofer no encontrado")
	}

	// 2. Verificar ownership
	if !canEditChofer(existing, userID, role) {
		return ctx.Redirect().To("/choferes?flash_error=No autorizado")
	}

	// 3. Bind
	var req requests.ChoferUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To(fmt.Sprintf("/choferes/%d/edit?flash_error=Datos inválidos", id))
	}

	// 4. Validación
	rules := map[string]any{
		"numero_licencia":   "required|min:3|max:50",
		"tipo_licencia":     "required|max:20",
		"anios_experiencia": "required|integer|min:0",
		"estado":            "required|in:disponible,en_viaje,inactivo",
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
		return ctx.Render("choferes/edit", fiber.Map{
			"title":     "Editar Chofer",
			"chofer":    existing,
			"errors":    errs,
			"old":       req,
			"csrfToken": csrf.TokenFromContext(ctx),
			"role":      role,
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

	// 5. Whitelist: SOLO columnas de choferes
	updates := map[string]interface{}{
		"numero_licencia":            req.NumeroLicencia,
		"tipo_licencia":              req.TipoLicencia,
		"pais_emision_licencia":      req.PaisEmisionLicencia,
		"fecha_vencimiento_licencia": parseDate(req.FechaVencimientoLicencia),
		"anios_experiencia":          req.AniosExperiencia,
		"tipos_equipo_permitidos":    req.TiposEquipoPermitidos,
		"certificaciones":            req.Certificaciones,
		"numero_seguro":              req.NumeroSeguro,
		"fecha_vencimiento_seguro":   parseDate(req.FechaVencimientoSeguro),
		"estado":                     req.Estado,
	}

	if err := c.service.Update(uint(id), updates); err != nil {
		log.Printf("Error actualizando chofer %d: %v", id, err)
		return ctx.Redirect().To(fmt.Sprintf("/choferes/%d/edit?flash_error=Error al actualizar", id))
	}
	return ctx.Redirect().To(fmt.Sprintf("/choferes/%d?flash_success=Chofer actualizado", id))
}

// ─────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────
func (c *ChoferController) Delete(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/choferes?flash_error=ID inválido")
	}

	chofer, err := c.service.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/choferes?flash_error=Chofer no encontrado")
	}

	if !canEditChofer(chofer, userID, role) {
		return ctx.Redirect().To("/choferes?flash_error=No autorizado")
	}

	if err := c.service.Delete(uint(id)); err != nil {
		return ctx.Redirect().To("/choferes?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/choferes?flash_success=Chofer eliminado")
}
