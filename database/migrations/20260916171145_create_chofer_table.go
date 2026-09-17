package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260916171145CreateChoferTable struct{}

// Signature The unique signature for the migration.
func (r *M20260916171145CreateChoferTable) Signature() string {
	return "20260916171145_create_chofer_table"
}

// Up Run the migrations.
func (r *M20260916171145CreateChoferTable) Up() error {
	if !facades.Schema().HasTable("chofers") {
		return facades.Schema().Create("chofers", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("user_id")
			table.UnsignedBigInteger("empresa_id")
			table.String("numero_licencia", 50)
			table.String("tipo_licencia", 20)
			table.String("pais_emision_licencia", 100).Default("Cuba")
			table.TimestampTz("fecha_vencimiento_licencia").Nullable()
			table.BigInteger("anios_experiencia").Default(0)
			table.Text("tipos_equipo_permitidos")
			table.Text("certificaciones")
			table.String("numero_seguro", 100)
			table.TimestampTz("fecha_vencimiento_seguro").Nullable()
			table.Text("estado").Default("disponible")
			table.TimestampTz("created_at")
			table.TimestampTz("updated_at").Nullable()
			table.TimestampTz("deleted_at").Nullable()

			table.Index("deleted_at").Name("idx_chofers_deleted_at")
			table.Index("empresa_id").Name("idx_chofers_empresa_id")
			table.Index("estado").Name("idx_chofers_estado")
			table.Unique("numero_licencia").Name("idx_chofers_numero_licencia")
			table.Unique("user_id").Name("idx_chofers_user_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260916171145CreateChoferTable) Down() error {
	return facades.Schema().DropIfExists("chofers")
}
