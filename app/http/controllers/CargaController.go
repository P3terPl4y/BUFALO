package controllers

import (
	"fmt"
	"goravel/app/community"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/requests"
	"goravel/app/services"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
)

type CargaController struct {
	cargaService      *services.CargaService
	direccionService  *services.DireccionService
	empresaService    *services.EmpresaService
	choferService     *services.ChoferService
	publicadorService *services.PublicadorService
}

func NewCargaController() *CargaController {
	return &CargaController{
		cargaService:      services.NewCargaService(),
		direccionService:  services.NewDireccionService(),
		empresaService:    services.NewEmpresaService(),
		choferService:     services.NewChoferService(),
		publicadorService: services.NewPublicadorService(),
	}
}

func (c *CargaController) getCurrentUserID(ctx fiber.Ctx) (uint, error) {
	if uid, ok := ctx.Locals("user_id").(uint); ok {
		return uid, nil
	}
	if uid, ok := session.FromContext(ctx).Get("user_id").(uint); ok {
		return uid, nil
	}
	return 0, fiber.ErrUnauthorized
}

func (c *CargaController) getAllDestinations(ctx fiber.Ctx) []models.Direccion {
	userID, role := currentUser(ctx)
	var dests []models.Direccion
	var err error
	if role == "admin" {
		dests, err = c.direccionService.GetAll()
	} else {
		dests, err = c.direccionService.GetOwnedByUserID(userID)
	}
	if err != nil {
		log.Printf("Error al obtener direcciones: %v", err)
		return []models.Direccion{}
	}
	return dests
}

// ─────────────────────────────────────────────────────────────
// Index
// ─────────────────────────────────────────────────────────────
func (c *CargaController) Index(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	filters := map[string]string{
		"status":      ctx.Query("status"),
		"tipo_equipo": ctx.Query("tipo_equipo"),
		"tipo_carga":  ctx.Query("tipo_carga"),
		"min_rate":    ctx.Query("min_rate"),
		"max_rate":    ctx.Query("max_rate"),
		"pickup_from": ctx.Query("pickup_from"),
		"pickup_to":   ctx.Query("pickup_to"),
	}

	// ── Filtrar por rol ──
	// publicador: ve las cargas que él publicó (publicador_id = su perfil)
	// chofer:     ve las cargas que él aceptó (chofer_id = su perfil)
	// admin:      ve todas
	switch role {
	case "publicador":
		pub, err := c.publicadorService.GetByUserID(userID)
		if err != nil {
			log.Printf("No se pudo resolver perfil publicador user_id=%d: %v", userID, err)
			return ctx.SendStatus(fiber.StatusForbidden)
		}
		filters["publicador_id"] = strconv.FormatUint(uint64(pub.ID), 10)
	case "chofer":
		ch, err := c.choferService.GetByUserID(userID)
		if err != nil {
			log.Printf("No se pudo resolver perfil chofer user_id=%d: %v", userID, err)
			return ctx.SendStatus(fiber.StatusForbidden)
		}
		filters["driver_board_id"] = strconv.FormatUint(uint64(ch.ID), 10)
	}

	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "10"))

	list, total, err := c.cargaService.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error al obtener cargas: %v", err)
		return ctx.Render("dashboard/index", fiber.Map{
			"title":       "Lista de Cargas",
			"flash_error": "Error al cargar las cargas",
			"role":        role,
		}, "layouts/base")
	}

	return ctx.Render("dashboard/index", fiber.Map{
		"title":   "Lista de Cargas",
		"loads":   list,
		"total":   total,
		"page":    page,
		"perPage": perPage,
		"filters": filters,
		"role":    role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Show
// ─────────────────────────────────────────────────────────────
func (c *CargaController) Show(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	carga, err := c.cargaService.GetByID(ctx.Params("id"))
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{
			"title": "Carga no encontrada",
			"role":  role,
		}, "layouts/base")
	}

	if !c.canAccessCarga(carga, userID, role) {
		return ctx.Render("dashboard/404", fiber.Map{
			"title": "Carga no encontrada",
			"role":  role,
		}, "layouts/base")
	}

	hasMap := carga.OrigenDireccion != nil && carga.DestinoDireccion != nil &&
		carga.OrigenDireccion.Latitud != nil && carga.OrigenDireccion.Longitud != nil &&
		carga.DestinoDireccion.Latitud != nil && carga.DestinoDireccion.Longitud != nil

	return ctx.Render("dashboard/show", fiber.Map{
		"title":        "Detalle de Carga",
		"load":         carga,
		"csrfToken":    csrf.TokenFromContext(ctx),
		"hasMap":       hasMap,
		"role":         role,
		"userID":       userID,
		"alreadyRated": services.NewDriverCommunityService().HasLoadRating(carga.ID),
	}, "layouts/base")
}

// canAccessCarga decide si el usuario puede VER la carga.
func (c *CargaController) canAccessCarga(carga *models.Carga, userID uint, role string) bool {
	switch role {
	case "admin":
		return true
	case "publicador":
		pub, err := c.publicadorService.GetByUserID(userID)
		if err != nil {
			return false
		}
		return carga.PublicadorID == pub.ID
	case "chofer":
		ch, err := c.choferService.GetByUserID(userID)
		if err != nil {
			return false
		}
		member := services.NewDriverCommunityService().IsCompanyMember(carga.EmpresaID, ch.ID)
		return community.DriverCanSeeLoad(string(carga.Estado), string(carga.Audiencia), carga.ChoferID != nil && *carga.ChoferID == ch.ID, member)
	}
	return false
}

// ─────────────────────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────────────────────
func (c *CargaController) Create(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	return ctx.Render("dashboard/create", fiber.Map{
		"title":        "Nueva Carga",
		"destinations": c.getAllDestinations(ctx),
		"csrfToken":    csrf.TokenFromContext(ctx),
		"role":         sess.Get("role"),
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Store
// ─────────────────────────────────────────────────────────────
func (c *CargaController) Store(ctx fiber.Ctx) error {
	userID, err := c.getCurrentUserID(ctx)
	if err != nil {
		return ctx.Redirect().To("/login")
	}

	// 1. Resolver el Publicador del user logueado
	publicador, err := c.publicadorService.GetByUserID(userID)
	if err != nil {
		return ctx.Redirect().To("/loads/create?flash_error=Debes tener un perfil de publicador")
	}

	var req requests.CargaStoreRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Render("dashboard/create", fiber.Map{
			"title":        "Nueva Carga",
			"flash_error":  "Datos inválidos",
			"destinations": c.getAllDestinations(ctx),
			"csrfToken":    csrf.TokenFromContext(ctx),
		}, "layouts/base")
	}

	errs, valid := validateCargaInput(ctx, req)
	if !valid {
		return ctx.Render("dashboard/create", fiber.Map{
			"title":        "Nueva Carga",
			"flash_error":  "Error de validación",
			"errors":       errs,
			"old":          req,
			"destinations": c.getAllDestinations(ctx),
			"csrfToken":    csrf.TokenFromContext(ctx),
		}, "layouts/base")
	}

	fechaRecogida, err := time.Parse("2006-01-02T15:04", req.FechaRecogida)
	if err != nil {
		return ctx.Redirect().To("/loads/create?flash_error=Fecha de recogida inválida")
	}
	var fechaEntrega *time.Time
	if req.FechaEntrega != "" {
		t, parseErr := time.Parse("2006-01-02T15:04", req.FechaEntrega)
		if parseErr != nil || t.Before(fechaRecogida) {
			return ctx.Redirect().To("/loads/create?flash_error=Fecha+de+entrega+inv%C3%A1lida")
		}
		fechaEntrega = &t
	}

	// tarifa_por_km
	tarifaKm := req.TarifaPorKm
	if (tarifaKm == nil || *tarifaKm == 0) && req.DistanciaKm > 0 && req.TarifaTotal != nil {
		val := *req.TarifaTotal / req.DistanciaKm
		tarifaKm = &val
	}

	audiencia := models.Audiencia(strings.TrimSpace(req.Audiencia))
	if audiencia == "" {
		audiencia = models.AudienciaLoadBoard
	}
	if err := community.ValidateAudience(string(audiencia)); err != nil {
		return ctx.Redirect().To("/loads/create?flash_error=Selecciona+una+audiencia+v%C3%A1lida")
	}
	if audiencia == models.AudienciaRedPrivada {
		members, err := facades.Orm().Query().Model(&models.RedChofer{}).Where("empresa_id = ?", publicador.EmpresaID).Count()
		if err != nil || members == 0 {
			return ctx.Redirect().To("/loads/create?flash_error=A%C3%B1ade+choferes+a+tu+red+antes+de+publicar+en+privado")
		}
	}

	carga := models.Carga{
		NumeroReferencia:   req.NumeroReferencia,
		PublicadorID:       publicador.ID,        // ✅ perfil del publicador
		EmpresaID:          publicador.EmpresaID, // ✅ empresa del publicador
		ChoferID:           nil,                  // sin asignar
		OrigenDireccionID:  req.OrigenDireccionID,
		DestinoDireccionID: req.DestinoDireccionID,
		FechaRecogida:      fechaRecogida,
		FechaEntrega:       fechaEntrega,
		TipoCarga:          models.TipoCarga(req.TipoCarga),
		TipoEquipo:         models.TipoEquipo(req.TipoEquipo),
		PesoKg:             req.PesoKg,
		Commodity:          strPtr(req.Commodity),
		DistanciaKm:        req.DistanciaKm,
		DistanciaRealKm:    req.DistanciaRealKm,
		TarifaTotal:        req.TarifaTotal,
		TarifaPorKm:        tarifaKm,
		Moneda:             models.Moneda(req.Moneda),
		Estado:             models.CargaPublicada,
		Audiencia:          audiencia,
	}

	if err := c.cargaService.Create(&carga); err != nil {
		log.Printf("Error al crear carga: %v", err)
		return ctx.Redirect().To("/loads/create?flash_error=Error al guardar")
	}
	return ctx.Redirect().To(fmt.Sprintf("/loads/%d", carga.ID))
}

// ─────────────────────────────────────────────────────────────
// Edit
// ─────────────────────────────────────────────────────────────
func (c *CargaController) Edit(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	carga, err := c.cargaService.GetByID(ctx.Params("id"))
	if err != nil || !c.canEditCarga(carga, userID, role) {
		return ctx.Render("dashboard/404", fiber.Map{
			"title": "Carga no encontrada",
			"role":  role,
		}, "layouts/base")
	}

	return ctx.Render("dashboard/edit", fiber.Map{
		"title":        "Editar Carga",
		"load":         carga,
		"role":         role,
		"destinations": c.getAllDestinations(ctx),
		"csrfToken":    csrf.TokenFromContext(ctx),
	}, "layouts/base")
}

// canEditCarga decide si el usuario puede EDITAR la carga.
func (c *CargaController) canEditCarga(carga *models.Carga, userID uint, role string) bool {
	if role == "admin" {
		return true
	}
	if role != "publicador" {
		return false
	}
	pub, err := c.publicadorService.GetByUserID(userID)
	if err != nil {
		return false
	}
	return carga.PublicadorID == pub.ID
}

// ─────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────
func (c *CargaController) Update(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)
	id := ctx.Params("id")

	existing, err := c.cargaService.GetByID(id)
	if err != nil || !c.canEditCarga(existing, userID, role) {
		return ctx.Render("dashboard/404", fiber.Map{
			"title": "Carga no encontrada",
			"role":  role,
		}, "layouts/base")
	}

	var req requests.CargaUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/loads/" + id + "/edit?flash_error=Datos inválidos")
	}
	if _, valid := validateCargaInput(ctx, req); !valid {
		return ctx.Redirect().To("/loads/" + id + "/edit?flash_error=Revisa+los+datos+de+la+carga")
	}
	audience := models.Audiencia(strings.TrimSpace(req.Audiencia))
	if audience == "" {
		audience = models.AudienciaLoadBoard
	}
	if err := community.ValidateAudience(string(audience)); err != nil {
		return ctx.Redirect().To("/loads/" + id + "/edit?flash_error=Audiencia+inv%C3%A1lida")
	}
	if audience == models.AudienciaRedPrivada {
		if existing.EmpresaID == 0 {
			return ctx.Redirect().To("/loads/" + id + "/edit?flash_error=La+empresa+no+tiene+una+red+privada")
		}
		members, countErr := facades.Orm().Query().Model(&models.RedChofer{}).Where("empresa_id = ?", existing.EmpresaID).Count()
		if countErr != nil || members == 0 {
			return ctx.Redirect().To("/loads/" + id + "/edit?flash_error=A%C3%B1ade+choferes+a+tu+red+antes+de+publicar+en+privado")
		}
	}

	fechaRecogida, err := time.Parse("2006-01-02T15:04", req.FechaRecogida)
	if err != nil {
		return ctx.Redirect().To("/loads/" + id + "/edit?flash_error=Fecha+de+recogida+inv%C3%A1lida")
	}
	var fechaEntrega *time.Time
	if req.FechaEntrega != "" {
		t, parseErr := time.Parse("2006-01-02T15:04", req.FechaEntrega)
		if parseErr != nil || t.Before(fechaRecogida) {
			return ctx.Redirect().To("/loads/" + id + "/edit?flash_error=Fecha+de+entrega+inv%C3%A1lida")
		}
		fechaEntrega = &t
	}

	tarifaKm := req.TarifaPorKm
	if (tarifaKm == nil || *tarifaKm == 0) && req.DistanciaKm > 0 && req.TarifaTotal != nil {
		val := *req.TarifaTotal / req.DistanciaKm
		tarifaKm = &val
	}

	// Whitelist: NUNCA se cambian publicador_id, empresa_id, chofer_id, estado
	updates := map[string]interface{}{
		"numero_referencia":    req.NumeroReferencia,
		"origen_direccion_id":  req.OrigenDireccionID,
		"destino_direccion_id": req.DestinoDireccionID,
		"fecha_recogida":       fechaRecogida,
		"fecha_entrega":        fechaEntrega,
		"tipo_carga":           req.TipoCarga,
		"tipo_equipo":          req.TipoEquipo,
		"peso_kg":              req.PesoKg,
		"commodity":            strPtr(req.Commodity),
		"distancia_km":         req.DistanciaKm,
		"distancia_real_km":    req.DistanciaRealKm,
		"tarifa_total":         req.TarifaTotal,
		"tarifa_por_km":        tarifaKm,
		"moneda":               req.Moneda,
		"audiencia":            audience,
	}

	if err := c.cargaService.Update(id, updates); err != nil {
		log.Printf("Error al actualizar carga: %v", err)
		return ctx.Redirect().To("/loads/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To(fmt.Sprintf("/loads/%s?flash_success=Carga actualizada correctamente", id))
}

func cargaValidationRules() map[string]any {
	return map[string]any{
		"numero_referencia":    "required|min:3|max:50",
		"origen_direccion_id":  "required|integer",
		"destino_direccion_id": "required|integer",
		"fecha_recogida":       "required",
		"tipo_carga":           "required|in:" + strings.Join(models.TiposCargaDisponibles(), ","),
		"tipo_equipo":          "required|in:" + strings.Join(models.TiposEquipoDisponibles(), ","),
		"peso_kg":              "required|numeric|min:0.01",
		"distancia_km":         "required|numeric|min:0.01",
		"tarifa_total":         "required|numeric|min:0.01",
		"moneda":               "required|in:CUP,MLC,USD,EUR",
	}
}

func validateCargaInput(ctx fiber.Ctx, req any) (map[string]string, bool) {
	validator, err := facades.Validation().Make(ctx.Context(), req, cargaValidationRules())
	if err != nil || validator == nil || validator.Fails() {
		errors := map[string]string{"tipo_carga": "Selecciona un tipo de carga válido."}
		if validator != nil {
			for field, fieldErrors := range validator.Errors().All() {
				for _, message := range fieldErrors {
					errors[field] = message
					break
				}
			}
		}
		return errors, false
	}
	return nil, true
}

// ─────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────
func (c *CargaController) Delete(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)
	id := ctx.Params("id")

	carga, err := c.cargaService.GetByID(id)
	if err != nil || !c.canEditCarga(carga, userID, role) {
		return ctx.Redirect().To("/loads?flash_error=Carga no encontrada")
	}

	if err := c.cargaService.Delete(id); err != nil {
		return ctx.Redirect().To("/loads?flash_error=Error al eliminar la carga")
	}
	return ctx.Redirect().To("/loads?flash_success=Carga eliminada correctamente")
}

// ─────────────────────────────────────────────────────────────
// AcceptLoad — un chofer acepta una carga publicada
// ─────────────────────────────────────────────────────────────
func (c *CargaController) AcceptLoad(ctx fiber.Ctx) error {
	userID, err := c.getCurrentUserID(ctx)
	if err != nil {
		return ctx.Redirect().To("/login")
	}

	// 1. Resolver el Chofer del user logueado
	chofer, err := c.choferService.GetByUserID(userID)
	if err != nil {
		return ctx.Redirect().To("/home?flash_error=Debes tener un perfil de chofer")
	}

	id := ctx.Params("id")
	if err := c.cargaService.AcceptLoadService(id, chofer.ID); err != nil {
		log.Printf("Error al aceptar carga %s: %v", id, err)
		return ctx.Redirect().To("/home?flash_error=No se pudo aceptar la carga")
	}
	return ctx.Redirect().To("/home?flash_success=Carga aceptada correctamente")
}

// StartTransit lets only the assigned driver begin the journey.
func (c *CargaController) StartTransit(ctx fiber.Ctx) error {
	userID, err := c.getCurrentUserID(ctx)
	if err != nil {
		return ctx.Redirect().To("/login")
	}
	chofer, err := c.choferService.GetByUserID(userID)
	if err != nil {
		return ctx.Redirect().To("/home?flash_error=Debes tener un perfil de chofer")
	}
	id := ctx.Params("id")
	if err := c.cargaService.StartTransit(id, chofer.ID); err != nil {
		return ctx.Redirect().To(fmt.Sprintf("/loads/%s?flash_error=No se pudo iniciar el tránsito", id))
	}
	return ctx.Redirect().To(fmt.Sprintf("/loads/%s?flash_success=Tránsito iniciado", id))
}

// MarkDelivered lets only the assigned driver complete a load already in transit.
func (c *CargaController) MarkDelivered(ctx fiber.Ctx) error {
	userID, err := c.getCurrentUserID(ctx)
	if err != nil {
		return ctx.Redirect().To("/login")
	}
	chofer, err := c.choferService.GetByUserID(userID)
	if err != nil {
		return ctx.Redirect().To("/home?flash_error=Debes tener un perfil de chofer")
	}
	id := ctx.Params("id")
	if err := c.cargaService.MarkDelivered(id, chofer.ID); err != nil {
		return ctx.Redirect().To(fmt.Sprintf("/loads/%s?flash_error=No se pudo registrar la entrega", id))
	}
	return ctx.Redirect().To(fmt.Sprintf("/loads/%s?flash_success=Carga entregada", id))
}

// ─────────────────────────────────────────────────────────────
// AssignChofer — el publicador dueño de la carga asigna un chofer
// ─────────────────────────────────────────────────────────────
func (c *CargaController) AssignChofer(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)

	carga, err := c.cargaService.GetByID(ctx.Params("id"))
	if err != nil {
		return ctx.Redirect().To("/loads?flash_error=Carga no encontrada")
	}
	if !c.canEditCarga(carga, userID, role) {
		return ctx.Redirect().To("/loads?flash_error=No autorizado")
	}

	choferIDStr := ctx.FormValue("chofer_id")
	choferID, err := strconv.ParseUint(choferIDStr, 10, 32)
	if err != nil {
		return ctx.Redirect().To("/loads/" + ctx.Params("id") + "?flash_error=Chofer inválido")
	}
	if carga.Audiencia == models.AudienciaRedPrivada && !services.NewDriverCommunityService().IsCompanyMember(carga.EmpresaID, uint(choferID)) {
		return ctx.Redirect().To("/loads/" + ctx.Params("id") + "?flash_error=Solo+puedes+asignar+choferes+de+tu+red+privada")
	}

	if err := c.cargaService.AssignChofer(ctx.Params("id"), uint(choferID)); err != nil {
		return ctx.Redirect().To("/loads/" + ctx.Params("id") + "?flash_error=Error al asignar")
	}
	return ctx.Redirect().To("/loads/" + ctx.Params("id") + "?flash_success=Chofer asignado")
}

func (c *CargaController) RateDriver(ctx fiber.Ctx) error {
	userID, err := c.getCurrentUserID(ctx)
	if err != nil {
		return ctx.Redirect().To("/login")
	}
	role, _ := session.FromContext(ctx).Get("role").(string)
	if role != "publicador" {
		return ctx.SendStatus(fiber.StatusForbidden)
	}
	profile, err := c.publicadorService.GetByUserID(userID)
	if err != nil {
		return ctx.SendStatus(fiber.StatusForbidden)
	}
	load, err := c.cargaService.GetByID(ctx.Params("id"))
	if err != nil || load.PublicadorID != profile.ID {
		return ctx.SendStatus(fiber.StatusNotFound)
	}
	score, err := strconv.Atoi(ctx.FormValue("puntaje"))
	if err != nil {
		return ctx.Redirect().To("/loads/" + ctx.Params("id") + "?flash_error=Puntaje+inv%C3%A1lido")
	}
	comment := strings.TrimSpace(ctx.FormValue("comentario"))
	if err := services.NewDriverCommunityService().RateCompletedLoad(load.ID, profile.ID, score, comment); err != nil {
		return ctx.Redirect().To("/loads/" + ctx.Params("id") + "?flash_error=No+se+pudo+registrar+la+calificaci%C3%B3n")
	}
	return ctx.Redirect().To("/loads/" + ctx.Params("id") + "?flash_success=Chofer+calificado")
}
