package middleware

import "github.com/gofiber/fiber/v3"

// IsActiveUserHandler remains for route compatibility; SessionAuth verifies
// the authoritative account state in the database before this middleware runs.
func IsActiveUserHandler() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if active, _ := ctx.Locals("is_active").(bool); active {
			return ctx.Next()
		}
		return ctx.Redirect().To("/login?flash_error=Usuario deshabilitado")
	}
}
