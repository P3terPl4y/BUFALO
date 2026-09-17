package services

import (
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
)

type EmpresaService struct{}

func NewEmpresaService() *EmpresaService {
	return &EmpresaService{}
}

func (s *EmpresaService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.Empresa, int64, error) {
	query := facades.Orm().Query().
		Model(&models.Empresa{}).
		With("Direccion")

	if tipo := filters["tipo"]; tipo != "" {
		query = query.Where("tipo = ?", tipo)
	}
	if estado := filters["estado"]; estado != "" {
		query = query.Where("estado = ?", estado)
	}
	if q := filters["q"]; q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"nombre_legal LIKE ? OR nombre_comercial LIKE ? OR mc_number LIKE ? OR dot_number LIKE ?",
			like, like, like, like,
		)
	}

	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	var list []models.Empresa
	err = query.Limit(perPage).Offset(offset).Order("created_at desc").Find(&list)
	return list, total, err
}

func (s *EmpresaService) GetByID(id string) (*models.Empresa, error) {
	var e models.Empresa
	err := facades.Orm().Query().
		With("Direccion").
		With("Owner").
		With("Choferes").
		Where("id = ?", id).
		First(&e)
	if err != nil || e.ID == 0 {
		return nil, errors.New("empresa not found")
	}
	return &e, nil
}

func (s *EmpresaService) Create(e *models.Empresa) error {
	return facades.Orm().Query().Create(e)
}

func (s *EmpresaService) Update(id string, updates map[string]interface{}) error {
	_, err := facades.Orm().Query().Model(&models.Empresa{}).Where("id = ?", id).Update(updates)
	return err
}

func (s *EmpresaService) Delete(id string) error {
	_, err := facades.Orm().Query().Where("id = ?", id).Delete(&models.Empresa{})
	return err
}

// GetChoferes lista los choferes de la empresa con su User preload.
func (s *EmpresaService) GetChoferes(empresaID string) ([]models.Chofer, error) {
	var list []models.Chofer
	err := facades.Orm().Query().
		With("User").
		Where("empresa_id = ?", empresaID).
		Order("id asc").
		Find(&list)
	return list, err
}
