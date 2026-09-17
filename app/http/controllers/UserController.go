package controllers

import (
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
)

type UserController struct {
	userService *services.UserService
}

func NewUserController() *UserController {
	return &UserController{
		userService: services.NewUserService(),
	}
}

// getCurrentUserID extrae el user_id de Locals (lo pone SessionAuth)
func (c *UserController) getCurrentUserID(ctx fiber.Ctx) (uint, bool) {
	id, ok := ctx.Locals("user_id").(uint)
	return id, ok
}

// Show - muestra el perfil del usuario autenticado
func (c *UserController) Show(ctx fiber.Ctx) error {
	userID, ok := c.getCurrentUserID(ctx)
	if !ok {
		return ctx.Redirect().To("/login")
	}

	user, err := c.userService.GetByID(userID)
	if err != nil {
		log.Printf("Error al obtener perfil: %v", err)
		return ctx.Render("profile/show", fiber.Map{
			"title":       "Mi Perfil",
			"flash_error": "Usuario no encontrado",
			"role":        ctx.Locals("role"),
		}, "layouts/base")
	}

	return ctx.Render("profile/show", fiber.Map{
		"title": "Mi Perfil",
		"user":  user,
		"role":  ctx.Locals("role"),
	}, "layouts/base")
}

// Edit - muestra el formulario de edición
func (c *UserController) Edit(ctx fiber.Ctx) error {
	userID, ok := c.getCurrentUserID(ctx)
	if !ok {
		return ctx.Redirect().To("/login")
	}

	user, err := c.userService.GetByID(userID)
	if err != nil {
		log.Printf("Error al obtener perfil: %v", err)
		return ctx.Render("profile/edit", fiber.Map{
			"title":       "Editar Perfil",
			"flash_error": "Usuario no encontrado",
			"role":        ctx.Locals("role"),
		}, "layouts/base")
	}

	return ctx.Render("profile/edit", fiber.Map{
		"title":     "Editar Perfil",
		"user":      user,
		"role":      ctx.Locals("role"),
		"csrfToken": csrf.TokenFromContext(ctx),
	}, "layouts/base")
}

// Update - procesa la actualización del perfil
func (c *UserController) Update(ctx fiber.Ctx) error {
	userID, ok := c.getCurrentUserID(ctx)
	if !ok {
		return ctx.Redirect().To("/login")
	}

	user, err := c.userService.GetByID(userID)
	if err != nil {
		return ctx.Redirect().To("/login")
	}

	// 1) Bind al request
	var req requests.UserUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Render("profile/edit", fiber.Map{
			"title":       "Editar Perfil",
			"flash_error": "Datos inválidos",
			"user":        user,
			"role":        ctx.Locals("role"),
			"csrfToken":   csrf.TokenFromContext(ctx),
		}, "layouts/base")
	}

	// 2) Validación
	rules := map[string]any{
		"name":         "nullable|min:3|max:100",
		"email":        "nullable|email",
		"password":     "nullable|min:8",
		"city":         "nullable|max:100",
		"state":        "nullable|max:100",
		"country":      "nullable|max:100",
		"postal_code":  "nullable|max:20",
		"radius":       "nullable|integer|min:0",
		"max_weight":   "nullable|numeric|min:0",
		"max_distance": "nullable|numeric|min:0",
	}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil {
		log.Printf("❌ Validation Make error: %v", err)
		return ctx.Render("profile/edit", fiber.Map{
			"title":       "Editar Perfil",
			"flash_error": "Error al validar los datos",
			"user":        user,
			"old":         req,
			"role":        ctx.Locals("role"),
			"csrfToken":   csrf.TokenFromContext(ctx),
		}, "layouts/base")
	}
	if validator.Fails() {
		errs := make(map[string]string)
		for field, fieldErrors := range validator.Errors().All() {
			for _, msg := range fieldErrors {
				errs[field] = msg
				break
			}
		}
		return ctx.Render("profile/edit", fiber.Map{
			"title":       "Editar Perfil",
			"flash_error": "Error de validación",
			"errors":      errs,
			"user":        user,
			"old":         req,
			"role":        ctx.Locals("role"),
			"csrfToken":   csrf.TokenFromContext(ctx),
		}, "layouts/base")
	}

	// 3) Construir updates
	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" && req.Email != user.Email {
		count, _ := facades.Orm().Query().
			Model(&models.User{}).
			Where("email = ?", req.Email).
			Where("id <> ?", user.ID).
			Count()
		if count > 0 {
			return ctx.Render("profile/edit", fiber.Map{
				"title":       "Editar Perfil",
				"flash_error": "El email ya está registrado",
				"user":        user,
				"old":         req,
				"role":        ctx.Locals("role"),
				"csrfToken":   csrf.TokenFromContext(ctx),
			}, "layouts/base")
		}
		updates["email"] = req.Email
	}
	if req.Password != "" {
		hashed, err := facades.Hash().Make(req.Password)
		if err != nil {
			return ctx.Render("profile/edit", fiber.Map{
				"title":       "Editar Perfil",
				"flash_error": "Error al procesar la contraseña",
				"user":        user,
				"role":        ctx.Locals("role"),
				"csrfToken":   csrf.TokenFromContext(ctx),
			}, "layouts/base")
		}
		updates["password"] = hashed
	}

	// Ubicación
	updates["address"]     = req.Address
	updates["city"]        = req.City
	updates["state"]       = req.State
	updates["country"]     = req.Country
	updates["postal_code"] = req.PostalCode
	updates["latitude"]    = req.Latitude
	updates["longitude"]   = req.Longitude
	updates["radius"]      = req.Radius

	// Preferencias
	updates["preferred_equipment_types"] = req.PreferredEquipmentTypes
	updates["preferred_cargo_types"]     = req.PreferredCargoTypes
	updates["max_weight"]                = req.MaxWeight
	updates["max_distance"]              = req.MaxDistance
	updates["preferred_routes"]          = req.PreferredRoutes

	// Disponibilidad
	if req.AvailableFrom != "" {
		if t, err := time.Parse("2006-01-02T15:04", req.AvailableFrom); err == nil {
			updates["available_from"] = t
		}
	}
	if req.AvailableTo != "" {
		if t, err := time.Parse("2006-01-02T15:04", req.AvailableTo); err == nil {
			updates["available_to"] = t
		}
	}
	updates["notes"] = req.Notes

	// 4) Persistir
	if err := c.userService.Update(userID, updates); err != nil {
		log.Printf("Error al actualizar perfil: %v", err)
		return ctx.Render("profile/edit", fiber.Map{
			"title":       "Editar Perfil",
			"flash_error": "Error al actualizar el perfil",
			"user":        user,
			"old":         req,
			"role":        ctx.Locals("role"),
			"csrfToken":   csrf.TokenFromContext(ctx),
		}, "layouts/base")
	}

	return ctx.Redirect().To("/profile?flash_success=Perfil actualizado correctamente")
}
