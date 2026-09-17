// middleware/rate_limit.go
package middleware

import (
    "time"
    "github.com/gofiber/fiber/v3"
    "github.com/gofiber/fiber/v3/middleware/limiter"
)

func LoginRateLimiter() fiber.Handler {
    return limiter.New(limiter.Config{
        Max:        5,                // Máximo 5 intentos
        Expiration: 1 * time.Minute,  // Por minuto
        KeyGenerator: func(c fiber.Ctx) string {
            return c.IP() // Limita por IP
        },
        LimitReached: func(c fiber.Ctx) error {
            return c.Status(429).SendString("Demasiados intentos. Inténtalo de nuevo en 1 minuto.")
        },
    })
}
