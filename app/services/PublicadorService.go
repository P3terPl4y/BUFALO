package services

import (
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
)

type PublicadorService struct{}

func NewPublicadorService() *PublicadorService { return &PublicadorService{} }

func (s *PublicadorService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.Publicador, int64, error) {
	query := facades.Orm().Query().
		Model(&models.Publicador{}).
		With("User").
		With("Empresa")

	if empresaID := filters["empresa_id"]; empresaID != "" {
		query = query.Where("empresa_id = ?", empresaID)
	}
	if estado := filters["estado"]; estado != "" {
		query = query.Where("estado = ?", estado)
	}
	if q := filters["q"]; q != "" {
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
	var list []models.Publicador
	err = query.Limit(perPage).Offset(offset).Order("id asc").Find(&list)
	return list, total, err
}

func (s *PublicadorService) GetByID(id uint) (*models.Publicador, error) {
	var p models.Publicador
	err := facades.Orm().Query().
		With("User").
		With("Empresa").
		Where("id = ?", id).
		First(&p)
	if err != nil || p.ID == 0 {
		return nil, errors.New("publicador not found")
	}
	return &p, nil
}

func (s *PublicadorService) GetByUserID(userID uint) (*models.Publicador, error) {
	var p models.Publicador
	err := facades.Orm().Query().
		With("User").
		With("Empresa").
		Where("user_id = ?", userID).
		First(&p)
	if err != nil || p.ID == 0 {
		return nil, errors.New("publicador not found")
	}
	return &p, nil
}

func (s *PublicadorService) Create(p *models.Publicador) error {
	return facades.Orm().Query().Create(p)
}

func (s *PublicadorService) Update(id uint, updates map[string]interface{}) error {
	_, err := facades.Orm().Query().
		Model(&models.Publicador{}).
		Where("id = ?", id).
		Update(updates)
	return err
}

func (s *PublicadorService) Delete(id uint) error {
	_, err := facades.Orm().Query().
		Where("id = ?", id).
		Delete(&models.Publicador{})
	return err
}

func (s *PublicadorService) ToggleEstado(id uint, estado models.EstadoPublicador) error {
	_, err := facades.Orm().Query().
		Model(&models.Publicador{}).
		Where("id = ?", id).
		Update(map[string]interface{}{"estado": estado})
	return err
}
