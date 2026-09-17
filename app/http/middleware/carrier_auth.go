package middleware

import (
	"goravel/app/facades"
	"goravel/app/models"
	"log"

	"github.com/gofiber/fiber/v3"
)

// CarrierAuth valida que el usuario autenticado tenga rol "carrier"
func CarrierAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(uint)
		if !ok {
			log.Println("❌ [CarrierAuth] user_id no encontrado")
			return c.Redirect().To("/login")
		}

		var user models.User
		if err := facades.Orm().Query().Where("id = ?", userID).First(&user); err != nil {
			log.Printf("❌ [CarrierAuth] Usuario no encontrado: %v", err)
			return c.Redirect().To("/login")
		}

		if user.Role != "carrier" {
			log.Printf("❌ [CarrierAuth] Usuario %s no es carrier, redirigiendo a /home", user.Email)
			return c.Redirect().To("/home")
		}

		log.Printf("✅ [CarrierAuth] Usuario carrier autorizado: %s", user.Email)
		return c.Next()
	}
}
