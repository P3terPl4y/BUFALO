package unit

import (
	"goravel/app/services"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"goravel/tests"
)

type EmailServiceTestSuite struct {
	suite.Suite
	tests.TestCase

	service *services.EmailService
}

func TestEmailServiceTestSuite(t *testing.T) {
	suite.Run(t, &EmailServiceTestSuite{
		service: services.NewEmailService(),
	})
}

// La suite unitaria no debe depender de SMTP ni enviar correo externo.
// La entrega real se cubre con un servidor SMTP local en app/maildelivery.
func (s *EmailServiceTestSuite) TestSend_FailsClosedWithoutSMTPConfiguration() {
	err := s.service.Send(
		"no-reply@truckfast.com",
		"test@example.com",
		"Test Subject",
		"<p>Hola</p>",
	)
	s.Error(err)
	s.True(strings.Contains(err.Error(), "SMTP is not configured"), "unexpected error: %v", err)
}
