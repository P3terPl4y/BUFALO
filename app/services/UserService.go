package services

import (
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
	"strings"

	"github.com/goravel/framework/contracts/database/orm"
	frameworkerrors "github.com/goravel/framework/errors"
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
	if user != nil {
		user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	}
	return facades.Orm().Query().Create(user)
}

// CreateWithRole crea User + (Empresa opcional) + (Chofer o Publicador) en una
// transacción. Debe llamarse en lugar de CreateWithEmpresa
// cuando el usuario tenga rol "chofer" o "publicador".
func (s *UserService) CreateWithRole(
	user *models.User,
	empresa *models.Empresa,
	chofer *models.Chofer,
	publicador *models.Publicador,
) error {
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.CreateWithRoleInTransaction(tx, user, empresa, chofer, publicador); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateWithRoleInTransaction persists an account and its role profile inside
// an existing transaction, allowing email-token consumption to be atomic.
func (s *UserService) CreateWithRoleInTransaction(
	tx orm.Query,
	user *models.User,
	empresa *models.Empresa,
	chofer *models.Chofer,
	publicador *models.Publicador,
) error {
	if user == nil {
		return errors.New("user is required")
	}
	if tx == nil {
		return errors.New("transaction is required")
	}
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	switch user.Role {
	case "chofer":
		if chofer == nil || publicador != nil {
			return errors.New("exactly one chofer profile is required for role=chofer")
		}
	case "publicador":
		if publicador == nil || chofer != nil {
			return errors.New("exactly one publicador profile is required for role=publicador")
		}
	default:
		return errors.New("unsupported user role")
	}

	// 1. Empresa (opcional)
	createdEmpresa := false
	if empresa != nil && empresa.ID == 0 {
		if err := tx.Create(empresa); err != nil {
			return err
		}
		createdEmpresa = true
	}
	if empresa != nil {
		user.EmpresaID = &empresa.ID
	}

	// 2. User
	if err := tx.Create(user); err != nil {
		return err
	}

	// 2.1 Si acabamos de crear la empresa, el creador es el owner
	if createdEmpresa && empresa != nil {
		if _, err := tx.
			Model(&models.Empresa{}).
			Where("id = ?", empresa.ID).
			Update("owner_id", user.ID); err != nil {
			return err
		}
		empresa.OwnerID = &user.ID // reflejarlo en memoria también
	}
	// 3. Perfil de rol
	switch user.Role {
	case "chofer":
		chofer.UserID = user.ID
		if empresa != nil {
			empID := empresa.ID
			chofer.EmpresaID = empID
		}
		if err := tx.Create(chofer); err != nil {
			return err
		}
		user.ChoferID = &chofer.ID

	case "publicador":
		publicador.UserID = user.ID
		if empresa != nil {
			empID := empresa.ID
			publicador.EmpresaID = empID
		}
		if err := tx.Create(publicador); err != nil {
			return err
		}
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
		if _, err := tx.
			Model(&models.User{}).
			Where("id = ?", user.ID).
			Update(updates); err != nil {
			return err
		}
	}

	return nil
}

// CreateWithEmpresa crea User + Empresa en una transacción.
// Si algo falla, se revierte todo (empresa y usuario).
// Si empresa == nil o empresa.ID != 0, solo se vincula.
func (s *UserService) CreateWithEmpresa(user *models.User, empresa *models.Empresa) error {
	if user == nil {
		return errors.New("user is required")
	}
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if empresa != nil && empresa.ID == 0 {
		if err := tx.Create(empresa); err != nil {
			return err
		}
	}

	if empresa != nil {
		user.EmpresaID = &empresa.ID
	}

	if err := tx.Create(user); err != nil {
		return err
	}
	return tx.Commit()
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
	if lookupErr := recordError(err, u.ID, "user"); lookupErr != nil {
		return nil, lookupErr
	}
	return &u, nil
}

// GetByEmail devuelve un usuario por email (útil para login).
func (s *UserService) GetByEmail(email string) (*models.User, error) {
	var u models.User
	err := facades.Orm().Query().
		Where("email = ?", email).
		First(&u)
	if lookupErr := recordError(err, u.ID, "user"); lookupErr != nil {
		return nil, lookupErr
	}
	return &u, nil
}

// GetAllWithFilters lista usuarios con filtros y paginación (panel admin).
func (s *UserService) GetAllWithFilters(filters map[string]string, page, perPage int) ([]models.User, int64, error) {
	page, perPage = NormalizePagination(page, perPage)
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
		Order("nombre_legal asc").Limit(100).
		Find(&list)
	return list, err
}

// ─────────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────────

// Update actualiza cualquier campo del User por ID (incluido EmpresaID).
func (s *UserService) Update(id uint, updates map[string]interface{}) error {
	if email, ok := updates["email"].(string); ok {
		updates["email"] = strings.ToLower(strings.TrimSpace(email))
	}
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
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SET LOCAL lock_timeout = '3s'"); err != nil {
		return err
	}
	var user models.User
	if err := tx.Where("id = ?", id).LockForUpdate().First(&user); err != nil {
		return err
	}
	if err := recordError(nil, user.ID, "usuario"); err != nil {
		return err
	}
	if user.Role == "admin" {
		return errors.New("no se puede eliminar a un administrador")
	}
	var driver models.Chofer
	if err := tx.Where("user_id = ?", id).LockForUpdate().First(&driver); err != nil && !errors.Is(err, frameworkerrors.OrmRecordNotFound) {
		return err
	}
	var publisher models.Publicador
	if err := tx.Where("user_id = ?", id).LockForUpdate().First(&publisher); err != nil && !errors.Is(err, frameworkerrors.OrmRecordNotFound) {
		return err
	}
	// Keep commercial identities when referenced. Deactivation is the supported
	// administrative action for an account with business history.
	hasLoads, err := tx.Model(&models.Carga{}).Where("(chofer_id = ? AND ? > 0) OR (publicador_id = ? AND ? > 0)", driver.ID, driver.ID, publisher.ID, publisher.ID).Exists()
	if err != nil {
		return err
	}
	ownsCompany, err := tx.Model(&models.Empresa{}).Where("owner_id = ?", id).Exists()
	if err != nil {
		return err
	}
	ownsAddress, err := tx.Model(&models.Direccion{}).Where("owner_id = ?", id).Exists()
	if err != nil {
		return err
	}
	hasInvoices, err := tx.Model(&models.Factura{}).Where("(chofer_id = ? AND ? > 0) OR (publicador_id = ? AND ? > 0)", driver.ID, driver.ID, publisher.ID, publisher.ID).Exists()
	if err != nil {
		return err
	}
	if hasLoads || hasInvoices || ownsCompany || ownsAddress {
		return errors.New("usuario con referencias comerciales; desactiva la cuenta")
	}
	if driver.ID != 0 {
		if _, err := tx.Where("id = ?", driver.ID).Delete(&models.Chofer{}); err != nil {
			return err
		}
	}
	if publisher.ID != 0 {
		if _, err := tx.Where("id = ?", publisher.ID).Delete(&models.Publicador{}); err != nil {
			return err
		}
	}
	if _, err := tx.Where("id = ?", id).Delete(&models.User{}); err != nil {
		return err
	}
	return tx.Commit()
}

// EmailTaken verifica si un email ya está registrado en la tabla users.
// Si excludeID > 0, excluye ese usuario de la verificación (útil en updates).
func (s *UserService) EmailTaken(email string, excludeID uint) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
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
	email = strings.ToLower(strings.TrimSpace(email))
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
