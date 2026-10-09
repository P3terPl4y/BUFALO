package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"goravel/app/services"
)

type NotificationController struct{}

func (NotificationController) Index(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return fiber.ErrUnauthorized
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	rows, total, err := services.ListUserNotifications(userID, page)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	unread, err := services.UnreadUserNotificationCount(userID)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	page, _ = services.NormalizePagination(page, 30)
	ctx.Set("Cache-Control", "no-store, private")
	return ctx.Render("notifications/index", fiber.Map{"title": "Notificaciones", "notifications": rows, "total": total, "unreadCount": unread, "page": page, "nextPage": page + 1, "hasNext": int64(page*30) < total, "role": ctx.Locals("role"), "csrfToken": csrf.TokenFromContext(ctx)}, "layouts/base")
}

func (NotificationController) Read(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return fiber.ErrUnauthorized
	}
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return fiber.ErrNotFound
	}
	if err := services.MarkUserNotificationRead(userID, uint(id)); err != nil {
		return fiber.ErrServiceUnavailable
	}
	return ctx.Redirect().To("/notifications")
}
