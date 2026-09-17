package feature

import (
	"testing"

	"goravel/app/services"
	"goravel/tests"

	"github.com/stretchr/testify/suite"
)

type ProfileTestSuite struct {
	suite.Suite
	tests.TestCase
}

func TestProfileTestSuite(t *testing.T) {
	suite.Run(t, &ProfileTestSuite{})
}

func (s *ProfileTestSuite) SetupTest() {
	s.RefreshDatabase()
}

func (s *ProfileTestSuite) TestUpdate_ChangesName() {
	u := seedUser(s.T(), "Old", "u@test.com", "password123", "carrier")
	client := login(s.T(), "u@test.com", "password123")

	resp := postForm(s.T(), client, "/profile/update", map[string]string{
		"name": "New",
	})
	resp.Body.Close()

	svc := services.NewUserService()
	found, err := svc.GetByID(u.ID)
	s.NoError(err)
	s.Equal("New", found.Name)
}

func (s *ProfileTestSuite) TestUpdate_PasswordChange() {
	u := seedUser(s.T(), "User", "pw@test.com", "oldpassword", "carrier")
	client := login(s.T(), "pw@test.com", "oldpassword")

	resp := postForm(s.T(), client, "/profile/update", map[string]string{
		"password": "newpassword123",
	})
	resp.Body.Close()

	svc := services.NewUserService()
	found, err := svc.GetByID(u.ID)
	s.NoError(err)
	s.True(found.CheckPassword("newpassword123"))
	s.False(found.CheckPassword("oldpassword"))
}

func (s *ProfileTestSuite) TestUpdate_DuplicateEmail() {
	seedUser(s.T(), "A", "a@test.com", "password123", "carrier")
	seedUser(s.T(), "B", "b@test.com", "password123", "carrier")

	client := login(s.T(), "a@test.com", "password123")
	resp := postForm(s.T(), client, "/profile/update", map[string]string{
		"email": "b@test.com",
	})
	body := readBody(s.T(), resp)
	s.Contains(body, "email")
}
