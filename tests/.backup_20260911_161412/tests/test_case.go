package tests

import (
	"goravel/app/facades"
	"log"

	"github.com/stretchr/testify/suite"
)

type TestCase struct {
	suite.Suite
}

func (t *TestCase) RefreshDatabase() {
	orm := facades.Orm()
	if orm == nil {
		t.FailNow("❌ facades.Orm() es nil — falta llamar app.Boot() en TestMain")
	}

	// Trucar todas las tablas relevantes
	tables := []string{"loads", "users"}
	for _, table := range tables {
		if _, err := orm.Query().Exec("TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE"); err != nil {
			// No falles si la tabla no existe aún; solo loguea
			log.Printf("RefreshDatabase: %s → %v", table, err)
		}
	}
}
