package services

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

func validateText(values map[string]interface{}, key string, min, max int) error {
	value, present := values[key]
	if !present {
		return nil
	}
	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("%s inválido", key)
	}
	size := utf8.RuneCountInString(strings.TrimSpace(text))
	if size < min || size > max {
		return fmt.Errorf("%s inválido", key)
	}
	return nil
}
func validateChoice(values map[string]interface{}, key string, choices ...string) error {
	value, present := values[key]
	if !present {
		return nil
	}
	text := fmt.Sprint(value)
	for _, choice := range choices {
		if text == choice {
			return nil
		}
	}
	return fmt.Errorf("%s inválido", key)
}
func ValidateCompanyUpdate(values map[string]interface{}) error {
	if err := validateText(values, "nombre_legal", 3, 255); err != nil {
		return err
	}
	if err := validateChoice(values, "tipo", "broker", "carrier", "shipper", "factoring", "mixto"); err != nil {
		return err
	}
	if err := validateChoice(values, "estado", "activo", "inactivo", "suspendido"); err != nil {
		return err
	}
	if photo, ok := values["profile_photo"]; ok {
		value, valid := photo.(string)
		if !valid || len(value) > 255 || (value != "" && !strings.HasPrefix(value, "/uploads/company-logos/")) {
			return fmt.Errorf("imagen de empresa inválida")
		}
	}
	if days, ok := values["days_to_pay"].(float64); ok && (math.IsNaN(days) || math.IsInf(days, 0) || days < 0 || days > 999) {
		return fmt.Errorf("plazo de pago inválido")
	}
	return nil
}
func ValidateDriverUpdate(values map[string]interface{}) error {
	if err := validateText(values, "numero_licencia", 3, 50); err != nil {
		return err
	}
	if err := validateText(values, "tipo_licencia", 1, 20); err != nil {
		return err
	}
	if years, ok := values["anios_experiencia"].(int); ok && (years < 0 || years > 100) {
		return fmt.Errorf("experiencia inválida")
	}
	return validateChoice(values, "estado", "disponible", "en_viaje", "inactivo")
}
func ValidatePublisherUpdate(values map[string]interface{}) error {
	if err := validateText(values, "numero_licencia_broker", 3, 50); err != nil {
		return err
	}
	if years, ok := values["anios_experiencia"].(int); ok && (years < 0 || years > 100) {
		return fmt.Errorf("experiencia inválida")
	}
	if commission, ok := values["comision"].(float64); ok && (math.IsNaN(commission) || math.IsInf(commission, 0) || commission < 0 || commission > 100) {
		return fmt.Errorf("comisión inválida")
	}
	return validateChoice(values, "estado", "activo", "inactivo", "suspendido")
}
func ValidateAddressUpdate(values map[string]interface{}) error {
	for _, key := range []string{"ciudad", "estado_provincia", "pais"} {
		if err := validateText(values, key, 2, 100); err != nil {
			return err
		}
	}
	lat, _ := values["latitud"].(*float64)
	lon, _ := values["longitud"].(*float64)
	if (lat == nil) != (lon == nil) {
		return fmt.Errorf("coordenadas incompletas")
	}
	if lat != nil && (math.IsNaN(*lat) || math.IsInf(*lat, 0) || *lat < -90 || *lat > 90 || math.IsNaN(*lon) || math.IsInf(*lon, 0) || *lon < -180 || *lon > 180) {
		return fmt.Errorf("coordenadas inválidas")
	}
	return nil
}
