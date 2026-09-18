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
	"github.com/gofiber/fiber/v3/middleware/session"
)

type AuthController struct{}

func NewAuthController() *AuthController { return &AuthController{} }

func (a *AuthController) ShowHome(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(uint)
	if !ok {
		return ctx.Redirect().To("/login")
	}

	var user models.User
	if err := facades.Orm().Query().
		With("Empresa").
		Where("id = ?", userID).
		First(&user); err != nil {
		return ctx.Redirect().To("/login")
	}

	data := fiber.Map{
		"title":        "Inicio",
		"user":         user,
		"isAdmin":      user.Role == "admin",
		"isPublicador": user.Role == "publicador",
		"isChofer":     user.Role == "chofer",
		"csrfToken":    csrf.TokenFromContext(ctx),
	}

	// ═══════════════════════════════════════════════════════════════
	// Datos para el panel principal
	// ═══════════════════════════════════════════════════════════════
	publicadorSvc := services.NewPublicadorService()
	choferSvc := services.NewChoferService()

	// ── ADMIN: contadores globales ──
	if user.Role == "admin" {
		var totalCargas, cargasPublicadas, cargasAsignadas, cargasEntregadas int64
		totalCargas, _ = facades.Orm().Query().Model(&models.Carga{}).Count()
		cargasPublicadas, _ = facades.Orm().Query().Model(&models.Carga{}).
			Where("estado = ?", "publicada").Count()
		cargasAsignadas, _ = facades.Orm().Query().Model(&models.Carga{}).
			Where("estado = ?", "asignada").Count()
		cargasEntregadas, _ = facades.Orm().Query().Model(&models.Carga{}).
			Where("estado = ?", "entregada").Count()
		data["totalCargas"]      = totalCargas
		data["cargasPublicadas"] = cargasPublicadas
		data["cargasAsignadas"]  = cargasAsignadas
		data["cargasEntregadas"] = cargasEntregadas
	}

	// ── CARGAS DISPONIBLES: visibles para todos ──
	var disponibles []models.Carga
	facades.Orm().Query().
		Model(&models.Carga{}).
		With("OrigenDireccion").
		With("DestinoDireccion").
		Where("estado = ?", "publicada").
		Order("fecha_recogida asc").
		Limit(20).
		Find(&disponibles)
	data["cargasDisponibles"] = disponibles

	// ── CARGAS ASIGNADAS / ENTREGADAS según rol ──
	var asignadas, entregadas []models.Carga

	switch user.Role {
	case "publicador":
		if pub, err := publicadorSvc.GetByUserID(userID); err == nil {
			facades.Orm().Query().
				Model(&models.Carga{}).
				With("OrigenDireccion").
				With("DestinoDireccion").
				Where("publicador_id = ?", pub.ID).
				Where("estado IN ?", []string{"asignada", "en_transito"}).
				Order("fecha_recogida asc").
				Find(&asignadas)

			facades.Orm().Query().
				Model(&models.Carga{}).
				With("OrigenDireccion").
				With("DestinoDireccion").
				Where("publicador_id = ?", pub.ID).
				Where("estado = ?", "entregada").
				Order("fecha_entrega desc").
				Find(&entregadas)
		}

	case "chofer":
		if ch, err := choferSvc.GetByUserID(userID); err == nil {
			facades.Orm().Query().
				Model(&models.Carga{}).
				With("OrigenDireccion").
				With("DestinoDireccion").
				Where("chofer_id = ?", ch.ID).
				Where("estado IN ?", []string{"asignada", "en_transito"}).
				Order("fecha_recogida asc").
				Find(&asignadas)

			facades.Orm().Query().
				Model(&models.Carga{}).
				With("OrigenDireccion").
				With("DestinoDireccion").
				Where("chofer_id = ?", ch.ID).
				Where("estado = ?", "entregada").
				Order("fecha_entrega desc").
				Find(&entregadas)
		}
	}

	data["cargasAsignadas"]  = asignadas
	data["cargasEntregadas"] = entregadas

	return ctx.Render("home", data, "layouts/base")
}

func (a *AuthController) Home(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(uint)
	if !ok {
		return ctx.Redirect().To("/login")
	}
	var user models.User
	if err := facades.Orm().Query().Where("id = ?", userID).First(&user); err != nil {
		return ctx.Redirect().To("/login")
	}
	// Todos van al mismo /home (el template decide qué mostrar por rol)
	return ctx.Redirect().To("/home")
}

func (a *AuthController) ShowLogin(ctx fiber.Ctx) error {
	return ctx.Render("auth/login", fiber.Map{
		"title":     "Iniciar Sesión",
		"csrfToken": csrf.TokenFromContext(ctx),
	})
}

func (a *AuthController) ShowRegister(ctx fiber.Ctx) error {
	userSvc := services.NewUserService()
	empresasBroker, _ := userSvc.GetEmpresasByTipo("broker")    // para publicadores
	empresasCarrier, _ := userSvc.GetEmpresasByTipo("carrier")  // para choferes

	return ctx.Render("auth/register", fiber.Map{
		"title":           "Crear Cuenta",
		"csrfToken":       csrf.TokenFromContext(ctx),
		"empresasBroker":  empresasBroker,
		"empresasCarrier": empresasCarrier,
	})
}

func (a *AuthController) HandleLogin(ctx fiber.Ctx) error {
	email := ctx.FormValue("email")
	password := ctx.FormValue("password")

	if email == "" || password == "" {
		return ctx.Render("auth/login", fiber.Map{
			"title":       "Iniciar Sesión",
			"flash_error": "Correo y contraseña son obligatorios",
			"csrfToken":   csrf.TokenFromContext(ctx),
		})
	}

	var user models.User
	if err := facades.Orm().Query().Where("email = ?", email).First(&user); err != nil {
		return ctx.Render("auth/login", fiber.Map{
			"title":       "Iniciar Sesión",
			"flash_error": "Credenciales incorrectas",
			"csrfToken":   csrf.TokenFromContext(ctx),
		})
	}

	if !facades.Hash().Check(password, user.Password) {
		return ctx.Render("auth/login", fiber.Map{
			"title":       "Iniciar Sesión",
			"flash_error": "Credenciales incorrectas",
			"csrfToken":   csrf.TokenFromContext(ctx),
		})
	}
	log.Println(user.IsActive)
	if !user.IsActive{
		return ctx.Redirect().To("/login?flash_error=Usuario deshabilitado")
	}
	sess := session.FromContext(ctx)
	if sess == nil {
		return ctx.Redirect().To("/login?flash_error=Error de sesión")
	}
	_ = sess.Regenerate()
	sess.Set("is_active",user.IsActive)
	sess.Set("user_id", user.ID)
	sess.Set("authenticated", true)
	sess.Set("role", user.Role)
	return ctx.Redirect().To("/home")
}

// HandleRegister — usa el request completo (identidad + empresa + ubicación + preferencias + disponibilidad).
func (a *AuthController) HandleRegister(ctx fiber.Ctx) error {
	// 1. Bind
	var req requests.UserRegisterRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return a.renderRegister(ctx, "Datos inválidos", nil, nil)
	}

	// 2. Rol válido
	if req.Role != "publicador" && req.Role != "chofer" {
		return a.renderRegister(ctx, "Debes seleccionar un tipo de cuenta válido", nil, &req)
	}

	// 3. Empresa: existing | new | none
	if req.EmpresaMode != "existing" && req.EmpresaMode != "new" && req.EmpresaMode != "none" {
		return a.renderRegister(ctx, "Selecciona una opción de empresa válida", nil, &req)
	}
	if req.EmpresaMode == "existing" && req.EmpresaID == 0 {
		return a.renderRegister(ctx, "Debes seleccionar una empresa", nil, &req)
	}
	if req.EmpresaMode == "new" && req.EmpresaNombreLegal == "" {
		return a.renderRegister(ctx, "El nombre legal de la empresa es obligatorio",
			map[string]string{"empresa_nombre_legal": "Requerido"}, &req)
	}

	// 4. Validación base + dinámica
	rules := map[string]any{
		"name":     "required|min:3|max:100",
		"email":    "required|email",
		"password": "required|min:8",
		"role":     "required|in:publicador,chofer",
		"city":     "required|max:100",
		"state":    "required|max:100",
		"country":  "required|max:100",
		"radius":   "required|integer|min:1",
		"phone":    "required|max:30",
		"whatsapp": "required|max:30",
	}
	if req.Role == "chofer" {
		rules["chofer_numero_licencia"]   = "required|min:3|max:50"
		rules["chofer_tipo_licencia"]     = "required|max:20"
		rules["chofer_anios_experiencia"] = "required|integer|min:0"
	}
	if req.Role == "publicador" {
		rules["publicador_numero_licencia_broker"] = "required|min:3|max:50"
		rules["publicador_anios_experiencia"]      = "required|integer|min:0"
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
		return a.renderRegister(ctx, "Revisa los campos marcados en rojo", errs, &req)
	}

	// 5. Email del dominio
	emailSvc := services.NewEmailService()
	if !emailSvc.EmailExists(req.Email) {
		return a.renderRegister(ctx, "El correo electrónico no es válido", nil, &req)
	}

	// 6. Email único
	userSvc := services.NewUserService()
	emailTaken, err := userSvc.EmailExists(req.Email, 0)
	if err != nil {
		return a.renderRegister(ctx, "Error al verificar el email", nil, &req)
	}
	if emailTaken {
		return a.renderRegister(ctx, "El email ya está registrado", nil, &req)
	}

	// 7. Hash
	hashed, err := facades.Hash().Make(req.Password)
	if err != nil {
		return a.renderRegister(ctx, "Error al procesar la contraseña", nil, &req)
	}

	// 8. Empresa (según modo)
	tipoEmpresa := "broker"
	if req.Role == "chofer" {
		tipoEmpresa = "carrier"
	}

	var empresa *models.Empresa
	switch req.EmpresaMode {
	case "existing":
		var found models.Empresa
		err := facades.Orm().Query().
			Where("id = ?", req.EmpresaID).
			Where("tipo = ?", tipoEmpresa).
			Where("estado = ?", "activo").
			First(&found)
		if err != nil || found.ID == 0 {
			return a.renderRegister(ctx,
				"La empresa seleccionada no es válida para tu rol", nil, &req)
		}
		empresa = &found
	case "new":
		empresa = &models.Empresa{
			Tipo:            models.TipoEmpresa(tipoEmpresa),
			NombreLegal:     req.EmpresaNombreLegal,
			NombreComercial: strPtr(req.EmpresaNombreComercial),
			TaxID:           strPtr(req.EmpresaTaxID),
			MCNumber:        strPtr(req.EmpresaMCNumber),
			DOTNumber:       strPtr(req.EmpresaDOTNumber),
			Telefono:        strPtr(req.EmpresaTelefono),
			Email:           strPtr(req.EmpresaEmail),
			SitioWeb:        strPtr(req.EmpresaSitioWeb),
			Estado:          models.EmpresaActiva,
		}
	case "none":
		empresa = nil
	}

	// 9. Parseo de fechas
	parseDate := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return &t
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return &t
		}
		return nil
	}

	// 10. User
	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashed,
		Role:     req.Role,
		IsActive: true,

		// Contacto
		Phone:          strPtr(req.Phone),
		PhoneAlt:       strPtr(req.PhoneAlt),
		WhatsApp:       strPtr(req.WhatsApp),
		Telegram:       strPtr(req.Telegram),
		EmergencyName:  strPtr(req.EmergencyName),
		EmergencyPhone: strPtr(req.EmergencyPhone),

		// Ubicación
		Address:    req.Address,
		City:       req.City,
		State:      req.State,
		Country:    req.Country,
		PostalCode: req.PostalCode,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		Radius:     req.Radius,

		// Preferencias
		PreferredEquipmentTypes: req.PreferredEquipmentTypes,
		PreferredCargoTypes:     req.PreferredCargoTypes,
		MaxWeight:               req.MaxWeight,
		MaxDistance:             req.MaxDistance,
		PreferredRoutes:         req.PreferredRoutes,

		// Disponibilidad
		AvailableFrom: parseDate(req.AvailableFrom),
		AvailableTo:   parseDate(req.AvailableTo),
		Notes:         req.Notes,
	}

	// 11. Perfil de rol
	var chofer *models.Chofer
	var publicador *models.Publicador

	if req.Role == "chofer" {
		chofer = &models.Chofer{
			NumeroLicencia:           req.ChoferNumeroLicencia,
			TipoLicencia:             req.ChoferTipoLicencia,
			PaisEmisionLicencia:      req.ChoferPaisEmisionLicencia,
			FechaVencimientoLicencia: parseDate(req.ChoferFechaVencimientoLicencia),
			AniosExperiencia:         req.ChoferAniosExperiencia,
			TiposEquipoPermitidos:    req.ChoferTiposEquipoPermitidos,
			Certificaciones:          req.ChoferCertificaciones,
			NumeroSeguro:             req.ChoferNumeroSeguro,
			FechaVencimientoSeguro:   parseDate(req.ChoferFechaVencimientoSeguro),
			Estado:                   models.ChoferDisponible,
		}
		if chofer.PaisEmisionLicencia == "" {
			chofer.PaisEmisionLicencia = "Cuba"
		}
	} else {
		publicador = &models.Publicador{
			NumeroLicenciaBroker:     req.PublicadorNumeroLicenciaBroker,
			PaisEmisionLicencia:      req.PublicadorPaisEmisionLicencia,
			FechaVencimientoLicencia: parseDate(req.PublicadorFechaVencimientoLicencia),
			AniosExperiencia:         req.PublicadorAniosExperiencia,
			Especialidad:             req.PublicadorEspecialidad,
			Comision:                 req.PublicadorComision,
			CreditScore:              req.PublicadorCreditScore,
			Estado:                   models.PublicadorActivo,
		}
		if publicador.PaisEmisionLicencia == "" {
			publicador.PaisEmisionLicencia = "Cuba"
		}
	}

	// 12. Persistir con rollback
	if err := userSvc.CreateWithRole(&user, empresa, chofer, publicador); err != nil {
		log.Printf("Error al crear usuario: %v", err)
		// Limpiamos el password para no re-exponerlo
		req.Password = ""
		return a.renderRegister(ctx, "No se pudo crear la cuenta. Intenta de nuevo.", nil, &req)
	}

	// 13. Éxito → login
	return ctx.Render("auth/login", fiber.Map{
		"title":         "Iniciar Sesión",
		"flash_success": "Registro exitoso. Ya puedes iniciar sesión.",
		"csrfToken":     csrf.TokenFromContext(ctx),
	})
}

func (a *AuthController) renderRegister(ctx fiber.Ctx, msg string, errs map[string]string, old *requests.UserRegisterRequest) error {
	userSvc := services.NewUserService()
	empresasBroker, _ := userSvc.GetEmpresasByTipo("broker")
	empresasCarrier, _ := userSvc.GetEmpresasByTipo("carrier")

	// Nunca re-enviar la contraseña al template
	if old != nil {
		old.Password = ""
	}

	return ctx.Render("auth/register", fiber.Map{
		"title":           "Crear Cuenta",
		"flash_error":     msg,
		"errors":          errs,
		"old":             old,
		"csrfToken":       csrf.TokenFromContext(ctx),
		"empresasBroker":  empresasBroker,
		"empresasCarrier": empresasCarrier,
	}, "layouts/base")
}
func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}


func (a *AuthController) Logout(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	if sess != nil {
		sess.Destroy()
	}
	return ctx.Redirect().To("/login")
}
