package viewhelpers

import (
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
}
