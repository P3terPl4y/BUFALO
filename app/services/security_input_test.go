package services_test

import (
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
	"testing"
)

func TestCompanySearchTreatsSQLInjectionAsLiteralInput(t *testing.T) {
	tests.ResetDB(t)
	tests.SeedEmpresa(t, "carrier", "Trusted Carrier")
	tests.SeedEmpresa(t, "broker", "Other Broker")

	rows, total, err := services.NewEmpresaService().GetAllWithFilters(
		map[string]string{"q": "%' OR 1=1 --"}, 1, 100,
	)
	if err != nil {
		t.Fatalf("search returned an error for hostile input: %v", err)
	}
	if total != 0 || len(rows) != 0 {
		t.Fatalf("hostile search widened the query: total=%d rows=%d", total, len(rows))
	}
	if count, err := facades.Orm().Query().Model(&models.Empresa{}).Count(); err != nil || count != 2 {
		t.Fatalf("hostile search changed company data: count=%d err=%v", count, err)
	}
}
