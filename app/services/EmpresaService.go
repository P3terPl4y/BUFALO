package services

import (
	"goravel/app/facades"
	"goravel/app/models"
)

type EmpresaService struct{}

func NewEmpresaService() *EmpresaService {
	return &EmpresaService{}
}

func (s *EmpresaService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.Empresa, int64, error) {
	page, perPage = NormalizePagination(page, perPage)
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
		Where("id = ?", id).
		First(&e)
	if lookupErr := recordError(err, e.ID, "empresa"); lookupErr != nil {
		return nil, lookupErr
	}
	return &e, nil
}

func (s *EmpresaService) Create(e *models.Empresa) error {
	return facades.Orm().Query().Create(e)
}

func (s *EmpresaService) Update(id string, updates map[string]interface{}) error {
	if err := ValidateCompanyUpdate(updates); err != nil {
		return err
	}
	_, err := facades.Orm().Query().Model(&models.Empresa{}).Where("id = ?", id).Update(updates)
	return err
}

func (s *EmpresaService) Delete(id string) error {
	_, err := facades.Orm().Query().Where("id = ?", id).Delete(&models.Empresa{})
	return err
}

// GetChoferes lista los choferes de la empresa con su User preload.
func (s *EmpresaService) GetChoferes(empresaID string, pages ...int) ([]models.Chofer, error) {
	page := 1
	if len(pages) > 0 {
		page, _ = NormalizePagination(pages[0], 100)
	}
	var list []models.Chofer
	err := facades.Orm().Query().
		With("User").
		Where("empresa_id = ?", empresaID).
		Order("id asc").Limit(100).Offset((page - 1) * 100).
		Find(&list)
	return list, err
}
