package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

// M20261003000001FixDireccionCoordinatePrecision preserves GPS precision.
// The original Decimal columns defaulted to scale 2 and rounded map picks.
type M20261003000001FixDireccionCoordinatePrecision struct{}

func (r *M20261003000001FixDireccionCoordinatePrecision) Signature() string {
	return "20261003000001_fix_direccion_coordinate_precision"
}

func (r *M20261003000001FixDireccionCoordinatePrecision) Up() error {
	if !facades.Schema().HasTable("direccions") {
		return nil
	}
	return facades.Schema().Table("direccions", func(table schema.Blueprint) {
		if facades.Schema().HasColumn("direccions", "latitud") {
			table.Decimal("latitud").Total(10).Places(7).Nullable().Change()
		}
		if facades.Schema().HasColumn("direccions", "longitud") {
			table.Decimal("longitud").Total(10).Places(7).Nullable().Change()
		}
	})
}

func (r *M20261003000001FixDireccionCoordinatePrecision) Down() error {
	if !facades.Schema().HasTable("direccions") {
		return nil
	}
	return facades.Schema().Table("direccions", func(table schema.Blueprint) {
		if facades.Schema().HasColumn("direccions", "latitud") {
			table.Decimal("latitud").Total(10).Places(2).Nullable().Change()
		}
		if facades.Schema().HasColumn("direccions", "longitud") {
			table.Decimal("longitud").Total(10).Places(2).Nullable().Change()
		}
	})
}
