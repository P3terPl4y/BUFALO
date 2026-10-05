package models

import "testing"

func TestLoadTypeAllowLists(t *testing.T) {
	for _, value := range []string{"FTL", "LTL", "parcel", "bulk", "liquid_bulk", "oversized"} {
		if !EsTipoCargaValido(value) {
			t.Errorf("EsTipoCargaValido(%q) = false", value)
		}
	}
	for _, value := range []string{"", "hazmat", "tipo_licencia", "admin", "ftl"} {
		if EsTipoCargaValido(value) {
			t.Errorf("EsTipoCargaValido(%q) = true", value)
		}
	}
	for _, value := range []string{"dry_van", "flatbed", "reefer", "step_deck", "double_drop", "lowboy", "cargo_van", "box_truck", "power_only"} {
		if !EsTipoEquipoValido(value) {
			t.Errorf("EsTipoEquipoValido(%q) = false", value)
		}
	}
	for _, value := range []string{"", "tipo_licencia", "broker", "hazmat"} {
		if EsTipoEquipoValido(value) {
			t.Errorf("EsTipoEquipoValido(%q) = true", value)
		}
	}
}
