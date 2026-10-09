package services

import (
	"context"
	"fmt"
	"goravel/app/facades"
	"goravel/app/maildelivery"
	"goravel/app/models"
	"html"
	"net/mail"
	"net/url"
)

type EmailService struct{}

func (s *EmailService) SendEmailChangeConfirmation(to, link string) error {
	from := (&mail.Address{Address: facades.Config().GetString("mail.from.address"), Name: facades.Config().GetString("mail.from.name")}).String()
	return s.send(from, to, "Confirma tu nuevo correo | BUFALO", "<p>Confirma el cambio de correo de tu cuenta BUFALO.</p><p><a href=\""+html.EscapeString(link)+"\">Confirmar nuevo correo</a></p><p>El enlace vence en 24 horas.</p>", "Confirma tu nuevo correo: "+link)
}

func (s *EmailService) SendLoadInterest(driver, publisher *models.User, load *models.Carga, comment string) error {
	base, err := url.Parse(facades.Config().GetString("http.url"))
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return fmt.Errorf("APP_URL inválida")
	}
	link := base.ResolveReference(&url.URL{Path: fmt.Sprintf("/loads/%d", load.ID)}).String()
	text := fmt.Sprintf("%s (%s) está interesado en tu carga %s.\n\n%s\n\n%s", driver.Name, driver.Email, load.NumeroReferencia, comment, link)
	body := fmt.Sprintf("<p><strong>%s</strong> (%s) está interesado en tu carga %s.</p><p style=\"white-space:pre-wrap\">%s</p><p><a href=\"%s\">Ver carga</a></p>", html.EscapeString(driver.Name), html.EscapeString(driver.Email), html.EscapeString(load.NumeroReferencia), html.EscapeString(comment), html.EscapeString(link))
	from := (&mail.Address{Address: facades.Config().GetString("mail.from.address"), Name: facades.Config().GetString("mail.from.name")}).String()
	return s.send(from, publisher.Email, "Interés en tu carga | BUFALO", body, text)
}

func NewEmailService() *EmailService {
	return &EmailService{}
}

func (s *EmailService) SendRegistrationVerification(to, confirmationURL string) error {
	host := facades.Config().GetString("mail.host")
	from := facades.Config().GetString("mail.from.address")
	if host == "" || from == "" {
		return fmt.Errorf("email delivery is not configured")
	}
	name := facades.Config().GetString("mail.from.name")
	link := html.EscapeString(confirmationURL)
	body := "<p>Confirma tu correo para crear tu cuenta en BUFALO.</p>" +
		"<p><a href=\"" + link + "\">Confirmar correo y crear cuenta</a></p>" +
		"<p>El enlace vence en 24 horas. Si no solicitaste esta cuenta, ignora este mensaje.</p>"
	text := "Confirma tu correo para crear tu cuenta en BUFALO: " + confirmationURL +
		"\nEl enlace vence en 24 horas. Si no solicitaste esta cuenta, ignora este mensaje."
	return s.send((&mail.Address{Address: from, Name: name}).String(), to,
		"Confirma tu correo electrónico | BUFALO", body, text)
}

// Send envía un correo con cuerpo HTML.
func (s *EmailService) Send(from, to, subject, body string) error {
	return s.send((&mail.Address{Address: from, Name: "TruckFast"}).String(), to, subject, body, "")
}

func (s *EmailService) send(from, to, subject, body, text string) error {
	cfg := maildelivery.Config{Host: facades.Config().GetString("mail.host"), Port: facades.Config().GetInt("mail.port"), Username: facades.Config().GetString("mail.username"), Password: facades.Config().GetString("mail.password")}
	return maildelivery.Send(context.Background(), cfg, from, to, subject, body, text)
}

// EmailExists preserves the public validation hook without making SMTP/DNS
// queries; actual ownership is verified by the one-time confirmation link.
func (s *EmailService) EmailExists(email string) bool {
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}
