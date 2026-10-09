package services

import (
	"errors"
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

var ErrMembershipDenied = errors.New("la asociación no está permitida o ya fue resuelta")

type CompanyMembershipRequest struct {
	ID        uint
	UserID    uint
	EmpresaID uint
	Status    string
	DecidedBy *uint
	DecidedAt *time.Time
	CreatedAt time.Time
}

func (CompanyMembershipRequest) TableName() string { return "company_membership_requests" }

func membershipProfile(tx orm.Query, user models.User, company models.Empresa, assign bool) error {
	if !user.IsActive || company.Estado != models.EmpresaActiva || company.DeletedAt != nil {
		return ErrMembershipDenied
	}
	switch user.Role {
	case "chofer":
		if company.Tipo != models.TipoEmpresa("carrier") {
			return ErrMembershipDenied
		}
		var profile models.Chofer
		if err := tx.Where("user_id = ?", user.ID).LockForUpdate().First(&profile); err != nil {
			return err
		}
		if profile.ID == 0 || profile.DeletedAt != nil || profile.EmpresaID != 0 {
			return ErrMembershipDenied
		}
		if assign {
			_, err := tx.Model(&models.Chofer{}).Where("id = ?", profile.ID).Update("empresa_id", company.ID)
			return err
		}
	case "publicador":
		if company.Tipo != models.TipoEmpresa("broker") {
			return ErrMembershipDenied
		}
		var profile models.Publicador
		if err := tx.Where("user_id = ?", user.ID).LockForUpdate().First(&profile); err != nil {
			return err
		}
		if profile.ID == 0 || profile.DeletedAt != nil || profile.EmpresaID != 0 {
			return ErrMembershipDenied
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
	if err := membershipProfile(tx, user, company, false); err != nil {
		return err
	}
	exists, err := tx.Model(&CompanyMembershipRequest{}).Where("user_id = ? AND status = ?", userID, "pending").Exists()
	if err != nil {
		return err
	}
	if exists {
		return ErrMembershipDenied
	}
	// Prevent repeated request/rejection spam without blocking future applications.
	count, err := tx.Model(&CompanyMembershipRequest{}).Where("user_id = ? AND created_at > ?", userID, time.Now().Add(-24*time.Hour)).Count()
	if err != nil {
		return err
	}
	if count >= 5 {
		return ErrMembershipDenied
	}
	row := CompanyMembershipRequest{UserID: userID, EmpresaID: companyID, Status: "pending", CreatedAt: time.Now()}
	if err := tx.Create(&row); err != nil {
		return err
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
	var admin models.User
	if err := tx.Where("id = ? AND role = ? AND is_active = ?", adminID, "admin", true).LockForUpdate().First(&admin); err != nil {
		return err
	}
	if admin.ID == 0 {
		return ErrMembershipDenied
	}
	var row CompanyMembershipRequest
	if err := tx.Where("id = ?", requestID).LockForUpdate().First(&row); err != nil {
		return err
	}
	if row.ID == 0 || row.Status != "pending" {
		return ErrMembershipDenied
	}
	status := "rejected"
	if approve {
		var user models.User
		if err := tx.Where("id = ?", row.UserID).LockForUpdate().First(&user); err != nil {
			return err
		}
		var company models.Empresa
		if err := tx.Where("id = ?", row.EmpresaID).LockForUpdate().First(&company); err != nil {
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
	if _, err := tx.Model(&CompanyMembershipRequest{}).Where("id = ?", row.ID).Update(map[string]any{"status": status, "decided_by": adminID, "decided_at": time.Now()}); err != nil {
		return err
	}
	return tx.Commit()
}
