package middleware

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

func ErrorHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			log.Printf("❌ Error: %v", err)
			// En producción, no mostrar detalles internos
			return c.Status(500).JSON(fiber.Map{
				"error": "Ha ocurrido un error interno",
			})
		}
		return nil
	}
}
