package controllers

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"goravel/app/facades"
	"goravel/app/services"
	"net/url"
	"strconv"
)

func (c *EmpresaController) RequestMembership(ctx fiber.Ctx) error {
	uid, ok := ctx.Locals("user_id").(uint)
	if !ok || uid == 0 {
		return fiber.ErrUnauthorized
	}
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return fiber.ErrBadRequest
	}
	if err := services.RequestCompanyMembership(uid, uint(id)); err != nil {
		if errors.Is(err, services.ErrMembershipDenied) || errors.Is(err, services.ErrNotFound) {
			message := "No se pudo solicitar la afiliación. Comprueba que tu perfil y la empresa estén activos."
			switch {
			case errors.Is(err, services.ErrMembershipAlreadyAssociated):
				message = "Tu perfil ya está asociado a una empresa."
			case errors.Is(err, services.ErrMembershipPending):
				message = "Ya tienes una solicitud de afiliación pendiente."
			case errors.Is(err, services.ErrMembershipRequestLimit):
				message = "Alcanzaste el límite de cinco solicitudes en 24 horas. Inténtalo más tarde."
			case errors.Is(err, services.ErrMembershipWrongCompanyType):
				message = "El tipo de empresa no corresponde a tu perfil."
			case errors.Is(err, services.ErrMembershipCompanyOwner):
				message = "Como propietario, administra las empresas que creaste desde Mis empresas; no puedes afiliarte a otra empresa."
			}
			return ctx.Redirect().To("/empresas/" + strconv.FormatUint(id, 10) + "?flash_error=" + url.QueryEscape(message))
		}
		return fiber.ErrServiceUnavailable
	}
	return ctx.Redirect().To("/empresas/" + strconv.FormatUint(id, 10) + "?flash_success=Solicitud+registrada.+Un+administrador+debe+aprobarla.")
}
func (c *AdminController) MembershipRequests(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	page, _ = services.NormalizePagination(page, 50)
	var rows []services.CompanyMembershipRequest
	if err := facades.Orm().Query().Where("status = ?", "pending").Order("id").Offset((page - 1) * 50).Limit(50).Find(&rows); err != nil {
		return fiber.ErrServiceUnavailable
	}
	return ctx.Render("admin/memberships", fiber.Map{"requests": rows, "page": page, "next": page + 1, "previous": page - 1, "hasNext": len(rows) == 50, "csrfToken": csrf.TokenFromContext(ctx), "role": "admin", "title": "Solicitudes de asociación"}, "layouts/base")
}
func (c *AdminController) DecideMembership(ctx fiber.Ctx) error {
	uid, ok := ctx.Locals("user_id").(uint)
	if !ok || uid == 0 {
		return fiber.ErrUnauthorized
	}
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return fiber.ErrBadRequest
	}
	decision := ctx.FormValue("decision")
	if decision != "approve" && decision != "reject" {
		return fiber.ErrBadRequest
	}
	if err := services.DecideCompanyMembership(uid, uint(id), decision == "approve"); err != nil {
		if errors.Is(err, services.ErrMembershipDenied) {
			return fiber.ErrConflict
		}
		return fiber.ErrServiceUnavailable
	}
	return ctx.Redirect().To("/admin/memberships")
}
