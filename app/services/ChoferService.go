package services

import (
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
)

type ChoferService struct{}

func NewChoferService() *ChoferService { return &ChoferService{} }

// GetAllWithFilters lista choferes con User y Empresa preload.
func (s *ChoferService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.Chofer, int64, error) {
	query := facades.Orm().Query().
		Model(&models.Chofer{}).
		With("User").
		With("Empresa")

	if empresaID := filters["empresa_id"]; empresaID != "" {
		query = query.Where("empresa_id = ?", empresaID)
	}
	if estado := filters["estado"]; estado != "" {
		query = query.Where("estado = ?", estado)
	}
	if q := filters["q"]; q != "" {
		// Búsqueda por nombre del User asociado
		query = query.Where(
			"user_id IN (SELECT id FROM users WHERE name LIKE ? OR email LIKE ?)",
			"%"+q+"%", "%"+q+"%",
		)
	}

	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	var list []models.Chofer
	err = query.Limit(perPage).Offset(offset).Order("id asc").Find(&list)
	return list, total, err
}

func (s *ChoferService) GetByID(id uint) (*models.Chofer, error) {
	var c models.Chofer
	err := facades.Orm().Query().
		With("User").
		With("Empresa").
		Where("id = ?", id).
		First(&c)
	if err != nil || c.ID == 0 {
		return nil, errors.New("chofer not found")
	}
	return &c, nil
}

func (s *ChoferService) GetByUserID(userID uint) (*models.Chofer, error) {
	var c models.Chofer
	err := facades.Orm().Query().
		With("User").
		With("Empresa").
		Where("user_id = ?", userID).
		First(&c)
	if err != nil || c.ID == 0 {
		return nil, errors.New("chofer not found")
	}
	return &c, nil
}

func (s *ChoferService) Create(c *models.Chofer) error {
	return facades.Orm().Query().Create(c)
}

// Update solo modifica campos de la tabla choferes.
func (s *ChoferService) Update(id uint, updates map[string]interface{}) error {
	_, err := facades.Orm().Query().
		Model(&models.Chofer{}).
		Where("id = ?", id).
		Update(updates)
	return err
}

func (s *ChoferService) Delete(id uint) error {
	_, err := facades.Orm().Query().
		Where("id = ?", id).
		Delete(&models.Chofer{})
	return err
}

// ToggleEstado cambia el estado operativo del chofer.
func (s *ChoferService) ToggleEstado(id uint, estado models.EstadoChofer) error {
	_, err := facades.Orm().Query().
		Model(&models.Chofer{}).
		Where("id = ?", id).
		Update(map[string]interface{}{"estado": estado})
	return err
}
