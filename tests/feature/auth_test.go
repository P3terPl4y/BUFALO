package feature

import (
	"net/http"
	"testing"

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
	s.RefreshDatabase()
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
	seedUser(s.T(), "X", "x@test.com", "correct", "carrier")

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
		"name":     "New User",
		"email":    "new@test.com",
		"password": "password123",
	})
	s.Equal(http.StatusOK, resp.StatusCode)

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
