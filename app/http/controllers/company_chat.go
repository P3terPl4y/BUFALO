package controllers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/viewhelpers"
)

type CompanyChatController struct{}

type companyChatMessageResponse struct {
	ID         uint      `json:"id"`
	UserID     uint      `json:"user_id"`
	UserName   string    `json:"user_name"`
	UserPhoto  string    `json:"user_photo,omitempty"`
	ProfileURL string    `json:"profile_url,omitempty"`
	Message    string    `json:"message"`
	Moderated  bool      `json:"moderated"`
	CreatedAt  time.Time `json:"created_at"`
}

func companyChatMessagesResponse(rows []models.EmpresaChatMensaje) []companyChatMessageResponse {
	response := make([]companyChatMessageResponse, 0, len(rows))
	for _, row := range rows {
		name := "Miembro"
		if row.User != nil && row.User.Name != "" {
			name = row.User.Name
		}
		photo, profileURL := "", ""
		if row.User != nil {
			photo = row.User.ProfilePhoto
			profileURL = viewhelpers.UserProfileURL(row.User)
		}
		moderated := row.ModeradoEn != nil
		message := row.Mensaje
		if moderated {
			message = "Este mensaje fue eliminado por el administrador de la empresa."
		}
		response = append(response, companyChatMessageResponse{ID: row.ID, UserID: row.UserID, UserName: name, UserPhoto: photo, ProfileURL: profileURL, Message: message, Moderated: moderated, CreatedAt: row.CreatedAt})
	}
	return response
}

func companyChatParams(ctx fiber.Ctx) (uint, uint, error) {
	companyID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil || companyID == 0 {
		return 0, 0, fiber.ErrNotFound
	}
	userID, ok := ctx.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return 0, 0, fiber.ErrUnauthorized
	}
	return uint(companyID), userID, nil
}

func companyChatDenied(err error) bool {
	return errors.Is(err, services.ErrCompanyChatForbidden) || errors.Is(err, services.ErrNotFound)
}

func (CompanyChatController) Show(ctx fiber.Ctx) error {
	companyID, userID, err := companyChatParams(ctx)
	if err != nil {
		return err
	}
	access, err := services.GetCompanyChatAccess(companyID, userID)
	if err != nil {
		if companyChatDenied(err) {
			return fiber.ErrNotFound
		}
		return fiber.ErrServiceUnavailable
	}
	company, err := services.NewEmpresaService().GetByID(strconv.FormatUint(uint64(companyID), 10))
	if err != nil || company == nil {
		return fiber.ErrServiceUnavailable
	}
	rows, err := services.ListCompanyChatMessages(companyID, userID)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	ctx.Set("Cache-Control", "no-store, private")
	return ctx.Render("empresas/chat", fiber.Map{
		"title": "Chat de " + company.NombreLegal, "empresa": company, "messages": companyChatMessagesResponse(rows),
		"moderator": access.Moderator, "userID": userID, "csrfToken": csrf.TokenFromContext(ctx),
		"flash_error": ctx.Query("flash_error"), "flash_success": ctx.Query("flash_success"),
	}, "layouts/base")
}

func (CompanyChatController) Messages(ctx fiber.Ctx) error {
	companyID, userID, err := companyChatParams(ctx)
	if err != nil {
		return err
	}
	rows, err := services.ListCompanyChatMessages(companyID, userID)
	if err != nil {
		if companyChatDenied(err) {
			return fiber.ErrNotFound
		}
		return fiber.ErrServiceUnavailable
	}
	ctx.Set("Cache-Control", "no-store, private")
	return ctx.JSON(fiber.Map{"messages": companyChatMessagesResponse(rows)})
}

func (CompanyChatController) Send(ctx fiber.Ctx) error {
	companyID, userID, err := companyChatParams(ctx)
	if err != nil {
		return err
	}
	if err := services.SendCompanyChatMessage(companyID, userID, ctx.FormValue("message")); err != nil {
		if companyChatDenied(err) {
			return fiber.ErrNotFound
		}
		if errors.Is(err, services.ErrCompanyChatMessage) {
			return ctx.Redirect().To("/empresas/" + strconv.FormatUint(uint64(companyID), 10) + "/chat?flash_error=El+mensaje+debe+tener+entre+1+y+1000+caracteres")
		}
		return fiber.ErrServiceUnavailable
	}
	return ctx.Redirect().To("/empresas/" + strconv.FormatUint(uint64(companyID), 10) + "/chat?flash_success=Mensaje+enviado")
}

func (CompanyChatController) Moderate(ctx fiber.Ctx) error {
	companyID, userID, err := companyChatParams(ctx)
	if err != nil {
		return err
	}
	messageID, err := strconv.ParseUint(ctx.Params("messageID"), 10, 32)
	if err != nil || messageID == 0 {
		return fiber.ErrNotFound
	}
	if err := services.ModerateCompanyChatMessage(companyID, uint(messageID), userID); err != nil {
		if errors.Is(err, services.ErrCompanyChatForbidden) {
			return fiber.ErrForbidden
		}
		if errors.Is(err, services.ErrCompanyChatMessageMissing) || errors.Is(err, services.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return fiber.ErrServiceUnavailable
	}
	return ctx.Redirect().To("/empresas/" + strconv.FormatUint(uint64(companyID), 10) + "/chat?flash_success=Mensaje+moderado")
}
