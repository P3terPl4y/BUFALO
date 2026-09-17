package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260916171836CreateCargasTable struct{}

// Signature The unique signature for the migration.
func (r *M20260916171836CreateCargasTable) Signature() string {
	return "20260916171836_create_cargas_table"
}

// Up Run the migrations.
func (r *M20260916171836CreateCargasTable) Up() error {
	if !facades.Schema().HasTable("cargas") {
		return facades.Schema().Create("cargas", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.String("numero_referencia", 50)
			table.UnsignedBigInteger("publicador_id")
			table.UnsignedBigInteger("empresa_id")
			table.UnsignedBigInteger("chofer_id").Nullable()
			table.UnsignedBigInteger("origen_direccion_id")
			table.UnsignedBigInteger("destino_direccion_id")
			table.TimestampTz("fecha_recogida")
			table.TimestampTz("fecha_entrega").Nullable()
			table.Text("tipo_carga")
			table.Text("tipo_equipo")
			table.Decimal("peso_kg").Nullable()
			table.String("commodity", 255).Nullable()
			table.Decimal("distancia_km").Default(0)
			table.Decimal("distancia_real_km").Nullable()
			table.Decimal("tarifa_total").Nullable()
			table.Decimal("tarifa_por_km").Nullable()
			table.String("moneda", 3).Default("CUP")
			table.Text("estado").Default("publicada")
			table.Text("audiencia").Default("load_board")
			table.TimestampTz("created_at")
			table.TimestampTz("updated_at").Nullable()
			table.TimestampTz("deleted_at").Nullable()

			table.Index("chofer_id").Name("idx_cargas_chofer_id")
			table.Index("deleted_at").Name("idx_cargas_deleted_at")
			table.Index("destino_direccion_id").Name("idx_cargas_destino_direccion_id")
			table.Index("distancia_km").Name("idx_cargas_distancia_km")
			table.Index("empresa_id").Name("idx_cargas_empresa_id")
			table.Index("estado").Name("idx_cargas_estado")
			table.Index("fecha_recogida").Name("idx_cargas_fecha_recogida")
			table.Unique("numero_referencia").Name("idx_cargas_numero_referencia")
			table.Index("origen_direccion_id").Name("idx_cargas_origen_direccion_id")
			table.Index("publicador_id").Name("idx_cargas_publicador_id")
			table.Index("tarifa_por_km").Name("idx_cargas_tarifa_por_km")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260916171836CreateCargasTable) Down() error {
	return facades.Schema().DropIfExists("cargas")
}
