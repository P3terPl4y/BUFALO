package feature

import (
	"net/http"
	"testing"

	"goravel/app/services"
	"goravel/tests"

	"github.com/stretchr/testify/suite"
)

type AdminUsersTestSuite struct {
	suite.Suite
	tests.TestCase
}

func TestAdminUsersTestSuite(t *testing.T) {
	suite.Run(t, &AdminUsersTestSuite{})
}

func (s *AdminUsersTestSuite) SetupTest() {
	s.RefreshDatabase()
}

func (s *AdminUsersTestSuite) TestStore_CreatesBroker() {
	seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")
	client := login(s.T(), "admin@test.com", "password123")

	resp := postForm(s.T(), client, "/admin/users", map[string]string{
		"name":     "Broker Test",
		"email":    "broker@test.com",
		"password": "password123",
		"role":     "broker",
	})
	defer resp.Body.Close()

	s.Equal(http.StatusSeeOther, resp.StatusCode)
	s.Contains(resp.Header.Get("Location"), "/admin/users")

	svc := services.NewUserService()
	users, total, _ := svc.GetAllWithFilters(map[string]string{"search": "broker@test.com"}, 1, 10)
	s.Equal(int64(1), total)
	s.Require().Len(users, 1)
	s.Equal("broker", users[0].Role)
	s.True(users[0].IsActive)
}

func (s *AdminUsersTestSuite) TestStore_RejectsInvalidRole() {
	seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")
	client := login(s.T(), "admin@test.com", "password123")

	resp := postForm(s.T(), client, "/admin/users", map[string]string{
		"name":     "Hacker",
		"email":    "hacker@test.com",
		"password": "password123",
		"role":     "admin",
	})
	resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode) // renderiza el form con error

	svc := services.NewUserService()
	_, total, _ := svc.GetAllWithFilters(map[string]string{"search": "hacker@test.com"}, 1, 10)
	s.Equal(int64(0), total)
}

func (s *AdminUsersTestSuite) TestStore_DuplicateEmail() {
	seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")
	seedUser(s.T(), "A", "dup@test.com", "password123", "carrier")

	client := login(s.T(), "admin@test.com", "password123")

	resp := postForm(s.T(), client, "/admin/users", map[string]string{
		"name":     "B",
		"email":    "dup@test.com",
		"password": "password123",
		"role":     "broker",
	})
	resp.Body.Close()

	svc := services.NewUserService()
	_, total, _ := svc.GetAllWithFilters(map[string]string{"search": "dup@test.com"}, 1, 10)
	s.Equal(int64(1), total)
}

func (s *AdminUsersTestSuite) TestDelete_AdminIsProtected() {
	admin := seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")
	client := login(s.T(), "admin@test.com", "password123")

	resp := postForm(s.T(), client, "/admin/users/"+itoa(admin.ID), map[string]string{
		"_method": "DELETE",
	})
	resp.Body.Close()

	svc := services.NewUserService()
	_, err := svc.GetByID(admin.ID)
	s.NoError(err, "el admin debe seguir existiendo")
}
