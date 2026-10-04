package controllers

import (
	"fmt"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
)

// MobileController provides the small, role-scoped JSON surface used by the
// installable React client. It deliberately returns DTOs instead of database
// models so private user and billing fields cannot leak through associations.
type MobileController struct {
	loads        *services.CargaService
	choferes     *services.ChoferService
	publicadores *services.PublicadorService
}

func NewMobileController() *MobileController {
	return &MobileController{
		loads:        services.NewCargaService(),
		choferes:     services.NewChoferService(),
		publicadores: services.NewPublicadorService(),
	}
}

type mobileUser struct {
	ID           uint    `json:"id"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Role         string  `json:"role"`
	ProfilePhoto string  `json:"profile_photo,omitempty"`
	DriverRating float64 `json:"driver_rating,omitempty"`
	RatingCount  int     `json:"rating_count,omitempty"`
}

type mobileAddress struct {
	City      string   `json:"city"`
	State     string   `json:"state"`
	Country   string   `json:"country"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

type mobileLoad struct {
	ID          uint          `json:"id"`
	Reference   string        `json:"reference"`
	Status      string        `json:"status"`
	CargoType   string        `json:"cargo_type"`
	Equipment   string        `json:"equipment"`
	WeightKG    *float64      `json:"weight_kg,omitempty"`
	DistanceKM  float64       `json:"distance_km"`
	Rate        *float64      `json:"rate,omitempty"`
	Currency    string        `json:"currency"`
	Pickup      string        `json:"pickup"`
	Origin      mobileAddress `json:"origin"`
	Destination mobileAddress `json:"destination"`
}

func (c *MobileController) CSRF(ctx fiber.Ctx) error {
	return ctx.JSON(fiber.Map{"token": csrf.TokenFromContext(ctx)})
}

func (c *MobileController) Me(ctx fiber.Ctx) error {
	id, ok := ctx.Locals("user_id").(uint)
	if !ok || id == 0 {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sesión requerida"})
	}
	var user models.User
	if err := facades.Orm().Query().Where("id = ?", id).First(&user); err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sesión inválida"})
	}
	dto := mobileUser{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, ProfilePhoto: user.ProfilePhoto}
	if user.Role == "chofer" {
		if driver, err := c.choferes.GetByUserID(user.ID); err == nil {
			dto.DriverRating, dto.RatingCount = driver.RatingAverage, driver.RatingCount
		}
	}
	return ctx.JSON(dto)
}

func (c *MobileController) Loads(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(uint)
	role, _ := ctx.Locals("role").(string)
	if !ok || userID == 0 {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sesión requerida"})
	}
	filters := map[string]string{}
	page, err := strconv.Atoi(ctx.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	if page > 10000 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Página fuera de rango"})
	}
	perPage := 25
	switch role {
	case "admin":
	case "publicador":
		profile, err := c.publicadores.GetByUserID(userID)
		if err != nil {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Perfil de publicador no disponible"})
		}
		filters["publicador_id"] = strconv.FormatUint(uint64(profile.ID), 10)
	case "chofer":
		profile, err := c.choferes.GetByUserID(userID)
		if err != nil {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Perfil de chofer no disponible"})
		}
		filters["driver_board_id"] = strconv.FormatUint(uint64(profile.ID), 10)
	default:
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Rol no habilitado"})
	}
	if status := strings.TrimSpace(ctx.Query("status")); status != "" {
		filters["status"] = status
	}
	list, total, err := c.loads.GetAllWithFilters(filters, page, perPage)
	if err != nil {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "No se pudieron cargar las cargas"})
	}
	return c.loadResponse(ctx, list, total, page, int64(page*perPage) < total)
}

func (c *MobileController) loadResponse(ctx fiber.Ctx, loads []models.Carga, total int64, page int, hasMore bool) error {
	result := make([]mobileLoad, 0, len(loads))
	seen := map[uint]bool{}
	for _, load := range loads {
		if seen[load.ID] {
			continue
		}
		seen[load.ID] = true
		item := mobileLoad{ID: load.ID, Reference: load.NumeroReferencia, Status: string(load.Estado), CargoType: string(load.TipoCarga), Equipment: string(load.TipoEquipo), WeightKG: load.PesoKg, DistanceKM: load.DistanciaKm, Rate: load.TarifaTotal, Currency: string(load.Moneda), Pickup: load.FechaRecogida.Format("2006-01-02T15:04:05Z07:00")}
		if load.OrigenDireccion != nil {
			item.Origin = mobileAddress{City: load.OrigenDireccion.Ciudad, State: load.OrigenDireccion.EstadoProvincia, Country: load.OrigenDireccion.Pais, Latitude: load.OrigenDireccion.Latitud, Longitude: load.OrigenDireccion.Longitud}
		}
		if load.DestinoDireccion != nil {
			item.Destination = mobileAddress{City: load.DestinoDireccion.Ciudad, State: load.DestinoDireccion.EstadoProvincia, Country: load.DestinoDireccion.Pais, Latitude: load.DestinoDireccion.Latitud, Longitude: load.DestinoDireccion.Longitud}
		}
		result = append(result, item)
	}
	return ctx.JSON(fiber.Map{"loads": result, "total": total, "page": page, "has_more": hasMore})
}

func (c *MobileController) Accept(ctx fiber.Ctx) error {
	return c.driverAction(ctx, func(id string, driver uint) error { return c.loads.AcceptLoadService(id, driver) })
}

func (c *MobileController) StartTransit(ctx fiber.Ctx) error {
	return c.driverAction(ctx, func(id string, driver uint) error { return c.loads.StartTransit(id, driver) })
}

func (c *MobileController) Deliver(ctx fiber.Ctx) error {
	return c.driverAction(ctx, func(id string, driver uint) error { return c.loads.MarkDelivered(id, driver) })
}

func (c *MobileController) driverAction(ctx fiber.Ctx, action func(string, uint) error) error {
	userID, ok := ctx.Locals("user_id").(uint)
	role, _ := ctx.Locals("role").(string)
	if !ok || userID == 0 {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sesión requerida"})
	}
	if role != "chofer" {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Solo un chofer puede cambiar el estado de una carga"})
	}
	driver, err := c.choferes.GetByUserID(userID)
	if err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Perfil de chofer no disponible"})
	}
	if err := action(ctx.Params("id"), driver.ID); err != nil {
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": fmt.Sprintf("No se pudo completar la operación: %v", err)})
	}
	return ctx.JSON(fiber.Map{"ok": true})
}
