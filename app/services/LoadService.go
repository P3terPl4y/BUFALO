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
func (s *CargaService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.Carga, int64, error) {
	query := facades.Orm().Query().
		Model(&models.Carga{}).
		With("Publicador.User").
		With("Empresa").
		With("Chofer.User").
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
	if driverBoardID := filters["driver_board_id"]; driverBoardID != "" {
		if _, err := strconv.ParseUint(driverBoardID, 10, 32); err == nil {
			query = query.Where("(chofer_id = ? OR (chofer_id IS NULL AND estado = ? AND (audiencia = ? OR (audiencia = ? AND empresa_id IN (SELECT empresa_id FROM red_choferes WHERE chofer_id = ?)))))", driverBoardID, models.CargaPublicada, models.AudienciaLoadBoard, models.AudienciaRedPrivada, driverBoardID)
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
		With("Publicador.User").
		With("Empresa").
		With("Chofer.User").
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
func (s *CargaService) AcceptLoadService(id string, choferID uint) error {
	result, err := facades.Orm().Query().
		Model(&models.Carga{}).
		Where("id = ?", id).
		Where("estado = ?", "publicada").
		Where("chofer_id IS NULL").
		Where("(audiencia = ? OR (audiencia = ? AND EXISTS (SELECT 1 FROM red_choferes WHERE red_choferes.empresa_id = cargas.empresa_id AND red_choferes.chofer_id = ?)))", models.AudienciaLoadBoard, models.AudienciaRedPrivada, choferID).
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
	if choferID == 0 {
		return errors.New("chofer inválido")
	}
	var chofer models.Chofer
	if err := facades.Orm().Query().Where("id = ?", choferID).First(&chofer); err != nil || chofer.ID == 0 {
		return errors.New("chofer no encontrado")
	}
	if chofer.Estado != models.ChoferDisponible {
		return errors.New("chofer no disponible")
	}
	result, err := facades.Orm().Query().
		Model(&models.Carga{}).
		Where("id = ?", cargaID).
		Where("estado = ?", models.CargaPublicada).
		Where("chofer_id IS NULL").
		Update(map[string]interface{}{"chofer_id": choferID, "estado": models.CargaAsignada})
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errors.New("carga no está publicada o ya fue asignada")
	}
	return nil
}

// StartTransit moves a load to transit only when it is still assigned to this driver.
func (s *CargaService) StartTransit(cargaID string, choferID uint) error {
	if choferID == 0 {
		return errors.New("chofer inválido")
	}
	result, err := facades.Orm().Query().Model(&models.Carga{}).
		Where("id = ?", cargaID).
		Where("chofer_id = ?", choferID).
		Where("estado = ?", models.CargaAsignada).
		Update("estado", models.CargaEnTransito)
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return errors.New("carga no asignada a este chofer o transición inválida")
	}
	return nil
}

// MarkDelivered completes a load only when the assigned driver has started transit.
func (s *CargaService) MarkDelivered(cargaID string, choferID uint) error {
	if choferID == 0 {
		return errors.New("chofer inválido")
	}
	result, err := facades.Orm().Query().Model(&models.Carga{}).
		Where("id = ?", cargaID).
		Where("chofer_id = ?", choferID).
		Where("estado = ?", models.CargaEnTransito).
		Update(map[string]interface{}{"estado": models.CargaEntregada, "fecha_entrega": time.Now().UTC()})
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return errors.New("carga no asignada a este chofer o transición inválida")
	}
	return nil
}
