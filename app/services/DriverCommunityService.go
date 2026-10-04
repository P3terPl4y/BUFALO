package services

import (
	"errors"
	"fmt"

	"goravel/app/community"
	"goravel/app/facades"
	"goravel/app/models"
)

type DriverCommunityService struct{}

func NewDriverCommunityService() *DriverCommunityService { return &DriverCommunityService{} }

func ValidateDriverRating(score int, comment string) error {
	return community.ValidateDriverRating(score, comment)
}

// RateCompletedLoad enforces one rating per delivered load and recalculates the
// driver's aggregate inside one database transaction.
func (s *DriverCommunityService) RateCompletedLoad(loadID, publisherID uint, score int, comment string) error {
	if err := ValidateDriverRating(score, comment); err != nil {
		return err
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var load models.Carga
	if err := tx.Where("id = ?", loadID).First(&load); err != nil || load.ID == 0 {
		return errors.New("carga no encontrada")
	}
	if load.PublicadorID != publisherID {
		return errors.New("solo el publicador de la carga puede calificarla")
	}
	if load.Estado != models.CargaEntregada || load.ChoferID == nil {
		return errors.New("solo se puede calificar una carga entregada")
	}

	exists, err := tx.Model(&models.ChoferCalificacion{}).Where("carga_id = ?", loadID).Exists()
	if err != nil {
		return err
	}
	if exists {
		return errors.New("esta carga ya fue calificada")
	}
	rating := models.ChoferCalificacion{CargaID: load.ID, ChoferID: *load.ChoferID, PublicadorID: publisherID, Puntaje: score, Comentario: comment}
	if err := tx.Create(&rating); err != nil {
		return fmt.Errorf("no se pudo guardar la calificación: %w", err)
	}

	var average float64
	query := tx.Model(&models.ChoferCalificacion{}).Where("chofer_id = ?", *load.ChoferID)
	if err := query.Avg("puntaje", &average); err != nil {
		return err
	}
	count, err := tx.Model(&models.ChoferCalificacion{}).Where("chofer_id = ?", *load.ChoferID).Count()
	if err != nil {
		return err
	}
	if _, err := tx.Model(&models.Chofer{}).Where("id = ?", *load.ChoferID).Update(map[string]interface{}{"rating_average": average, "rating_count": count}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *DriverCommunityService) HasLoadRating(loadID uint) bool {
	exists, err := facades.Orm().Query().Model(&models.ChoferCalificacion{}).Where("carga_id = ?", loadID).Exists()
	return err == nil && exists
}

func (s *DriverCommunityService) ListCompanyNetwork(companyID uint) ([]models.RedChofer, error) {
	var members []models.RedChofer
	err := facades.Orm().Query().Model(&models.RedChofer{}).With("Chofer.User").Where("empresa_id = ?", companyID).OrderBy("id", "desc").Find(&members)
	return members, err
}

func (s *DriverCommunityService) AddToCompanyNetwork(companyID, driverID uint) error {
	if companyID == 0 || driverID == 0 {
		return errors.New("empresa o chofer inválido")
	}
	var driver models.Chofer
	if err := facades.Orm().Query().Where("id = ?", driverID).First(&driver); err != nil || driver.ID == 0 {
		return errors.New("chofer no encontrado")
	}
	var user models.User
	if err := facades.Orm().Query().Where("id = ?", driver.UserID).First(&user); err != nil || user.ID == 0 || !user.IsActive {
		return errors.New("el chofer no tiene una cuenta activa")
	}
	return facades.Orm().Query().Create(&models.RedChofer{EmpresaID: companyID, ChoferID: driverID})
}

func (s *DriverCommunityService) RemoveFromCompanyNetwork(companyID, driverID uint) error {
	result, err := facades.Orm().Query().Where("empresa_id = ? AND chofer_id = ?", companyID, driverID).Delete(&models.RedChofer{})
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errors.New("el chofer no pertenece a esta red")
	}
	return nil
}

func (s *DriverCommunityService) IsCompanyMember(companyID, driverID uint) bool {
	exists, err := facades.Orm().Query().Model(&models.RedChofer{}).Where("empresa_id = ? AND chofer_id = ?", companyID, driverID).Exists()
	return err == nil && exists
}
