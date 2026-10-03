package controllers

import (
	"fmt"
	"goravel/app/billing"
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
	user, err := c.currentUser(ctx)
	if err != nil {
		return fiber.ErrForbidden
	}
	role := user.Role
	filters := map[string]string{"estado": ctx.Query("estado"), "fecha_desde": ctx.Query("fecha_desde"), "fecha_hasta": ctx.Query("fecha_hasta")}
	if role != "admin" {
		if user.EmpresaID == nil || *user.EmpresaID == 0 {
			return fiber.ErrForbidden
		}
		if role == "publicador" {
			filters["emisor_id"] = strconv.FormatUint(uint64(*user.EmpresaID), 10)
		} else if role == "chofer" {
			filters["receptor_id"] = strconv.FormatUint(uint64(*user.EmpresaID), 10)
		} else {
			return fiber.ErrForbidden
		}
	}

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
		"title":       "Facturas",
		"facturas":    list,
		"total":       total,
		"page":        page,
		"perPage":     perPage,
		"hasNext":     int64(page)*int64(perPage) < total,
		"flash_error": ctx.Query("flash_error"), "flash_success": ctx.Query("flash_success"),
		"filters": filters,
		"role":    role,
	}, "layouts/base")

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
		if err := c.authorize(ctx, &models.Factura{EmisorID: carga.EmpresaID}, true); err != nil {
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
		"emisor_id":      "required|integer",
		"receptor_id":    "required|integer",
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

	f := models.Factura{
		CargaID:          req.CargaID,
		EmisorID:         req.EmisorID,
		ReceptorID:       req.ReceptorID,
		ChoferID:         req.ChoferID,
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
	if err != nil || !billing.CanAccess(user.ID, user.Role, user.EmpresaID, f, write) {
		return fiber.ErrForbidden
	}
	return nil
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
