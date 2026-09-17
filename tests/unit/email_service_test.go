package unit

import (
	"goravel/app/services"
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

// Este test usa "log" como driver de mail para no enviar correos reales.
// Configura MAIL_MAILER=log en .env.testing
func (s *EmailServiceTestSuite) TestSend_DoesNotPanic() {
	err := s.service.Send(
		"no-reply@truckfast.com",
		"test@example.com",
		"Test Subject",
		"<p>Hola</p>",
	)
	s.NoError(err)
}
