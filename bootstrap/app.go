package bootstrap

import (
	"github.com/goravel/framework/contracts/database/schema"
	contractsfoundation "github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/foundation"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/config"
)

func Boot() contractsfoundation.Application {
	return foundation.Setup().
		WithMigrations(Migrations).
		WithProviders(Providers).
		WithConfig(config.Boot).
		WithCallback(func() {
			facades.Schema().Extend(schema.Extension{
				Models: []any{
					&models.Direccion{},
&models.Empresa{},
&models.User{},        // FK a Empresa
&models.Chofer{},      // FK a User + Empresa
&models.Publicador{},  // FK a User + Empresa
&models.Carga{},       // FK a Publicador + Empresa + Chofer + Direcciones
&models.Factura{},     // FK a Carga + Empresas + Chofer
					models.CargaHistorial{},
				},
			})
		}).
		Create()
}
