package services

import (
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
	"strconv"
	"time"
)

type CargaService struct{}

func NewCargaService() *CargaService {
	return &CargaService{}
}

// ─────────────────────────────────────────────────────────────
// Listar con filtros
// ─────────────────────────────────────────────────────────────
// GetAllWithFilters lista cargas visibles para el usuario según filtros y
// paginación. La autorización de ownership se aplica antes de editar o borrar.
func (s *CargaService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.Carga, int64, error) {
	query := facades.Orm().Query().
		Model(&models.Carga{}).
		With("Publicador").
		With("Empresa").
		With("Chofer").
		With("OrigenDireccion").
		With("DestinoDireccion")

	if status := filters["status"]; status != "" {
		query = query.Where("estado = ?", status)
	}
	if tipoEquipo := filters["tipo_equipo"]; tipoEquipo != "" {
		query = query.Where("tipo_equipo = ?", tipoEquipo)
	}
	if tipoCarga := filters["tipo_carga"]; tipoCarga != "" {
		query = query.Where("tipo_carga = ?", tipoCarga)
	}
	if publicadorID := filters["publicador_id"]; publicadorID != "" {
		if _, err := strconv.ParseUint(publicadorID, 10, 32); err == nil {
			query = query.Where("publicador_id = ?", publicadorID)
		}
	}
	if choferID := filters["chofer_id"]; choferID != "" {
		if _, err := strconv.ParseUint(choferID, 10, 32); err == nil {
			query = query.Where("chofer_id = ?", choferID)
		}
	}
	if empresaID := filters["empresa_id"]; empresaID != "" {
		if _, err := strconv.ParseUint(empresaID, 10, 32); err == nil {
			query = query.Where("empresa_id = ?", empresaID)
		}
	}
	if minRate := filters["min_rate"]; minRate != "" {
		if _, err := strconv.ParseFloat(minRate, 64); err == nil {
			query = query.Where("tarifa_total >= ?", minRate)
		}
	}
	if maxRate := filters["max_rate"]; maxRate != "" {
		if _, err := strconv.ParseFloat(maxRate, 64); err == nil {
			query = query.Where("tarifa_total <= ?", maxRate)
		}
	}
	if pickupFrom := filters["pickup_from"]; pickupFrom != "" {
		if t, err := time.Parse("2006-01-02", pickupFrom); err == nil {
			query = query.Where("fecha_recogida >= ?", t)
		}
	}
	if pickupTo := filters["pickup_to"]; pickupTo != "" {
		if t, err := time.Parse("2006-01-02", pickupTo); err == nil {
			query = query.Where("fecha_recogida <= ?", t.Add(24*time.Hour))
		}
	}

	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	var list []models.Carga
	err = query.Limit(perPage).Offset(offset).Order("fecha_recogida desc").Find(&list)
	return list, total, err
}

// ─────────────────────────────────────────────────────────────
// Obtener una por ID
// ─────────────────────────────────────────────────────────────
func (s *CargaService) GetByID(id string) (*models.Carga, error) {
	var c models.Carga
	err := facades.Orm().Query().
		With("Publicador").
		With("Empresa").
		With("Chofer").
		With("OrigenDireccion").
		With("DestinoDireccion").
		With("Factura").
		Where("id = ?", id).
		First(&c)
	if err != nil || c.ID == 0 {
		return nil, errors.New("carga not found")
	}
	return &c, nil
}

func (s *CargaService) Create(c *models.Carga) error {
	return facades.Orm().Query().Create(c)
}

func (s *CargaService) Update(id string, updates map[string]interface{}) error {
	_, err := facades.Orm().Query().Model(&models.Carga{}).Where("id = ?", id).Update(updates)
	return err
}

func (s *CargaService) Delete(id string) error {
	_, err := facades.Orm().Query().Where("id = ?", id).Delete(&models.Carga{})
	return err
}

// ─────────────────────────────────────────────────────────────
// Aceptar carga — el chofer la toma
// ─────────────────────────────────────────────────────────────
// AcceptLoadService asigna una carga a un chofer y actualiza su estado dentro
// de la transición operativa permitida.
func (s *CargaService) AcceptLoadService(id string, choferID uint) error {
	result, err := facades.Orm().Query().
		Model(&models.Carga{}).
		Where("id = ?", id).
		Where("estado = ?", "publicada").
		Where("chofer_id IS NULL").
		Update(map[string]interface{}{
			"chofer_id": choferID,
			"estado":    "asignada",
		})
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errors.New("carga no está publicada o ya fue asignada")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────
// Asignar chofer — el publicador lo hace explícitamente
// ─────────────────────────────────────────────────────────────
func (s *CargaService) AssignChofer(cargaID string, choferID uint) error {
	_, err := facades.Orm().Query().
		Model(&models.Carga{}).
		Where("id = ?", cargaID).
		Update(map[string]interface{}{"chofer_id": choferID})
	return err
}
