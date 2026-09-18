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

func (c *CargaController) getAllDestinations() []models.Direccion {
	dests, err := c.direccionService.GetAll()
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
		if pub, err := c.publicadorService.GetByUserID(userID); err == nil {
			filters["publicador_id"] = strconv.FormatUint(uint64(pub.ID), 10)
		}
	case "chofer":
		if ch, err := c.choferService.GetByUserID(userID); err == nil {
			filters["chofer_id"] = strconv.FormatUint(uint64(ch.ID), 10)
		}
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
		"title":         "Lista de Cargas",
		"loads":         list,
		"total":         total,
		"page":          page,
		"perPage":       int64(perPage),
		"filters":       filters,
		"role":          role,
		"flash_success": ctx.Query("flash_success"), "created_id": ctx.Query("created_id"),
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
		"title":     "Detalle de Carga",
		"load":      carga,
		"csrfToken": csrf.TokenFromContext(ctx),
		"hasMap":    hasMap,
		"role":      role,
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
		// Abierta: cualquier chofer la puede ver
		if carga.Estado == "publicada" {
			return true
		}
		// Asignada a este chofer
		ch, err := c.choferService.GetByUserID(userID)
		if err != nil {
			return false
		}
		return carga.ChoferID != nil && *carga.ChoferID == ch.ID
	}
	return false
}

// ─────────────────────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────────────────────
func (c *CargaController) Create(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	return ctx.Render("dashboard/create", fiber.Map{
		"title":         "Nueva Carga",
		"flash_error":   ctx.Query("flash_error"),
		"flash_success": ctx.Query("flash_success"),
		"destinations":  c.getAllDestinations(),
		"csrfToken":     csrf.TokenFromContext(ctx),
		"role":          sess.Get("role"),
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
	// El administrador puede publicar cargas de prueba usando el primer perfil
	// publicador disponible; los demás roles deben tener su propio perfil.
	if err != nil {
		role, _ := session.FromContext(ctx).Get("role").(string)
		if role == "admin" {
			var fallback models.Publicador
			err = facades.Orm().Query().Order("id asc").First(&fallback)
			if err == nil && fallback.ID > 0 {
				publicador = &fallback
			}
		}
	}
	if err != nil {
		return ctx.Redirect().To("/loads/create?flash_error=Debes tener un perfil de publicador")
	}

	var req requests.CargaStoreRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Render("dashboard/create", fiber.Map{
			"title":        "Nueva Carga",
			"flash_error":  "Datos inválidos",
			"destinations": c.getAllDestinations(),
			"csrfToken":    csrf.TokenFromContext(ctx),
		}, "layouts/base")
	}

	rules := map[string]any{
		"numero_referencia":    "required|min:3|max:50",
		"origen_direccion_id":  "required|integer",
		"destino_direccion_id": "required|integer",
		"fecha_recogida":       "required",
		"tipo_carga":           "required|in:FTL,LTL",
		"tipo_equipo":          "required|in:dry_van,flatbed,reefer,step_deck,double_drop,lowboy,cargo_van,box_truck,power_only",
		"peso_kg":              "required|numeric|min:0.01",
		"distancia_km":         "required|numeric|min:0.01",
		"tarifa_total":         "required|numeric|min:0.01",
		"moneda":               "required|in:CUP,MLC,USD,EUR",
	}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil || validator.Fails() {
		errs := map[string]string{}
		if validator != nil {
			for field, fieldErrors := range validator.Errors().All() {
				for _, msg := range fieldErrors {
					errs[field] = msg
					break
				}
			}
		}
		return ctx.Render("dashboard/create", fiber.Map{
			"title":        "Nueva Carga",
			"flash_error":  "Error de validación",
			"errors":       errs,
			"old":          req,
			"destinations": c.getAllDestinations(),
			"csrfToken":    csrf.TokenFromContext(ctx),
		}, "layouts/base")
	}

	fechaRecogida, err := time.Parse("2006-01-02T15:04", req.FechaRecogida)
	if err != nil {
		return ctx.Redirect().To("/loads/create?flash_error=Fecha de recogida inválida")
	}
	var fechaEntrega *time.Time
	if req.FechaEntrega != "" {
		if t, err := time.Parse("2006-01-02T15:04", req.FechaEntrega); err == nil {
			fechaEntrega = &t
		}
	}

	// tarifa_por_km
	tarifaKm := req.TarifaPorKm
	if (tarifaKm == nil || *tarifaKm == 0) && req.DistanciaKm > 0 && req.TarifaTotal != nil {
		val := *req.TarifaTotal / req.DistanciaKm
		tarifaKm = &val
	}

	audiencia := models.Audiencia(req.Audiencia)
	if audiencia == "" {
		audiencia = models.AudienciaLoadBoard
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
	return ctx.Redirect().To(fmt.Sprintf("/loads?flash_success=Carga publicada correctamente&created_id=%d", carga.ID))
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
		"destinations": c.getAllDestinations(),
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

	fechaRecogida, _ := time.Parse("2006-01-02T15:04", req.FechaRecogida)
	var fechaEntrega *time.Time
	if req.FechaEntrega != "" {
		if t, err := time.Parse("2006-01-02T15:04", req.FechaEntrega); err == nil {
			fechaEntrega = &t
		}
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
		"audiencia":            req.Audiencia,
	}

	if err := c.cargaService.Update(id, updates); err != nil {
		log.Printf("Error al actualizar carga: %v", err)
		return ctx.Redirect().To("/loads/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To(fmt.Sprintf("/loads/%s?flash_success=Carga actualizada correctamente", id))
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

	if err := c.cargaService.AssignChofer(ctx.Params("id"), uint(choferID)); err != nil {
		return ctx.Redirect().To("/loads/" + ctx.Params("id") + "?flash_error=Error al asignar")
	}
	return ctx.Redirect().To("/loads/" + ctx.Params("id") + "?flash_success=Chofer asignado")
}
