package controllers

import (
	"github.com/gofiber/fiber/v3"
	"goravel/app/services"
	"net/url"
	"strconv"
)

type LoadInterestController struct{ sender services.InterestSender }

func NewLoadInterestController() *LoadInterestController {
	return &LoadInterestController{}
}
func NewLoadInterestControllerWithSender(sender services.InterestSender) *LoadInterestController {
	return &LoadInterestController{sender: sender}
}
func (c *LoadInterestController) SendInterest(ctx fiber.Ctx) error {
	uid, ok := ctx.Locals("user_id").(uint)
	if !ok || uid == 0 {
		return fiber.ErrUnauthorized
	}
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return fiber.ErrBadRequest
	}
	if err := services.SendLoadInterest(uid, uint(id), ctx.FormValue("comment"), c.sender); err != nil {
		if !services.IsPublicInterestError(err) {
			return fiber.ErrServiceUnavailable
		}
		return ctx.Redirect().To("/home?flash_error=" + url.QueryEscape(err.Error()))
	}
	return ctx.Redirect().To("/home?flash_success=" + url.QueryEscape("Tu comentario fue registrado para enviarlo al publicador de la carga"))
}
