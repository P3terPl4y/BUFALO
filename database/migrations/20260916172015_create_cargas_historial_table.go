package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260916172015CreateCargasHistorialTable struct{}

// Signature The unique signature for the migration.
func (r *M20260916172015CreateCargasHistorialTable) Signature() string {
	return "20260916172015_create_cargas_historial_table"
}

// Up Run the migrations.
func (r *M20260916172015CreateCargasHistorialTable) Up() error {
	if !facades.Schema().HasTable("carga_historials") {
		return facades.Schema().Create("carga_historials", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("carga_id")
			table.Text("estado")
			table.String("comentario", 500).Nullable()
			table.UnsignedBigInteger("usuario_id").Nullable()
			table.TimestampTz("created_at")

			table.Index("carga_id").Name("idx_carga_historials_carga_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260916172015CreateCargasHistorialTable) Down() error {
	return facades.Schema().DropIfExists("carga_historials")
}
