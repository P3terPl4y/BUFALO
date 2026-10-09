package viewhelpers

import (
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v3"
	"goravel/app/models"
)

func TestNotificationTargetsAreRoleAndResourceScoped(t *testing.T) {
	choferID, publicadorID, resourceID := uint(8), uint(12), uint(55)
	if got := UserProfileURL(&models.User{Role: "chofer", ChoferID: &choferID}); got != "/choferes/8" {
		t.Fatalf("driver profile URL = %q", got)
	}
	if got := UserProfileURL(&models.User{Role: "publicador", PublicadorID: &publicadorID}); got != "/publicadores/12" {
		t.Fatalf("publisher profile URL = %q", got)
	}
	if got := UserProfileURL(&models.User{Role: "admin", ChoferID: &choferID}); got != "" {
		t.Fatalf("admin should not be linked to a role profile, got %q", got)
	}
	if got := NotificationResourceURL("load", &resourceID); got != "/loads/55" {
		t.Fatalf("load URL = %q", got)
	}
	if got := NotificationResourceURL("company", &resourceID); got != "/empresas/55" {
		t.Fatalf("company URL = %q", got)
	}
	if got := NotificationResourceURL("external", &resourceID); got != "" {
		t.Fatalf("unapproved resource type should not create a link, got %q", got)
	}
}

func TestNotificationFeedRendersActorResourceAndReadState(t *testing.T) {
	profileID, loadID, actorID := uint(14), uint(23), uint(9)
	actor := &models.User{Role: "chofer", ChoferID: &profileID, Name: "Camila Chofer", ProfilePhoto: "/storage/camila.webp"}
	notice := models.UserNotification{
		ID: 7, UserID: 4, ActorUserID: &actorID, Actor: actor,
		EventType: "load_assigned", Title: "Carga asignada", Message: "La carga BFL-23 fue asignada.",
		ResourceType: "load", ResourceID: &loadID, CreatedAt: time.Date(2026, 10, 9, 15, 30, 0, 0, time.UTC),
	}
	engine := html.New(filepath.Join("..", "views"), ".html")
	Register(engine)
	app := fiber.New(fiber.Config{Views: engine})
	app.Get("/notifications-preview", func(c fiber.Ctx) error {
		return c.Render("notifications/index", fiber.Map{
			"title": "Notificaciones", "notifications": []models.UserNotification{notice},
			"unreadCount": int64(1), "csrfToken": "csrf-preview", "hasNext": false,
		})
	})
	response, err := app.Test(httptest.NewRequest("GET", "/notifications-preview", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	page := string(body)
	for _, expected := range []string{
		"Asignación de carga", "Carga asignada", "Camila Chofer", "/storage/camila.webp",
		`href="/choferes/14"`, `href="/loads/23"`, "Ver carga", "Marcar como leída", "csrf-preview", "notification-card--unread",
	} {
		if !strings.Contains(page, expected) {
			t.Errorf("notification feed does not render %q", expected)
		}
	}
}
