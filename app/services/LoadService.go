package services

import (
	"errors"
	"github.com/goravel/framework/contracts/database/orm"
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
	page, perPage = NormalizePagination(page, perPage)
	query := facades.Orm().Query().
		Model(&models.Carga{}).
		With("Publicador.User").
		With("Empresa").
		With("Chofer.User").
		With("OrigenDireccion").
		With("DestinoDireccion")

	if filters["active_work"] == "true" {
		query = query.Where("estado IN ?", []string{"asignada", "en_transito"})
	}
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
	if lookupErr := recordError(err, c.ID, "carga"); lookupErr != nil {
		return nil, lookupErr
	}
	return &c, nil
}

func (s *CargaService) Create(c *models.Carga) error {
	return facades.Orm().Query().Create(c)
}

func (s *CargaService) Delete(id string) error {
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var load models.Carga
	err = tx.Where("id = ?", id).LockForUpdate().First(&load)
	if err := recordError(err, load.ID, "carga"); err != nil {
		return err
	}
	if load.Estado != models.CargaPublicada || load.ChoferID != nil {
		return ErrConflict
	}
	exists, err := tx.Model(&models.Factura{}).Where("carga_id = ?", load.ID).Exists()
	if err != nil {
		return err
	}
	if exists {
		return ErrConflict
	}
	if _, err := tx.Where("id = ?", load.ID).Delete(&models.Carga{}); err != nil {
		return err
	}
	return tx.Commit()
}

// ─────────────────────────────────────────────────────────────
// Aceptar carga — el chofer la toma
// ─────────────────────────────────────────────────────────────
func (s *CargaService) AcceptLoadService(id string, choferID uint, actorUserID ...uint) error {
	loadID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return errors.New("carga inválida")
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var load models.Carga
	err = tx.Where("id = ?", loadID).LockForUpdate().First(&load)
	if err := recordError(err, load.ID, "carga"); err != nil {
		return err
	}
	if load.Estado != models.CargaPublicada || load.ChoferID != nil {
		return errors.New("carga no está publicada o ya fue asignada")
	}
	if load.Audiencia == models.AudienciaRedPrivada {
		member, err := tx.Model(&models.RedChofer{}).Where("empresa_id = ? AND chofer_id = ?", load.EmpresaID, choferID).Exists()
		if err != nil {
			return err
		}
		if !member {
			return errors.New("chofer fuera de la red privada")
		}
	}
	if _, err := tx.Model(&models.Carga{}).Where("id = ?", load.ID).Update(map[string]interface{}{"chofer_id": choferID, "estado": models.CargaAsignada}); err != nil {
		return err
	}
	if err := recordLoadActivity(tx, load.ID, models.CargaAsignada, "Carga aceptada por el chofer", firstActor(actorUserID)); err != nil {
		return err
	}
	if err := notifyLoadAssigned(tx, load, choferID, firstActor(actorUserID)); err != nil {
		return err
	}
	return tx.Commit()
}

// ─────────────────────────────────────────────────────────────
// Asignar chofer — el publicador lo hace explícitamente
// ─────────────────────────────────────────────────────────────
func (s *CargaService) AssignChofer(cargaID string, choferID uint, actorUserID ...uint) error {
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
	loadID, err := strconv.ParseUint(cargaID, 10, 32)
	if err != nil {
		return errors.New("carga inválida")
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var load models.Carga
	err = tx.Where("id = ?", loadID).LockForUpdate().First(&load)
	if err := recordError(err, load.ID, "carga"); err != nil {
		return err
	}
	if load.Estado != models.CargaPublicada || load.ChoferID != nil {
		return errors.New("carga no está publicada o ya fue asignada")
	}
	if _, err := tx.Model(&models.Carga{}).Where("id = ?", load.ID).Update(map[string]interface{}{"chofer_id": choferID, "estado": models.CargaAsignada}); err != nil {
		return err
	}
	if err := recordLoadActivity(tx, load.ID, models.CargaAsignada, "Chofer asignado por el publicador", firstActor(actorUserID)); err != nil {
		return err
	}
	if err := notifyLoadAssigned(tx, load, choferID, firstActor(actorUserID)); err != nil {
		return err
	}
	return tx.Commit()
}

func notifyLoadAssigned(tx orm.Query, load models.Carga, choferID, actorUserID uint) error {
	var driver models.Chofer
	if err := tx.Where("id = ?", choferID).First(&driver); err != nil {
		return err
	}
	if driver.ID == 0 {
		return ErrNotFound
	}
	var publisher models.Publicador
	if err := tx.Where("id = ?", load.PublicadorID).First(&publisher); err != nil {
		return err
	}
	if actorUserID == 0 {
		actorUserID = publisher.UserID
	}
	if err := CreateUserNotification(tx, driver.UserID, "load_assigned", "Carga asignada", "Se te asignó la carga "+load.NumeroReferencia+".", "load", load.ID, true, actorUserID); err != nil {
		return err
	}
	return CreateUserNotification(tx, publisher.UserID, "load_assigned", "Chofer asignado", "Un chofer aceptó o recibió la carga "+load.NumeroReferencia+".", "load", load.ID, false, driver.UserID)
}

// StartTransit moves a load to transit only when it is still assigned to this driver.
func (s *CargaService) StartTransit(cargaID string, choferID uint, actorUserID ...uint) error {
	if choferID == 0 {
		return errors.New("chofer inválido")
	}
	return s.transitionForDriver(cargaID, choferID, models.CargaAsignada, models.CargaEnTransito, "Chofer confirmó el inicio del tránsito", nil, firstActor(actorUserID))
}

// MarkDelivered completes a load only when the assigned driver has started transit.
func (s *CargaService) MarkDelivered(cargaID string, choferID uint, actorUserID ...uint) error {
	if choferID == 0 {
		return errors.New("chofer inválido")
	}
	now := time.Now().UTC()
	return s.transitionForDriver(cargaID, choferID, models.CargaEnTransito, models.CargaEntregada, "Chofer confirmó la entrega", &now, firstActor(actorUserID))
}

func firstActor(ids []uint) uint {
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

func (s *CargaService) transitionForDriver(cargaID string, choferID uint, from, to models.EstadoCarga, message string, deliveredAt *time.Time, actorUserID uint) error {
	loadID, err := strconv.ParseUint(cargaID, 10, 32)
	if err != nil {
		return errors.New("carga inválida")
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var load models.Carga
	err = tx.Where("id = ?", loadID).LockForUpdate().First(&load)
	if err := recordError(err, load.ID, "carga"); err != nil {
		return err
	}
	if load.ChoferID == nil || *load.ChoferID != choferID || load.Estado != from {
		return errors.New("carga no asignada a este chofer o transición inválida")
	}
	updates := map[string]interface{}{"estado": to}
	if deliveredAt != nil {
		updates["fecha_entrega"] = *deliveredAt
	}
	if _, err := tx.Model(&models.Carga{}).Where("id = ?", load.ID).Update(updates); err != nil {
		return err
	}
	if err := recordLoadActivity(tx, load.ID, to, message, actorUserID); err != nil {
		return err
	}
	var publisher models.Publicador
	if err := tx.Where("id = ?", load.PublicadorID).First(&publisher); err != nil {
		return err
	}
	if actorUserID == 0 {
		var driver models.Chofer
		if err := tx.Where("id = ?", choferID).First(&driver); err != nil {
			return err
		}
		actorUserID = driver.UserID
	}
	if err := CreateUserNotification(tx, publisher.UserID, "load_status", "Cambio de estado de carga", "La carga "+load.NumeroReferencia+" cambió a estado: "+string(to)+".", "load", load.ID, false, actorUserID); err != nil {
		return err
	}
	return tx.Commit()
}
