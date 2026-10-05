package controllers

import (
	"fmt"
	"goravel/app/billing"
	"goravel/app/exports"
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

type FacturaController struct {
	service      *services.FacturaService
	cargaService *services.CargaService
}

func NewFacturaController() *FacturaController {
	return &FacturaController{
		service:      services.NewFacturaService(),
		cargaService: services.NewCargaService(),
	}
}

func (c *FacturaController) Index(ctx fiber.Ctx) error {
	filters, err := c.invoiceFiltersForCurrentUser(ctx)
	if err != nil {
		return fiber.ErrForbidden
	}
	role, _ := session.FromContext(ctx).Get("role").(string)
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "10"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}

	list, total, err := c.service.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		log.Printf("Error listando facturas: %v", err)
		return fiber.ErrInternalServerError
	}

	return ctx.Render("facturas/index", fiber.Map{
		"title": "Facturas", "facturas": list, "total": total, "page": page, "perPage": perPage,
		"hasNext":     int64(page)*int64(perPage) < total,
		"flash_error": ctx.Query("flash_error"), "flash_success": ctx.Query("flash_success"),
		"filters": filters, "role": role,
	}, "layouts/base")
}

func (c *FacturaController) invoiceFiltersForCurrentUser(ctx fiber.Ctx) (map[string]string, error) {
	user, err := c.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	filters := map[string]string{
		"estado": ctx.Query("estado"), "emisor_id": ctx.Query("emisor_id"), "receptor_id": ctx.Query("receptor_id"),
		"fecha_desde": ctx.Query("fecha_desde"), "fecha_hasta": ctx.Query("fecha_hasta"),
	}
	switch user.Role {
	case "admin":
	case "publicador":
		publicador, err := c.publicadorForUser(user)
		if err != nil {
			return nil, err
		}
		filters["publicador_id"] = strconv.FormatUint(uint64(publicador.ID), 10)
	case "chofer":
		chofer, err := c.choferForUser(user)
		if err != nil {
			return nil, err
		}
		filters["chofer_id"] = strconv.FormatUint(uint64(chofer.ID), 10)
	default:
		return nil, fiber.ErrForbidden
	}
	return filters, nil
}

func (c *FacturaController) Export(ctx fiber.Ctx) error {
	filters, err := c.invoiceFiltersForCurrentUser(ctx)
	if err != nil {
		return fiber.ErrForbidden
	}
	const pageSize = 100
	const maxExportRows = 10000
	rows, total, err := c.service.GetAllWithFilters(filters, 1, pageSize)
	if err != nil {
		log.Printf("Error preparando exportación de facturas: %v", err)
		return fiber.ErrInternalServerError
	}
	if total > maxExportRows {
		return ctx.Status(fiber.StatusRequestEntityTooLarge).SendString("La exportación supera 10000 facturas; aplica filtros de fecha o estado.")
	}
	for page := 2; int64((page-1)*pageSize) < total; page++ {
		next, _, pageErr := c.service.GetAllWithFilters(filters, page, pageSize)
		if pageErr != nil {
			log.Printf("Error paginando exportación de facturas: %v", pageErr)
			return fiber.ErrInternalServerError
		}
		rows = append(rows, next...)
	}
	data := make([][]exports.Cell, 0, len(rows))
	for _, invoice := range rows {
		publisherName, driverName := "", ""
		if invoice.Publicador != nil && invoice.Publicador.User != nil {
			publisherName = invoice.Publicador.User.Name
		}
		if invoice.Chofer != nil && invoice.Chofer.User != nil {
			driverName = invoice.Chofer.User.Name
		}
		issuerName, recipientName := "", ""
		if invoice.Emisor != nil {
			issuerName = invoice.Emisor.NombreLegal
		}
		if invoice.Receptor != nil {
			recipientName = invoice.Receptor.NombreLegal
		}
		data = append(data, []exports.Cell{
			{Value: invoice.NumeroFactura}, {Value: invoice.FechaEmision.Format("2006-01-02")},
			{Value: string(invoice.Estado)}, {Value: fmt.Sprintf("%.2f", invoice.Subtotal), Numeric: true},
			{Value: fmt.Sprintf("%.2f", invoice.Impuestos), Numeric: true}, {Value: fmt.Sprintf("%.2f", invoice.Total), Numeric: true},
			{Value: string(invoice.Moneda)}, {Value: strconv.FormatUint(uint64(invoice.CargaID), 10), Numeric: true},
			{Value: issuerName}, {Value: recipientName}, {Value: publisherName}, {Value: driverName},
		})
	}
	body, contentType, extension, err := exports.Render(exports.Report{
		Title: "Facturas BUFALO", Headers: []string{"Número", "Fecha de emisión", "Estado", "Subtotal", "Impuestos", "Total", "Moneda", "Carga ID", "Empresa emisora", "Empresa receptora", "Publicador", "Chofer"}, Rows: data,
	}, ctx.Query("format"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	filename := "bufalo-facturas-" + time.Now().Format("20060102-150405") + "." + extension
	ctx.Set("Content-Type", contentType)
	ctx.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	ctx.Set("Cache-Control", "no-store, private")
	return ctx.Send(body)
}

func (c *FacturaController) Show(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	f, err := c.service.GetByID(ctx.Params("id"))
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
	}
	if err := c.authorize(ctx, f, false); err != nil {
		return err
	}
	return ctx.Render("facturas/show", fiber.Map{"title": "Detalle Factura", "flash_error": ctx.Query("flash_error"), "flash_success": ctx.Query("flash_success"), "factura": f, "csrfToken": csrf.TokenFromContext(ctx), "role": sess.Get("role")}, "layouts/base")
}

func (c *FacturaController) Create(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	cargaID := ctx.Query("carga_id")
	var carga *models.Carga
	if cargaID != "" {
		var err error
		carga, err = c.cargaService.GetByID(cargaID)
		if err != nil {
			return fiber.ErrNotFound
		}
		draft, err := c.invoiceForCurrentIssuer(ctx, carga)
		if err != nil {
			return fiber.ErrForbidden
		}
		if err := c.authorize(ctx, draft, true); err != nil {
			return err
		}
	}
	return ctx.Render("facturas/create", fiber.Map{
		"title": "Nueva Factura", "flash_error": ctx.Query("flash_error"),
		"carga":     carga,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      sess.Get("role"),
	}, "layouts/base")
}

func (c *FacturaController) Store(ctx fiber.Ctx) error {
	var req requests.FacturaStoreRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/facturas/create?flash_error=Datos inválidos")
	}
	rules := map[string]any{
		"carga_id":       "required|integer",
		"numero_factura": "required|min:3|max:50",
		"fecha_emision":  "required",
		"subtotal":       "required|numeric|min:0",

		"moneda": "required|in:CUP,MLC,USD,EUR",
	}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil || validator.Fails() {
		return ctx.Redirect().To("/facturas/create?flash_error=Error de validación")
	}

	fechaEmision, err := time.Parse("2006-01-02", req.FechaEmision)
	if err != nil {
		return ctx.Redirect().To("/facturas/create?flash_error=Fecha de emisión inválida")
	}
	var fechaVenc *time.Time
	if req.FechaVencimiento != "" {
		if t, err := time.Parse("2006-01-02", req.FechaVencimiento); err == nil {
			fechaVenc = &t
		} else {
			return fiber.ErrBadRequest
		}
	}

	user, err := c.currentUser(ctx)
	if err != nil {
		return fiber.ErrForbidden
	}
	f := models.Factura{
		CargaID:          req.CargaID,
		EmisorTipo:       models.TipoEmisorFactura(user.Role),
		NumeroFactura:    req.NumeroFactura,
		FechaEmision:     fechaEmision,
		FechaVencimiento: fechaVenc,
		DistanciaKm:      req.DistanciaKm,
		TarifaPorKm:      req.TarifaPorKm,
		Subtotal:         req.Subtotal,
		Impuestos:        req.Impuestos,
		Total:            req.Total,
		Moneda:           models.Moneda(req.Moneda),
		Estado:           models.FacturaBorrador,
		MetodoPago:       strPtr(req.MetodoPago),
	}
	switch user.Role {
	case "publicador":
		profile, err := c.publicadorForUser(user)
		if err != nil {
			return fiber.ErrForbidden
		}
		f.PublicadorID = &profile.ID
	case "chofer":
		profile, err := c.choferForUser(user)
		if err != nil {
			return fiber.ErrForbidden
		}
		f.ChoferID = &profile.ID
	case "admin":
		load, err := c.cargaService.GetByID(strconv.FormatUint(uint64(req.CargaID), 10))
		if err != nil || load.PublicadorID == 0 {
			return fiber.ErrBadRequest
		}
		f.EmisorTipo = models.EmisorFacturaPublicador
		f.PublicadorID = &load.PublicadorID
	default:
		return fiber.ErrForbidden
	}
	if err := c.authorize(ctx, &f, true); err != nil {
		return err
	}
	if err := c.service.Create(&f); err != nil {
		log.Printf("Error creando factura: %v", err)
		return ctx.Redirect().To("/facturas/create?flash_error=Error al guardar")
	}
	return ctx.Redirect().To(fmt.Sprintf("/facturas/%d", f.ID))
}

func (c *FacturaController) Edit(ctx fiber.Ctx) error {
	sess := session.FromContext(ctx)
	f, err := c.service.GetByID(ctx.Params("id"))
	if err != nil {
		return ctx.Render("dashboard/404", fiber.Map{"title": "No encontrada", "role": sess.Get("role")}, "layouts/base")
	}
	if err := c.authorize(ctx, f, true); err != nil {
		return err
	}
	return ctx.Render("facturas/edit", fiber.Map{
		"title": "Editar Factura", "flash_error": ctx.Query("flash_error"),
		"factura":   f,
		"csrfToken": csrf.TokenFromContext(ctx),
		"role":      sess.Get("role"),
	}, "layouts/base")
}

func (c *FacturaController) Update(ctx fiber.Ctx) error {
	f, err := c.service.GetByID(ctx.Params("id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	if err := c.authorize(ctx, f, true); err != nil {
		return err
	}
	id := ctx.Params("id")
	var req requests.FacturaUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/facturas/" + id + "/edit?flash_error=Datos inválidos")
	}
	updates := map[string]interface{}{
		"distancia_km":  req.DistanciaKm,
		"tarifa_por_km": req.TarifaPorKm,
		"subtotal":      req.Subtotal,
		"impuestos":     req.Impuestos,
		"total":         req.Total,
		"moneda":        req.Moneda,
		"metodo_pago":   req.MetodoPago,
	}
	if err := c.service.Update(id, updates); err != nil {
		return ctx.Redirect().To("/facturas/" + id + "/edit?flash_error=Error al actualizar")
	}
	return ctx.Redirect().To("/facturas/" + id + "?flash_success=Actualizada")
}

func (c *FacturaController) Delete(ctx fiber.Ctx) error {
	f, err := c.service.GetByID(ctx.Params("id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	if err := c.authorize(ctx, f, true); err != nil {
		return err
	}
	if err := c.service.Delete(ctx.Params("id")); err != nil {
		return ctx.Redirect().To("/facturas?flash_error=Error al eliminar")
	}
	return ctx.Redirect().To("/facturas?flash_success=Eliminada")
}

// MarcarPagada — cambia estado a pagada
func (c *FacturaController) MarcarPagada(ctx fiber.Ctx) error {
	f, err := c.service.GetByID(ctx.Params("id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	if err := c.authorize(ctx, f, true); err != nil {
		return err
	}
	id := ctx.Params("id")
	metodo := ctx.FormValue("metodo_pago")

	if err := c.service.MarcarPagada(id, metodo); err != nil {
		return ctx.Redirect().To("/facturas/" + id + "?flash_error=Error al marcar como pagada")
	}
	return ctx.Redirect().To("/facturas/" + id + "?flash_success=Factura marcada como pagada")
}

func (c *FacturaController) currentUser(ctx fiber.Ctx) (*models.User, error) {
	sess := session.FromContext(ctx)
	if sess == nil {
		return nil, fiber.ErrForbidden
	}
	id, ok := sess.Get("user_id").(uint)
	if !ok || id == 0 {
		return nil, fiber.ErrForbidden
	}
	var user models.User
	err := facades.Orm().Query().Where("id = ?", id).First(&user)
	if err != nil || user.ID == 0 || !user.IsActive {
		return nil, fiber.ErrForbidden
	}
	return &user, nil
}
func (c *FacturaController) authorize(ctx fiber.Ctx, f *models.Factura, write bool) error {
	user, err := c.currentUser(ctx)
	if err != nil {
		return fiber.ErrForbidden
	}
	var publicadorID, choferID uint
	if user.Role == "publicador" {
		profile, err := c.publicadorForUser(user)
		if err != nil {
			return fiber.ErrForbidden
		}
		publicadorID = profile.ID
	} else if user.Role == "chofer" {
		chofer, err := c.choferForUser(user)
		if err != nil {
			return fiber.ErrForbidden
		}
		choferID = chofer.ID
	}
	if !billing.CanAccess(user.ID, user.Role, publicadorID, choferID, f, write) {
		return fiber.ErrForbidden
	}
	return nil
}

func (c *FacturaController) publicadorForUser(user *models.User) (*models.Publicador, error) {
	if user == nil || user.ID == 0 {
		return nil, fiber.ErrForbidden
	}
	return services.NewPublicadorService().GetByUserID(user.ID)
}

func (c *FacturaController) invoiceForCurrentIssuer(ctx fiber.Ctx, carga *models.Carga) (*models.Factura, error) {
	user, err := c.currentUser(ctx)
	if err != nil || carga == nil {
		return nil, fiber.ErrForbidden
	}
	f := &models.Factura{PublicadorID: &carga.PublicadorID, ChoferID: carga.ChoferID}
	switch user.Role {
	case "publicador":
		f.EmisorTipo = models.EmisorFacturaPublicador
	case "chofer":
		f.EmisorTipo = models.EmisorFacturaChofer
	case "admin":
		f.EmisorTipo = models.EmisorFacturaPublicador
	default:
		return nil, fiber.ErrForbidden
	}
	return f, nil
}

func (c *FacturaController) choferForUser(user *models.User) (*models.Chofer, error) {
	if user == nil || user.ID == 0 {
		return nil, fiber.ErrForbidden
	}
	return services.NewChoferService().GetByUserID(user.ID)
}
func (c *FacturaController) Emitir(ctx fiber.Ctx) error {
	f, err := c.service.GetByID(ctx.Params("id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	if err := c.authorize(ctx, f, true); err != nil {
		return err
	}
	if err := c.service.CambiarEstado(ctx.Params("id"), models.FacturaEmitida, ""); err != nil {
		return fiber.ErrConflict
	}
	return ctx.Redirect().To(fmt.Sprintf("/facturas/%d", f.ID))
}
