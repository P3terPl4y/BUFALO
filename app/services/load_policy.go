package services

import (
	"errors"
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"sort"
	"strconv"
)

func LoadVisibilityFilters(userID uint, role string) (map[string]string, error) {
	filters := map[string]string{}
	switch role {
	case "admin":
	case "publicador":
		profile, err := NewPublicadorService().GetByUserID(userID)
		if err != nil {
			return nil, err
		}
		filters["publicador_id"] = strconv.FormatUint(uint64(profile.ID), 10)
	case "chofer":
		profile, err := NewChoferService().GetByUserID(userID)
		if err != nil {
			return nil, err
		}
		filters["driver_board_id"] = strconv.FormatUint(uint64(profile.ID), 10)
	default:
		return nil, errors.New("unsupported role")
	}
	return filters, nil
}

func validateLoadAddresses(tx orm.Query, load *models.Carga, userID uint, role string) error {
	ids := []uint{load.OrigenDireccionID, load.DestinoDireccionID}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		var address models.Direccion
		err := tx.Where("id = ?", id).LockForUpdate().First(&address)
		if err := recordError(err, address.ID, "direccion"); err != nil {
			return err
		}
		if role != "admin" && (address.OwnerID == nil || *address.OwnerID != userID) {
			return errors.New("la dirección no te pertenece")
		}
	}
	return nil
}

func (s *CargaService) CreateForActor(load *models.Carga, userID uint, role string) error {
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if userID == 0 || (role != "admin" && role != "publicador") {
		return ErrNotFound
	}
	var publisher models.Publicador
	query := tx.Where("id = ?", load.PublicadorID)
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}
	err = query.LockForUpdate().First(&publisher)
	if err := recordError(err, publisher.ID, "publicador"); err != nil {
		return err
	}
	var company models.Empresa
	err = tx.Where("id = ? AND tipo = ? AND estado = ?", publisher.EmpresaID, models.TipoBroker, models.EmpresaActiva).First(&company)
	if err := recordError(err, company.ID, "empresa"); err != nil {
		return err
	}
	load.EmpresaID = publisher.EmpresaID
	if err := validateLoadAddresses(tx, load, userID, role); err != nil {
		return err
	}
	if err := tx.Create(load); err != nil {
		return err
	}
	return tx.Commit()
}

// Serialize edits with assignment and reject changes to completed commercial work.
func (s *CargaService) UpdateForActor(id string, updates map[string]interface{}, userID uint, role string) error {
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
	if role != "admin" {
		var publisher models.Publicador
		err := tx.Where("id = ? AND user_id = ?", load.PublicadorID, userID).First(&publisher)
		if err := recordError(err, publisher.ID, "publicador"); err != nil {
			return err
		}
	}
	if load.Estado != models.CargaPublicada || load.ChoferID != nil {
		return ErrConflict
	}
	invoiced, err := tx.Model(&models.Factura{}).Where("carga_id = ?", load.ID).Exists()
	if err != nil {
		return err
	}
	if invoiced {
		return ErrConflict
	}
	origin, ok := updates["origen_direccion_id"].(uint)
	if !ok {
		return errors.New("origen inválido")
	}
	destination, ok := updates["destino_direccion_id"].(uint)
	if !ok {
		return errors.New("destino inválido")
	}
	load.OrigenDireccionID, load.DestinoDireccionID = origin, destination
	if err := validateLoadAddresses(tx, &load, userID, role); err != nil {
		return err
	}
	for key := range updates {
		switch key {
		case "numero_referencia", "origen_direccion_id", "destino_direccion_id", "fecha_recogida", "fecha_entrega", "tipo_carga", "tipo_equipo", "peso_kg", "commodity", "distancia_km", "distancia_real_km", "tarifa_total", "tarifa_por_km", "moneda", "audiencia":
		default:
			return errors.New("campo de carga no editable")
		}
	}
	if _, err := tx.Model(&models.Carga{}).Where("id = ?", load.ID).Update(updates); err != nil {
		return err
	}
	return tx.Commit()
}
