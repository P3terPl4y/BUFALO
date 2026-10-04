package controllers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goravel/app/models"

	"github.com/gofiber/fiber/v3"
)

func TestMobileLoadResponseUsesSafeMobileDTO(t *testing.T) {
	app := fiber.New()
	controller := NewMobileController()
	secret := "private-account-value"
	weight, rate := 1250.0, 980.5
	lat, lng := 23.113592, -82.366592
	app.Get("/loads", func(ctx fiber.Ctx) error {
		return controller.loadResponse(ctx, []models.Carga{{
			ID: 7, NumeroReferencia: "MOBILE-7", Estado: models.CargaPublicada,
			FechaRecogida: time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC),
			PesoKg:        &weight, TarifaTotal: &rate, OrigenDireccion: &models.Direccion{Ciudad: "La Habana", EstadoProvincia: "La Habana", Latitud: &lat, Longitud: &lng},
			DestinoDireccion: &models.Direccion{Ciudad: "Matanzas", EstadoProvincia: "Matanzas"},
			Publicador:       &models.Publicador{User: &models.User{Name: "No debe salir", Password: secret}},
		}}, 1, 1, false)
	})
	response, err := app.Test(httptest.NewRequest("GET", "/loads", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body struct {
		Loads   []mobileLoad `json:"loads"`
		Total   int64        `json:"total"`
		Page    int          `json:"page"`
		HasMore bool         `json:"has_more"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Loads) != 1 || body.Total != 1 || body.Page != 1 || body.HasMore {
		t.Fatalf("unexpected mobile page: %+v", body)
	}
	load := body.Loads[0]
	if load.Reference != "MOBILE-7" || load.Origin.City != "La Habana" || load.Destination.City != "Matanzas" || load.Origin.Latitude == nil {
		t.Fatalf("mobile route fields were lost: %+v", load)
	}
	encoded, _ := json.Marshal(body)
	if strings.Contains(string(encoded), secret) || strings.Contains(string(encoded), "No debe salir") || strings.Contains(string(encoded), "password") {
		t.Fatalf("mobile response leaked account data: %s", encoded)
	}
}

func TestMobileLoadActionsRejectNonDriverBeforeDatabaseAccess(t *testing.T) {
	app := fiber.New()
	app.Use(func(ctx fiber.Ctx) error {
		ctx.Locals("user_id", uint(9))
		ctx.Locals("role", "publicador")
		return ctx.Next()
	})
	app.Post("/loads/:id<int>/accept", NewMobileController().Accept)
	response, err := app.Test(httptest.NewRequest("POST", "/loads/4/accept", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] == "" {
		t.Fatal("expected a clear role error")
	}
}
