package services

import (
	"errors"
	"fmt"
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
	// Se deduplica por los datos visibles. Esto protege la grilla frente a
	// registros demo antiguos que repetían la misma dirección con otro ID.
	var all []models.Direccion
	err := facades.Orm().Query().Model(&models.Direccion{}).
		Order("estado_provincia asc, ciudad asc, id asc").Find(&all)
	if err != nil {
		return nil, 0, err
	}
	unique := make([]models.Direccion, 0, len(all))
	seen := make(map[string]struct{}, len(all))
	for _, d := range all {
		key := fmt.Sprintf("%s|%s|%s|%s|%v|%v", valueString(d.Calle), d.Ciudad, d.EstadoProvincia, d.Pais, d.Latitud, d.Longitud)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, d)
	}
	total := int64(len(unique))
	start := (page - 1) * perPage
	if start >= len(unique) {
		return []models.Direccion{}, total, nil
	}
	end := start + perPage
	if end > len(unique) {
		end = len(unique)
	}
	return unique[start:end], total, nil
}

func valueString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
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
