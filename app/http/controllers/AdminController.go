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

type AdminController struct {
	userService *services.UserService
}

func NewAdminController() *AdminController {
	return &AdminController{
		userService: services.NewUserService(),
	}
}

// helper: obtiene el ID del admin autenticado
func (c *AdminController) getCurrentAdminID(ctx fiber.Ctx) (uint, bool) {
	id, ok := ctx.Locals("user_id").(uint)
	return id, ok
}

// Index - lista todos los usuarios con filtros y paginación
func (c *AdminController) Index(ctx fiber.Ctx) error {
	filters := map[string]string{
		"role":   ctx.Query("role"),
		"search": ctx.Query("search"),
		"status": ctx.Query("status"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))

	users, total, err := c.userService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error al listar usuarios: %v", err)
		return ctx.Render("admin/users/index", fiber.Map{
			"title":       "Gestión de Usuarios",
			"flash_error": "Error al cargar los usuarios",
			"role":        ctx.Locals("role"),
		}, "layouts/base")
	}

	// ... después de obtener `users, total, err`
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}
	nextPage := page + 1
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if nextPage > totalPages {
		nextPage = totalPages
	}

	return ctx.Render("admin/users/index", fiber.Map{
		"title":      "Gestión de Usuarios",
		"users":      users,
		"total":      total,
		"page":       page,
		"perPage":    perPage,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"filters":    filters,
		"role":       ctx.Locals("role"),
	}, "layouts/base")
}

// Create - muestra el formulario de creación de usuario
func (c *AdminController) Create(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	empresasCarrier, _ := c.userService.GetEmpresasByTipo("carrier")
	empresasBroker, _ := c.userService.GetEmpresasByTipo("broker")

	return ctx.Render("admin/users/create", fiber.Map{
		"title":           "Crear Usuario",
		"csrfToken":       csrf.TokenFromContext(ctx),
		"role":            sess.Get("role"),
		"empresasCarrier": empresasCarrier,
		"empresasBroker":  empresasBroker,
	}, "layouts/base")
}
func (c *AdminController) Store(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	adminID, _ := c.getCurrentAdminID(ctx)

	// ── 1. Bind ──
	var req requests.UserRegisterRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Render("admin/users/create", fiber.Map{
			"title":       "Crear Usuario",
			"flash_error": "Datos inválidos",
			"csrfToken":   csrf.TokenFromContext(ctx),
			"role":        sess.Get("role"),
		}, "layouts/base")
	}

	// Helper para re-render con datos y errores
	renderErr := func(msg string, errs map[string]string) error {
		userSvc := services.NewUserService()
		empresasCarrier, _ := userSvc.GetEmpresasByTipo("carrier")
		empresasBroker, _ := userSvc.GetEmpresasByTipo("broker")
		return ctx.Render("admin/users/create", fiber.Map{
			"title":           "Crear Usuario",
			"flash_error":     msg,
			"errors":          errs,
			"old":             req,
			"csrfToken":       csrf.TokenFromContext(ctx),
			"role":            sess.Get("role"),
			"empresasCarrier": empresasCarrier,
			"empresasBroker":  empresasBroker,
		}, "layouts/base")
	}

	// ── 2. Rol ──
	if req.Role != "broker" && req.Role != "carrier" {
		return renderErr("Solo se pueden crear usuarios con rol broker o carrier", nil)
	}

	// ── 3. Validación: básicos + ubicación + preferencias + disponibilidad ──
	rules := map[string]any{
		"name":                      "required|min:3|max:100",
		"email":                     "required|email",
		"password":                  "required|min:8",
		"role":                      "required|in:broker,carrier",
		"address":                   "required|max:500",
		"city":                      "required|max:100",
		"state":                     "required|max:100",
		"country":                   "required|max:100",
		"postal_code":               "required|max:20",
		"radius":                    "required|integer|min:1",
		"preferred_equipment_types": "required|max:255",
		"preferred_cargo_types":     "required|max:255",
		"max_weight":                "required|numeric|min:0.01",
		"max_distance":              "required|numeric|min:0.01",
		"available_from":            "required",
		"available_to":              "required",
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
		return renderErr("Error de validación", errs)
	}

	// ── 4. Parsear disponibilidad ──
	parseTime := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
			return &t
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return &t
		}
		return nil
	}
	availFrom := parseTime(req.AvailableFrom)
	availTo := parseTime(req.AvailableTo)

	// ── 5. Email único ──
	userSvc := services.NewUserService()
	exists, err := userSvc.EmailExists(req.Email, 0)
	if err != nil {
		log.Printf("Error al comprobar email: %v", err)
		return renderErr("Error al validar el email", nil)
	}
	if exists {
		return renderErr("El email ya está registrado", nil)
	}

	// ── 6. Construir User ──
	user := models.User{
		Name:                    req.Name,
		Email:                   req.Email,
		Role:                    req.Role,
		IsActive:                true,
		Address:                 req.Address,
		City:                    req.City,
		State:                   req.State,
		Country:                 req.Country,
		PostalCode:              req.PostalCode,
		Latitude:                req.Latitude,
		Longitude:               req.Longitude,
		Radius:                  req.Radius,
		PreferredEquipmentTypes: req.PreferredEquipmentTypes,
		PreferredCargoTypes:     req.PreferredCargoTypes,
		MaxWeight:               req.MaxWeight,
		MaxDistance:             req.MaxDistance,
		PreferredRoutes:         req.PreferredRoutes,
		AvailableFrom:           availFrom,
		AvailableTo:             availTo,
		Notes:                   req.Notes,
	}
	if adminID > 0 {
		user.CreatedBy = &adminID
	}
	if err := user.SetPassword(req.Password); err != nil {
		return renderErr("Error al procesar la contraseña", nil)
	}

	// ── 7. Resolver empresa según modo ──
	var empresa *models.Empresa
	switch req.EmpresaMode {
	case "existing":
    if req.EmpresaID == 0 {
        return renderErr("Debes seleccionar una empresa", nil)
    }
    var found models.Empresa
    err := facades.Orm().Query().
        Where("id = ?", req.EmpresaID).
        Where("tipo = ?", req.Role).
        Where("estado = ?", "activo").
        First(&found)
		if err != nil || found.ID == 0 {
			return renderErr("La empresa seleccionada no es válida para el rol", nil)
		}
		empresa = &found

	case "new":
		if req.EmpresaNombreLegal == "" {
			return renderErr("El nombre legal de la empresa es obligatorio", map[string]string{
				"empresa_nombre_legal": "Requerido",
			})
		}
		empresa = &models.Empresa{
			Tipo:            models.TipoEmpresa(req.Role),
			NombreLegal:     req.EmpresaNombreLegal,
			NombreComercial: strOrNil(req.EmpresaNombreComercial),
			TaxID:           strOrNil(req.EmpresaTaxID),
			MCNumber:        strOrNil(req.EmpresaMCNumber),
			DOTNumber:       strOrNil(req.EmpresaDOTNumber),
			Telefono:        strOrNil(req.EmpresaTelefono),
			Email:           strOrNil(req.EmpresaEmail),
			SitioWeb:        strOrNil(req.EmpresaSitioWeb),
			Estado:          models.EmpresaActiva,
		}

	case "none", "":
		empresa = nil
	}

	// ── 8. Persistir en transacción (rollback si algo falla) ──
	if err := userSvc.CreateWithEmpresa(&user, empresa); err != nil {
		log.Printf("Error al crear usuario (rollback aplicado): %v", err)
		return renderErr("Error al guardar el usuario. Intenta de nuevo.", nil)
	}

	return ctx.Redirect().To("/admin/users?flash_success=Usuario creado correctamente")
}

// Edit - muestra el formulario de edición
func (c *AdminController) Edit(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=ID inválido")
	}

	user, err := c.userService.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Usuario no encontrado")
	}

	return ctx.Render("admin/users/edit", fiber.Map{
		"title":     "Editar Usuario",
		"user":      user,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      sess.Get("role"),
	}, "layouts/base")
}

func (c *AdminController) Update(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	adminID, ok := c.getCurrentAdminID(ctx)
	if !ok {
		return ctx.Redirect().To("/login")
	}

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=ID inválido")
	}

	user, err := c.userService.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Usuario no encontrado")
	}
	if user.Role == "admin" {
		return ctx.Redirect().To("/admin/users?flash_error=No se puede editar a un administrador")
	}

	// ── 1. Bind ──
	var req requests.AdminUpdateRequestByUser
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Render("admin/users/edit", fiber.Map{
			"title":       "Editar Usuario",
			"flash_error": "Datos inválidos",
			"user":        user,
			"csrfToken":   csrf.TokenFromContext(ctx),
			"role":        sess.Get("role"),
		}, "layouts/base")
	}

	// ── 2. Validación (todo nullable en edición) ──
	rules := map[string]any{
		"name":                      "nullable|min:3|max:100",
		"email":                     "nullable|email",
		"password":                  "nullable|min:8",
		"role":                      "nullable|in:broker,carrier",
		"address":                   "nullable|max:500",
		"city":                      "nullable|max:100",
		"state":                     "nullable|max:100",
		"country":                   "nullable|max:100",
		"postal_code":               "nullable|max:20",
		"radius":                    "nullable|integer|min:0",
		"preferred_equipment_types": "nullable|max:255",
		"preferred_cargo_types":     "nullable|max:255",
		"max_weight":                "nullable|numeric|min:0",
		"max_distance":              "nullable|numeric|min:0",
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
		return ctx.Render("admin/users/edit", fiber.Map{
			"title":       "Editar Usuario",
			"flash_error": "Error de validación",
			"errors":      errs,
			"user":        user,
			"old":         req,
			"csrfToken":   csrf.TokenFromContext(ctx),
			"role":        sess.Get("role"),
		}, "layouts/base")
	}

	// ── 3. Construir updates ──
	updates := map[string]interface{}{
		"updated_by": adminID,
	}

	if req.Name != "" {
		updates["name"] = req.Name
	}

	if req.Email != "" && req.Email != user.Email {
		exists, err := c.userService.EmailExists(req.Email, user.ID)
		if err != nil {
			return ctx.Redirect().To("/admin/users?flash_error=Error al validar email")
		}
		if exists {
			return ctx.Redirect().To("/admin/users?flash_error=El email ya está registrado")
		}
		updates["email"] = req.Email
	}

	if req.Role != "" && (req.Role == "broker" || req.Role == "carrier") {
		updates["role"] = req.Role
	}

	if req.Password != "" {
		tmp := *user
		if err := tmp.SetPassword(req.Password); err != nil {
			return ctx.Redirect().To("/admin/users?flash_error=Error al procesar la contraseña")
		}
		updates["password"] = tmp.Password
	}

	// Ubicación
	if req.Address != "" {
		updates["address"] = req.Address
	}
	if req.City != "" {
		updates["city"] = req.City
	}
	if req.State != "" {
		updates["state"] = req.State
	}
	if req.Country != "" {
		updates["country"] = req.Country
	}
	if req.PostalCode != "" {
		updates["postal_code"] = req.PostalCode
	}
	if req.Latitude != 0 {
		updates["latitude"] = req.Latitude
	}
	if req.Longitude != 0 {
		updates["longitude"] = req.Longitude
	}
	if req.Radius != 0 {
		updates["radius"] = req.Radius
	}

	// Preferencias
	if req.PreferredEquipmentTypes != "" {
		updates["preferred_equipment_types"] = req.PreferredEquipmentTypes
	}
	if req.PreferredCargoTypes != "" {
		updates["preferred_cargo_types"] = req.PreferredCargoTypes
	}
	if req.MaxWeight != 0 {
		updates["max_weight"] = req.MaxWeight
	}
	if req.MaxDistance != 0 {
		updates["max_distance"] = req.MaxDistance
	}
	if req.PreferredRoutes != "" {
		updates["preferred_routes"] = req.PreferredRoutes
	}

	// Disponibilidad
	if req.AvailableFrom != "" {
		if t, err := time.Parse("2006-01-02T15:04", req.AvailableFrom); err == nil {
			updates["available_from"] = t
		} else if t, err := time.Parse(time.RFC3339, req.AvailableFrom); err == nil {
			updates["available_from"] = t
		}
	}
	if req.AvailableTo != "" {
		if t, err := time.Parse("2006-01-02T15:04", req.AvailableTo); err == nil {
			updates["available_to"] = t
		} else if t, err := time.Parse(time.RFC3339, req.AvailableTo); err == nil {
			updates["available_to"] = t
		}
	}
	if req.Notes != "" {
		updates["notes"] = req.Notes
	}

	// ── 4. Persistir ──
	if err := c.userService.Update(user.ID, updates); err != nil {
		log.Printf("Error al actualizar usuario: %v", err)
		return ctx.Redirect().To("/admin/users?flash_error=Error al actualizar")
	}

	return ctx.Redirect().To("/admin/users?flash_success=Usuario actualizado correctamente")
}

// Delete - elimina un usuario (solo broker o carrier, nunca admin)
func (c *AdminController) Delete(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=ID inválido")
	}

	user, err := c.userService.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Usuario no encontrado")
	}

	if user.Role == "admin" {
		return ctx.Redirect().To("/admin/users?flash_error=No se puede eliminar a un administrador")
	}

	if err := c.userService.Delete(user.ID); err != nil {
		log.Printf("Error al eliminar usuario: %v", err)
		return ctx.Redirect().To("/admin/users?flash_error=Error al eliminar el usuario")
	}

	return ctx.Redirect().To("/admin/users?flash_success=Usuario eliminado correctamente")
}

// ToggleActive - activa/desactiva un usuario
func (c *AdminController) ToggleActive(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=ID inválido")
	}

	user, err := c.userService.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Usuario no encontrado")
	}

	if user.Role == "admin" {
		return ctx.Redirect().To("/admin/users?flash_error=No se puede modificar a un administrador")
	}

	if err := c.userService.ToggleActive(user.ID, !user.IsActive); err != nil {
		log.Printf("Error al cambiar estado: %v", err)
		return ctx.Redirect().To("/admin/users?flash_error=Error al cambiar el estado")
	}

	status := "activado"
	if user.IsActive {
		status = "desactivado"
	}
	return ctx.Redirect().To(fmt.Sprintf("/admin/users?flash_success=Usuario %s correctamente", status))
}
