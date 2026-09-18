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
	userService       *services.UserService
	empresaService    *services.EmpresaService
	choferService     *services.ChoferService
	publicadorService *services.PublicadorService
	cargaService      *services.CargaService
	direccionService  *services.DireccionService
	facturaService    *services.FacturaService
}

func NewAdminController() *AdminController {
	return &AdminController{
		userService:       services.NewUserService(),
		empresaService:    services.NewEmpresaService(),
		choferService:     services.NewChoferService(),
		publicadorService: services.NewPublicadorService(),
		cargaService:      services.NewCargaService(),
		direccionService:  services.NewDireccionService(),
		facturaService:    services.NewFacturaService(),
	}
}

func (c *AdminController) getCurrentAdminID(ctx fiber.Ctx) (uint, bool) {
	id, ok := ctx.Locals("user_id").(uint)
	return id, ok
}

// ═══════════════════════════════════════════════════════════════
// DASHBOARD
// ═══════════════════════════════════════════════════════════════

func (c *AdminController) Dashboard(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)

	// Conteos globales
	totalUsers, _ := facades.Orm().Query().Model(&models.User{}).Count()
	totalEmpresas, _ := facades.Orm().Query().Model(&models.Empresa{}).Count()
	totalChoferes, _ := facades.Orm().Query().Model(&models.Chofer{}).Count()
	totalPublicadores, _ := facades.Orm().Query().Model(&models.Publicador{}).Count()
	totalDirecciones, _ := facades.Orm().Query().Model(&models.Direccion{}).Count()

	totalCargas, _ := facades.Orm().Query().Model(&models.Carga{}).Count()
	cargasPublicadas, _ := facades.Orm().Query().Model(&models.Carga{}).
		Where("estado = ?", "publicada").Count()
	cargasAsignadas, _ := facades.Orm().Query().Model(&models.Carga{}).
		Where("estado = ?", "asignada").Count()
	cargasEntregadas, _ := facades.Orm().Query().Model(&models.Carga{}).
		Where("estado = ?", "entregada").Count()
	cargasCanceladas, _ := facades.Orm().Query().Model(&models.Carga{}).
		Where("estado = ?", "cancelada").Count()

	totalFacturas, _ := facades.Orm().Query().Model(&models.Factura{}).Count()
	facturasPagadas, _ := facades.Orm().Query().Model(&models.Factura{}).
		Where("estado = ?", "pagada").Count()
	facturasPendientes, _ := facades.Orm().Query().Model(&models.Factura{}).
		Where("estado IN ?", []string{"emitida", "vencida"}).Count()

	// Últimos usuarios y cargas
	var ultimosUsers []models.User
	var ultimasCargas []models.Carga

	facades.Orm().Query().Model(&models.User{}).
		Order("created_at desc").Limit(5).Find(&ultimosUsers)
	facades.Orm().Query().Model(&models.Carga{}).
		With("Publicador").With("Empresa").
		Order("created_at desc").Limit(5).Find(&ultimasCargas)

	return ctx.Render("admin/dashboard", fiber.Map{
		"title": "Panel de Administración",
		"role":  sess.Get("role"),

		"totalUsers":        totalUsers,
		"totalEmpresas":     totalEmpresas,
		"totalChoferes":     totalChoferes,
		"totalPublicadores": totalPublicadores,
		"totalDirecciones":  totalDirecciones,

		"totalCargas":      totalCargas,
		"cargasPublicadas": cargasPublicadas,
		"cargasAsignadas":  cargasAsignadas,
		"cargasEntregadas": cargasEntregadas,
		"cargasCanceladas": cargasCanceladas,

		"totalFacturas":      totalFacturas,
		"facturasPagadas":    facturasPagadas,
		"facturasPendientes": facturasPendientes,

		"ultimosUsers":  ultimosUsers,
		"ultimasCargas": ultimasCargas,
	}, "layouts/base")
}

// ═══════════════════════════════════════════════════════════════
// USERS
// ═══════════════════════════════════════════════════════════════

func (c *AdminController) UsersIndex(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	filters := map[string]string{
		"role":   ctx.Query("role"),
		"search": ctx.Query("search"),
		"status": ctx.Query("status"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))

	users, total, err := c.userService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando usuarios: %v", err)
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

	// Email único
	exists, _ := c.userService.EmailExists(req.Email, 0)
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
		if req.EmpresaID > 0 {
			var found models.Empresa
			if err := facades.Orm().Query().
				Where("id = ?", req.EmpresaID).
				Where("tipo = ?", tipoEmpresa).
				First(&found); err == nil && found.ID > 0 {
				empresa = &found
			}
		}
	case "new":
		if req.EmpresaNombreLegal != "" {
			empresa = &models.Empresa{
				Tipo:        models.TipoEmpresa(tipoEmpresa),
				NombreLegal: req.EmpresaNombreLegal,
				Estado:      models.EmpresaActiva,
			}
		}
	}

	parseTime := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
			return &t
		}
		return nil
	}

	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashed,
		Role:     req.Role,
		IsActive: true,
		Address:  req.Address,
		City:     req.City,
		State:    req.State,
		Country:  req.Country,
		PostalCode: req.PostalCode,
		Latitude: req.Latitude,
		Longitude: req.Longitude,
		Radius:   req.Radius,
		Phone:    strPtr(req.Phone),
		WhatsApp: strPtr(req.WhatsApp),
		AvailableFrom: parseTime(req.AvailableFrom),
		AvailableTo:   parseTime(req.AvailableTo),
		Notes:    req.Notes,
	}
	if adminID > 0 {
		user.CreatedBy = &adminID
	}

	var chofer *models.Chofer
	var publicador *models.Publicador
	if req.Role == "chofer" {
		chofer = &models.Chofer{
			NumeroLicencia:      req.ChoferNumeroLicencia,
			TipoLicencia:        req.ChoferTipoLicencia,
			PaisEmisionLicencia: "Cuba",
			AniosExperiencia:    req.ChoferAniosExperiencia,
			Estado:              models.ChoferDisponible,
		}
	} else {
		publicador = &models.Publicador{
			NumeroLicenciaBroker: req.PublicadorNumeroLicenciaBroker,
			PaisEmisionLicencia:  "Cuba",
			AniosExperiencia:     req.PublicadorAniosExperiencia,
			Estado:               models.PublicadorActivo,
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

	updates := map[string]interface{}{"updated_by": adminID}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" && req.Email != user.Email {
		exists, _ := c.userService.EmailExists(req.Email, user.ID)
		if exists {
			return ctx.Redirect().To(fmt.Sprintf("/admin/users/%d/edit?flash_error=El email ya está registrado", id))
		}
		updates["email"] = req.Email
	}
	if req.Password != "" {
		tmp := *user
		if err := tmp.SetPassword(req.Password); err == nil {
			updates["password"] = tmp.Password
		}
	}
	// Ubicación
	if req.Address != "" { updates["address"] = req.Address }
	if req.City != "" { updates["city"] = req.City }
	if req.State != "" { updates["state"] = req.State }
	if req.Country != "" { updates["country"] = req.Country }
	if req.PostalCode != "" { updates["postal_code"] = req.PostalCode }
	if req.Latitude != 0 { updates["latitude"] = req.Latitude }
	if req.Longitude != 0 { updates["longitude"] = req.Longitude }
	if req.Radius != 0 { updates["radius"] = req.Radius }

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

func (c *AdminController) EmpresasIndex(ctx fiber.Ctx) error {
	filters := map[string]string{
		"tipo":   ctx.Query("tipo"),
		"estado": ctx.Query("estado"),
		"q":      ctx.Query("q"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))

	list, total, err := c.empresaService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando empresas: %v", err)
	}

	prevPage := page - 1
	if prevPage < 1 { prevPage = 1 }
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	nextPage := page + 1
	if nextPage > totalPages { nextPage = totalPages }

	return ctx.Render("admin/empresas/index", fiber.Map{
		"title":      "Empresas",
		"empresas":   list,
		"total":      total,
		"page":       page,
		"perPage":    perPage,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"filters":    filters,
		"role":       ctx.Locals("role"),
		"csrfToken":  csrf.TokenFromContext(ctx),
	}, "layouts/base")
}

func (c *AdminController) EmpresasEdit(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/empresas?flash_error=ID inválido")
	}
	e, err := c.empresaService.GetByID(strconv.FormatUint(uint64(id), 10))
	if err != nil {
		return ctx.Redirect().To("/admin/empresas?flash_error=Empresa no encontrada")
	}
	dests, _ := c.direccionService.GetAll()

	return ctx.Render("admin/empresas/edit", fiber.Map{
		"title":        "Editar Empresa",
		"empresa":      e,
		"destinations": dests,
		"csrfToken":    csrf.TokenFromContext(ctx),
		"role":         ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) EmpresasUpdate(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	var req requests.EmpresaUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/admin/empresas/" + id + "/edit?flash_error=Datos inválidos")
	}
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
	if err := c.empresaService.Update(id, updates); err != nil {
		return ctx.Redirect().To("/admin/empresas/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To("/admin/empresas?flash_success=Empresa actualizada")
}

func (c *AdminController) EmpresasDelete(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.empresaService.Delete(id); err != nil {
		return ctx.Redirect().To("/admin/empresas?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/empresas?flash_success=Empresa eliminada")
}

// ═══════════════════════════════════════════════════════════════
// CHOFERES
// ═══════════════════════════════════════════════════════════════

func (c *AdminController) ChoferesIndex(ctx fiber.Ctx) error {
	filters := map[string]string{
		"estado":     ctx.Query("estado"),
		"empresa_id": ctx.Query("empresa_id"),
		"q":          ctx.Query("q"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))

	list, total, err := c.choferService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando choferes: %v", err)
	}

	prevPage := page - 1
	if prevPage < 1 { prevPage = 1 }
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	nextPage := page + 1
	if nextPage > totalPages { nextPage = totalPages }

	return ctx.Render("admin/choferes/index", fiber.Map{
		"title":      "Choferes",
		"choferes":   list,
		"total":      total,
		"page":       page,
		"perPage":    perPage,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"filters":    filters,
		"role":       ctx.Locals("role"),
		"csrfToken":  csrf.TokenFromContext(ctx),
	}, "layouts/base")
}

func (c *AdminController) ChoferesEdit(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/choferes?flash_error=ID inválido")
	}
	ch, err := c.choferService.GetByID(uint(id))
	if err != nil {
		return ctx.Redirect().To("/admin/choferes?flash_error=Chofer no encontrado")
	}

	return ctx.Render("admin/choferes/edit", fiber.Map{
		"title":     "Editar Chofer",
		"chofer":    ch,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) ChoferesUpdate(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/choferes?flash_error=ID inválido")
	}
	var req requests.ChoferUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To(fmt.Sprintf("/admin/choferes/%d/edit?flash_error=Datos inválidos", id))
	}

	parseDate := func(s string) *time.Time {
		if s == "" { return nil }
		if t, err := time.Parse("2006-01-02", s); err == nil { return &t }
		return nil
	}

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
	if err := c.choferService.Update(uint(id), updates); err != nil {
		return ctx.Redirect().To(fmt.Sprintf("/admin/choferes/%d/edit?flash_error=Error al actualizar", id))
	}
	return ctx.Redirect().To("/admin/choferes?flash_success=Chofer actualizado")
}

func (c *AdminController) ChoferesDelete(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Redirect().To("/admin/choferes?flash_error=ID inválido")
	}
	if err := c.choferService.Delete(uint(id)); err != nil {
		return ctx.Redirect().To("/admin/choferes?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/choferes?flash_success=Chofer eliminado")
}

// ═══════════════════════════════════════════════════════════════
// PUBLICADORES
// ═══════════════════════════════════════════════════════════════

func (c *AdminController) PublicadoresIndex(ctx fiber.Ctx) error {
	filters := map[string]string{
		"estado":     ctx.Query("estado"),
		"empresa_id": ctx.Query("empresa_id"),
		"q":          ctx.Query("q"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))

	list, total, err := c.publicadorService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando publicadores: %v", err)
	}

	prevPage := page - 1
	if prevPage < 1 { prevPage = 1 }
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	nextPage := page + 1
	if nextPage > totalPages { nextPage = totalPages }

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

	parseDate := func(s string) *time.Time {
		if s == "" { return nil }
		if t, err := time.Parse("2006-01-02", s); err == nil { return &t }
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

func (c *AdminController) CargasIndex(ctx fiber.Ctx) error {
	filters := map[string]string{
		"status":      ctx.Query("status"),
		"tipo_equipo": ctx.Query("tipo_equipo"),
		"tipo_carga":  ctx.Query("tipo_carga"),
		"publicador_id": ctx.Query("publicador_id"),
		"chofer_id":   ctx.Query("chofer_id"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))

	list, total, err := c.cargaService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando cargas: %v", err)
	}

	prevPage := page - 1
	if prevPage < 1 { prevPage = 1 }
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	nextPage := page + 1
	if nextPage > totalPages { nextPage = totalPages }

	return ctx.Render("admin/cargas/index", fiber.Map{
		"title":      "Cargas",
		"loads":      list,
		"total":      total,
		"page":       page,
		"perPage":    perPage,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"filters":    filters,
		"role":       ctx.Locals("role"),
		"csrfToken":  csrf.TokenFromContext(ctx),
	}, "layouts/base")
}

func (c *AdminController) CargasShow(ctx fiber.Ctx) error {
	carga, err := c.cargaService.GetByID(ctx.Params("id"))
	if err != nil {
		return ctx.Redirect().To("/admin/cargas?flash_error=Carga no encontrada")
	}

	hasMap := carga.OrigenDireccion != nil && carga.DestinoDireccion != nil &&
		carga.OrigenDireccion.Latitud != nil && carga.OrigenDireccion.Longitud != nil &&
		carga.DestinoDireccion.Latitud != nil && carga.DestinoDireccion.Longitud != nil

	return ctx.Render("admin/cargas/show", fiber.Map{
		"title":     "Detalle de Carga",
		"load":      carga,
		"hasMap":    hasMap,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) CargasDelete(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.cargaService.Delete(id); err != nil {
		return ctx.Redirect().To("/admin/cargas?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/cargas?flash_success=Carga eliminada")
}

// ═══════════════════════════════════════════════════════════════
// DIRECCIONES
// ═══════════════════════════════════════════════════════════════

func (c *AdminController) DireccionesIndex(ctx fiber.Ctx) error {
	list, err := c.direccionService.GetAll()
	if err != nil {
		log.Printf("Error listando direcciones: %v", err)
	}

	return ctx.Render("admin/direcciones/index", fiber.Map{
		"title":       "Direcciones",
		"direcciones": list,
		"csrfToken":   csrf.TokenFromContext(ctx),
		"role":        ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) DireccionesEdit(ctx fiber.Ctx) error {
	d, err := c.direccionService.GetByID(ctx.Params("id"))
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

func (c *AdminController) FacturasIndex(ctx fiber.Ctx) error {
	filters := map[string]string{
		"estado":       ctx.Query("estado"),
		"emisor_id":    ctx.Query("emisor_id"),
		"receptor_id":  ctx.Query("receptor_id"),
		"fecha_desde":  ctx.Query("fecha_desde"),
		"fecha_hasta":  ctx.Query("fecha_hasta"),
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "15"))

	list, total, err := c.facturaService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando facturas: %v", err)
	}

	prevPage := page - 1
	if prevPage < 1 { prevPage = 1 }
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	nextPage := page + 1
	if nextPage > totalPages { nextPage = totalPages }

	return ctx.Render("admin/facturas/index", fiber.Map{
		"title":      "Facturas",
		"facturas":   list,
		"total":      total,
		"page":       page,
		"perPage":    perPage,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"filters":    filters,
		"role":       ctx.Locals("role"),
		"csrfToken":  csrf.TokenFromContext(ctx),
	}, "layouts/base")
}

func (c *AdminController) FacturasShow(ctx fiber.Ctx) error {
	f, err := c.facturaService.GetByID(ctx.Params("id"))
	if err != nil {
		return ctx.Redirect().To("/admin/facturas?flash_error=Factura no encontrada")
	}
	return ctx.Render("admin/facturas/show", fiber.Map{
		"title":     "Detalle de Factura",
		"factura":   f,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      ctx.Locals("role"),
	}, "layouts/base")
}

func (c *AdminController) FacturasUpdate(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	var req requests.FacturaUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/admin/facturas/" + id + "/edit?flash_error=Datos inválidos")
	}
	updates := map[string]interface{}{
		"distancia_km":  req.DistanciaKm,
		"tarifa_por_km": req.TarifaPorKm,
		"subtotal":      req.Subtotal,
		"impuestos":     req.Impuestos,
		"total":         req.Total,
		"moneda":        req.Moneda,
		"metodo_pago":   strPtr(req.MetodoPago),
	}
	if err := c.facturaService.Update(id, updates); err != nil {
		return ctx.Redirect().To("/admin/facturas/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To("/admin/facturas?flash_success=Factura actualizada")
}

func (c *AdminController) FacturasDelete(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.facturaService.Delete(id); err != nil {
		return ctx.Redirect().To("/admin/facturas?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/admin/facturas?flash_success=Factura eliminada")
}

func (c *AdminController) FacturasPagar(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	metodo := ctx.FormValue("metodo_pago")
	if metodo == "" {
		metodo = "transferencia"
	}
	if err := c.facturaService.MarcarPagada(id, metodo); err != nil {
		return ctx.Redirect().To("/admin/facturas/" + id + "?flash_error=Error al marcar pagada")
	}
	return ctx.Redirect().To("/admin/facturas/" + id + "?flash_success=Factura marcada como pagada")
}
