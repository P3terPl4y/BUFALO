package middleware

import (
	"log"
	"os"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
)

// JWTAuth es el middleware de autenticación JWT para la API
func JWTAuth() fiber.Handler {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "tu-secreto-super-seguro-cambiar-en-produccion"
		log.Println("⚠️ [JWTAuth] Usando JWT_SECRET por defecto (inseguro)")
	} else {
		log.Println("✅ [JWTAuth] JWT_SECRET cargado desde .env")
	}

	log.Println("🔐 [JWTAuth] Middleware JWT inicializado")

	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{
			Key: []byte(secret),
		},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			log.Printf("❌ [JWTAuth] Error de autenticación: %v", err)
			return c.Status(401).JSON(fiber.Map{
				"error": "Token inválido o expirado",
			})
		},
		SuccessHandler: func(c fiber.Ctx) error {
			log.Println("✅ [JWTAuth] Token JWT válido")
			return c.Next()
		},
	})
}
