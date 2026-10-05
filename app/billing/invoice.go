// Package billing contains invoice rules independent of HTTP and persistence.
package billing

import (
	"errors"
	"goravel/app/models"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func CanAccess(userID uint, role string, publicadorID, choferID uint, f *models.Factura, write bool) bool {
	if userID == 0 || f == nil {
		return false
	}
	if role == "admin" {
		return true
	}
	if write {
		if role == "publicador" {
			return publicadorID != 0 && f.EmisorTipo == models.EmisorFacturaPublicador && f.PublicadorID != nil && *f.PublicadorID == publicadorID
		}
		return role == "chofer" && choferID != 0 && f.EmisorTipo == models.EmisorFacturaChofer && f.ChoferID != nil && *f.ChoferID == choferID
	}
	return (role == "publicador" && publicadorID != 0 && f.PublicadorID != nil && *f.PublicadorID == publicadorID) ||
		(role == "chofer" && choferID != 0 && f.ChoferID != nil && *f.ChoferID == choferID)
}

// Validate normalizes amounts to database precision and derives the total.
func Validate(f *models.Factura) error {
	if f == nil {
		return errors.New("factura requerida")
	}
	f.NumeroFactura = strings.TrimSpace(f.NumeroFactura)
	if f.CargaID == 0 || len(f.NumeroFactura) < 3 || len(f.NumeroFactura) > 50 ||
		(f.EmisorID != nil && *f.EmisorID == 0) || (f.ReceptorID != nil && *f.ReceptorID == 0) ||
		(f.EmisorID != nil && f.ReceptorID != nil && *f.EmisorID == *f.ReceptorID) ||
		(f.PublicadorID == nil && f.ChoferID == nil) ||
		(f.PublicadorID != nil && *f.PublicadorID == 0) || (f.ChoferID != nil && *f.ChoferID == 0) {
		return errors.New("identificadores o número inválidos")
	}
	if f.EmisorTipo != models.EmisorFacturaPublicador && f.EmisorTipo != models.EmisorFacturaChofer {
		return errors.New("perfil emisor inválido")
	}
	if (f.EmisorTipo == models.EmisorFacturaPublicador && f.PublicadorID == nil) || (f.EmisorTipo == models.EmisorFacturaChofer && f.ChoferID == nil) {
		return errors.New("la factura debe estar asociada al perfil que la emite")
	}
	if f.FechaEmision.IsZero() || (f.FechaVencimiento != nil && f.FechaVencimiento.Before(f.FechaEmision)) {
		return errors.New("fechas inválidas")
	}
	switch f.Moneda {
	case models.MonedaCUP, models.MonedaMLC, models.MonedaUSD, models.MonedaEUR:
	default:
		return errors.New("moneda inválida")
	}
	for _, n := range []float64{f.Subtotal, f.Impuestos, f.Total, f.DistanciaKm, f.TarifaPorKm} {
		if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 9999999999.99 {
			return errors.New("importe inválido")
		}
	}
	subtotal, impuestos := cents(f.Subtotal), cents(f.Impuestos)
	f.Subtotal = float64(subtotal) / 100
	f.Impuestos = float64(impuestos) / 100
	f.Total = float64(subtotal+impuestos) / 100
	if f.Total > 9999999999.99 {
		return errors.New("total fuera de rango")
	}
	return nil
}

func Transition(f *models.Factura, target models.EstadoFactura, method string, now time.Time) error {
	if f == nil {
		return errors.New("factura requerida")
	}
	allowed := false
	switch target {
	case models.FacturaEmitida:
		allowed = f.Estado == models.FacturaBorrador
	case models.FacturaPagada:
		allowed = f.Estado == models.FacturaEmitida || f.Estado == models.FacturaVencida
	case models.FacturaCancelada:
		allowed = f.Estado == models.FacturaBorrador || f.Estado == models.FacturaEmitida || f.Estado == models.FacturaVencida
	case models.FacturaVencida:
		allowed = f.Estado == models.FacturaEmitida && f.FechaVencimiento != nil && now.After(*f.FechaVencimiento)
	}
	if !allowed {
		return errors.New("transición de factura inválida")
	}
	if target == models.FacturaPagada {
		method = strings.TrimSpace(method)
		if method == "" || len(method) > 50 {
			return errors.New("método de pago requerido (máximo 50 caracteres)")
		}
		f.MetodoPago, f.FechaPago = &method, &now
	}
	f.Estado = target
	return nil
}

// cents rounds the decimal representation rather than its binary approximation.
// This preserves half-up rounding for values such as 1.005.
func cents(value float64) int64 {
	rational, _ := new(big.Rat).SetString(strconv.FormatFloat(value, 'f', -1, 64))
	rational.Mul(rational, big.NewRat(100, 1))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(rational.Num(), rational.Denom(), remainder)
	if remainder.Mul(remainder, big.NewInt(2)).Cmp(rational.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return quotient.Int64()
}
