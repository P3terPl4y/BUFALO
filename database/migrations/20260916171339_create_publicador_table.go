package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260916171339CreatePublicadorTable struct{}

// Signature The unique signature for the migration.
func (r *M20260916171339CreatePublicadorTable) Signature() string {
	return "20260916171339_create_publicador_table"
}

// Up Run the migrations.
func (r *M20260916171339CreatePublicadorTable) Up() error {
	if !facades.Schema().HasTable("publicadors") {
		return facades.Schema().Create("publicadors", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("user_id")
			table.UnsignedBigInteger("empresa_id")
			table.String("numero_licencia_broker", 50)
			table.String("pais_emision_licencia", 100).Default("Cuba")
			table.TimestampTz("fecha_vencimiento_licencia").Nullable()
			table.BigInteger("anios_experiencia").Default(0)
			table.String("especialidad", 100)
			table.Decimal("comision").Default(0)
			table.BigInteger("credit_score").Default(0)
			table.Text("estado").Default("activo")
			table.TimestampTz("created_at")
			table.TimestampTz("updated_at").Nullable()
			table.TimestampTz("deleted_at").Nullable()

			table.Index("deleted_at").Name("idx_publicadors_deleted_at")
			table.Index("empresa_id").Name("idx_publicadors_empresa_id")
			table.Index("estado").Name("idx_publicadors_estado")
			table.Unique("numero_licencia_broker").Name("idx_publicadors_numero_licencia_broker")
			table.Unique("user_id").Name("idx_publicadors_user_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260916171339CreatePublicadorTable) Down() error {
	return facades.Schema().DropIfExists("publicadors")
}
