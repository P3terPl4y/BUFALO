package services_test

import (
	"fmt"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
	"testing"
	"time"
)

func TestFacturaPersistenceLifecycle(t *testing.T) {
	tests.ResetDB(t)
	emisor := tests.SeedEmpresa(t, "broker", "Emisor")
	receptor := tests.SeedEmpresa(t, "carrier", "Receptor")
	brokerUser := tests.NewUserPublicador("billing-broker@test.local")
	if err := facades.Orm().Query().Create(brokerUser); err != nil {
		t.Fatal(err)
	}
	broker := tests.NewPublicadorProfile()
	broker.UserID, broker.EmpresaID = brokerUser.ID, emisor.ID
	if err := facades.Orm().Query().Create(broker); err != nil {
		t.Fatal(err)
	}
	user := tests.NewUserChofer("billing@test.local")
	if err := facades.Orm().Query().Create(user); err != nil {
		t.Fatal(err)
	}
	driver := tests.NewChoferProfile()
	driver.UserID = user.ID
	driver.EmpresaID = receptor.ID
	if err := facades.Orm().Query().Create(driver); err != nil {
		t.Fatal(err)
	}
	load := &models.Carga{NumeroReferencia: "BILL-LOAD", PublicadorID: broker.ID, EmpresaID: emisor.ID, ChoferID: &driver.ID, OrigenDireccionID: 1, DestinoDireccionID: 2, FechaRecogida: time.Now(), TipoCarga: models.CargaFTL, TipoEquipo: models.EquipoDryVan, Estado: models.CargaEntregada, Moneda: models.MonedaUSD}
	if err := facades.Orm().Query().Create(load); err != nil {
		t.Fatal(err)
	}
	svc := services.NewFacturaService()
	f := &models.Factura{CargaID: load.ID, EmisorTipo: models.EmisorFacturaPublicador, PublicadorID: &broker.ID, NumeroFactura: "BILL-001", FechaEmision: time.Now(), Subtotal: 100, Impuestos: 20, Total: 999, Moneda: models.MonedaUSD}
	wrong := *f
	wrong.PublicadorID = tests.PtrUint(999999)
	if svc.Create(&wrong) == nil {
		t.Fatal("mismatched parties accepted")
	}
	missing := *f
	missing.CargaID = 999999
	if svc.Create(&missing) == nil {
		t.Fatal("missing load accepted")
	}
	if err := svc.Create(f); err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprint(f.ID)
	if f.Total != 120 || f.Estado != models.FacturaBorrador || f.ChoferID == nil || *f.ChoferID != driver.ID || f.EmisorID == nil || *f.EmisorID != emisor.ID || f.ReceptorID == nil || *f.ReceptorID != receptor.ID {
		t.Fatalf("incorrect invoice %+v", f)
	}
	duplicate := *f
	duplicate.ID = 0
	if svc.Create(&duplicate) == nil {
		t.Fatal("duplicate invoice accepted")
	}
	if svc.MarcarPagada(id, "transferencia") == nil {
		t.Fatal("paid draft")
	}
	if err := svc.Update(id, map[string]interface{}{"subtotal": float64(150), "impuestos": float64(30), "total": float64(0)}); err != nil {
		t.Fatal(err)
	}
	if err := svc.CambiarEstado(id, models.FacturaEmitida, ""); err != nil {
		t.Fatal(err)
	}
	if svc.Update(id, map[string]interface{}{"subtotal": float64(1)}) == nil {
		t.Fatal("edited issued invoice")
	}
	if err := svc.MarcarPagada(id, "transferencia"); err != nil {
		t.Fatal(err)
	}
	saved, err := svc.GetByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Total != 180 || saved.Estado != models.FacturaPagada || saved.FechaPago == nil {
		t.Fatalf("incorrect persisted invoice %+v", saved)
	}
	if svc.MarcarPagada(id, "efectivo") == nil || svc.Delete(id) == nil {
		t.Fatal("modified paid invoice")
	}
	if svc.MarcarPagada("999999", "transferencia") == nil {
		t.Fatal("nonexistent invoice paid")
	}
	loadForDriver := *load
	loadForDriver.ID = 0
	loadForDriver.NumeroReferencia = "BILL-LOAD-DRIVER"
	if err := facades.Orm().Query().Create(&loadForDriver); err != nil {
		t.Fatal(err)
	}
	driverInvoice := &models.Factura{CargaID: loadForDriver.ID, ChoferID: &driver.ID, EmisorTipo: models.EmisorFacturaChofer, NumeroFactura: "BILL-DRIVER-001", FechaEmision: time.Now(), Subtotal: 50, Moneda: models.MonedaUSD}
	if err := svc.Create(driverInvoice); err != nil {
		t.Fatalf("driver-issued invoice rejected: %v", err)
	}
	if driverInvoice.PublicadorID == nil || *driverInvoice.PublicadorID != broker.ID || driverInvoice.ChoferID == nil || *driverInvoice.ChoferID != driver.ID {
		t.Fatalf("invoice participants were not associated: %+v", driverInvoice)
	}
}

func TestFacturaCreationCurrencyIssuerAndDueDateMatrix(t *testing.T) {
	tests.ResetDB(t)
	emisorEmpresa := tests.SeedEmpresa(t, "broker", "Emisor matriz")
	receptorEmpresa := tests.SeedEmpresa(t, "carrier", "Receptor matriz")

	publicadorUser := tests.NewUserPublicador("billing-matrix-publisher@test.local")
	if err := facades.Orm().Query().Create(publicadorUser); err != nil {
		t.Fatal(err)
	}
	publicador := tests.NewPublicadorProfile()
	publicador.UserID, publicador.EmpresaID = publicadorUser.ID, emisorEmpresa.ID
	if err := facades.Orm().Query().Create(publicador); err != nil {
		t.Fatal(err)
	}

	choferUser := tests.NewUserChofer("billing-matrix-driver@test.local")
	if err := facades.Orm().Query().Create(choferUser); err != nil {
		t.Fatal(err)
	}
	chofer := tests.NewChoferProfile()
	chofer.UserID, chofer.EmpresaID = choferUser.ID, receptorEmpresa.ID
	if err := facades.Orm().Query().Create(chofer); err != nil {
		t.Fatal(err)
	}
	templateService := services.NewFacturaPlantillaService()
	defaultTemplate, err := templateService.Save(publicador.ID, services.FacturaPlantillaInput{
		Nombre: "Diseño principal", Preset: "classic", Formato: "a4", Color: "#253746",
		Bloques: []string{"brand", "parties", "details", "items", "totals"}, Predeterminada: true,
	})
	if err != nil {
		t.Fatalf("save default template: %v", err)
	}
	alternateTemplate, err := templateService.Save(publicador.ID, services.FacturaPlantillaInput{
		Nombre: "Diseño recibo", Preset: "receipt", Formato: "receipt", Color: "#123456",
		Bloques: []string{"brand", "parties", "details", "totals"},
	})
	if err != nil {
		t.Fatalf("save alternate template: %v", err)
	}

	monedas := []models.Moneda{models.MonedaCUP, models.MonedaMLC, models.MonedaUSD, models.MonedaEUR}
	issuerTypes := []models.TipoEmisorFactura{models.EmisorFacturaPublicador, models.EmisorFacturaChofer}
	svc := services.NewFacturaService()
	caseNumber := 0
	for _, currency := range monedas {
		for _, issuer := range issuerTypes {
			for _, withDueDate := range []bool{false, true} {
				caseNumber++
				load := &models.Carga{
					NumeroReferencia:   fmt.Sprintf("BILL-MATRIX-%02d", caseNumber),
					PublicadorID:       publicador.ID,
					EmpresaID:          emisorEmpresa.ID,
					ChoferID:           &chofer.ID,
					OrigenDireccionID:  1,
					DestinoDireccionID: 2,
					FechaRecogida:      time.Now(),
					TipoCarga:          models.CargaFTL,
					TipoEquipo:         models.EquipoDryVan,
					Estado:             models.CargaEntregada,
					Moneda:             currency,
				}
				if err := facades.Orm().Query().Create(load); err != nil {
					t.Fatalf("case %d create delivered load: %v", caseNumber, err)
				}

				invoice := &models.Factura{
					CargaID: load.ID, EmisorTipo: issuer,
					NumeroFactura: fmt.Sprintf("MATRIX-%02d", caseNumber),
					FechaEmision:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					Subtotal:      123.45, Impuestos: 6.55, Total: 999,
					Moneda: currency, MetodoPago: tests.StrPtr("previsto"),
				}
				if caseNumber%2 == 0 {
					invoice.PlantillaID = &alternateTemplate.ID
				}
				if issuer == models.EmisorFacturaPublicador {
					invoice.PublicadorID = &publicador.ID
				} else {
					invoice.ChoferID = &chofer.ID
				}
				if withDueDate {
					dueDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
					invoice.FechaVencimiento = &dueDate
				}
				if err := svc.Create(invoice); err != nil {
					t.Fatalf("case %d (%s, %s, due=%v): create: %v", caseNumber, currency, issuer, withDueDate, err)
				}
				saved, err := svc.GetByID(fmt.Sprint(invoice.ID))
				if err != nil {
					t.Fatalf("case %d reload: %v", caseNumber, err)
				}
				if saved.Moneda != currency || saved.EmisorTipo != issuer || saved.Total != 130 || saved.Estado != models.FacturaBorrador || saved.MetodoPago != nil {
					t.Errorf("case %d persisted unexpected invoice: %+v", caseNumber, saved)
				}
				if (saved.FechaVencimiento != nil) != withDueDate {
					t.Errorf("case %d due date present=%v, want %v", caseNumber, saved.FechaVencimiento != nil, withDueDate)
				}
				if saved.PublicadorID == nil || *saved.PublicadorID != publicador.ID || saved.ChoferID == nil || *saved.ChoferID != chofer.ID {
					t.Errorf("case %d participant association missing: %+v", caseNumber, saved)
				}
				wantTemplateID := defaultTemplate.ID
				if caseNumber%2 == 0 {
					wantTemplateID = alternateTemplate.ID
				}
				if saved.PlantillaID == nil || *saved.PlantillaID != wantTemplateID {
					t.Errorf("case %d selected template id=%v, want %d", caseNumber, saved.PlantillaID, wantTemplateID)
				}
			}
		}
	}
	firstID := fmt.Sprint(1)
	if err := services.NewFacturaService().SetTemplate(firstID, publicador.ID, &alternateTemplate.ID); err != nil {
		t.Fatalf("change invoice template: %v", err)
	}
	changed, err := services.NewFacturaService().GetByID(firstID)
	if err != nil || changed.Plantilla == nil || changed.Plantilla.ID != alternateTemplate.ID || changed.Total != 130 || changed.Estado != models.FacturaBorrador {
		t.Fatalf("template change altered invoice or did not persist: %+v err=%v", changed, err)
	}
}

func TestFacturaTemplateIsolationAndRequiredBlocks(t *testing.T) {
	tests.ResetDB(t)
	company := tests.SeedEmpresa(t, "broker", "Plantillas")
	users := []*models.User{tests.NewUserPublicador("template-owner@test.local"), tests.NewUserPublicador("template-other@test.local")}
	profiles := make([]*models.Publicador, 0, len(users))
	for _, user := range users {
		if err := facades.Orm().Query().Create(user); err != nil {
			t.Fatal(err)
		}
		profile := tests.NewPublicadorProfile()
		profile.NumeroLicenciaBroker = fmt.Sprintf("TEMPLATE-%d", user.ID)
		profile.UserID, profile.EmpresaID = user.ID, company.ID
		if err := facades.Orm().Query().Create(profile); err != nil {
			t.Fatal(err)
		}
		profiles = append(profiles, profile)
	}
	service := services.NewFacturaPlantillaService()
	input := services.FacturaPlantillaInput{
		Nombre: "Solo mío", Formato: "letter", Color: "#ABCDEF",
		Bloques: []string{"parties", "details", "totals"},
	}
	owned, err := service.Save(profiles[0].ID, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ID = owned.ID
	if _, err := service.Save(profiles[1].ID, input); err == nil {
		t.Fatal("another broker modified a template they do not own")
	}
	for _, invalid := range []services.FacturaPlantillaInput{
		{Nombre: "Sin importes", Formato: "a4", Color: "#123456", Bloques: []string{"brand", "parties", "details"}},
		{Nombre: "Color inválido", Formato: "a4", Color: "url(javascript:alert(1))", Bloques: []string{"parties", "details", "totals"}},
		{Nombre: "Bloque inyectado", Formato: "a4", Color: "#123456", Bloques: []string{"parties", "details", "totals", "script"}},
	} {
		if _, err := service.Save(profiles[0].ID, invalid); err == nil {
			t.Errorf("invalid template accepted: %+v", invalid)
		}
	}
}

func TestFacturaCancelPreservesRecord(t *testing.T) {
	tests.ResetDB(t)
	publicadorID := uint(1)
	f := &models.Factura{CargaID: 1, PublicadorID: &publicadorID, EmisorTipo: models.EmisorFacturaPublicador, NumeroFactura: "CANCEL-001", FechaEmision: time.Now(), Subtotal: 100, Total: 100, Moneda: models.MonedaUSD, Estado: models.FacturaBorrador}
	if err := facades.Orm().Query().Create(f); err != nil {
		t.Fatal(err)
	}
	svc := services.NewFacturaService()
	id := fmt.Sprint(f.ID)
	if err := svc.Delete(id); err != nil {
		t.Fatal(err)
	}
	saved, err := svc.GetByID(id)
	if err != nil || saved.Estado != models.FacturaCancelada {
		t.Fatalf("record not preserved %+v %v", saved, err)
	}
	if svc.Delete(id) == nil || svc.CambiarEstado(id, models.FacturaEmitida, "") == nil {
		t.Fatal("cancelled invoice mutated")
	}
	if svc.Update(id, map[string]interface{}{"subtotal": float64(1)}) == nil {
		t.Fatal("cancelled invoice edited")
	}
}
