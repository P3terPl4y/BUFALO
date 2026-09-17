package services

import (
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

// ─────────────────────────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────────────────────────

// Create inserta un usuario simple (sin empresa asociada).
func (s *UserService) Create(user *models.User) error {
	return facades.Orm().Query().Create(user)
}
// CreateWithRole crea User + (Empresa opcional) + (Chofer o Publicador) en un
// solo flujo con rollback manual. Debe llamarse en lugar de CreateWithEmpresa
// cuando el usuario tenga rol "chofer" o "publicador".
func (s *UserService) CreateWithRole(
	user *models.User,
	empresa *models.Empresa,
	chofer *models.Chofer,
	publicador *models.Publicador,
) error {
	createdEmpresa := false
	createdChofer := false
	createdPublicador := false

	rollback := func() {
		if createdPublicador && publicador != nil {
			_, _ = facades.Orm().Query().Where("id = ?", publicador.ID).Delete(&models.Publicador{})
		}
		if createdChofer && chofer != nil {
			_, _ = facades.Orm().Query().Where("id = ?", chofer.ID).Delete(&models.Chofer{})
		}
		if createdEmpresa && empresa != nil {
			_, _ = facades.Orm().Query().Where("id = ?", empresa.ID).Delete(&models.Empresa{})
		}
		if user.ID > 0 {
			_, _ = facades.Orm().Query().Where("id = ?", user.ID).Delete(&models.User{})
		}
	}

	// 1. Empresa (opcional)
	if empresa != nil && empresa.ID == 0 {
		if err := facades.Orm().Query().Create(empresa); err != nil {
			return err
		}
		createdEmpresa = true
	}
	if empresa != nil {
		user.EmpresaID = &empresa.ID
	}

	// 2. User
	if err := facades.Orm().Query().Create(user); err != nil {
    	rollback()
    	return err
	}

	// 2.1 Si acabamos de crear la empresa, el creador es el owner
	if createdEmpresa && empresa != nil {
    	if _, err := facades.Orm().Query().
      	  Model(&models.Empresa{}).
          Where("id = ?", empresa.ID).
          Update("owner_id", user.ID); err != nil {
          rollback()
          return err
        }
    	empresa.OwnerID = &user.ID // reflejarlo en memoria también
	}
	// 3. Perfil de rol
	switch user.Role {
	case "chofer":
		if chofer == nil {
			rollback()
			return errors.New("chofer profile is required for role=chofer")
		}
		chofer.UserID = user.ID
		if empresa != nil {
			empID := empresa.ID
			chofer.EmpresaID = empID
		}
		if err := facades.Orm().Query().Create(chofer); err != nil {
			rollback()
			return err
		}
		createdChofer = true
		user.ChoferID = &chofer.ID

	case "publicador":
		if publicador == nil {
			rollback()
			return errors.New("publicador profile is required for role=publicador")
		}
		publicador.UserID = user.ID
		if empresa != nil {
			empID := empresa.ID
			publicador.EmpresaID = empID
		}
		if err := facades.Orm().Query().Create(publicador); err != nil {
			rollback()
			return err
		}
		createdPublicador = true
		user.PublicadorID = &publicador.ID
	}

	// 4. Guardar los FK de rol en el User
	updates := map[string]interface{}{}
	if user.ChoferID != nil {
		updates["chofer_id"] = *user.ChoferID
	}
	if user.PublicadorID != nil {
		updates["publicador_id"] = *user.PublicadorID
	}
	if len(updates) > 0 {
		if _, err := facades.Orm().Query().
			Model(&models.User{}).
			Where("id = ?", user.ID).
			Update(updates); err != nil {
			rollback()
			return err
		}
	}

	return nil
}
// CreateWithEmpresa crea User + Empresa en una transacción.
// Si algo falla, se revierte todo (empresa y usuario).
// Si empresa == nil o empresa.ID != 0, solo se vincula.
func (s *UserService) CreateWithEmpresa(user *models.User, empresa *models.Empresa) error {
	// Rollback manual (compatible con cualquier versión de Goravel)
	createdEmpresa := false

	if empresa != nil && empresa.ID == 0 {
		if err := facades.Orm().Query().Create(empresa); err != nil {
			return err
		}
		createdEmpresa = true
	}

	if empresa != nil {
		user.EmpresaID = &empresa.ID
	}

	if err := facades.Orm().Query().Create(user); err != nil {
		// Rollback: borrar la empresa que acabamos de crear
		if createdEmpresa {
			_, _ = facades.Orm().Query().
				Where("id = ?", empresa.ID).
				Delete(&models.Empresa{})
		}
		return err
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────
// Read
// ─────────────────────────────────────────────────────────────────

func (s *UserService) GetByID(id uint) (*models.User, error) {
	var u models.User
	err := facades.Orm().Query().
		With("Empresa").
		Where("id = ?", id).
		First(&u)
	if err != nil || u.ID == 0 {
		return nil, errors.New("user not found")
	}
	return &u, nil
}

// GetByEmail devuelve un usuario por email (útil para login).
func (s *UserService) GetByEmail(email string) (*models.User, error) {
	var u models.User
	err := facades.Orm().Query().
		Where("email = ?", email).
		First(&u)
	if err != nil || u.ID == 0 {
		return nil, errors.New("user not found")
	}
	return &u, nil
}

// GetAllWithFilters lista usuarios con filtros y paginación (panel admin).
func (s *UserService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.User, int64, error) {
	query := facades.Orm().Query().
		Model(&models.User{}).
		With("Empresa")

	if role := filters["role"]; role != "" {
		query = query.Where("role = ?", role)
	}
	if search := filters["search"]; search != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status := filters["status"]; status != "" {
		switch status {
		case "active":
			query = query.Where("is_active = ?", true)
		case "inactive":
			query = query.Where("is_active = ?", false)
		}
	}

	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	var list []models.User
	err = query.
		Limit(perPage).
		Offset(offset).
		Order("created_at desc").
		Find(&list)
	return list, total, err
}

// GetEmpresasByTipo lista empresas activas del tipo dado (carrier / broker).
func (s *UserService) GetEmpresasByTipo(tipo string) ([]models.Empresa, error) {
	var list []models.Empresa
	err := facades.Orm().Query().
		Model(&models.Empresa{}).
		Where("tipo = ?", tipo).
		Where("estado = ?", "activo").
		Order("nombre_legal asc").
		Find(&list)
	return list, err
}

// ─────────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────────

// Update actualiza cualquier campo del User por ID (incluido EmpresaID).
func (s *UserService) Update(id uint, updates map[string]interface{}) error {
	_, err := facades.Orm().Query().
		Model(&models.User{}).
		Where("id = ?", id).
		Update(updates)
	return err
}

// UpdateProfile actualiza el perfil del usuario autenticado.
// Alias semántico de Update.
func (s *UserService) UpdateProfile(id uint, updates map[string]interface{}) error {
	return s.Update(id, updates)
}

// ToggleActive activa o desactiva un usuario.
func (s *UserService) ToggleActive(id uint, active bool) error {
	_, err := facades.Orm().Query().
		Model(&models.User{}).
		Where("id = ?", id).
		Update(map[string]interface{}{"is_active": active})
	return err
}

// ─────────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────────

func (s *UserService) Delete(id uint) error {
	_, err := facades.Orm().Query().
		Where("id = ?", id).
		Delete(&models.User{})
	return err
}
// EmailTaken verifica si un email ya está registrado en la tabla users.
// Si excludeID > 0, excluye ese usuario de la verificación (útil en updates).
func (s *UserService) EmailTaken(email string, excludeID uint) (bool, error) {
	query := facades.Orm().Query().
		Model(&models.User{}).
		Where("email = ?", email)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	count, err := query.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
// EmailExists verifica si un email ya está registrado en la tabla users.
// Si excludeID > 0, excluye ese usuario de la verificación (útil en updates).
func (s *UserService) EmailExists(email string, excludeID uint) (bool, error) {
	query := facades.Orm().Query().
		Model(&models.User{}).
		Where("email = ?", email)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	count, err := query.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
// Al guardar User, exactamente uno de los dos FK debe estar seteado
func ValidateUserRole(user *models.User) error {
    hasPublicador := user.PublicadorID != nil
    hasChofer := user.ChoferID != nil
    if user.Role == "admin" {
        if hasPublicador || hasChofer {
            return errors.New("admin no debe tener perfil de rol")
        }
        return nil
    }
    if user.Role == "publicador" && (!hasPublicador || hasChofer) {
        return errors.New("publicador requiere PublicadorID y no ChoferID")
    }
    if user.Role == "chofer" && (!hasChofer || hasPublicador) {
        return errors.New("chofer requiere ChoferID y no PublicadorID")
    }
    return nil
}
