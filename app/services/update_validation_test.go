package services

import (
	"errors"
	"math"
	"testing"
)

func TestResourceUpdateValidation(t *testing.T) {
	tests := []struct {
		name           string
		validate       func(map[string]interface{}) error
		valid, invalid map[string]interface{}
	}{
		{"company", ValidateCompanyUpdate, map[string]interface{}{"nombre_legal": "Transport Company", "tipo": "broker", "estado": "activo", "days_to_pay": float64(30)}, map[string]interface{}{"days_to_pay": math.NaN()}},
		{"driver", ValidateDriverUpdate, map[string]interface{}{"numero_licencia": "LIC-123", "tipo_licencia": "A", "estado": "disponible", "anios_experiencia": 5}, map[string]interface{}{"anios_experiencia": -1}},
		{"publisher", ValidatePublisherUpdate, map[string]interface{}{"numero_licencia_broker": "BRK-123", "estado": "activo", "comision": float64(10)}, map[string]interface{}{"comision": math.Inf(1)}},
		{"address", ValidateAddressUpdate, map[string]interface{}{"ciudad": "Habana", "estado_provincia": "Habana", "pais": "Cuba"}, map[string]interface{}{"ciudad": ""}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.validate(test.valid); err != nil {
				t.Fatal(err)
			}
			if err := test.validate(test.invalid); err == nil {
				t.Fatal("invalid update accepted")
			}
		})
	}
	lat := 91.0
	lon := 0.0
	if ValidateAddressUpdate(map[string]interface{}{"latitud": &lat, "longitud": &lon}) == nil {
		t.Fatal("invalid latitude accepted")
	}
	if ValidateAddressUpdate(map[string]interface{}{"longitud": &lon}) == nil {
		t.Fatal("incomplete coordinates accepted")
	}
}
func TestLookupPreservesInfrastructureFailure(t *testing.T) {
	failure := errors.New("database unavailable")
	err := recordError(failure, 0, "load")
	if !errors.Is(err, failure) || !IsInfrastructureError(err) {
		t.Fatal("database failure was masked")
	}
	err = recordError(nil, 0, "load")
	if !errors.Is(err, ErrNotFound) || IsInfrastructureError(err) {
		t.Fatal("not found was misclassified")
	}
}
