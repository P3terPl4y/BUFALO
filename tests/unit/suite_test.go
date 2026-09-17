package unit

import (
	"os"
	"testing"

	"goravel/bootstrap"
)

// TestMain arranca la app (ORM, config, providers) antes de correr los tests.
// Único TestMain por paquete — no añadas otro en este directorio.
func TestMain(m *testing.M) {
	os.Setenv("APP_ENV", "testing")
	os.Setenv("MAIL_MAILER", "log")
	os.Setenv("SESSION_DRIVER", "memory")

	app := bootstrap.Boot()
	app.Boot()
	os.Exit(m.Run())
}
