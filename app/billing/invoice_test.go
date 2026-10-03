package billing

import (
	"goravel/app/models"
	"math"
	"testing"
	"time"
)

func invoice() *models.Factura {
	return &models.Factura{CargaID: 1, EmisorID: 10, ReceptorID: 20, NumeroFactura: "INV-001", FechaEmision: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Moneda: models.MonedaUSD, Subtotal: 100.10, Impuestos: 20.20, Estado: models.FacturaBorrador}
}
func TestAmounts(t *testing.T) {
	f := invoice()
	f.Total = 999
	if err := Validate(f); err != nil || f.Total != 120.30 {
		t.Fatalf("total %v err %v", f.Total, err)
	}
	for _, n := range []float64{-1, math.NaN(), math.Inf(1), 1e12} {
		f := invoice()
		f.Subtotal = n
		if Validate(f) == nil {
			t.Errorf("accepted %v", n)
		}
	}
}
func TestDecimalHalfUp(t *testing.T) {
	f := invoice()
	f.Subtotal = 1.005
	f.Impuestos = 0.005
	if err := Validate(f); err != nil || f.Subtotal != 1.01 || f.Impuestos != 0.01 || f.Total != 1.02 {
		t.Fatalf("half-up %+v err %v", f, err)
	}
}
func TestInvalidData(t *testing.T) {
	cases := []func(*models.Factura){func(f *models.Factura) { f.CargaID = 0 }, func(f *models.Factura) { f.EmisorID = f.ReceptorID }, func(f *models.Factura) { f.Moneda = "XYZ" }, func(f *models.Factura) { f.NumeroFactura = " " }, func(f *models.Factura) { d := f.FechaEmision.Add(-time.Hour); f.FechaVencimiento = &d }}
	for i, mutate := range cases {
		f := invoice()
		mutate(f)
		if Validate(f) == nil {
			t.Errorf("case %d accepted", i)
		}
	}
}
func TestLifecycle(t *testing.T) {
	f := invoice()
	now := time.Now()
	if Transition(f, models.FacturaPagada, "transferencia", now) == nil {
		t.Fatal("paid draft")
	}
	if err := Transition(f, models.FacturaEmitida, "", now); err != nil {
		t.Fatal(err)
	}
	if Transition(f, models.FacturaPagada, " ", now) == nil {
		t.Fatal("empty method")
	}
	if err := Transition(f, models.FacturaPagada, "transferencia", now); err != nil {
		t.Fatal(err)
	}
	if f.FechaPago == nil || !f.FechaPago.Equal(now) || *f.MetodoPago != "transferencia" {
		t.Fatal("payment metadata")
	}
	if Transition(f, models.FacturaPagada, "efectivo", now) == nil || Transition(f, models.FacturaCancelada, "", now) == nil {
		t.Fatal("terminal state mutated")
	}
}
func TestAllTransitions(t *testing.T) {
	states := []models.EstadoFactura{models.FacturaBorrador, models.FacturaEmitida, models.FacturaVencida, models.FacturaPagada, models.FacturaCancelada}
	for _, from := range states {
		for _, to := range states {
			f := invoice()
			f.Estado = from
			d := time.Now().Add(-time.Hour)
			f.FechaVencimiento = &d
			allowed := (from == models.FacturaBorrador && (to == models.FacturaEmitida || to == models.FacturaCancelada)) || (from == models.FacturaEmitida && (to == models.FacturaPagada || to == models.FacturaVencida || to == models.FacturaCancelada)) || (from == models.FacturaVencida && (to == models.FacturaPagada || to == models.FacturaCancelada))
			if got := Transition(f, to, "transferencia", time.Now()) == nil; got != allowed {
				t.Errorf("%s -> %s allowed %v", from, to, got)
			}
		}
	}
}
func TestAccess(t *testing.T) {
	f := invoice()
	emisor, receptor, other := uint(10), uint(20), uint(99)
	cases := []struct {
		role        string
		empresa     *uint
		write, want bool
	}{{"admin", nil, true, true}, {"publicador", &emisor, true, true}, {"publicador", &other, true, false}, {"chofer", &receptor, false, true}, {"chofer", &receptor, true, false}, {"publicador", nil, false, false}, {"broker", &emisor, false, false}}
	for _, c := range cases {
		if got := CanAccess(10, c.role, c.empresa, f, c.write); got != c.want {
			t.Errorf("%+v got %v", c, got)
		}
	}
	if CanAccess(0, "admin", nil, f, true) {
		t.Fatal("anonymous admin")
	}
}
