package controllers

import (
	"fmt"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"math"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
)

type DireccionController struct {
	service *services.DireccionService
}

func NewDireccionController() *DireccionController {
	return &DireccionController{service: services.NewDireccionService()}
}

func canManageAddress(d *models.Direccion, userID uint, role string) bool {
	if d == nil || userID == 0 {
		return false
	}
	if role == "admin" {
		return true
	}
	return d.OwnerID != nil && *d.OwnerID == userID
}

func (c *DireccionController) authorized(ctx fiber.Ctx, d *models.Direccion) bool {
	userID, role := currentUser(ctx)
	return canManageAddress(d, userID, role)
}

func (c *DireccionController) Index(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, role := currentUser(ctx)
	var list []models.Direccion
	var err error
	if role == "admin" {
		list, err = c.service.GetAll()
	} else {
		list, err = c.service.GetOwnedByUserID(userID)
	}
	if err != nil {
		log.Printf("Error listando direcciones: %v", err)
		return fiber.ErrInternalServerError
	}
	return ctx.Render("direcciones/index", fiber.Map{
		"title":       "Direcciones",
		"direcciones": list,
		"csrfToken":   csrf.TokenFromContext(ctx),
		"role":        sess.Get("role"),
	}, "layouts/base")
}

func (c *DireccionController) Show(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	d, err := c.service.GetByID(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
	}
	if !c.authorized(ctx, d) {
		return ctx.Status(fiber.StatusNotFound).Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
	}
	return ctx.Render("direcciones/show", fiber.Map{
		"title":     "Detalle Dirección",
		"direccion": d,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      sess.Get("role"),
	}, "layouts/base")
}

func (c *DireccionController) Create(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	return ctx.Render("direcciones/create", fiber.Map{
		"title":     "Nueva Dirección",
		"flash_error": ctx.Query("flash_error"),
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      sess.Get("role"),
	}, "layouts/base")
}

func (c *DireccionController) Store(ctx fiber.Ctx) error {
	userID, _ := currentUser(ctx)
	var req requests.DireccionStoreRequest
	if err := ctx.Bind().Body(&req); err != nil {
		log.Printf("[Direccion.Store] bind error: %v", err)
		return ctx.Redirect().To("/direcciones/create?flash_error=Datos inválidos")
	}

	// Log del payload para diagnóstico
	log.Printf("[Direccion.Store] payload: ciudad=%q provincia=%q pais=%q lat=%v lng=%v",
		req.Ciudad, req.EstadoProvincia, req.Pais, req.Latitud, req.Longitud)

	rules := map[string]any{
		"ciudad":           "required|min:2|max:100",
		"estado_provincia": "required|min:2|max:100",
		"pais":             "required|max:100",
	}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil {
		log.Printf("[Direccion.Store] validator error: %v", err)
		return ctx.Redirect().To("/direcciones/create?flash_error=Error al validar")
	}
	if validator.Fails() {
		// Log de los campos que fallaron
		for f, msgs := range validator.Errors().All() {
			log.Printf("[Direccion.Store] validación falló en %s: %v", f, msgs)
		}
		return ctx.Redirect().To("/direcciones/create?flash_error=Error de validación")
	}

	// Convertir lat/lng string → *float64

	if req.Latitud != nil && (math.IsNaN(*req.Latitud) || math.IsInf(*req.Latitud, 0) || *req.Latitud < -90 || *req.Latitud > 90) {
		return ctx.Redirect().To("/direcciones/create?flash_error=Latitud inválida")
	}
	if req.Longitud != nil && (math.IsNaN(*req.Longitud) || math.IsInf(*req.Longitud, 0) || *req.Longitud < -180 || *req.Longitud > 180) {
		return ctx.Redirect().To("/direcciones/create?flash_error=Longitud inválida")
	}
	pais := req.Pais
	if pais == "" {
		pais = "Cuba"
	}
	ownerID := userID
	d := models.Direccion{
		OwnerID:         &ownerID,
		Calle:           strPtr(req.Calle),
		Ciudad:          req.Ciudad,
		EstadoProvincia: req.EstadoProvincia,
		CodigoPostal:    strPtr(req.CodigoPostal),
		Pais:            pais,
		Latitud:         req.Latitud,
		Longitud:        req.Longitud,
	}

	if err := c.service.Create(&d); err != nil {
		log.Printf("[Direccion.Store] create error: %v", err)
		return ctx.Redirect().To("/direcciones/create?flash_error=Error al guardar")
	}
	return ctx.Redirect().To(fmt.Sprintf("/direcciones/%d", d.ID))
}

func (c *DireccionController) Edit(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	d, err := c.service.GetByID(ctx.Params("id"))
	if err != nil || !c.authorized(ctx, d) {
		return ctx.Status(fiber.StatusNotFound).Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
	}
	return ctx.Render("direcciones/edit", fiber.Map{"title": "Editar Dirección", "direccion": d, "csrfToken": csrf.TokenFromContext(ctx), "role": sess.Get("role")}, "layouts/base")
}

func (c *DireccionController) Update(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	d, err := c.service.GetByID(id)
	if err != nil {
		return fiber.ErrNotFound
	}
	if !c.authorized(ctx, d) {
		return fiber.ErrForbidden
	}
	var req requests.DireccionUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return fiber.ErrBadRequest
	}
	rules := map[string]any{"ciudad": "required|min:2|max:100", "estado_provincia": "required|min:2|max:100", "pais": "required|min:2|max:100"}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil || validator.Fails() {
		return fiber.ErrBadRequest
	}
	if req.Latitud != nil && (math.IsNaN(*req.Latitud) || math.IsInf(*req.Latitud, 0) || *req.Latitud < -90 || *req.Latitud > 90) {
		return fiber.ErrBadRequest
	}
	if req.Longitud != nil && (math.IsNaN(*req.Longitud) || math.IsInf(*req.Longitud, 0) || *req.Longitud < -180 || *req.Longitud > 180) {
		return fiber.ErrBadRequest
	}
	updates := map[string]interface{}{"calle": strPtr(req.Calle), "ciudad": req.Ciudad, "estado_provincia": req.EstadoProvincia, "codigo_postal": strPtr(req.CodigoPostal), "pais": req.Pais, "latitud": req.Latitud, "longitud": req.Longitud}
	if err := c.service.Update(id, updates); err != nil {
		return ctx.Redirect().To("/direcciones/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To("/direcciones/" + id + "?flash_success=Actualizada")
}

func (c *DireccionController) Delete(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	d, err := c.service.GetByID(id)
	if err != nil {
		return fiber.ErrNotFound
	}
	if !c.authorized(ctx, d) {
		return fiber.ErrForbidden
	}
	if err := c.service.Delete(id); err != nil {
		return ctx.Redirect().To("/direcciones?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/direcciones?flash_success=Eliminada")
}

// Helper expuesto para selects en formularios
func (c *DireccionController) All(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)
	var list []models.Direccion
	var err error
	if role == "admin" {
		list, err = c.service.GetAll()
	} else {
		list, err = c.service.GetOwnedByUserID(userID)
	}
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return ctx.JSON(fiber.Map{"data": list})
}
