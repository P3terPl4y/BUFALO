package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"goravel/app/community"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/requests"
	"goravel/app/services"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
)

type EmpresaController struct {
	service          *services.EmpresaService
	direccionService *services.DireccionService
}

func NewEmpresaController() *EmpresaController {
	return &EmpresaController{
		service:          services.NewEmpresaService(),
		direccionService: services.NewDireccionService(),
	}
}

// canEditEmpresa devuelve true si el usuario puede modificar la empresa.
// Regla: admin siempre; si no, solo el OwnerID.
func canEditEmpresa(empresa *models.Empresa, userID uint, role string) bool {
	if role == "admin" {
		return true
	}
	if empresa == nil || empresa.OwnerID == nil {
		return false
	}
	return *empresa.OwnerID == userID
}

// currentUser lee user_id y role de la sesión una sola vez.
func currentUser(ctx fiber.Ctx) (uint, string) {
	sess := session.FromContext(ctx)
	userID, _ := sess.Get("user_id").(uint)
	role, _ := sess.Get("role").(string)
	return userID, role
}

func canUseCompanyAddress(direccion *models.Direccion, userID uint, role string) bool {
	if direccion == nil {
		return true // company address is optional
	}
	return canManageAddress(direccion, userID, role)
}

func optionalAddressID(addressID *uint) *uint {
	if addressID == nil || *addressID == 0 {
		return nil
	}
	return addressID
}

func saveEmpresaPhoto(ctx fiber.Ctx) (string, error) {
	file, err := ctx.FormFile("profile_photo")
	if err != nil || file == nil {
		return "", nil
	}
	if file.Size <= 0 || file.Size > community.MaxProfilePhotoBytes {
		return "", errors.New("la imagen debe pesar menos de 5 MB")
	}
	input, err := file.Open()
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(io.LimitReader(input, community.MaxProfilePhotoBytes+1))
	_ = input.Close()
	if err != nil {
		return "", err
	}
	ext, err := community.ValidateProfilePhoto(data)
	if err != nil {
		return "", err
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	filename := hex.EncodeToString(random[:]) + ext
	dir := filepath.Join("public", "uploads", "company-logos")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		return "", err
	}
	return "/uploads/company-logos/" + filename, nil
}

func removeEmpresaPhoto(photoURL string) {
	if !strings.HasPrefix(photoURL, "/uploads/company-logos/") {
		return
	}
	name := filepath.Base(photoURL)
	if name == "." || name == string(filepath.Separator) {
		return
	}
	_ = os.Remove(filepath.Join("public", "uploads", "company-logos", name))
}

func (c *EmpresaController) companyAddresses(userID uint, role string) []models.Direccion {
	var addresses []models.Direccion
	var err error
	if role == "admin" {
		addresses, err = c.direccionService.GetAll()
	} else {
		addresses, err = c.direccionService.GetOwnedByUserID(userID)
	}
	if err != nil {
		log.Printf("Error obteniendo direcciones disponibles para empresa: %v", err)
		return []models.Direccion{}
	}
	return addresses
}

func (c *EmpresaController) addressCanBeAssigned(addressID *uint, userID uint, role string) bool {
	if addressID == nil || *addressID == 0 {
		return true
	}
	direccion, err := c.direccionService.GetByID(strconv.FormatUint(uint64(*addressID), 10))
	if services.IsInfrastructureError(err) {
		return false
	}
	return err == nil && canUseCompanyAddress(direccion, userID, role)
}

// ─────────────────────────────────────────────────────────────
// Index
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Index(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)
	browse := ctx.Query("disponibles") == "1"
	if role == "admin" {
		browse = false
	}
	filters := map[string]string{
		"tipo":   ctx.Query("tipo"),
		"estado": ctx.Query("estado"),
		"q":      ctx.Query("q"),
		"role":   role,
	}
	hasSearch := strings.TrimSpace(filters["q"]) != "" || strings.TrimSpace(filters["tipo"]) != ""
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page", "10"))
	page, perPage = services.NormalizePagination(page, perPage)
	ownsCompany := false
	if role != "admin" {
		var checkErr error
		ownsCompany, checkErr = facades.Orm().Query().Model(&models.Empresa{}).Where("owner_id = ? AND deleted_at IS NULL", userID).Exists()
		if checkErr != nil {
			return fiber.ErrServiceUnavailable
		}
	}

	var list []models.Empresa
	var total int64
	var err error
	if role == "admin" {
		list, total, err = c.service.GetAllWithFilters(filters, page, perPage)
	} else if browse {
		if ownsCompany {
			// Owners manage their own companies; they are never offered affiliation actions.
			browse = false
			list, total, err = c.service.GetCompaniesForUser(userID, page, perPage)
		} else if hasSearch {
			list, total, err = c.service.GetAvailableCompanies(userID, filters, page, perPage)
		}
	} else {
		list, total, err = c.service.GetCompaniesForUser(userID, page, perPage)
	}
	if err != nil {
		log.Printf("Error listando empresas: %v", err)
		return fiber.ErrServiceUnavailable
	}
	for i := range list {
		list[i].CanManage = canEditEmpresa(&list[i], userID, role)
		if role != "admin" {
			access, accessErr := services.GetCompanyChatAccess(list[i].ID, userID)
			list[i].CanChat = accessErr == nil && access.Member
			if browse && list[i].CanChat {
				// Defensive: associated companies never belong in browse results.
				list[i].CanRequestMembership = false
			}
			if browse && !list[i].CanChat && role != "admin" {
				eligible, eligibleErr := services.CanRequestCompanyMembership(userID, list[i].ID)
				if eligibleErr != nil && services.IsInfrastructureError(eligibleErr) {
					return fiber.ErrServiceUnavailable
				}
				list[i].CanRequestMembership = eligible
			}
		}
	}
	if browse {
		pending := make(map[uint]bool)
		var rows []services.CompanyMembershipRequest
		if err := facades.Orm().Query().Where("user_id = ? AND status = ?", userID, "pending").Limit(10).Find(&rows); err != nil {
			return fiber.ErrServiceUnavailable
		}
		for _, row := range rows {
			pending[row.EmpresaID] = true
		}
		for i := range list {
			list[i].MembershipPending = pending[list[i].ID]
		}
	}

	return ctx.Render("empresas/index", fiber.Map{
		"title":       "Empresas",
		"empresas":    list,
		"total":       total,
		"page":        page,
		"perPage":     perPage,
		"filters":     filters,
		"role":        role,
		"browse":      browse,
		"ownsCompany": ownsCompany,
		"hasSearch":   hasSearch,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Show
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Show(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	e, err := c.service.GetByID(ctx.Params("id"))
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).Render("dashboard/404", fiber.Map{
			"title": "No encontrada",
			"role":  role,
		}, "layouts/base")
	}

	driverPage, _ := strconv.Atoi(ctx.Query("driver_page", "1"))
	driverPage, _ = services.NormalizePagination(driverPage, 100)
	members, err := c.service.GetCompanyMembers(ctx.Params("id"), driverPage)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	chatAccess, chatErr := services.GetCompanyChatAccess(e.ID, userID)
	if chatErr != nil && !errors.Is(chatErr, services.ErrCompanyChatForbidden) && !errors.Is(chatErr, services.ErrNotFound) {
		return fiber.ErrServiceUnavailable
	}
	canManage := canEditEmpresa(e, userID, role)
	isOwner := e.OwnerID != nil && *e.OwnerID == userID
	canRequestMembership := false
	if !canManage && (role == "chofer" || role == "publicador") {
		canRequestMembership, err = services.CanRequestCompanyMembership(userID, e.ID)
		if err != nil && services.IsInfrastructureError(err) {
			return fiber.ErrServiceUnavailable
		}
	}
	var membershipRequests []services.CompanyMembershipRequest
	if canManage {
		if err := facades.Orm().Query().With("User").Where("empresa_id = ? AND status = ?", e.ID, "pending").Order("created_at").Limit(100).Find(&membershipRequests); err != nil {
			return fiber.ErrServiceUnavailable
		}
	}

	return ctx.Render("empresas/show", fiber.Map{
		"title":          "Detalle Empresa",
		"driverPrevious": driverPage - 1, "driverNext": driverPage + 1, "driverHasNext": len(members) == 100,
		"empresa":              e,
		"members":              members,
		"userID":               userID,
		"csrfToken":            csrf.TokenFromContext(ctx),
		"role":                 role,
		"canChat":              chatErr == nil && chatAccess.Member,
		"canManage":            canManage,
		"isOwner":              isOwner,
		"canRequestMembership": canRequestMembership,
		"membershipRequests":   membershipRequests,
	}, "layouts/base")
}

func (c *EmpresaController) DecideMembership(ctx fiber.Ctx) error {
	uid, ok := ctx.Locals("user_id").(uint)
	if !ok || uid == 0 {
		return fiber.ErrUnauthorized
	}
	id, err := strconv.ParseUint(ctx.Params("requestID"), 10, 32)
	if err != nil || id == 0 {
		return fiber.ErrBadRequest
	}
	companyID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil || companyID == 0 {
		return fiber.ErrBadRequest
	}
	decision := ctx.FormValue("decision")
	if decision != "approve" && decision != "reject" {
		return fiber.ErrBadRequest
	}
	var request services.CompanyMembershipRequest
	if err := facades.Orm().Query().Where("id = ? AND empresa_id = ?", id, companyID).First(&request); err != nil {
		return fiber.ErrNotFound
	}
	if err := services.DecideCompanyMembership(uid, uint(id), decision == "approve"); err != nil {
		if errors.Is(err, services.ErrMembershipDenied) {
			return fiber.ErrForbidden
		}
		return fiber.ErrServiceUnavailable
	}
	return ctx.Redirect().To("/empresas/" + strconv.FormatUint(companyID, 10) + "?flash_success=Solicitud+resuelta")
}

// ─────────────────────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Create(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	dests := c.companyAddresses(userID, role)

	return ctx.Render("empresas/create", fiber.Map{
		"title":        "Nueva Empresa",
		"destinations": dests,
		"csrfToken":    csrf.TokenFromContext(ctx),
		"role":         role,
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Store
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Store(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	var req requests.EmpresaStoreRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/empresas/create?flash_error=Datos inválidos")
	}

	// ── Regla de tipo según rol ──
	// admin → cualquier tipo
	// publicador → solo broker
	// chofer → solo carrier
	if role != "admin" {
		expected := "broker"
		if role == "chofer" {
			expected = "carrier"
		}
		if req.Tipo != expected {
			return ctx.Redirect().To(fmt.Sprintf(
				"/empresas/create?flash_error=Como %s solo puedes crear empresas tipo %s",
				role, expected,
			))
		}
	}

	rules := map[string]any{
		"tipo":         "required|in:broker,carrier,shipper,factoring,mixto",
		"nombre_legal": "required|min:3|max:255",
		"estado":       "required|in:activo,inactivo,suspendido",
	}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil || validator.Fails() {
		return ctx.Redirect().To("/empresas/create?flash_error=Error de validación")
	}
	photoURL, err := saveEmpresaPhoto(ctx)
	if err != nil {
		return ctx.Redirect().To("/empresas/create?flash_error=Foto+inv%C3%A1lida:+usa+JPEG+o+PNG+de+hasta+5+MB")
	}
	if !c.addressCanBeAssigned(req.DireccionID, userID, role) {
		removeEmpresaPhoto(photoURL)
		return ctx.Redirect().To("/empresas/create?flash_error=La+direcci%C3%B3n+no+te+pertenece")
	}

	// ── OwnerID = quien la crea ──
	ownerID := userID

	e := models.Empresa{
		Tipo:            models.TipoEmpresa(req.Tipo),
		NombreLegal:     req.NombreLegal,
		NombreComercial: strPtr(req.NombreComercial),
		TaxID:           strPtr(req.TaxID),
		MCNumber:        strPtr(req.MCNumber),
		DOTNumber:       strPtr(req.DOTNumber),
		Telefono:        strPtr(req.Telefono),
		Email:           strPtr(req.Email),
		SitioWeb:        strPtr(req.SitioWeb),
		DireccionID:     optionalAddressID(req.DireccionID),
		CreditScore:     req.CreditScore,
		DaysToPay:       &req.DaysToPay,
		OwnerID:         &ownerID,
		Estado:          models.EstadoEmpresa(req.Estado),
		ProfilePhoto:    photoURL,
	}

	if err := c.service.Create(&e); err != nil {
		removeEmpresaPhoto(photoURL)
		log.Printf("Error creando empresa: %v", err)
		return ctx.Redirect().To("/empresas/create?flash_error=Error al guardar")
	}
	return ctx.Redirect().To(fmt.Sprintf("/empresas/%d", e.ID))
}

// ─────────────────────────────────────────────────────────────
// Edit
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Edit(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	id := ctx.Params("id")

	// 1. Cargar la entidad PRIMERO
	e, err := c.service.GetByID(id)
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/empresas?flash_error=Empresa no encontrada")
	}

	// 2. Verificar ownership
	if !canEditEmpresa(e, userID, role) {
		return ctx.Redirect().To("/empresas?flash_error=No autorizado")
	}

	dests := c.companyAddresses(userID, role)

	return ctx.Render("empresas/edit", fiber.Map{
		"title":         "Editar Empresa",
		"empresa":       e,
		"destinations":  dests,
		"userID":        userID,
		"csrfToken":     csrf.TokenFromContext(ctx),
		"role":          role,
		"flash_error":   ctx.Query("flash_error"),
		"flash_success": ctx.Query("flash_success"),
	}, "layouts/base")
}

// ─────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Update(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	id := ctx.Params("id")

	// 1. Cargar
	existing, err := c.service.GetByID(id)
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/empresas?flash_error=Empresa no encontrada")
	}

	// 2. Verificar ownership
	if !canEditEmpresa(existing, userID, role) {
		return ctx.Redirect().To("/empresas?flash_error=No autorizado")
	}

	// 3. Bind
	var req requests.EmpresaUpdateRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Redirect().To("/empresas/" + id + "/edit?flash_error=Datos inválidos")
	}
	if role != "admin" {
		expected := "broker"
		if role == "chofer" {
			expected = "carrier"
		}
		if req.Tipo != expected {
			return ctx.Redirect().To("/empresas/" + id + "/edit?flash_error=Tipo+de+empresa+no+permitido+para+tu+rol")
		}
	}
	if !c.addressCanBeAssigned(req.DireccionID, userID, role) {
		return ctx.Redirect().To("/empresas/" + id + "/edit?flash_error=La+direcci%C3%B3n+no+te+pertenece")
	}

	// 4. Validación
	rules := map[string]any{
		"tipo":         "required|in:broker,carrier,shipper,factoring,mixto",
		"nombre_legal": "required|min:3|max:255",
		"estado":       "required|in:activo,inactivo,suspendido",
	}
	validator, err := facades.Validation().Make(ctx.Context(), req, rules)
	if err != nil || validator.Fails() {
		return ctx.Redirect().To("/empresas/" + id + "/edit?flash_error=Error de validación")
	}

	// 5. Whitelist de updates (NO se toca OwnerID, ni ID, ni CreatedAt)
	updates := map[string]interface{}{
		"tipo":             req.Tipo,
		"nombre_legal":     req.NombreLegal,
		"nombre_comercial": strPtr(req.NombreComercial),
		"tax_id":           strPtr(req.TaxID),
		"mc_number":        strPtr(req.MCNumber),
		"dot_number":       strPtr(req.DOTNumber),
		"direccion_id":     optionalAddressID(req.DireccionID),
		"telefono":         strPtr(req.Telefono),
		"email":            strPtr(req.Email),
		"sitio_web":        strPtr(req.SitioWeb),
		"credit_score":     req.CreditScore,
		"days_to_pay":      req.DaysToPay,
		"estado":           req.Estado,
	}
	photoURL, photoErr := saveEmpresaPhoto(ctx)
	if photoErr != nil {
		return ctx.Redirect().To("/empresas/" + id + "/edit?flash_error=Foto+inv%C3%A1lida:+usa+JPEG+o+PNG+de+hasta+5+MB")
	}
	if photoURL != "" {
		updates["profile_photo"] = photoURL
	}
	if err := c.service.Update(id, updates); err != nil {
		removeEmpresaPhoto(photoURL)
		log.Printf("Error actualizando empresa %s: %v", id, err)
		return ctx.Redirect().To("/empresas/" + id + "/edit?flash_error=Error al actualizar")
	}
	if photoURL != "" {
		removeEmpresaPhoto(existing.ProfilePhoto)
	}
	return ctx.Redirect().To("/empresas/" + id + "?flash_success=Empresa actualizada")
}

// ─────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────
func (c *EmpresaController) Delete(ctx fiber.Ctx) error {
	userID, role := currentUser(ctx)

	id := ctx.Params("id")

	// 1. Cargar
	e, err := c.service.GetByID(id)
	if services.IsInfrastructureError(err) {
		return fiber.ErrServiceUnavailable
	}
	if err != nil {
		return ctx.Redirect().To("/empresas?flash_error=Empresa no encontrada")
	}

	// 2. Verificar ownership
	if !canEditEmpresa(e, userID, role) {
		return ctx.Redirect().To("/empresas?flash_error=No autorizado")
	}

	// 3. Eliminar
	if err := c.service.Delete(id); err != nil {
		log.Printf("Error eliminando empresa %s: %v", id, err)
		return ctx.Redirect().To("/empresas?flash_error=Error al eliminar")
	}
	removeEmpresaPhoto(e.ProfilePhoto)
	return ctx.Redirect().To("/empresas?flash_success=Empresa eliminada")
}
