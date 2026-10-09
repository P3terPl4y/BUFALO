package services

import (
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
	"strconv"
)

type DireccionService struct{}

func NewDireccionService() *DireccionService {
	return &DireccionService{}
}

func (s *DireccionService) GetAll(pagination ...int) ([]models.Direccion, error) {
	page, size := 1, 100
	if len(pagination) == 2 {
		page, size = NormalizePagination(pagination[0], pagination[1])
	}
	var list []models.Direccion
	err := facades.Orm().Query().
		Model(&models.Direccion{}).
		Order("estado_provincia asc, ciudad asc, id asc").Limit(size).Offset((page - 1) * size).
		Find(&list)
	return list, err
}

func (s *DireccionService) GetOwnedByUserID(userID uint, pagination ...int) ([]models.Direccion, error) {
	page, size := 1, 100
	if len(pagination) == 2 {
		page, size = NormalizePagination(pagination[0], pagination[1])
	}
	var list []models.Direccion
	if userID == 0 {
		return list, errors.New("usuario requerido para listar direcciones")
	}
	err := facades.Orm().Query().
		Model(&models.Direccion{}).
		Where("owner_id = ?", userID).
		Order("estado_provincia asc, ciudad asc, id asc").Limit(size).Offset((page - 1) * size).
		Find(&list)
	return list, err
}

func (s *DireccionService) GetByID(id string) (*models.Direccion, error) {
	var d models.Direccion
	err := facades.Orm().Query().With("Owner").Where("id = ?", id).First(&d)
	if lookupErr := recordError(err, d.ID, "direccion"); lookupErr != nil {
		return nil, lookupErr
	}
	return &d, nil
}

func (s *DireccionService) Create(d *models.Direccion) error {
	return facades.Orm().Query().Create(d)
}

func (s *DireccionService) Update(id string, updates map[string]interface{}) error {
	if err := ValidateAddressUpdate(updates); err != nil {
		return err
	}
	_, err := facades.Orm().Query().Model(&models.Direccion{}).Where("id = ?", id).Update(updates)
	return err
}

func (s *DireccionService) Delete(id string) error {
	_, err := facades.Orm().Query().Where("id = ?", id).Delete(&models.Direccion{})
	return err
}

// Helper: parsea uint desde string
func parseUint(s string) (uint, bool) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint(v), err == nil
}
