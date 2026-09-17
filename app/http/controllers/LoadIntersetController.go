package controllers

import (
	"fmt"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"log"

	"github.com/gofiber/fiber/v3"
)

type LoadInterestController struct {
	emailService *services.EmailService
}

func NewLoadInterestController() *LoadInterestController {
	return &LoadInterestController{
		emailService: services.NewEmailService(),
	}
}

// SendInterest - el carrier muestra interés por una carga abierta
// Recibe: :id (load_id) en la URL, broker_id y status como hidden en el form
// Obtiene: user_id (carrier) desde la sesión
func (c *LoadInterestController) SendInterest(ctx fiber.Ctx) error {
	// 1) ID de la carga desde la URL
	loadID := ctx.Params("id")
	if loadID == "" {
		return ctx.Redirect().To("/home?flash_error=Carga no especificada")
	}

	// 2) user_id (carrier) desde la sesión — sin consultar BD
	carrierID, ok := ctx.Locals("user_id").(uint)
	if !ok {
		return ctx.Redirect().To("/login")
	}

	// 3) broker_id y status vienen en el form (hidden inputs del botón)
	brokerID := ctx.FormValue("broker_id")
	status := ctx.FormValue("status")
	msg:=ctx.FormValue("msg")
	if brokerID == "" {
		return ctx.Redirect().To("/home?flash_error=Datos incompletos")
	}

	// 4) Validar estado sin tocar la tabla loads
	if status != "open" {
		return ctx.Redirect().To("/home?flash_error=Esta carga ya no está disponible")
	}

	// 5) Obtener datos del carrier (nombre/email) y del broker (email) — 2 queries mínimas
	var carrier models.User
	if err := facades.Orm().Query().Where("id = ?", carrierID).First(&carrier); err != nil {
		log.Printf("Error al obtener carrier %d: %v", carrierID, err)
		return ctx.Redirect().To("/home?flash_error=Error al obtener tus datos")
	}

	var broker models.User
	if err := facades.Orm().Query().Where("id = ?", brokerID).First(&broker); err != nil {
		log.Printf("Error al obtener broker %s: %v", brokerID, err)
		return ctx.Redirect().To("/home?flash_error=No se encontró al publicador de la carga")
	}

	// 6) Armar y enviar el correo
	subject := fmt.Sprintf("Nuevo interés en tu carga #%s", loadID)
	body := fmt.Sprintf(`
		<h2>¡Alguien está interesado en tu carga!</h2>
		<p><strong>Transportista:</strong> %s</p>
		<p><strong>Email:</strong> %s</p>
		<p><strong>Carga:</strong> #%s</p>
		<p>%s</p>
		<hr>
		<p>Ingresa a <a href="http://localhost:3000/dashboard">tu panel</a> para ver más detalles y asignar la carga.</p>
	`, carrier.Name, carrier.Email, loadID,msg)

	if err := c.emailService.Send("wagoassistant@gmail.com", broker.Email, subject, body); err != nil {
		log.Printf("Error al enviar correo al broker: %v", err)
		return ctx.Redirect().To("/home?flash_error=No se pudo enviar la notificación")
	}

	return ctx.Redirect().To("/home?flash_success=Tu interés ha sido enviado al publicador")
}
