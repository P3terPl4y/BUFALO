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

// GetPage devuelve una página 1-based y el total de direcciones para construir
// los controles de navegación. El tamaño por defecto es 10.
func (s *DireccionService) GetPage(page, perPage int) ([]models.Direccion, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	q := facades.Orm().Query().Model(&models.Direccion{})
	total, err := q.Count()
	if err != nil {
		return nil, 0, err
	}
	var list []models.Direccion
	err = q.Order("estado_provincia asc, ciudad asc").Limit(perPage).Offset((page - 1) * perPage).Find(&list)
	return list, total, err
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
