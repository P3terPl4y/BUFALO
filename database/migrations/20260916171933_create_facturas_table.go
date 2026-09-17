package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260916171933CreateFacturasTable struct{}

// Signature The unique signature for the migration.
func (r *M20260916171933CreateFacturasTable) Signature() string {
	return "20260916171933_create_facturas_table"
}

// Up Run the migrations.
func (r *M20260916171933CreateFacturasTable) Up() error {
	if !facades.Schema().HasTable("facturas") {
		return facades.Schema().Create("facturas", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("carga_id")
			table.UnsignedBigInteger("emisor_id")
			table.UnsignedBigInteger("receptor_id")
			table.UnsignedBigInteger("chofer_id").Nullable()
			table.String("numero_factura", 50)
			table.TimestampTz("fecha_emision")
			table.TimestampTz("fecha_vencimiento").Nullable()
			table.Decimal("distancia_km").Default(0)
			table.Decimal("tarifa_por_km").Default(0)
			table.Decimal("subtotal")
			table.Decimal("impuestos").Default(0)
			table.Decimal("total")
			table.String("moneda", 3).Default("CUP")
			table.Text("estado").Default("borrador")
			table.String("metodo_pago", 50).Nullable()
			table.TimestampTz("fecha_pago").Nullable()
			table.TimestampTz("created_at")
			table.TimestampTz("updated_at").Nullable()
			table.TimestampTz("deleted_at").Nullable()

			table.Unique("carga_id").Name("idx_facturas_carga_id")
			table.Index("chofer_id").Name("idx_facturas_chofer_id")
			table.Index("deleted_at").Name("idx_facturas_deleted_at")
			table.Index("distancia_km").Name("idx_facturas_distancia_km")
			table.Index("emisor_id").Name("idx_facturas_emisor_id")
			table.Index("estado").Name("idx_facturas_estado")
			table.Unique("numero_factura").Name("idx_facturas_numero_factura")
			table.Index("receptor_id").Name("idx_facturas_receptor_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260916171933CreateFacturasTable) Down() error {
	return facades.Schema().DropIfExists("facturas")
}
