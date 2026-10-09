package services

import (
	"errors"
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

var ErrMembershipDenied = errors.New("la asociación no está permitida o ya fue resuelta")
var ErrMembershipAlreadyAssociated = errors.New("el perfil ya pertenece a una empresa")
var ErrMembershipPending = errors.New("ya existe una solicitud pendiente")
var ErrMembershipRequestLimit = errors.New("se alcanzó el límite de solicitudes")
var ErrMembershipWrongCompanyType = errors.New("el tipo de empresa no corresponde al perfil")
var ErrMembershipCompanyOwner = errors.New("el propietario de una empresa no puede afiliarse a otra")

func membershipDenied(reason error) error { return errors.Join(ErrMembershipDenied, reason) }

type CompanyMembershipRequest struct {
	ID        uint
	UserID    uint
	EmpresaID uint
	Status    string
	DecidedBy *uint
	DecidedAt *time.Time
	CreatedAt time.Time
	User      *models.User `gorm:"foreignKey:UserID"`
}

// CanRequestCompanyMembership is the read-only counterpart to
// RequestCompanyMembership; the latter repeats every check transactionally.
func CanRequestCompanyMembership(userID, companyID uint) (bool, error) {
	if userID == 0 || companyID == 0 {
		return false, nil
	}
	query := facades.Orm().Query()
	var user models.User
	if err := query.Where("id = ? AND is_active = ?", userID, true).First(&user); err != nil {
		if errors.Is(recordError(err, user.ID, "usuario"), ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	var company models.Empresa
	if err := query.Where("id = ? AND estado = ? AND deleted_at IS NULL", companyID, models.EmpresaActiva).First(&company); err != nil {
		if errors.Is(recordError(err, company.ID, "empresa"), ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if company.OwnerID != nil && *company.OwnerID == userID {
		return false, nil
	}
	ownsCompany, err := query.Model(&models.Empresa{}).Where("owner_id = ? AND deleted_at IS NULL", userID).Exists()
	if err != nil {
		return false, err
	}
	if ownsCompany || (user.EmpresaID != nil && *user.EmpresaID != 0) {
		return false, nil
	}
	var pending bool
	if pending, err = query.Model(&CompanyMembershipRequest{}).Where("user_id = ? AND status = ?", userID, "pending").Exists(); err != nil || pending {
		return false, err
	}
	switch user.Role {
	case "chofer":
		if company.Tipo != models.TipoEmpresa("carrier") {
			return false, nil
		}
		var profile models.Chofer
		if err := query.Where("user_id = ?", userID).First(&profile); err != nil {
			if errors.Is(recordError(err, profile.ID, "chofer"), ErrNotFound) {
				return false, nil
			}
			return false, err
		}
		return profile.ID != 0 && profile.EmpresaID == 0 && profile.DeletedAt == nil, nil
	case "publicador":
		if company.Tipo != models.TipoEmpresa("broker") {
			return false, nil
		}
		var profile models.Publicador
		if err := query.Where("user_id = ?", userID).First(&profile); err != nil {
			if errors.Is(recordError(err, profile.ID, "publicador"), ErrNotFound) {
				return false, nil
			}
			return false, err
		}
		return profile.ID != 0 && profile.EmpresaID == 0 && profile.DeletedAt == nil, nil
	default:
		return false, nil
	}
}

func (CompanyMembershipRequest) TableName() string { return "company_membership_requests" }

func membershipProfile(tx orm.Query, user models.User, company models.Empresa, assign bool) error {
	if !user.IsActive || company.Estado != models.EmpresaActiva || company.DeletedAt != nil {
		return ErrMembershipDenied
	}
	if user.EmpresaID != nil && *user.EmpresaID != 0 {
		return membershipDenied(ErrMembershipAlreadyAssociated)
	}
	switch user.Role {
	case "chofer":
		if company.Tipo != models.TipoEmpresa("carrier") {
			return membershipDenied(ErrMembershipWrongCompanyType)
		}
		var profile models.Chofer
		if err := tx.Where("user_id = ?", user.ID).LockForUpdate().First(&profile); err != nil {
			return err
		}
		if profile.ID == 0 || profile.DeletedAt != nil {
			return ErrMembershipDenied
		}
		if profile.EmpresaID != 0 {
			return membershipDenied(ErrMembershipAlreadyAssociated)
		}
		if assign {
			_, err := tx.Model(&models.Chofer{}).Where("id = ?", profile.ID).Update("empresa_id", company.ID)
			return err
		}
	case "publicador":
		if company.Tipo != models.TipoEmpresa("broker") {
			return membershipDenied(ErrMembershipWrongCompanyType)
		}
		var profile models.Publicador
		if err := tx.Where("user_id = ?", user.ID).LockForUpdate().First(&profile); err != nil {
			return err
		}
		if profile.ID == 0 || profile.DeletedAt != nil {
			return ErrMembershipDenied
		}
		if profile.EmpresaID != 0 {
			return membershipDenied(ErrMembershipAlreadyAssociated)
		}
		if assign {
			_, err := tx.Model(&models.Publicador{}).Where("id = ?", profile.ID).Update("empresa_id", company.ID)
			return err
		}
	default:
		return ErrMembershipDenied
	}
	return nil
}
func RequestCompanyMembership(userID, companyID uint) error {
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SET LOCAL lock_timeout = '3s'"); err != nil {
		return err
	}
	var user models.User
	if err := tx.Where("id = ?", userID).LockForUpdate().First(&user); err != nil {
		return err
	}
	if user.ID == 0 {
		return ErrMembershipDenied
	}
	var company models.Empresa
	if err := tx.Where("id = ?", companyID).LockForUpdate().First(&company); err != nil {
		return err
	}
	if company.ID == 0 {
		return ErrMembershipDenied
	}
	ownsCompany, err := tx.Model(&models.Empresa{}).Where("owner_id = ? AND deleted_at IS NULL", userID).Exists()
	if err != nil {
		return err
	}
	if ownsCompany {
		return membershipDenied(ErrMembershipCompanyOwner)
	}
	if err := membershipProfile(tx, user, company, false); err != nil {
		return err
	}
	exists, err := tx.Model(&CompanyMembershipRequest{}).Where("user_id = ? AND status = ?", userID, "pending").Exists()
	if err != nil {
		return err
	}
	if exists {
		return membershipDenied(ErrMembershipPending)
	}
	// Prevent repeated request/rejection spam without blocking future applications.
	count, err := tx.Model(&CompanyMembershipRequest{}).Where("user_id = ? AND created_at > ?", userID, time.Now().Add(-24*time.Hour)).Count()
	if err != nil {
		return err
	}
	if count >= 5 {
		return membershipDenied(ErrMembershipRequestLimit)
	}
	row := CompanyMembershipRequest{UserID: userID, EmpresaID: companyID, Status: "pending", CreatedAt: time.Now()}
	if err := tx.Create(&row); err != nil {
		return err
	}
	if company.OwnerID != nil {
		if err := CreateUserNotification(tx, *company.OwnerID, "membership_requested", "Nueva solicitud de afiliación", "Un usuario solicitó unirse a tu empresa.", "company", company.ID, false, userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func DecideCompanyMembership(adminID, requestID uint, approve bool) error {
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SET LOCAL lock_timeout = '3s'"); err != nil {
		return err
	}
	var actor models.User
	if err := tx.Where("id = ? AND is_active = ?", adminID, true).LockForUpdate().First(&actor); err != nil {
		return err
	}
	if actor.ID == 0 {
		return ErrMembershipDenied
	}
	var row CompanyMembershipRequest
	if err := tx.Where("id = ?", requestID).LockForUpdate().First(&row); err != nil {
		return err
	}
	if row.ID == 0 || row.Status != "pending" {
		return ErrMembershipDenied
	}
	var company models.Empresa
	if err := tx.Where("id = ?", row.EmpresaID).LockForUpdate().First(&company); err != nil {
		return err
	}
	if actor.Role != "admin" && (actor.Role == "" || company.OwnerID == nil || *company.OwnerID != actor.ID) {
		return ErrMembershipDenied
	}
	status := "rejected"
	if approve {
		var user models.User
		if err := tx.Where("id = ?", row.UserID).LockForUpdate().First(&user); err != nil {
			return err
		}
		if company.ID == 0 || user.ID == 0 {
			return ErrMembershipDenied
		}
		if err := membershipProfile(tx, user, company, true); err != nil {
			return err
		}
		if _, err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("empresa_id", company.ID); err != nil {
			return err
		}
		status = "approved"
	}
	if _, err := tx.Model(&CompanyMembershipRequest{}).Where("id = ?", row.ID).Update(map[string]any{"status": status, "decided_by": actor.ID, "decided_at": time.Now()}); err != nil {
		return err
	}
	title, message := "Solicitud de afiliación rechazada", "El propietario rechazó tu solicitud. Tu cuenta sigue sin empresa; puedes solicitar unirte a otra empresa o volver a solicitarlo después."
	if approve {
		title, message = "Solicitud de afiliación aprobada", "El propietario aprobó tu solicitud y tu cuenta ya está asociada a la empresa."
	}
	if err := CreateUserNotification(tx, row.UserID, "membership_decision", title, message, "company", company.ID, false, adminID); err != nil {
		return err
	}
	return tx.Commit()
}
