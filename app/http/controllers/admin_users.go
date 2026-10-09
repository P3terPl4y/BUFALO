package controllers

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"strconv"
	"strings"
	"time"
)

func (c *AdminController) UsersIndex(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	filters := map[string]string{
		"role":   ctx.Query("role"),
		"search": ctx.Query("search"),
		"status": ctx.Query("status"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))
	page, perPage = services.NormalizePagination(page, perPage)

	users, total, err := c.userService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando usuarios: %v", err)
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

	return ctx.Render("admin/users/index", fiber.Map{
		"title":      "Usuarios",
		"users":      users,
		"total":      total,
		"page":       page,
		"perPage":    perPage,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"filters":    filters,
		"role":       sess.Get("role"),
		"csrfToken":  csrf.TokenFromContext(ctx),
	}, "layouts/base")
}

func (c *AdminController) UsersCreate(ctx fiber.Ctx) error {
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

func (c *AdminController) UsersStore(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	adminID, _ := c.getCurrentAdminID(ctx)

	var req requests.UserRegisterRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/admin/users/create?flash_error=Datos inválidos")
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	renderErr := func(msg string, errs map[string]string) error {
		empresasCarrier, _ := c.userService.GetEmpresasByTipo("carrier")
		empresasBroker, _ := c.userService.GetEmpresasByTipo("broker")
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

	if req.Role != "publicador" && req.Role != "chofer" {
		return renderErr("Solo se pueden crear usuarios con rol publicador o chofer", nil)
	}
	if req.EmpresaMode != "none" && req.EmpresaMode != "existing" && req.EmpresaMode != "new" {
		return renderErr("Selecciona cómo asociar la empresa o elige sin empresa", map[string]string{"empresa_mode": "Opción de empresa inválida"})
	}

	rules := map[string]any{
		"name":     "required|min:3|max:100",
		"email":    "required|email",
		"password": "required|min:8",
		"role":     "required|in:publicador,chofer",
		"city":     "required|max:100",
		"state":    "required|max:100",
		"country":  "required|max:100",
		"radius":   "required|integer|min:1",
	}
	if req.Role == "chofer" {
		rules["chofer_numero_licencia"] = "required|min:3|max:50"
		rules["chofer_tipo_licencia"] = "required|max:20"
		rules["chofer_anios_experiencia"] = "required|integer|min:0"
	} else {
		rules["publicador_numero_licencia_broker"] = "required|min:3|max:50"
		rules["publicador_anios_experiencia"] = "required|integer|min:0"
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
		return renderErr("Revisa los campos marcados en rojo", errs)
	}
	if req.EmpresaMode == "new" && strings.TrimSpace(req.EmpresaNombreLegal) == "" {
		return renderErr("Completa los datos de la empresa", map[string]string{"empresa_nombre_legal": "El nombre legal es obligatorio"})
	}

	// Email único
	exists, err := c.userService.EmailExists(req.Email, 0)
	if err != nil {
		log.Printf("Error comprobando correo al crear usuario: %v", err)
		return renderErr("No se pudo validar el correo en este momento", nil)
	}
	if exists {
		return renderErr("El email ya está registrado", nil)
	}

	hashed, err := facades.Hash().Make(req.Password)
	if err != nil {
		return renderErr("Error al procesar la contraseña", nil)
	}

	// Empresa
	tipoEmpresa := "broker"
	if req.Role == "chofer" {
		tipoEmpresa = "carrier"
	}

	var empresa *models.Empresa
	switch req.EmpresaMode {
	case "existing":
		if req.EmpresaID == 0 {
			return renderErr("Selecciona una empresa válida", map[string]string{"empresa_id": "Selecciona una empresa"})
		}
		var found models.Empresa
		if err := facades.Orm().Query().Where("id = ?", req.EmpresaID).Where("tipo = ?", tipoEmpresa).Where("estado = ?", "activo").First(&found); err != nil || found.ID == 0 {
			return renderErr("La empresa seleccionada no existe, no está activa o no corresponde al rol", map[string]string{"empresa_id": "Selecciona una empresa activa del tipo correspondiente"})
		}
		empresa = &found
	case "new":
		empresa = &models.Empresa{
			Tipo: models.TipoEmpresa(tipoEmpresa), NombreLegal: strings.TrimSpace(req.EmpresaNombreLegal),
			NombreComercial: optionalString(req.EmpresaNombreComercial), TaxID: optionalString(req.EmpresaTaxID),
			MCNumber: optionalString(req.EmpresaMCNumber), DOTNumber: optionalString(req.EmpresaDOTNumber),
			Telefono: optionalString(req.EmpresaTelefono), Email: optionalString(req.EmpresaEmail),
			SitioWeb: optionalString(req.EmpresaSitioWeb), Estado: models.EmpresaActiva,
		}
	}

	parseTime := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02"} {
			if t, err := time.Parse(layout, s); err == nil {
				return &t
			}
		}
		return nil
	}

	user := models.User{
		Name:           req.Name,
		Email:          req.Email,
		Password:       hashed,
		Role:           req.Role,
		IsActive:       true,
		Address:        req.Address,
		City:           req.City,
		State:          req.State,
		Country:        req.Country,
		PostalCode:     req.PostalCode,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		Radius:         req.Radius,
		Phone:          strPtr(req.Phone),
		PhoneAlt:       optionalString(req.PhoneAlt),
		WhatsApp:       strPtr(req.WhatsApp),
		Telegram:       optionalString(req.Telegram),
		EmergencyName:  optionalString(req.EmergencyName),
		EmergencyPhone: optionalString(req.EmergencyPhone),
		AvailableFrom:  parseTime(req.AvailableFrom),
		AvailableTo:    parseTime(req.AvailableTo),
		Notes:          req.Notes,
	}
	if adminID > 0 {
		user.CreatedBy = &adminID
	}

	var chofer *models.Chofer
	var publicador *models.Publicador
	if req.Role == "chofer" {
		chofer = &models.Chofer{
			NumeroLicencia:        req.ChoferNumeroLicencia,
			TipoLicencia:          req.ChoferTipoLicencia,
			PaisEmisionLicencia:   defaultString(req.ChoferPaisEmisionLicencia, req.Country),
			AniosExperiencia:      req.ChoferAniosExperiencia,
			TiposEquipoPermitidos: req.ChoferTiposEquipoPermitidos,
			Certificaciones:       req.ChoferCertificaciones,
			NumeroSeguro:          req.ChoferNumeroSeguro,
			Estado:                models.ChoferDisponible,
		}
		if t := parseTime(req.ChoferFechaVencimientoLicencia); t != nil {
			chofer.FechaVencimientoLicencia = t
		}
		if t := parseTime(req.ChoferFechaVencimientoSeguro); t != nil {
			chofer.FechaVencimientoSeguro = t
		}
	} else {
		publicador = &models.Publicador{
			NumeroLicenciaBroker: req.PublicadorNumeroLicenciaBroker,
			PaisEmisionLicencia:  defaultString(req.PublicadorPaisEmisionLicencia, req.Country),
			AniosExperiencia:     req.PublicadorAniosExperiencia,
			Especialidad:         req.PublicadorEspecialidad,
			Comision:             req.PublicadorComision,
			CreditScore:          req.PublicadorCreditScore,
			Estado:               models.PublicadorActivo,
		}
		if t := parseTime(req.PublicadorFechaVencimientoLicencia); t != nil {
			publicador.FechaVencimientoLicencia = t
		}
	}

	if err := c.userService.CreateWithRole(&user, empresa, chofer, publicador); err != nil {
		log.Printf("Error al crear usuario: %v", err)
		return renderErr("Error al guardar el usuario", nil)
	}

	return ctx.Redirect().To("/admin/users?flash_success=Usuario creado correctamente")
}

func (c *AdminController) UsersEdit(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=ID inválido")
	}

	user, err := c.userService.GetByID(uint(id))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Usuario no encontrado")
	}

	empresasCarrier, _ := c.userService.GetEmpresasByTipo("carrier")
	empresasBroker, _ := c.userService.GetEmpresasByTipo("broker")

	return ctx.Render("admin/users/edit", fiber.Map{
		"title":           "Editar Usuario",
		"user":            user,
		"csrfToken":       csrf.TokenFromContext(ctx),
		"role":            sess.Get("role"),
		"empresasCarrier": empresasCarrier,
		"empresasBroker":  empresasBroker,
	}, "layouts/base")
}

func (c *AdminController) UsersUpdate(ctx fiber.Ctx) error {
	//sess := session.FromContext(ctx)
	adminID, _ := c.getCurrentAdminID(ctx)

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=ID inválido")
	}

	user, err := c.userService.GetByID(uint(id))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Usuario no encontrado")
	}
	if user.Role == "admin" {
		return ctx.Redirect().To("/admin/users?flash_error=No se puede editar a un administrador")
	}

	var req requests.AdminUpdateRequestByUser
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To(fmt.Sprintf("/admin/users/%d/edit?flash_error=Datos inválidos", id))
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	validator, validationErr := facades.Validation().Make(ctx.Context(), req, map[string]any{"name": "nullable|min:3|max:100", "email": "nullable|email|max:100"})
	if validationErr != nil || validator == nil || validator.Fails() || (req.Password != "" && (len(req.Password) < 8 || len(req.Password) > 72)) {
		return fiber.ErrBadRequest
	}

	updates := map[string]interface{}{"updated_by": adminID}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" && req.Email != user.Email {
		exists, lookupErr := c.userService.EmailExists(req.Email, user.ID)
		if lookupErr != nil {
			return fiber.ErrServiceUnavailable
		}
		if exists {
			return ctx.Redirect().To(fmt.Sprintf("/admin/users/%d/edit?flash_error=El email ya está registrado", id))
		}
		updates["email"] = req.Email
	}
	if req.Password != "" {
		tmp := *user
		if err := tmp.SetPassword(req.Password); err != nil {
			return fiber.ErrBadRequest
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

	if err := c.userService.Update(user.ID, updates); err != nil {
		log.Printf("Error al actualizar usuario: %v", err)
		return ctx.Redirect().To(fmt.Sprintf("/admin/users/%d/edit?flash_error=Error al actualizar", id))
	}
	return ctx.Redirect().To("/admin/users?flash_success=Usuario actualizado correctamente")
}

func (c *AdminController) UsersDelete(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=ID inválido")
	}

	user, err := c.userService.GetByID(uint(id))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Usuario no encontrado")
	}
	if user.Role == "admin" {
		return ctx.Redirect().To("/admin/users?flash_error=No se puede eliminar a un administrador")
	}

	if err := c.userService.Delete(user.ID); err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/users?flash_success=Usuario eliminado correctamente")
}

func (c *AdminController) UsersToggleActive(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=ID inválido")
	}

	user, err := c.userService.GetByID(uint(id))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Usuario no encontrado")
	}
	if user.Role == "admin" {
		return ctx.Redirect().To("/admin/users?flash_error=No se puede modificar a un administrador")
	}

	if err := c.userService.ToggleActive(user.ID, !user.IsActive); err != nil {
		return ctx.Redirect().To("/admin/users?flash_error=Error al cambiar el estado")
	}

	status := "activado"
	if user.IsActive {
		status = "desactivado"
	}
	return ctx.Redirect().To(fmt.Sprintf("/admin/users?flash_success=Usuario %s correctamente", status))
}

// ═══════════════════════════════════════════════════════════════
// EMPRESAS
// ═══════════════════════════════════════════════════════════════
