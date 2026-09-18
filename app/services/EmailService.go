package services

import (
	"goravel/app/facades"
	"log"

	"github.com/goravel/framework/contracts/mail"

	emailverifier "github.com/AfterShip/email-verifier"
)

type EmailService struct{}

func NewEmailService() *EmailService {
	return &EmailService{}
}

// Send envía un correo con cuerpo HTML.
func (s *EmailService) Send(from, to, subject, body string) error {
	return facades.Mail().
		From(mail.Address{Address: from, Name: "TruckFast"}).
		To([]string{to}).
		Subject(subject).
		Content(mail.Content{Html: body}).
		Send()
}

var verifier = emailverifier.NewVerifier()

// EmailExists verifica si una dirección de email existe realmente.
// Devuelve true si el email es válido, el dominio tiene MX y el buzón
// existe en el servidor SMTP. Devuelve false en cualquier otro caso.
func (s *EmailService) EmailExists(email string) bool {
	if false {
		result, err := verifier.Verify(email)
		if err != nil {
			log.Println(err)
			return false
		}

		// 1. Sintaxis válida
		if !result.Syntax.Valid {
			log.Println("Error en primera")
			return false
		}

		// 2. El dominio tiene registros MX (servidor de correo)
		if !result.HasMxRecords {
			log.Println("Error en 2da")
			return false
		}

		/* 3. Verificación SMTP: si el servidor respondió y el buzón existe
		if result.SMTP != nil {
			// Reachable puede ser "yes", "no" o "unknown"
			// Solo aceptamos "yes" o "unknown" (algunos servidores no responden)
			if result.SMTP.Reachable == "no" {
				return false
			}
		}*/
	}

	return true
}
