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

type DireccionController struct {
	service *services.DireccionService
}

func NewDireccionController() *DireccionController {
	return &DireccionController{service: services.NewDireccionService()}
}

func (c *DireccionController) Index(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	list, err := c.service.GetAll()
	if err != nil {
		log.Printf("Error listando direcciones: %v", err)
	}
	return ctx.Render("direcciones/index", fiber.Map{
		"title":      "Direcciones",
		"direcciones": list,
		"csrfToken":   csrf.TokenFromContext(ctx),
		"role":       sess.Get("role"),
	}, "layouts/base")
}

func (c *DireccionController) Show(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	d, err := c.service.GetByID(ctx.Params("id"))
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
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
	log.Printf("[Direccion.Store] payload: ciudad=%q provincia=%q pais=%q lat=%q lng=%q",
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
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
	}
	if sess.Get("user_id").(uint)==*d.OwnerID{
	return ctx.Render("direcciones/edit", fiber.Map{
		"title":     "Editar Dirección",
		"direccion": d,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      sess.Get("role"),
	}, "layouts/base")}else{
	return ctx.Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
	}
}

func (c *DireccionController) Update(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	id := ctx.Params("id")
	d, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
	}
	if sess.Get("user_id").(uint)==*d.OwnerID{
	var req requests.DireccionUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/direcciones/" + id + "/edit?flash_error=Datos inválidos")
	}

	
log.Println(req)
	updates := map[string]interface{}{
		"calle":            strPtr(req.Calle),
		"ciudad":           req.Ciudad,
		"estado_provincia": req.EstadoProvincia,
		"codigo_postal":    strPtr(req.CodigoPostal),
		"pais":             req.Pais,
		"latitud":          req.Latitud,
		"longitud":        req.Longitud,
	}
	if err := c.service.Update(id, updates); err != nil {
		return ctx.Redirect().To("/direcciones/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To("/direcciones/" + id + "?flash_success=Actualizada")
	}else{
	return ctx.Redirect().To("/direcciones/" + id + "/edit?flash_error=Error al actualizar")
	}
}

func (c *DireccionController) Delete(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	id := ctx.Params("id")
	d, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
	}
	if sess.Get("user_id").(uint)==*d.OwnerID{
	if err := c.service.Delete(id); err != nil {
		return ctx.Redirect().To("/direcciones?flash_error=Error al eliminar")
	}}
	return ctx.Redirect().To("/direcciones?flash_success=Eliminada")
}

// Helper expuesto para selects en formularios
func (c *DireccionController) All(ctx fiber.Ctx) error {
	list, _ := c.service.GetAll()
	return ctx.JSON(fiber.Map{"data": list})
}

var _ = strconv.Itoa // evitar import no usado si no se usa strconv
