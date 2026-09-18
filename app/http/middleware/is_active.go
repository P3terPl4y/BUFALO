package middleware

import (
	"log"
"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3"
)

func IsActiveUserHandler() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		isActive,_ := session.FromContext(ctx).Get("is_active").(bool)
		log.Printf("Esta activo el usuario %v ?",isActive)
		if isActive{
			return ctx.Next()
		}
		return ctx.Redirect().To("/login?flash_error=Usuario deshabilitado") 
	}
}
