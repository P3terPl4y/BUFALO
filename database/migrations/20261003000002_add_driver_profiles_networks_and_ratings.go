package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"goravel/app/facades"
)

type M20261003000002AddDriverProfilesNetworksAndRatings struct{}

func (m *M20261003000002AddDriverProfilesNetworksAndRatings) Signature() string {
	return "20261003000002_add_driver_profiles_networks_and_ratings"
}

func (m *M20261003000002AddDriverProfilesNetworksAndRatings) Up() error {
	if facades.Schema().HasTable("users") && !facades.Schema().HasColumn("users", "profile_photo") {
		if err := facades.Schema().Table("users", func(table schema.Blueprint) {
			table.String("profile_photo", 255).Nullable()
		}); err != nil {
			return err
		}
	}
	if facades.Schema().HasTable("chofers") {
		if !facades.Schema().HasColumn("chofers", "rating_average") {
			if err := facades.Schema().Table("chofers", func(table schema.Blueprint) {
				table.Decimal("rating_average").Total(3).Places(2).Default(0)
			}); err != nil {
				return err
			}
		}
		if !facades.Schema().HasColumn("chofers", "rating_count") {
			if err := facades.Schema().Table("chofers", func(table schema.Blueprint) {
				table.BigInteger("rating_count").Default(0)
			}); err != nil {
				return err
			}
		}
	}
	if !facades.Schema().HasTable("chofer_calificaciones") {
		if err := facades.Schema().Create("chofer_calificaciones", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("carga_id")
			table.UnsignedBigInteger("chofer_id")
			table.UnsignedBigInteger("publicador_id")
			table.Integer("puntaje")
			table.Text("comentario").Nullable()
			table.TimestampTz("created_at")
			table.Unique("carga_id").Name("idx_chofer_calificaciones_carga")
			table.Index("chofer_id").Name("idx_chofer_calificaciones_chofer")
			table.Index("publicador_id").Name("idx_chofer_calificaciones_publicador")
		}); err != nil {
			return err
		}
	}
	if !facades.Schema().HasTable("red_choferes") {
		if err := facades.Schema().Create("red_choferes", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("empresa_id")
			table.UnsignedBigInteger("chofer_id")
			table.TimestampTz("created_at")
			table.Unique("empresa_id", "chofer_id").Name("idx_red_chofer_empresa_chofer")
			table.Index("chofer_id").Name("idx_red_chofer_chofer")
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m *M20261003000002AddDriverProfilesNetworksAndRatings) Down() error {
	if err := facades.Schema().DropIfExists("red_choferes"); err != nil {
		return err
	}
	if err := facades.Schema().DropIfExists("chofer_calificaciones"); err != nil {
		return err
	}
	if facades.Schema().HasTable("chofers") {
		if facades.Schema().HasColumn("chofers", "rating_count") {
			if err := facades.Schema().Table("chofers", func(table schema.Blueprint) { table.DropColumn("rating_count") }); err != nil {
				return err
			}
		}
		if facades.Schema().HasColumn("chofers", "rating_average") {
			if err := facades.Schema().Table("chofers", func(table schema.Blueprint) { table.DropColumn("rating_average") }); err != nil {
				return err
			}
		}
	}
	if facades.Schema().HasTable("users") && facades.Schema().HasColumn("users", "profile_photo") {
		return facades.Schema().Table("users", func(table schema.Blueprint) { table.DropColumn("profile_photo") })
	}
	return nil
}
