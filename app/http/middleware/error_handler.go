package middleware

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"goravel/app/sessionstore"
	"log"
	"net"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/jackc/pgx/v5/pgconn"
)

func ErrorHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			return RenderError(c, err)
		}
		return nil
	}
}

// RenderError is shared by Fiber's final handler and the CSRF middleware.
func RenderError(c fiber.Ctx, err error) error {
	log.Printf("❌ Error: %v", err)
	status := errorStatus(err)
	page, ok := errorPage(status)
	if !ok {
		status = fiber.StatusInternalServerError
		page, _ = errorPage(status)
	}
	c.Status(status)
	if c.Accepts("html") != "html" && c.Accepts("json") == "json" {
		return c.JSON(fiber.Map{"error": page.Message, "status": status})
	}
	if renderErr := c.Render("errors/status", fiber.Map{
		"code": status, "title": page.Title, "message": page.Message,
		"detail": page.Detail, "image": page.Image,
	}); renderErr != nil {
		return c.SendString(page.Message)
	}
	return nil
}

func errorStatus(err error) int {
	for _, rejection := range []error{csrf.ErrTokenNotFound, csrf.ErrTokenInvalid,
		csrf.ErrFetchSiteInvalid, csrf.ErrRefererNotFound, csrf.ErrRefererInvalid,
		csrf.ErrRefererNoMatch, csrf.ErrOriginInvalid, csrf.ErrOriginNoMatch} {
		if errors.Is(err, rejection) {
			return fiber.StatusForbidden
		}
	}
	if errors.Is(err, sessionstore.ErrCapacity) || errors.Is(err, sessionstore.ErrUnavailable) {
		return fiber.StatusServiceUnavailable
	}
	if errors.Is(err, sql.ErrConnDone) || errors.Is(err, driver.ErrBadConn) || errors.Is(err, context.DeadlineExceeded) {
		return fiber.StatusServiceUnavailable
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "57014", "55P03", "53300", "57P01", "57P02", "57P03":
			return fiber.StatusServiceUnavailable
		}
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		return fiber.StatusServiceUnavailable
	}
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return fiberErr.Code
	}
	return fiber.StatusInternalServerError
}

type errorPageContent struct{ Title, Message, Detail, Image string }

func errorPage(status int) (errorPageContent, bool) {
	pages := map[int]errorPageContent{
		400: {"Solicitud inválida", "No pudimos entender esta solicitud.", "Revisa los datos enviados e inténtalo de nuevo.", "400"},
		401: {"Inicia sesión", "Necesitas identificarte para continuar.", "Tu sesión puede haber terminado. Vuelve a iniciar sesión.", "401"},
		403: {"Acceso restringido", "No tienes permiso para ver este contenido.", "Si crees que es un error, contacta al administrador de tu cuenta.", "403"},
		404: {"Página no encontrada", "No encontramos esta página.", "La dirección puede haber cambiado o ya no estar disponible.", "404"},
		405: {"Acción no disponible", "Esta acción no está permitida desde esta página.", "Regresa y utiliza una opción disponible.", "405"},
		408: {"La solicitud tardó demasiado", "La conexión demoró más de lo esperado.", "Comprueba tu conexión y vuelve a intentarlo.", "408"},
		409: {"Conflicto de operación", "Los datos cambiaron mientras procesábamos la solicitud.", "Actualiza la página para continuar con la información más reciente.", "409"},
		413: {"Archivo demasiado grande", "El archivo supera el tamaño permitido.", "Elige un archivo más pequeño e inténtalo otra vez.", "413"},
		415: {"Formato no admitido", "El formato de la solicitud no está permitido.", "Utiliza el formulario de esta página.", "400"},
		419: {"Sesión vencida", "La sesión de seguridad ya no es válida.", "Recarga la página e inicia sesión si te lo solicita.", "419"},
		422: {"Revisa los datos", "Algunos datos no se pudieron validar.", "Corrige los campos señalados y vuelve a guardar.", "422"},
		429: {"Demasiados intentos", "Recibimos varias solicitudes en poco tiempo.", "Espera un momento antes de volver a intentarlo.", "429"},
		500: {"Error interno", "Algo salió mal al procesar la operación.", "El equipo de BUFALO ha recibido el registro técnico. Inténtalo más tarde.", "500"},
		502: {"Servicio temporalmente inaccesible", "Un servicio necesario no respondió correctamente.", "Vuelve a intentarlo en unos minutos.", "502"},
		503: {"Servicio en mantenimiento", "BUFALO no está disponible por el momento.", "Estamos trabajando para restablecer el servicio.", "503"},
		504: {"Tiempo de espera agotado", "Un servicio tardó demasiado en responder.", "Vuelve a intentarlo en unos minutos.", "504"},
	}
	page, ok := pages[status]
	return page, ok
}
