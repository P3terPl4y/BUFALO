package tests

import (
	"goravel/app/facades"
	"log"

	"github.com/stretchr/testify/suite"
)

type TestCase struct {
	suite.Suite
}

// RefreshDatabase limpia las tablas entre tests.
// Requiere que la app esté arrancada (ver TestMain en cada paquete).
func (t *TestCase) RefreshDatabase() {
	orm := facades.Orm()
	if orm == nil {
		t.FailNow("❌ facades.Orm() es nil — revisa que TestMain llame a bootstrap.Boot() y app.Boot()")
	}

	for _, table := range []string{"loads", "users"} {
		if _, err := orm.Query().Exec("TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE"); err != nil {
			log.Printf("RefreshDatabase: %s → %v", table, err)
		}
	}
}
