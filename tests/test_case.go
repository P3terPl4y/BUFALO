package tests

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type TestCase struct {
	suite.Suite
}

// RefreshDatabase limpia las tablas entre tests.
// Requiere que la app esté arrancada (ver TestMain en cada paquete).
func (t *TestCase) RefreshDatabase(current *testing.T) {
	if err := RequireTestDatabase(); err != nil {
		current.Fatal(err)
		return
	}
	ResetDB(current)
}
