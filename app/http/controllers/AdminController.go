package controllers

import (
	"github.com/gofiber/fiber/v3"
	"goravel/app/services"
	"strings"
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

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func defaultString(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}
