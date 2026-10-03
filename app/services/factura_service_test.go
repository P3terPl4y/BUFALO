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
	load := &models.Carga{NumeroReferencia: "BILL-LOAD", PublicadorID: 1, EmpresaID: emisor.ID, ChoferID: &driver.ID, OrigenDireccionID: 1, DestinoDireccionID: 2, FechaRecogida: time.Now(), TipoCarga: models.CargaFTL, TipoEquipo: models.EquipoDryVan, Estado: models.CargaEntregada, Moneda: models.MonedaUSD}
	if err := facades.Orm().Query().Create(load); err != nil {
		t.Fatal(err)
	}
	svc := services.NewFacturaService()
	f := &models.Factura{CargaID: load.ID, EmisorID: emisor.ID, ReceptorID: receptor.ID, NumeroFactura: "BILL-001", FechaEmision: time.Now(), Subtotal: 100, Impuestos: 20, Total: 999, Moneda: models.MonedaUSD}
	wrong := *f
	wrong.ReceptorID = emisor.ID
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
	if f.Total != 120 || f.Estado != models.FacturaBorrador || f.ChoferID == nil || *f.ChoferID != driver.ID {
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
}

func TestFacturaCancelPreservesRecord(t *testing.T) {
	tests.ResetDB(t)
	f := &models.Factura{CargaID: 1, EmisorID: 1, ReceptorID: 2, NumeroFactura: "CANCEL-001", FechaEmision: time.Now(), Subtotal: 100, Total: 100, Moneda: models.MonedaUSD, Estado: models.FacturaBorrador}
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
