package requests

import (
	"errors"
	"strings"
)

// NormalizePreferences preserves the stored CSV format while rejecting values
// outside the same catalog accepted by load creation. Empty means no filter.
func NormalizePreferences(raw string, valid func(string) bool) (string, error) {
	if len(raw) > 256 {
		return "", errors.New("selección demasiado larga")
	}
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	seen := map[string]bool{}
	values := []string{}
	for _, part := range strings.Split(raw, ",") {
		value := strings.TrimSpace(part)
		if !valid(value) {
			return "", errors.New("selecciona opciones válidas")
		}
		if !seen[value] {
			values = append(values, value)
			seen[value] = true
		}
	}
	return strings.Join(values, ","), nil
}
