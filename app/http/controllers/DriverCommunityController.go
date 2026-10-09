package controllers

import (
	"goravel/app/services"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
)

type DriverCommunityController struct {
	choferes     *services.ChoferService
	publicadores *services.PublicadorService
	community    *services.DriverCommunityService
}

func NewDriverCommunityController() *DriverCommunityController {
	return &DriverCommunityController{choferes: services.NewChoferService(), publicadores: services.NewPublicadorService(), community: services.NewDriverCommunityService()}
}

func (c *DriverCommunityController) companyID(ctx fiber.Ctx) (uint, error) {
	uid, ok := ctx.Locals("user_id").(uint)
	if !ok || uid == 0 {
		return 0, fiber.ErrUnauthorized
	}
	role, _ := ctx.Locals("role").(string)
	if role != "publicador" {
		return 0, fiber.ErrForbidden
	}
	profile, err := c.publicadores.GetByUserID(uid)
	if services.IsInfrastructureError(err) {
		return 0, fiber.ErrServiceUnavailable
	}
	if err != nil || profile.EmpresaID == 0 {
		return 0, fiber.ErrForbidden
	}
	return profile.EmpresaID, nil
}

func (c *DriverCommunityController) Index(ctx fiber.Ctx) error {
	companyID, err := c.companyID(ctx)
	if err != nil {
		return err
	}
	memberPage, _ := strconv.Atoi(ctx.Query("member_page", "1"))
	memberPage, _ = services.NormalizePagination(memberPage, 100)
	driverPage, _ := strconv.Atoi(ctx.Query("page", "1"))
	driverPage, _ = services.NormalizePagination(driverPage, 100)
	members, err := c.community.ListCompanyNetwork(companyID, memberPage)
	if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}
	memberIDs := make(map[uint]bool, len(members))
	for _, member := range members {
		memberIDs[member.ChoferID] = true
	}
	filters := map[string]string{"q": ctx.Query("q"), "estado": "disponible", "orden": "puntaje", "active_user": "true"}
	list, total, err := c.choferes.GetAllWithFilters(filters, driverPage, 100)
	if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}
	memberIDs, err = c.community.CompanyMemberIDs(companyID, list)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	return ctx.Render("community/drivers", fiber.Map{"memberPrevious": memberPage - 1, "memberNext": memberPage + 1, "memberHasNext": len(members) == 100, "previousPage": driverPage - 1, "nextPage": driverPage + 1, "hasNext": int64(driverPage*100) < total, "title": "Mi red de choferes", "role": "publicador", "csrfToken": csrf.TokenFromContext(ctx), "drivers": list, "members": members, "memberIDs": memberIDs, "total": total, "query": filters["q"]}, "layouts/base")
}

func (c *DriverCommunityController) Add(ctx fiber.Ctx) error {
	companyID, err := c.companyID(ctx)
	if err != nil {
		return ctx.SendStatus(fiber.StatusForbidden)
	}
	driverID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil || driverID == 0 {
		return ctx.Redirect().To("/red-choferes?flash_error=Chofer+inv%C3%A1lido")
	}
	if err := c.community.AddToCompanyNetwork(companyID, uint(driverID)); err != nil {
		return ctx.Redirect().To("/red-choferes?flash_error=No+se+pudo+a%C3%B1adir+el+chofer")
	}
	return ctx.Redirect().To("/red-choferes?flash_success=Chofer+a%C3%B1adido+a+la+red")
}

func (c *DriverCommunityController) Remove(ctx fiber.Ctx) error {
	companyID, err := c.companyID(ctx)
	if err != nil {
		return ctx.SendStatus(fiber.StatusForbidden)
	}
	driverID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil || driverID == 0 {
		return ctx.Redirect().To("/red-choferes?flash_error=Chofer+inv%C3%A1lido")
	}
	if err := c.community.RemoveFromCompanyNetwork(companyID, uint(driverID)); err != nil {
		return ctx.Redirect().To("/red-choferes?flash_error=No+se+pudo+quitar+el+chofer")
	}
	return ctx.Redirect().To("/red-choferes?flash_success=Chofer+quitado+de+la+red")
}
