package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260917043858CreateDireccionesTable struct{}

// Signature The unique signature for the migration.
func (r *M20260917043858CreateDireccionesTable) Signature() string {
	return "20260917043858_create_direcciones_table"
}

// Up Run the migrations.
func (r *M20260917043858CreateDireccionesTable) Up() error {
	if !facades.Schema().HasTable("direccions") {
		return facades.Schema().Create("direccions", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("owner_id").Nullable()
			table.String("calle", 255).Nullable()
			table.String("ciudad", 100)
			table.String("estado_provincia", 100)
			table.String("codigo_postal", 20).Nullable()
			table.String("pais", 100).Default("Cuba")
			table.Decimal("latitud").Nullable()
			table.Decimal("longitud").Nullable()
			table.TimestampTz("created_at")
			table.TimestampTz("updated_at").Nullable()

			table.Index("ciudad").Name("idx_direccions_ciudad")
			table.Index("codigo_postal").Name("idx_direccions_codigo_postal")
			table.Index("estado_provincia").Name("idx_direccions_estado_provincia")
			table.Index("owner_id").Name("idx_direccions_owner_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260917043858CreateDireccionesTable) Down() error {
	return facades.Schema().DropIfExists("direccions")
}
