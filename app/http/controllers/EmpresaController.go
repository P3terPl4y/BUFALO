package controllers

import (
	"fmt"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
)

type EmpresaController struct {
	service          *services.EmpresaService
	direccionService *services.DireccionService
}

func NewEmpresaController() *EmpresaController {
	return &EmpresaController{
		service:          services.NewEmpresaService(),
		direccionService: services.NewDireccionService(),
	}
}

// canEditEmpresa devuelve true si el usuario puede modificar la empresa.
// Regla: admin siempre; si no, solo el OwnerID.
func canEditEmpresa(empresa *models.Empresa, userID uint, role string) bool {
	if role == "admin" {
		return true
	}
	if empresa == nil || empresa.OwnerID == nil {
		return false
	}
	return *empresa.OwnerID == userID
}

// currentUser lee user_id y role de la sesión una sola vez.
func currentUser(ctx fiber.Ctx) (uint, string) {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)
	return userID, role
}

// ─────────────────────────────────────────────────────────────
// Index
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Index(ctx fiber.Ctx) error {
	_, role := currentUser(ctx)

	filters := map[string]string{
		"tipo":   ctx.Query("tipo"),
		"estado": ctx.Query("estado"),
		"q":      ctx.Query("q"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "10"))

	list, total, err := c.service.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando empresas: %v", err)
	}

	return ctx.Render("empresas/index", fiber.Map{
		"title":    "Empresas",
		"empresas": list,
		"total":    total,
		"page":     page,
		"perPage":  int64(perPage),
		"filters":  filters,
		"role":     role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Show
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Show(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	e, err := c.service.GetByID(ctx.Params("id"))
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{
			"title": "No encontrada",
			"role":  role,
		}, "layouts/base")
	}

	choferes, _ := c.service.GetChoferes(ctx.Params("id"))

	return ctx.Render("empresas/show", fiber.Map{
		"title":     "Detalle Empresa",
		"empresa":   e,
		"choferes":  choferes,
		"userID":    userID,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Create(ctx fiber.Ctx) error {
	_, role := currentUser(ctx)

	dests, _ := c.direccionService.GetAll()

	return ctx.Render("empresas/create", fiber.Map{
		"title":        "Nueva Empresa",
		"destinations": dests,
		"csrfToken":    csrf.TokenFromContext(ctx),
		"role":         role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Store
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Store(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	var req requests.EmpresaStoreRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/empresas/create?flash_error=Datos inválidos")
	}

	// ── Regla de tipo según rol ──
	// admin → cualquier tipo
	// publicador → solo broker
	// chofer → solo carrier
	if role != "admin" {
		expected := "broker"
		if role == "chofer" {
			expected = "carrier"
		}
		if req.Tipo != expected {
			return ctx.Redirect().To(fmt.Sprintf(
				"/empresas/create?flash_error=Como %s solo puedes crear empresas tipo %s",
				role, expected,
			))
		}
	}

	rules := map[string]any{
		"tipo":         "required|in:broker,carrier,shipper,factoring,mixto",
		"nombre_legal": "required|min:3|max:255",
		"estado":       "required|in:activo,inactivo,suspendido",
	}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil || validator.Fails() {
		return ctx.Redirect().To("/empresas/create?flash_error=Error de validación")
	}

	// ── OwnerID = quien la crea ──
	ownerID := userID

	e := models.Empresa{
		Tipo:            models.TipoEmpresa(req.Tipo),
		NombreLegal:     req.NombreLegal,
		NombreComercial: strPtr(req.NombreComercial),
		TaxID:           strPtr(req.TaxID),
		MCNumber:        strPtr(req.MCNumber),
		DOTNumber:       strPtr(req.DOTNumber),
		Telefono:        strPtr(req.Telefono),
		Email:           strPtr(req.Email),
		SitioWeb:        strPtr(req.SitioWeb),
		DireccionID:     req.DireccionID,
		CreditScore:     req.CreditScore,
		DaysToPay:       &req.DaysToPay,
		OwnerID:         &ownerID,
		Estado:          models.EstadoEmpresa(req.Estado),
	}

	if err := c.service.Create(&e); err != nil {
		log.Printf("Error creando empresa: %v", err)
		return ctx.Redirect().To("/empresas/create?flash_error=Error al guardar")
	}
	return ctx.Redirect().To(fmt.Sprintf("/empresas/%d", e.ID))
}

// ─────────────────────────────────────────────────────────────
// Edit
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Edit(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	id := ctx.Params("id")

	// 1. Cargar la entidad PRIMERO
	e, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Redirect().To("/empresas?flash_error=Empresa no encontrada")
	}

	// 2. Verificar ownership
	if !canEditEmpresa(e, userID, role) {
		return ctx.Redirect().To("/empresas?flash_error=No autorizado")
	}

	dests, _ := c.direccionService.GetAll()

	return ctx.Render("empresas/edit", fiber.Map{
		"title":        "Editar Empresa",
		"empresa":      e,
		"destinations": dests,
		"userID":       userID,
		"csrfToken":    csrf.TokenFromContext(ctx),
		"role":         role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Update(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	id := ctx.Params("id")

	// 1. Cargar
	existing, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Redirect().To("/empresas?flash_error=Empresa no encontrada")
	}

	// 2. Verificar ownership
	if !canEditEmpresa(existing, userID, role) {
		return ctx.Redirect().To("/empresas?flash_error=No autorizado")
	}

	// 3. Bind
	var req requests.EmpresaUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/empresas/" + id + "/edit?flash_error=Datos inválidos")
	}

	// 4. Validación
	rules := map[string]any{
		"tipo":         "required|in:broker,carrier,shipper,factoring,mixto",
		"nombre_legal": "required|min:3|max:255",
		"estado":       "required|in:activo,inactivo,suspendido",
	}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil || validator.Fails() {
		return ctx.Redirect().To("/empresas/" + id + "/edit?flash_error=Error de validación")
	}

	// 5. Whitelist de updates (NO se toca OwnerID, ni ID, ni CreatedAt)
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
	if req.DireccionID != nil && *req.DireccionID > 0 {
		updates["direccion_id"] = uintToUUID(*req.DireccionID)
	} else {
		updates["direccion_id"] = nil
	}
	if err := c.service.Update(id, updates); err != nil {
		log.Printf("Error actualizando empresa %s: %v", id, err)
		return ctx.Redirect().To("/empresas/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To("/empresas/" + id + "?flash_success=Empresa actualizada")
}

// ─────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Delete(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	id := ctx.Params("id")

	// 1. Cargar
	e, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Redirect().To("/empresas?flash_error=Empresa no encontrada")
	}

	// 2. Verificar ownership
	if !canEditEmpresa(e, userID, role) {
		return ctx.Redirect().To("/empresas?flash_error=No autorizado")
	}

	// 3. Eliminar
	if err := c.service.Delete(id); err != nil {
		log.Printf("Error eliminando empresa %s: %v", id, err)
		return ctx.Redirect().To("/empresas?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/empresas?flash_success=Empresa eliminada")
}
