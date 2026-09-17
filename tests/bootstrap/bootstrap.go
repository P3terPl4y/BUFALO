// tests/bootstrap/bootstrap.go
package bootstrap

import (
	"github.com/goravel/framework/foundation"
)

func Boot() {
	app := foundation.NewApplication()
	app.Boot()
}
