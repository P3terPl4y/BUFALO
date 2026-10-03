package viewhelpers

import "github.com/gofiber/template/html/v3"

// Register keeps production and test template functions identical.
func Register(engine *html.Engine) {
	engine.AddFunc("deref", func(p *uint) uint {
		if p == nil {
			return 0
		}
		return *p
	})
	engine.AddFunc("add", func(a, b int) int { return a + b })
	engine.AddFunc("sub", func(a, b int) int { return a - b })
}
