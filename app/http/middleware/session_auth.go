package middleware

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// SessionAuth valida la sesión y asigna user_id al contexto
func SessionAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		log.Println("🔐 [SessionAuth] Validando sesión")

		sess := session.FromContext(c)
		if sess == nil {
			log.Println("❌ [SessionAuth] sesión es nil")
			return c.Redirect().To("/login")
		}
		log.Println("✅ [SessionAuth] sesión obtenida correctamente")

		userID := sess.Get("user_id")
		if userID == nil {
			log.Println("❌ [SessionAuth] user_id no encontrado")
			return c.Redirect().To("/login")
		}

		userIDUint, ok := userID.(uint)
		if !ok {
			log.Printf("❌ [SessionAuth] user_id no es uint, es %T", userID)
			return c.Redirect().To("/login")
		}

		log.Printf("✅ [SessionAuth] user_id = %v", userIDUint)
		c.Locals("user_id", userIDUint)
		return c.Next()
	}
}
