package feature

import (
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
	"net/http"
	"net/url"
	"testing"
)

func TestSensitiveProfileChangesRequireReauthenticationAndRevokeSessions(t *testing.T) {
	// This suite's reset helper is invoked explicitly for top-level regression tests.
	tests.ResetDB(t)
	user := seedUser(t, "Credential Test", "credential@test.invalid", "oldpassword123", "admin")
	if _, err := facades.Orm().Query().Model(&models.User{}).Where("id = ?", user.ID).Update("city", "Preserved City"); err != nil {
		t.Fatal(err)
	}
	first := login(t, user.Email, "oldpassword123")
	second := login(t, user.Email, "oldpassword123")
	response := postForm(t, first, "/profile/update", map[string]string{"password": "newpassword123"})
	response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatalf("password changed without reauthentication: %d", response.StatusCode)
	}
	response = postForm(t, first, "/profile/update", map[string]string{"password": "newpassword123", "current_password": "oldpassword123"})
	response.Body.Close()
	found, err := services.NewUserService().GetByID(user.ID)
	if err != nil || !found.CheckPassword("newpassword123") || found.City != "Preserved City" {
		t.Fatal("authorized password change failed")
	}
	response = get(t, second, "/profile")
	response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatal("old session retained access")
	}
	first = login(t, user.Email, "newpassword123")
	response = postForm(t, first, "/profile/update", map[string]string{"email": "changed@test.invalid", "current_password": "newpassword123"})
	response.Body.Close()
	found, _ = services.NewUserService().GetByID(user.ID)
	if found.Email != user.Email {
		t.Fatal("unverified email activated")
	}
	link, err := url.Parse(lastRegistrationConfirmationURL)
	if err != nil || link.Query().Get("token") == "" {
		t.Fatal("no confirmation captured")
	}
	response = postForm(t, first, "/register/confirm", map[string]string{"token": link.Query().Get("token")})
	response.Body.Close()
	found, _ = services.NewUserService().GetByID(user.ID)
	if found.Email != "changed@test.invalid" {
		t.Fatal("confirmed email not activated")
	}
	response = get(t, first, "/profile")
	response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatal("email change did not revoke old session")
	}
}

func TestLogoutGetOnlyConfirmsAndPostDestroysSession(t *testing.T) {
	tests.ResetDB(t)
	user := seedUser(t, "Logout User", "logout@test.invalid", "logoutpassword123", "admin")
	client := login(t, user.Email, "logoutpassword123")
	response := get(t, client, "/logout")
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("logout confirmation status %d", response.StatusCode)
	}
	response = get(t, client, "/profile")
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("GET logout destroyed the session")
	}
	response = postForm(t, client, "/logout", map[string]string{})
	response.Body.Close()
	if response.StatusCode != 303 {
		t.Fatalf("POST logout status %d", response.StatusCode)
	}
	response = get(t, client, "/profile")
	response.Body.Close()
	if response.StatusCode != 303 {
		t.Fatal("POST logout retained session")
	}
}
