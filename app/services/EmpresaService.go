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
		With("Direccion").With("Owner")

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

// GetCompaniesForUser returns companies the account owns or is associated with.
// User and role-profile associations are both considered to support legacy rows.
func (s *EmpresaService) GetCompaniesForUser(userID uint, page, perPage int) ([]models.Empresa, int64, error) {
	page, perPage = NormalizePagination(page, perPage)
	query := facades.Orm().Query().Model(&models.Empresa{}).With("Direccion").With("Owner").
		Where(`owner_id = ? OR id IN (SELECT empresa_id FROM users WHERE id = ? AND empresa_id IS NOT NULL) OR id IN (SELECT empresa_id FROM publicadors WHERE user_id = ? AND empresa_id > 0) OR id IN (SELECT empresa_id FROM chofers WHERE user_id = ? AND empresa_id > 0)`, userID, userID, userID, userID)
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	var list []models.Empresa
	err = query.Order("created_at desc").Limit(perPage).Offset((page - 1) * perPage).Find(&list)
	return list, total, err
}

// GetAvailableCompanies returns active directory entries except companies the
// user already owns or belongs to. Filters are intentionally only applied when
// the user explicitly opens the directory.
func (s *EmpresaService) GetAvailableCompanies(userID uint, filters map[string]string, page, perPage int) ([]models.Empresa, int64, error) {
	page, perPage = NormalizePagination(page, perPage)
	query := facades.Orm().Query().Model(&models.Empresa{}).With("Direccion").With("Owner").
		Where("estado = ? AND deleted_at IS NULL", models.EmpresaActiva).
		Where(`owner_id IS DISTINCT FROM ? AND id NOT IN (SELECT empresa_id FROM users WHERE id = ? AND empresa_id IS NOT NULL) AND id NOT IN (SELECT empresa_id FROM publicadors WHERE user_id = ? AND empresa_id > 0) AND id NOT IN (SELECT empresa_id FROM chofers WHERE user_id = ? AND empresa_id > 0)`, userID, userID, userID, userID)
	if role := filters["role"]; role == "chofer" {
		query = query.Where("tipo = ?", models.TipoEmpresa("carrier"))
	} else if role == "publicador" {
		query = query.Where("tipo = ?", models.TipoEmpresa("broker"))
	}
	if tipo := filters["tipo"]; tipo != "" {
		query = query.Where("tipo = ?", tipo)
	}
	if q := filters["q"]; q != "" {
		like := "%" + q + "%"
		query = query.Where("nombre_legal LIKE ? OR nombre_comercial LIKE ? OR mc_number LIKE ? OR dot_number LIKE ?", like, like, like, like)
	}
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	var list []models.Empresa
	err = query.Order("nombre_legal asc").Limit(perPage).Offset((page - 1) * perPage).Find(&list)
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

// GetCompanyMembers returns a bounded page of user accounts linked to a company.
func (s *EmpresaService) GetCompanyMembers(empresaID string, page int) ([]models.User, error) {
	page, perPage := NormalizePagination(page, 100)
	var list []models.User
	err := facades.Orm().Query().Model(&models.User{}).
		With("Chofer").With("Publicador").
		Where("empresa_id = ?", empresaID).
		Order("name asc").Limit(perPage).Offset((page - 1) * perPage).Find(&list)
	return list, err
}
