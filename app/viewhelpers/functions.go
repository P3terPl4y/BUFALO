package viewhelpers

import (
	"fmt"

	"github.com/gofiber/template/html/v3"
	"goravel/app/models"
	"strings"
)

// Register keeps production and test template functions identical.
func Register(engine *html.Engine) {
	engine.AddFunc("equipmentChoices", models.EquipmentChoices)
	engine.AddFunc("cargoChoices", models.CargoChoices)
	engine.AddFunc("csvContains", func(raw, value string) bool {
		for _, part := range strings.Split(raw, ",") {
			if strings.TrimSpace(part) == value {
				return true
			}
		}
		return false
	})
	engine.AddFunc("deref", func(p *uint) uint {
		if p == nil {
			return 0
		}
		return *p
	})
	engine.AddFunc("add", func(a, b int) int { return a + b })
	engine.AddFunc("sub", func(a, b int) int { return a - b })
	engine.AddFunc("userProfileURL", UserProfileURL)
	engine.AddFunc("notificationResourceURL", NotificationResourceURL)
	engine.AddFunc("notificationLabel", NotificationLabel)
	engine.AddFunc("notificationIcon", NotificationIcon)
}

// UserProfileURL exposes only the public profile route for known role profiles.
func UserProfileURL(user *models.User) string {
	if user == nil {
		return ""
	}
	switch user.Role {
	case "chofer":
		if user.ChoferID != nil && *user.ChoferID > 0 {
			return fmt.Sprintf("/choferes/%d", *user.ChoferID)
		}
	case "publicador":
		if user.PublicadorID != nil && *user.PublicadorID > 0 {
			return fmt.Sprintf("/publicadores/%d", *user.PublicadorID)
		}
	}
	return ""
}

func NotificationResourceURL(resourceType string, resourceID *uint) string {
	if resourceID == nil || *resourceID == 0 {
		return ""
	}
	switch resourceType {
	case "load":
		return fmt.Sprintf("/loads/%d", *resourceID)
	case "company":
		return fmt.Sprintf("/empresas/%d", *resourceID)
	default:
		return ""
	}
}

func NotificationLabel(eventType string) string {
	switch eventType {
	case "load_assigned":
		return "Asignación de carga"
	case "load_published":
		return "Carga nueva"
	case "load_status":
		return "Estado de carga"
	case "load_interest":
		return "Interés recibido"
	case "membership_requested":
		return "Solicitud de empresa"
	case "membership_decision":
		return "Afiliación"
	default:
		return "Actividad"
	}
}

func NotificationIcon(eventType string) string {
	switch eventType {
	case "load_assigned", "load_published", "load_status":
		return "bi-truck"
	case "load_interest":
		return "bi-chat-left-text"
	case "membership_requested", "membership_decision":
		return "bi-building"
	default:
		return "bi-bell"
	}
}
