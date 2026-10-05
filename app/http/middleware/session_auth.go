package middleware

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	frameworkerrors "github.com/goravel/framework/errors"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/monitoring"
	"log"
	"time"
)

// SessionAuth treats the database account as authoritative on every request.
// This revokes stale sessions after disabling an account or changing its role.
func SessionAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		sess := session.FromContext(c)
		if sess == nil {
			return c.Redirect().To("/login")
		}
		rawID := sess.Get("user_id")
		userID, ok := rawID.(uint)
		if !ok || userID == 0 {
			return c.Redirect().To("/login")
		}
		var user models.User
		if err := facades.Orm().Query().Where("id = ?", userID).First(&user); err != nil {
			if !errors.Is(err, frameworkerrors.OrmRecordNotFound) {
				log.Printf("session account lookup failed: %v", err)
				return fiber.ErrServiceUnavailable
			}
			if destroyErr := sess.Destroy(); destroyErr != nil {
				log.Printf("session revocation failed: %v", destroyErr)
			}
			return c.Redirect().To("/login?flash_error=Sesión inválida o usuario deshabilitado")
		}
		if user.ID == 0 || !user.IsActive {
			if err := sess.Destroy(); err != nil {
				log.Printf("session revocation failed: %v", err)
			}
			return c.Redirect().To("/login?flash_error=Sesión inválida o usuario deshabilitado")
		}
		sess.Set("role", user.Role)
		sess.Set("is_active", user.IsActive)
		// Keep the database-validated user for controllers in this request. This
		// avoids a second lookup with unrelated preloads for the current profile.
		c.Locals("authenticated_user", &user)
		c.Locals("user_id", user.ID)
		c.Locals("role", user.Role)
		c.Locals("is_active", user.IsActive)
		monitoring.RecordUserActivity(user.ID, time.Now())
		return c.Next()
	}
}
