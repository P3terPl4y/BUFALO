package bootstrap

import (
	"github.com/goravel/framework/contracts/database/schema"
	"goravel/database/migrations"
)

func Migrations() []schema.Migration {
	return []schema.Migration{
		&migrations.M20260916171145CreateChoferTable{},
		&migrations.M20260916171339CreatePublicadorTable{},
		&migrations.M20260916171747CreateEmpresasTable{},
		&migrations.M20260916171836CreateCargasTable{},
		&migrations.M20260916171933CreateFacturasTable{},
		&migrations.M20260916172015CreateCargasHistorialTable{},
		&migrations.M20260916204352CreateUsersTable{},
		&migrations.M20260917043858CreateDireccionesTable{},
	}
}
