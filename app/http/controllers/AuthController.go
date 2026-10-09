package controllers

import (
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/goravel/framework/contracts/database/orm"
	frameworkerrors "github.com/goravel/framework/errors"
	"golang.org/x/crypto/bcrypt"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"
)

type AuthController struct {
	sendVerification func(email, confirmationURL string) error
}

// El hash ficticio impide omitir bcrypt cuando el correo no tiene cuenta.
var dummyLoginHash = sync.OnceValue(func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte("BUFALO unused authentication identity"), 12)
	if err != nil {
		panic(err)
	}
	return string(hash)
})

func NewAuthController() *AuthController {
	return NewAuthControllerWithVerificationSender(services.NewEmailService().SendRegistrationVerification)
}

// NewAuthControllerWithVerificationSender allows tests to capture confirmation
// links without sending real email.
func NewAuthControllerWithVerificationSender(sender func(email, confirmationURL string) error) *AuthController {
	_ = dummyLoginHash()
	return &AuthController{sendVerification: sender}
}

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
		"title":       "Inicio",
		"flash_error": ctx.Query("flash_error"), "flash_success": ctx.Query("flash_success"),
		"user":         user,
		"role":         user.Role,
		"isAdmin":      user.Role == "admin",
		"isPublicador": user.Role == "publicador",
		"isChofer":     user.Role == "chofer",
		"csrfToken":    csrf.TokenFromContext(ctx),
	}

	// ═══════════════════════════════════════════════════════════════
	// Datos para el panel principal
	// ═══════════════════════════════════════════════════════════════

	// ── ADMIN: contadores globales ──
	if user.Role == "admin" {
		var totalCargas, cargasPublicadas, cargasAsignadas, cargasEntregadas int64
		var err error
		totalCargas, err = facades.Orm().Query().Model(&models.Carga{}).Count()
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
		cargasPublicadas, err = facades.Orm().Query().Model(&models.Carga{}).
			Where("estado = ?", "publicada").Count()
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
		cargasAsignadas, err = facades.Orm().Query().Model(&models.Carga{}).
			Where("estado = ?", "asignada").Count()
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
		cargasEntregadas, err = facades.Orm().Query().Model(&models.Carga{}).
			Where("estado = ?", "entregada").Count()
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
		data["totalCargas"] = totalCargas
		data["cargasPublicadas"] = cargasPublicadas
		data["totalAsignadas"] = cargasAsignadas
		data["totalEntregadas"] = cargasEntregadas
	}

	filters, err := services.LoadVisibilityFilters(userID, user.Role)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			return fiber.ErrForbidden
		}
		return fiber.ErrServiceUnavailable
	}
	filters["status"] = "publicada"
	disponibles, _, err := services.NewCargaService().GetAllWithFilters(filters, 1, 20)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	data["cargasDisponibles"] = disponibles

	// Each summary is bounded and shares the same authorization as the board.
	var asignadas, entregadas []models.Carga
	if user.Role != "admin" {
		assignedFilters, err := services.LoadVisibilityFilters(userID, user.Role)
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
		if board := assignedFilters["driver_board_id"]; board != "" {
			delete(assignedFilters, "driver_board_id")
			assignedFilters["chofer_id"] = board
		}
		assignedFilters["active_work"] = "true"
		asignadas, _, err = services.NewCargaService().GetAllWithFilters(assignedFilters, 1, 20)
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
		delete(assignedFilters, "active_work")
		assignedFilters["status"] = "entregada"
		entregadas, _, err = services.NewCargaService().GetAllWithFilters(assignedFilters, 1, 20)
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
	}

	data["cargasAsignadas"] = asignadas
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
	ctx.Set("Cache-Control", "no-store, private")
	return ctx.Render("auth/login", fiber.Map{
		"title":     "Iniciar Sesión",
		"csrfToken": csrf.TokenFromContext(ctx),
	})
}

func (a *AuthController) ShowRegister(ctx fiber.Ctx) error {
	ctx.Set("Cache-Control", "no-store, private")
	empresasBroker := []models.Empresa{}
	empresasCarrier := []models.Empresa{}

	return ctx.Render("auth/register", fiber.Map{
		"title":           "Crear Cuenta",
		"csrfToken":       csrf.TokenFromContext(ctx),
		"empresasBroker":  empresasBroker,
		"empresasCarrier": empresasCarrier,
	})
}

func (a *AuthController) HandleLogin(ctx fiber.Ctx) error {
	ctx.Set("Cache-Control", "no-store, private")
	email := strings.ToLower(strings.TrimSpace(ctx.FormValue("email")))
	password := ctx.FormValue("password")

	if email == "" || len(email) > 100 || password == "" || len(password) > 72 {
		return ctx.Render("auth/login", fiber.Map{
			"title":       "Iniciar Sesión",
			"flash_error": "Correo y contraseña son obligatorios",
			"csrfToken":   csrf.TokenFromContext(ctx),
		})
	}

	var user models.User
	err := facades.Orm().Query().Where("email = ?", email).First(&user)
	if err != nil && !errors.Is(err, frameworkerrors.OrmRecordNotFound) {
		return fiber.ErrServiceUnavailable
	}
	hash := user.Password
	if user.ID == 0 {
		hash = dummyLoginHash()
	}
	valid := facades.Hash().Check(password, hash)
	if user.ID == 0 || !valid || !user.IsActive {
		return ctx.Render("auth/login", fiber.Map{
			"title":       "Iniciar Sesión",
			"flash_error": "Credenciales incorrectas",
			"csrfToken":   csrf.TokenFromContext(ctx),
		})
	}
	if cost, err := bcrypt.Cost([]byte(user.Password)); err == nil && cost < 12 {
		upgraded, err := facades.Hash().Make(password)
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
		result, err := facades.Orm().Query().Model(&models.User{}).Where("id = ? AND password = ?", user.ID, user.Password).Update("password", upgraded)
		if err != nil || result.RowsAffected != 1 {
			return fiber.ErrServiceUnavailable
		}
		user.Password = upgraded
	}
	sess := session.FromContext(ctx)
	if sess == nil {
		return ctx.Redirect().To("/login?flash_error=Error de sesión")
	}
	if err := sess.Regenerate(); err != nil {
		log.Printf("Error regenerando sesión al iniciar sesión: %v", err)
		return fiber.ErrInternalServerError
	}
	sess.Set("is_active", user.IsActive)
	sess.Set("user_id", user.ID)
	sess.Set("authenticated", true)
	sess.Set("role", user.Role)
	sess.Set("credential_stamp", user.CredentialStamp())
	return ctx.Redirect().To("/home")
}

// HandleRegister — usa el request completo (identidad + empresa + ubicación + preferencias + disponibilidad).
func (a *AuthController) HandleRegister(ctx fiber.Ctx) error {
	ctx.Set("Cache-Control", "no-store, private")
	// 1. Bind
	var req requests.UserRegisterRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return a.renderRegister(ctx, "Datos inválidos", nil, nil)
	}
	preferenceFields := []struct {
		name   string
		target *string
		valid  func(string) bool
	}{
		{"preferred_equipment_types", &req.PreferredEquipmentTypes, models.EsTipoEquipoValido},
		{"preferred_cargo_types", &req.PreferredCargoTypes, models.EsTipoCargaValido},
		{"chofer_tipos_equipo_permitidos", &req.ChoferTiposEquipoPermitidos, models.EsTipoEquipoValido},
	}
	for _, field := range preferenceFields {
		if ctx.FormValue(field.name+"_present") == "1" || ctx.Request().PostArgs().Has(field.name+"_choice") {
			parts := ctx.Request().PostArgs().PeekMulti(field.name + "_choice")
			selected := make([]string, 0, len(parts))
			for _, part := range parts {
				selected = append(selected, string(part))
			}
			*field.target = strings.Join(selected, ",")
		}
		normalized, err := requests.NormalizePreferences(*field.target, field.valid)
		if err != nil {
			return a.renderRegister(ctx, "Revisa las preferencias de carga y equipo", map[string]string{field.name: err.Error()}, &req)
		}
		*field.target = normalized
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if len(req.Password) > 72 {
		return a.renderRegister(ctx, "La contraseña no puede superar 72 bytes", nil, &req)
	}

	// 2. Rol válido
	if req.Role != "publicador" && req.Role != "chofer" {
		return a.renderRegister(ctx, "Debes seleccionar un tipo de cuenta válido", nil, &req)
	}

	// 3. Empresa: existing | new | none
	if req.EmpresaMode == "existing" {
		return a.renderRegister(ctx, "La asociación a una empresa existente requiere aprobación de un administrador", nil, &req)
	}
	if req.EmpresaMode != "new" && req.EmpresaMode != "none" {
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
		"email":    "required|email|max:100",
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
		rules["chofer_numero_licencia"] = "required|min:3|max:50"
		rules["chofer_tipo_licencia"] = "required|max:20"
		rules["chofer_anios_experiencia"] = "required|integer|min:0"
	}
	if req.Role == "publicador" {
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
		return a.renderRegister(ctx, "Revisa los campos marcados en rojo", errs, &req)
	}

	// 5. Email del dominio
	emailSvc := services.NewEmailService()
	if !emailSvc.EmailExists(req.Email) {
		return a.renderRegister(ctx, "El correo electrónico no es válido", nil, &req)
	}

	// Mantener el coste de hash también para direcciones ya registradas.
	hashed, err := facades.Hash().Make(req.Password)
	if err != nil {
		return a.renderRegister(ctx, "Error al procesar la contraseña", nil, &req)
	}
	// 6. Email único
	userSvc := services.NewUserService()
	emailTaken, err := userSvc.EmailExists(req.Email, 0)
	if err != nil {
		return a.renderRegister(ctx, "Error al verificar el email", nil, &req)
	}
	if emailTaken {
		return a.verificationSent(ctx)
	}

	// Validate a selected company now, then validate it again inside the final
	// transaction because the user may leave this flow until the email arrives.
	if req.EmpresaMode == "existing" {
		tipoEmpresa := "broker"
		if req.Role == "chofer" {
			tipoEmpresa = "carrier"
		}
		var found models.Empresa
		if err := facades.Orm().Query().Where("id = ?", req.EmpresaID).
			Where("tipo = ?", tipoEmpresa).Where("estado = ?", "activo").First(&found); err != nil || found.ID == 0 {
			return a.renderRegister(ctx, "La empresa seleccionada no es válida para tu rol", nil, &req)
		}
	}

	// Only an encrypted pending payload is stored here. No user, company, or
	// role profile is created until the owner of the email confirms the token.
	req.Password = hashed
	payload, err := json.Marshal(req)
	if err != nil {
		return a.renderRegister(ctx, "No se pudo preparar el registro", nil, &req)
	}
	verification := services.NewEmailVerificationService()
	token, err := verification.Start(req.Email, string(payload))
	if errors.Is(err, services.ErrRegistrationUnavailable) {
		return a.verificationSent(ctx)
	}
	if err != nil {
		log.Printf("No se pudo guardar el registro pendiente: %v", err)
		req.Password = ""
		return a.renderRegister(ctx, "No se pudo iniciar el registro. Intenta de nuevo.", nil, &req)
	}
	confirmationURL, err := registrationConfirmationURL(token)
	if err != nil {
		_ = verification.Cancel(token)
		log.Printf("No se pudo construir el enlace de confirmación: %v", err)
		return a.renderRegister(ctx, "No se pudo enviar la confirmación. Intenta de nuevo.", nil, &req)
	}
	if a.sendVerification == nil || a.sendVerification(req.Email, confirmationURL) != nil {
		_ = verification.Cancel(token)
		log.Printf("No se pudo enviar el correo de confirmación")
		req.Password = ""
		return a.renderRegister(ctx, "No pudimos enviar el correo de confirmación. Revisa los datos e inténtalo de nuevo.", nil, &req)
	}
	ticket, err := verification.IssueResendTicket(token)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	if sess := session.FromContext(ctx); sess != nil {
		sess.Set("verification_resend_ticket", ticket)
	}
	return a.verificationSent(ctx)
}

func (a *AuthController) ResendVerification(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	if sess == nil {
		return fiber.ErrUnauthorized
	}
	ticket, _ := sess.Get("verification_resend_ticket").(string)
	err := services.NewEmailVerificationService().Resend(ticket, func(email, token string) error {
		link, err := registrationConfirmationURL(token)
		if err != nil {
			return err
		}
		if a.sendVerification == nil {
			return fiber.ErrServiceUnavailable
		}
		return a.sendVerification(email, link)
	})
	if err != nil && !errors.Is(err, services.ErrInvalidVerificationToken) {
		return fiber.ErrServiceUnavailable
	}
	return a.verificationSent(ctx)
}

func (a *AuthController) verificationSent(ctx fiber.Ctx) error {
	canResend := false
	if sess := session.FromContext(ctx); sess != nil {
		canResend = sess.Get("verification_resend_ticket") != nil
	}
	ctx.Set("Cache-Control", "no-store, private")
	return ctx.Render("auth/verification_sent", fiber.Map{
		"title":     "Confirma tu correo",
		"canResend": canResend,
		"csrfToken": csrf.TokenFromContext(ctx),
	})
}

func (a *AuthController) ShowEmailConfirmation(ctx fiber.Ctx) error {
	ctx.Set("Cache-Control", "no-store, private")
	// Keep the activation token out of Referer while allowing HTTPS CSRF
	// validation in browsers that omit Origin on a same-origin form POST.
	ctx.Set("Referrer-Policy", "origin")
	return ctx.Render("auth/confirm_email", fiber.Map{
		"title":     "Confirmar correo",
		"token":     ctx.Query("token"),
		"csrfToken": csrf.TokenFromContext(ctx),
	})
}

func (a *AuthController) ConfirmEmail(ctx fiber.Ctx) error {
	ctx.Set("Cache-Control", "no-store, private")
	ctx.Set("Referrer-Policy", "origin")
	token := strings.TrimSpace(ctx.FormValue("token"))
	verification := services.NewEmailVerificationService()
	emailChanged := false
	err := verification.Confirm(token, func(tx orm.Query, payload string) error {
		var purpose struct {
			Purpose string `json:"purpose"`
		}
		if err := json.Unmarshal([]byte(payload), &purpose); err != nil {
			return err
		}
		if purpose.Purpose == "email_change" {
			emailChanged = true
			return services.ConfirmEmailChange(tx, payload)
		}
		var req requests.UserRegisterRequest
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			return errors.New("invalid pending registration payload")
		}
		return createConfirmedRegistration(tx, req)
	})
	if err != nil {
		if errors.Is(err, services.ErrInvalidVerificationToken) {
			return ctx.Render("auth/login", fiber.Map{
				"title": "Iniciar Sesión", "flash_error": "El enlace de confirmación no es válido o venció.",
				"csrfToken": csrf.TokenFromContext(ctx),
			})
		}
		log.Printf("No se pudo confirmar el registro: %v", err)
		return ctx.Render("auth/confirm_email", fiber.Map{
			"title": "Confirmar correo", "token": token,
			"flash_error": "No pudimos completar el registro. Intenta confirmar de nuevo.",
			"csrfToken":   csrf.TokenFromContext(ctx),
		})
	}
	if emailChanged {
		return ctx.Render("auth/login", fiber.Map{"title": "Iniciar sesión", "flash_success": "Correo actualizado. Inicia sesión con tu nuevo correo.", "csrfToken": csrf.TokenFromContext(ctx)})
	}
	return ctx.Render("auth/login", fiber.Map{
		"title": "Iniciar Sesión", "flash_success": "Correo confirmado. Tu cuenta fue creada; ya puedes iniciar sesión.",
		"csrfToken": csrf.TokenFromContext(ctx),
	})
}

func registrationConfirmationURL(token string) (string, error) {
	baseURL := strings.TrimSpace(facades.Config().GetString("http.url"))
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/") + "/register/confirm")
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return "", errors.New("APP_URL must be an absolute HTTP(S) URL")
	}
	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func createConfirmedRegistration(tx orm.Query, req requests.UserRegisterRequest) error {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Role != "publicador" && req.Role != "chofer" || req.Email == "" || req.Password == "" {
		return errors.New("invalid pending registration")
	}
	exists, err := tx.Model(&models.User{}).Where("email = ?", req.Email).Exists()
	if err != nil {
		return err
	}
	if exists {
		return errors.New("registration email already exists")
	}

	parseDate := func(value string) *time.Time {
		if value == "" {
			return nil
		}
		if parsed, err := time.Parse("2006-01-02", value); err == nil {
			return &parsed
		}
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return &parsed
		}
		return nil
	}
	tipoEmpresa := "broker"
	if req.Role == "chofer" {
		tipoEmpresa = "carrier"
	}
	var empresa *models.Empresa
	switch req.EmpresaMode {
	case "existing":
		return errors.New("existing company membership requires administrator approval")
	case "new":
		if strings.TrimSpace(req.EmpresaNombreLegal) == "" {
			return errors.New("company name is required")
		}
		empresa = &models.Empresa{
			Tipo: models.TipoEmpresa(tipoEmpresa), NombreLegal: req.EmpresaNombreLegal,
			NombreComercial: strPtr(req.EmpresaNombreComercial), TaxID: strPtr(req.EmpresaTaxID),
			MCNumber: strPtr(req.EmpresaMCNumber), DOTNumber: strPtr(req.EmpresaDOTNumber),
			Telefono: strPtr(req.EmpresaTelefono), Email: strPtr(req.EmpresaEmail),
			SitioWeb: strPtr(req.EmpresaSitioWeb), Estado: models.EmpresaActiva,
		}
	case "none":
	default:
		return errors.New("invalid company selection")
	}
	user := models.User{
		Name: req.Name, Email: req.Email, Password: req.Password, Role: req.Role, IsActive: true,
		Phone: strPtr(req.Phone), PhoneAlt: strPtr(req.PhoneAlt), WhatsApp: strPtr(req.WhatsApp),
		Telegram: strPtr(req.Telegram), EmergencyName: strPtr(req.EmergencyName), EmergencyPhone: strPtr(req.EmergencyPhone),
		Address: req.Address, City: req.City, State: req.State, Country: req.Country, PostalCode: req.PostalCode,
		Latitude: req.Latitude, Longitude: req.Longitude, Radius: req.Radius,
		PreferredEquipmentTypes: req.PreferredEquipmentTypes, PreferredCargoTypes: req.PreferredCargoTypes,
		MaxWeight: req.MaxWeight, MaxDistance: req.MaxDistance, PreferredRoutes: req.PreferredRoutes,
		AvailableFrom: parseDate(req.AvailableFrom), AvailableTo: parseDate(req.AvailableTo), Notes: req.Notes,
	}
	userService := services.NewUserService()
	if req.Role == "chofer" {
		pais := req.ChoferPaisEmisionLicencia
		if pais == "" {
			pais = "Cuba"
		}
		profile := &models.Chofer{
			NumeroLicencia: req.ChoferNumeroLicencia, TipoLicencia: req.ChoferTipoLicencia,
			PaisEmisionLicencia: pais, FechaVencimientoLicencia: parseDate(req.ChoferFechaVencimientoLicencia),
			AniosExperiencia: req.ChoferAniosExperiencia, TiposEquipoPermitidos: req.ChoferTiposEquipoPermitidos,
			Certificaciones: req.ChoferCertificaciones, NumeroSeguro: req.ChoferNumeroSeguro,
			FechaVencimientoSeguro: parseDate(req.ChoferFechaVencimientoSeguro), Estado: models.ChoferDisponible,
		}
		return userService.CreateWithRoleInTransaction(tx, &user, empresa, profile, nil)
	}
	pais := req.PublicadorPaisEmisionLicencia
	if pais == "" {
		pais = "Cuba"
	}
	profile := &models.Publicador{
		NumeroLicenciaBroker: req.PublicadorNumeroLicenciaBroker, PaisEmisionLicencia: pais,
		FechaVencimientoLicencia: parseDate(req.PublicadorFechaVencimientoLicencia),
		AniosExperiencia:         req.PublicadorAniosExperiencia, Especialidad: req.PublicadorEspecialidad,
		Comision: req.PublicadorComision, CreditScore: req.PublicadorCreditScore, Estado: models.PublicadorActivo,
	}
	return userService.CreateWithRoleInTransaction(tx, &user, empresa, nil, profile)
}

func (a *AuthController) renderRegister(ctx fiber.Ctx, msg string, errs map[string]string, old *requests.UserRegisterRequest) error {
	empresasBroker := []models.Empresa{}
	empresasCarrier := []models.Empresa{}

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
		if err := sess.Destroy(); err != nil {
			return fiber.ErrServiceUnavailable
		}
	}
	return ctx.Redirect().To("/login")
}

func (a *AuthController) ShowLogout(ctx fiber.Ctx) error {
	return ctx.Render("auth/logout", fiber.Map{"title": "Cerrar sesión", "csrfToken": csrf.TokenFromContext(ctx)})
}
