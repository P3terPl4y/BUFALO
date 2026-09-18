package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260916204352CreateUsersTable struct{}

// Signature The unique signature for the migration.
func (r *M20260916204352CreateUsersTable) Signature() string {
	return "20260916204352_create_users_table"
}

// Up Run the migrations.
func (r *M20260916204352CreateUsersTable) Up() error {
	// Supabase incluye auth.users en otro esquema. HasTable("users") puede
	// detectarla y saltarse por error la tabla pública de BUFALO, por lo que
	// esta migración debe crear la tabla del esquema configurado explícitamente.
	return facades.Schema().Create("users", func(table schema.Blueprint) {
		table.TimestampTz("created_at").Nullable()
		table.TimestampTz("updated_at").Nullable()
		table.BigIncrements("id")
		table.String("name", 100)
		table.String("email", 100)
		table.String("password", 255)
		table.String("role", 50)
		table.String("phone", 30).Nullable()
		table.String("phone_alt", 30).Nullable()
		table.String("whatsapp", 30).Nullable()
		table.String("telegram", 60).Nullable()
		table.String("emergency_name", 100).Nullable()
		table.String("emergency_phone", 30).Nullable()
		table.UnsignedBigInteger("empresa_id").Nullable()
		table.UnsignedBigInteger("publicador_id").Nullable()
		table.UnsignedBigInteger("chofer_id").Nullable()
		table.Boolean("is_active").Default(true)
		table.TimestampTz("last_login").Nullable()
		table.UnsignedBigInteger("created_by").Nullable()
		table.UnsignedBigInteger("updated_by").Nullable()
		table.Text("address")
		table.String("city", 100)
		table.String("state", 100)
		table.String("country", 100).Default("Cuba")
		table.String("postal_code", 20)
		table.Decimal("latitude")
		table.Decimal("longitude")
		table.BigInteger("radius").Default(100)
		table.Text("preferred_equipment_types")
		table.Text("preferred_cargo_types")
		table.Decimal("max_weight")
		table.Decimal("max_distance")
		table.Text("preferred_routes")
		table.TimestampTz("available_from").Nullable()
		table.TimestampTz("available_to").Nullable()
		table.Text("notes")

		table.Index("chofer_id").Name("idx_users_chofer_id")
		table.Unique("email").Name("idx_users_email")
		table.Index("empresa_id").Name("idx_users_empresa_id")
		table.Index("is_active").Name("idx_users_is_active")
		table.Index("publicador_id").Name("idx_users_publicador_id")
		table.Index("role").Name("idx_users_role")
	})
}

// Down Reverse the migrations.
func (r *M20260916204352CreateUsersTable) Down() error {
	return facades.Schema().DropIfExists("users")
}
