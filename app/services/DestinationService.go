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

func (s *DireccionService) GetAll() ([]models.Direccion, error) {
	var list []models.Direccion
	err := facades.Orm().Query().
		Model(&models.Direccion{}).
		Order("estado_provincia asc, ciudad asc").
		Find(&list)
	return list, err
}

func (s *DireccionService) GetByID(id string) (*models.Direccion, error) {
	var d models.Direccion
	err := facades.Orm().Query().With("Owner").Where("id = ?", id).First(&d)
	if err != nil || d.ID == 0 {
		return nil, errors.New("direccion not found")
	}
	return &d, nil
}

func (s *DireccionService) Create(d *models.Direccion) error {
	return facades.Orm().Query().Create(d)
}

func (s *DireccionService) Update(id string, updates map[string]interface{}) error {
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
