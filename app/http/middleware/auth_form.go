package middleware

import (
	"mime"

	"github.com/gofiber/fiber/v3"
)

// AuthFormOnly limita el tamaño y evita discrepancias entre el campo que se
// limita por tasa y el que procesa el controlador (JSON, query o duplicados).
func AuthFormOnly(c fiber.Ctx) error {
	typeName, _, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	if err != nil || typeName != "application/x-www-form-urlencoded" {
		return fiber.ErrUnsupportedMediaType
	}
	if len(c.Body()) > 16<<10 {
		return fiber.ErrRequestEntityTooLarge
	}
	for _, key := range []string{"email", "password", "role", "empresa_mode", "empresa_id", "token", "_csrf"} {
		if len(c.Request().PostArgs().PeekMulti(key)) > 1 || c.Request().URI().QueryArgs().Has(key) {
			return fiber.ErrBadRequest
		}
	}
	return c.Next()
}
