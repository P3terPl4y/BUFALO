package services

import (
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

type FacturaService struct{}

func NewFacturaService() *FacturaService {
	return &FacturaService{}
}

func (s *FacturaService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.Factura, int64, error) {
	query := facades.Orm().Query().
		Model(&models.Factura{}).
		With("Carga").
		With("Emisor").
		With("Receptor").
		With("Chofer")

	if estado := filters["estado"]; estado != "" {
		query = query.Where("estado = ?", estado)
	}
	if emisorID := filters["emisor_id"]; emisorID != "" {
		query = query.Where("emisor_id = ?", emisorID)
	}
	if receptorID := filters["receptor_id"]; receptorID != "" {
		query = query.Where("receptor_id = ?", receptorID)
	}
	if desde := filters["fecha_desde"]; desde != "" {
		if t, err := time.Parse("2006-01-02", desde); err == nil {
			query = query.Where("fecha_emision >= ?", t)
		}
	}
	if hasta := filters["fecha_hasta"]; hasta != "" {
		if t, err := time.Parse("2006-01-02", hasta); err == nil {
			query = query.Where("fecha_emision <= ?", t.Add(24*time.Hour))
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
		With("Chofer").
		Where("id = ?", id).
		First(&f)
	if err != nil || f.ID == 0 {
		return nil, errors.New("factura not found")
	}
	return &f, nil
}

func (s *FacturaService) Create(f *models.Factura) error {
	return facades.Orm().Query().Create(f)
}

func (s *FacturaService) Update(id string, updates map[string]interface{}) error {
	_, err := facades.Orm().Query().Model(&models.Factura{}).Where("id = ?", id).Update(updates)
	return err
}

func (s *FacturaService) Delete(id string) error {
	_, err := facades.Orm().Query().Where("id = ?", id).Delete(&models.Factura{})
	return err
}

// MarcarPagada cambia el estado de la factura a pagada.
func (s *FacturaService) MarcarPagada(id string, metodo string) error {
	now := time.Now()
	_, err := facades.Orm().Query().
		Model(&models.Factura{}).
		Where("id = ?", id).
		Update(map[string]interface{}{
			"estado":      "pagada",
			"metodo_pago": metodo,
			"fecha_pago":  now,
		})
	return err
}
