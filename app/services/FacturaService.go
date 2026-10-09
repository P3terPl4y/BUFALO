package services

import (
	"errors"
	"goravel/app/billing"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

type FacturaService struct{}

func NewFacturaService() *FacturaService {
	return &FacturaService{}
}

func (s *FacturaService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.Factura, int64, error) {
	page, perPage = NormalizePagination(page, perPage)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	query := facades.Orm().Query().
		Model(&models.Factura{}).
		With("Carga").
		With("Emisor").
		With("Receptor").
		With("Publicador.User").
		With("Chofer.User")

	if estado := filters["estado"]; estado != "" {
		query = query.Where("estado = ?", estado)
	}
	if emisorID := filters["emisor_id"]; emisorID != "" {
		query = query.Where("emisor_id = ?", emisorID)
	}
	if publicadorID := filters["publicador_id"]; publicadorID != "" {
		query = query.Where("publicador_id = ?", publicadorID)
	}
	if receptorID := filters["receptor_id"]; receptorID != "" {
		query = query.Where("receptor_id = ?", receptorID)
	}
	if choferID := filters["chofer_id"]; choferID != "" {
		query = query.Where("chofer_id = ?", choferID)
	}
	if desde := filters["fecha_desde"]; desde != "" {
		if t, err := time.Parse("2006-01-02", desde); err == nil {
			query = query.Where("fecha_emision >= ?", t)
		}
	}
	if hasta := filters["fecha_hasta"]; hasta != "" {
		if t, err := time.Parse("2006-01-02", hasta); err == nil {
			query = query.Where("fecha_emision < ?", t.Add(24*time.Hour))
		}
	}

	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	var list []models.Factura
	err = query.Limit(perPage).Offset(offset).Order("fecha_emision desc").Find(&list)
	return list, total, err
}

func (s *FacturaService) GetByID(id string) (*models.Factura, error) {
	var f models.Factura
	err := facades.Orm().Query().
		With("Carga").
		With("Emisor").
		With("Receptor").
		With("Publicador.User").
		With("Chofer.User").
		Where("id = ?", id).
		First(&f)
	if lookupErr := recordError(err, f.ID, "factura"); lookupErr != nil {
		return nil, lookupErr
	}
	return &f, nil
}

func (s *FacturaService) Create(f *models.Factura) error {
	if err := billing.Validate(f); err != nil {
		return err
	}
	var carga models.Carga
	if err := facades.Orm().Query().With("Chofer").Where("id = ?", f.CargaID).First(&carga); err != nil {
		return err
	}
	if carga.ID == 0 || carga.Estado != models.CargaEntregada || carga.PublicadorID == 0 {
		return errors.New("la factura requiere una carga entregada con un publicador válido")
	}
	var publicador models.Publicador
	if err := facades.Orm().Query().Where("id = ?", carga.PublicadorID).First(&publicador); err != nil || publicador.ID == 0 {
		return errors.New("el perfil publicador asociado a la carga no existe")
	}
	if carga.ChoferID != nil && (carga.Chofer == nil || carga.Chofer.ID != *carga.ChoferID) {
		return errors.New("el perfil chofer asociado a la carga no existe")
	}
	if f.EmisorTipo == models.EmisorFacturaPublicador {
		if f.PublicadorID == nil || *f.PublicadorID != carga.PublicadorID {
			return errors.New("el publicador no pertenece a la carga")
		}
	} else if f.EmisorTipo == models.EmisorFacturaChofer {
		if carga.ChoferID == nil || f.ChoferID == nil || *f.ChoferID != *carga.ChoferID {
			return errors.New("el chofer no está asignado a la carga")
		}
	} else {
		return errors.New("tipo de emisor inválido")
	}

	publicadorID, choferID := carga.PublicadorID, carga.ChoferID
	f.PublicadorID = &publicadorID
	f.ChoferID = choferID
	emisorID := carga.EmpresaID
	if emisorID > 0 {
		f.EmisorID = &emisorID
	} else {
		f.EmisorID = nil
	}
	f.ReceptorID = nil
	if carga.Chofer != nil && carga.Chofer.EmpresaID > 0 {
		receptorID := carga.Chofer.EmpresaID
		f.ReceptorID = &receptorID
	}
	f.Estado = models.FacturaBorrador
	f.FechaPago = nil
	f.MetodoPago = nil
	return facades.Orm().Query().Create(f)
}

func (s *FacturaService) Update(id string, updates map[string]interface{}) error {
	f, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if f.Estado != models.FacturaBorrador {
		return errors.New("solo se editan borradores")
	}
	for key, val := range updates {
		if key != "moneda" && key != "metodo_pago" {
			if _, ok := val.(float64); !ok {
				return errors.New("importe inválido")
			}
		}
		switch key {
		case "subtotal":
			f.Subtotal, _ = val.(float64)
		case "impuestos":
			f.Impuestos, _ = val.(float64)
		case "total":
			f.Total, _ = val.(float64)
		case "distancia_km":
			f.DistanciaKm, _ = val.(float64)
		case "tarifa_por_km":
			f.TarifaPorKm, _ = val.(float64)
		case "moneda":
			currency, ok := val.(string)
			if !ok {
				return errors.New("moneda inválida")
			}
			f.Moneda = models.Moneda(currency)
		case "metodo_pago": // Payment data is set only when recording payment.
		default:
			return errors.New("campo no editable")
		}
	}
	if err := billing.Validate(f); err != nil {
		return err
	}
	values := map[string]interface{}{"subtotal": f.Subtotal, "impuestos": f.Impuestos, "total": f.Total, "distancia_km": f.DistanciaKm, "tarifa_por_km": f.TarifaPorKm, "moneda": f.Moneda}
	result, err := facades.Orm().Query().Model(&models.Factura{}).Where("id = ? AND estado = ?", id, models.FacturaBorrador).Update(values)
	if err == nil && result.RowsAffected != 1 {
		return errors.New("factura cambió concurrentemente")
	}
	return err
}

// Delete preserves invoice records by cancelling them.
func (s *FacturaService) Delete(id string) error {
	return s.CambiarEstado(id, models.FacturaCancelada, "")
}
func (s *FacturaService) MarcarPagada(id, metodo string) error {
	return s.CambiarEstado(id, models.FacturaPagada, metodo)
}
func (s *FacturaService) CambiarEstado(id string, target models.EstadoFactura, method string) error {
	f, err := s.GetByID(id)
	if err != nil {
		return err
	}
	previous := f.Estado
	if err := billing.Transition(f, target, method, time.Now()); err != nil {
		return err
	}
	values := map[string]interface{}{"estado": f.Estado}
	if target == models.FacturaPagada {
		values["metodo_pago"] = f.MetodoPago
		values["fecha_pago"] = f.FechaPago
	}
	result, err := facades.Orm().Query().Model(&models.Factura{}).Where("id = ? AND estado = ?", id, previous).Update(values)
	if err == nil && result.RowsAffected != 1 {
		return errors.New("factura cambió concurrentemente")
	}
	return err
}
