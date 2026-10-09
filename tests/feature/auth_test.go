package feature

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"goravel/app/services"
	"goravel/tests"

	"github.com/stretchr/testify/suite"
)

type AuthTestSuite struct {
	suite.Suite
	tests.TestCase
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, &AuthTestSuite{})
}

func (s *AuthTestSuite) SetupTest() {
	s.RefreshDatabase(s.T())
}

func (s *AuthTestSuite) TestLogin_Success() {
	seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")

	client := newClient()
	resp := postForm(s.T(), client, "/login", map[string]string{
		"email":    "admin@test.com",
		"password": "password123",
	})
	defer resp.Body.Close()

	s.Equal(http.StatusSeeOther, resp.StatusCode)
	s.Contains(resp.Header.Get("Location"), "/home")
}

func (s *AuthTestSuite) TestLogin_WrongPassword() {
	seedUser(s.T(), "X", "x@test.com", "correctpassword", "carrier")

	client := newClient()
	resp := postForm(s.T(), client, "/login", map[string]string{
		"email":    "x@test.com",
		"password": "wrong",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(readBody(s.T(), resp), "Credenciales incorrectas")
}

func (s *AuthTestSuite) TestRegister_Success() {
	client := newClient()
	resp := postForm(s.T(), client, "/register", map[string]string{
		"name": "New User", "role": "publicador", "city": "La Habana", "state": "La Habana", "country": "Cuba", "radius": "100", "phone": "+5355555555", "whatsapp": "+5355555555", "empresa_mode": "none", "publicador_numero_licencia_broker": "NEW-USER", "publicador_anios_experiencia": "1",
		"email":    "new@test.com",
		"password": "password123",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	if body := readBody(s.T(), resp); !strings.Contains(body, "Confirma tu correo") {
		s.T().Fatalf("registration should wait for email confirmation: %s", body)
	}
	if count := tests.CountUsers(s.T()); count != 0 {
		s.T().Fatalf("user was persisted before email confirmation: %d", count)
	}
	confirmationURL, err := url.Parse(lastRegistrationConfirmationURL)
	s.NoError(err)
	token := confirmationURL.Query().Get("token")
	s.NotEmpty(token)
	confirmed := postForm(s.T(), client, "/register/confirm", map[string]string{"token": token})
	s.Contains(readBody(s.T(), confirmed), "Correo confirmado")

	// Login works for the new user
	client2 := login(s.T(), "new@test.com", "password123")
	resp2 := get(s.T(), client2, "/home")
	s.Equal(http.StatusOK, resp2.StatusCode)
}

func (s *AuthTestSuite) TestRegister_DuplicateEmail() {
	seedUser(s.T(), "A", "dup@test.com", "password123", "carrier")

	client := newClient()
	resp := postForm(s.T(), client, "/register", map[string]string{
		"name":     "B",
		"email":    "dup@test.com",
		"password": "password123",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(readBody(s.T(), resp), "email")
}

func (s *AuthTestSuite) TestSessionRoleChangeRevokesAdminAccess() {
	user := seedUser(s.T(), "Former admin", "former-admin@test.com", "password123", "admin")
	client := login(s.T(), user.Email, "password123")
	s.NoError(services.NewUserService().Update(user.ID, map[string]interface{}{"role": "chofer"}))
	resp := get(s.T(), client, "/admin")
	defer resp.Body.Close()
	s.Equal(http.StatusSeeOther, resp.StatusCode)
	s.Contains(resp.Header.Get("Location"), "/home")
}

func (s *AuthTestSuite) TestDisabledAccountSessionIsDestroyed() {
	user := seedUser(s.T(), "Disabled", "disabled@test.com", "password123", "admin")
	client := login(s.T(), user.Email, "password123")
	s.NoError(services.NewUserService().ToggleActive(user.ID, false))
	resp := get(s.T(), client, "/admin")
	defer resp.Body.Close()
	s.Equal(http.StatusSeeOther, resp.StatusCode)
	s.Contains(resp.Header.Get("Location"), "/login")
	resp = get(s.T(), client, "/admin")
	defer resp.Body.Close()
	s.Equal(http.StatusSeeOther, resp.StatusCode)
	s.Contains(resp.Header.Get("Location"), "/login")
}
