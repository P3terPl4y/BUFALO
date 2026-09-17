
package controllers

import "fmt"

// strPtr convierte "" → nil, y "algo" → *"algo".
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// uintToUUID formatea un uint como UUID v4 con el valor embebido al final.
// Ej: 4 → "00000000-0000-0000-0000-000000000004"
func uintToUUID(u uint) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", u)
}
