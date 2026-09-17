package tests

import (
	"os"
	"testing"

	"github.com/goravel/framework/foundation"
)

func TestMain(m *testing.M) {
	// Arranca la app y registra providers (ORM, config, etc.)
	app := foundation.NewApplication()
	app.Boot()

	os.Exit(m.Run())
}
