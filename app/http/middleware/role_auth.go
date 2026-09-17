package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// RoleAuth devuelve un middleware que solo deja pasar si el rol de la
// sesión está en la lista permitida. Asume que SessionAuth ya corrió.
func RoleAuth(allowed ...string) fiber.Handler {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}

	return func(ctx fiber.Ctx) error {
		sess := session.FromContext(ctx)
		if sess == nil {
			return ctx.Redirect().To("/login")
		}

		role, _ := sess.Get("role").(string)
		if role == "" {
			// Fallback: Locals (por si SessionAuth lo puso ahí)
			role, _ = ctx.Locals("role").(string)
		}

		if _, ok := allowedSet[role]; !ok {
			return ctx.Redirect().To("/home?flash_error=No autorizado")
		}
		return ctx.Next()
	}
}

// Atajos semánticos por rol.
func PublicadorAuth() fiber.Handler {
	return RoleAuth("publicador", "admin")
}

func ChoferAuth() fiber.Handler {
	return RoleAuth("chofer", "admin")
}

func AdminAuth() fiber.Handler {
	return RoleAuth("admin")
}
