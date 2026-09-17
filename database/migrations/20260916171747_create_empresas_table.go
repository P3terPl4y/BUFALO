package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260916171747CreateEmpresasTable struct{}

// Signature The unique signature for the migration.
func (r *M20260916171747CreateEmpresasTable) Signature() string {
	return "20260916171747_create_empresas_table"
}

// Up Run the migrations.
func (r *M20260916171747CreateEmpresasTable) Up() error {
	if !facades.Schema().HasTable("empresas") {
		return facades.Schema().Create("empresas", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.Text("tipo")
			table.String("nombre_legal", 255)
			table.String("nombre_comercial", 255).Nullable()
			table.String("tax_id", 50).Nullable()
			table.String("mc_number", 20).Nullable()
			table.String("dot_number", 20).Nullable()
			table.UnsignedBigInteger("direccion_id").Nullable()
			table.String("telefono", 30).Nullable()
			table.String("email", 255).Nullable()
			table.String("sitio_web", 255).Nullable()
			table.BigInteger("credit_score").Nullable()
			table.Decimal("days_to_pay").Nullable()
			table.UnsignedBigInteger("owner_id").Nullable()
			table.Text("estado").Default("activo")
			table.TimestampTz("created_at")
			table.TimestampTz("updated_at").Nullable()
			table.TimestampTz("deleted_at").Nullable()

			table.Index("deleted_at").Name("idx_empresas_deleted_at")
			table.Index("direccion_id").Name("idx_empresas_direccion_id")
			table.Index("estado").Name("idx_empresas_estado")
			table.Index("mc_number").Name("idx_empresas_mc_number")
			table.Index("owner_id").Name("idx_empresas_owner_id")
			table.Unique("tax_id").Name("idx_empresas_tax_id")
			table.Index("tipo").Name("idx_empresas_tipo")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260916171747CreateEmpresasTable) Down() error {
	return facades.Schema().DropIfExists("empresas")
}
