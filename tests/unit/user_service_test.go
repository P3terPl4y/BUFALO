package unit

import (
	"goravel/app/models"
	"goravel/app/services"
	"testing"

	"github.com/stretchr/testify/suite"
	"goravel/tests"
)

type UserServiceTestSuite struct {
	suite.Suite
	tests.TestCase

	service *services.UserService
}

func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, &UserServiceTestSuite{
		service: services.NewUserService(),
	})
}

// Se ejecuta antes de cada test
func (s *UserServiceTestSuite) SetupTest() {
	// Opcional: limpiar tabla users
	s.RefreshDatabase()
}

func (s *UserServiceTestSuite) TestCreate_Success() {
	user := &models.User{
		Name:     "Test User",
		Email:    "test@example.com",
		Role:     "carrier",
		IsActive: true,
	}
	s.NoError(user.SetPassword("password123"))

	err := s.service.Create(user)
	s.NoError(err)
	s.NotZero(user.ID)
}

func (s *UserServiceTestSuite) TestCreate_DuplicateEmail() {
	u1 := &models.User{Name: "A", Email: "dup@example.com", Role: "carrier", IsActive: true}
	_ = u1.SetPassword("password123")
	s.NoError(s.service.Create(u1))

	u2 := &models.User{Name: "B", Email: "dup@example.com", Role: "broker", IsActive: true}
	_ = u2.SetPassword("password123")
	err := s.service.Create(u2)
	s.Error(err) // viola uniqueIndex
}

func (s *UserServiceTestSuite) TestGetByID_Found() {
	user := &models.User{Name: "Find Me", Email: "find@example.com", Role: "carrier", IsActive: true}
	_ = user.SetPassword("password123")
	s.NoError(s.service.Create(user))

	found, err := s.service.GetByID(user.ID)
	s.NoError(err)
	s.Equal("find@example.com", found.Email)
}

func (s *UserServiceTestSuite) TestGetByID_NotFound() {
	_, err := s.service.GetByID(999999)
	s.Error(err)
}

func (s *UserServiceTestSuite) TestGetAllWithFilters_ByRole() {
	for i := 0; i < 3; i++ {
		u := &models.User{
			Name: "carrier", Email: "c" + string(rune('0'+i)) + "@x.com",
			Role: "carrier", IsActive: true,
		}
		_ = u.SetPassword("password123")
		s.NoError(s.service.Create(u))
	}
	b := &models.User{Name: "broker", Email: "b@x.com", Role: "broker", IsActive: true}
	_ = b.SetPassword("password123")
	s.NoError(s.service.Create(b))

	users, total, err := s.service.GetAllWithFilters(map[string]string{"role": "carrier"}, 1, 10)
	s.NoError(err)
	s.Equal(int64(3), total)
	s.Len(users, 3)
}

func (s *UserServiceTestSuite) TestUpdate() {
	user := &models.User{Name: "Old", Email: "old@x.com", Role: "carrier", IsActive: true}
	_ = user.SetPassword("password123")
	s.NoError(s.service.Create(user))

	err := s.service.Update(user.ID, map[string]interface{}{
		"name": "New",
	})
	s.NoError(err)

	found, _ := s.service.GetByID(user.ID)
	s.Equal("New", found.Name)
}

func (s *UserServiceTestSuite) TestToggleActive() {
	user := &models.User{Name: "Toggle", Email: "tog@x.com", Role: "carrier", IsActive: true}
	_ = user.SetPassword("password123")
	s.NoError(s.service.Create(user))

	s.NoError(s.service.ToggleActive(user.ID, false))
	found, _ := s.service.GetByID(user.ID)
	s.False(found.IsActive)

	s.NoError(s.service.ToggleActive(user.ID, true))
	found, _ = s.service.GetByID(user.ID)
	s.True(found.IsActive)
}

func (s *UserServiceTestSuite) TestDelete() {
	user := &models.User{Name: "Delete", Email: "del@x.com", Role: "carrier", IsActive: true}
	_ = user.SetPassword("password123")
	s.NoError(s.service.Create(user))

	s.NoError(s.service.Delete(user.ID))

	_, err := s.service.GetByID(user.ID)
	s.Error(err)
}
