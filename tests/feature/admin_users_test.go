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
	s.RefreshDatabase(s.T())
}

func (s *AdminUsersTestSuite) TestStore_CreatesPublicador() {
	seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")
	client := login(s.T(), "admin@test.com", "password123")

	resp := postForm(s.T(), client, "/admin/users", map[string]string{
		"name":     "Broker Test",
		"email":    "broker@test.com",
		"password": "password123",
		"role":     "publicador", "city": "La Habana", "state": "La Habana", "country": "Cuba", "radius": "100", "empresa_mode": "none", "publicador_numero_licencia_broker": "BROKER-TEST", "publicador_anios_experiencia": "1",
	})
	defer resp.Body.Close()

	s.Equal(http.StatusSeeOther, resp.StatusCode)
	s.Contains(resp.Header.Get("Location"), "/admin/users")

	svc := services.NewUserService()
	users, total, _ := svc.GetAllWithFilters(map[string]string{"search": "broker@test.com"}, 1, 10)
	s.Equal(int64(1), total)
	s.Require().Len(users, 1)
	s.Equal("publicador", users[0].Role)
	s.True(users[0].IsActive)
}

func (s *AdminUsersTestSuite) TestCreateFormOffersSupportedDomainRoles() {
	seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")
	client := login(s.T(), "admin@test.com", "password123")
	resp := get(s.T(), client, "/admin/users/create")
	body := readBody(s.T(), resp)

	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(body, `name="role" value="publicador"`)
	s.Contains(body, `name="role" value="chofer"`)
	s.NotContains(body, `name="role" value="broker"`)
	s.NotContains(body, `name="role" value="carrier"`)
	for _, field := range []string{
		"city", "state", "country", "radius", "chofer_numero_licencia", "chofer_tipo_licencia",
		"chofer_anios_experiencia", "publicador_numero_licencia_broker", "publicador_anios_experiencia",
		"empresa_mode", "empresa_id", "empresa_nombre_legal",
	} {
		s.Contains(body, `name="`+field+`"`, "form must submit server-required/profile field %s", field)
	}
}

func (s *AdminUsersTestSuite) TestStore_RejectsIncompleteProfileWithoutCreatingUser() {
	seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")
	client := login(s.T(), "admin@test.com", "password123")
	resp := postForm(s.T(), client, "/admin/users", map[string]string{
		"name": "Incomplete", "email": "incomplete@test.com", "password": "password123", "role": "chofer",
	})
	body := readBody(s.T(), resp)
	resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(body, "Ciudad")
	s.Contains(body, "Número de licencia")
	svc := services.NewUserService()
	_, total, _ := svc.GetAllWithFilters(map[string]string{"search": "incomplete@test.com"}, 1, 10)
	s.Equal(int64(0), total)
}

func (s *AdminUsersTestSuite) TestStore_RejectsInvalidCompanyMode() {
	seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")
	client := login(s.T(), "admin@test.com", "password123")
	resp := postForm(s.T(), client, "/admin/users", map[string]string{
		"name": "Bad Mode", "email": "bad-mode@test.com", "password": "password123", "role": "publicador",
		"city": "La Habana", "state": "La Habana", "country": "Cuba", "radius": "100",
		"publicador_numero_licencia_broker": "BROKER-TEST", "publicador_anios_experiencia": "1", "empresa_mode": "unexpected",
	})
	body := readBody(s.T(), resp)
	resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(body, "Opción de empresa inválida")
	svc := services.NewUserService()
	_, total, _ := svc.GetAllWithFilters(map[string]string{"search": "bad-mode@test.com"}, 1, 10)
	s.Equal(int64(0), total)
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
		"role":     "publicador", "city": "La Habana", "state": "La Habana", "country": "Cuba", "radius": "100", "empresa_mode": "none", "publicador_numero_licencia_broker": "BROKER-TEST", "publicador_anios_experiencia": "1",
	})
	resp.Body.Close()

	svc := services.NewUserService()
	_, total, _ := svc.GetAllWithFilters(map[string]string{"search": "dup@test.com"}, 1, 10)
	s.Equal(int64(1), total)
}

func (s *AdminUsersTestSuite) TestIndexMutationFormsIncludeCSRFAndUseDeleteRoute() {
	seedUser(s.T(), "Admin", "admin@test.com", "password123", "admin")
	target := seedUser(s.T(), "Publicador", "publisher@test.com", "password123", "publicador")
	client := login(s.T(), "admin@test.com", "password123")

	resp := get(s.T(), client, "/admin/users")
	body := readBody(s.T(), resp)
	id := itoa(target.ID)

	s.Contains(body, `<form action="/admin/users/`+id+`/toggle" method="POST"><input type="hidden" name="_csrf" value="`)
	s.Contains(body, `<form action="/admin/users/`+id+`/delete" method="POST"`)
	s.Contains(body, `<input type="hidden" name="_csrf" value="`)
	s.NotContains(body, `<form action="/admin/users/`+id+`" method="POST" onsubmit=`)
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
