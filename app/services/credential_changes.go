package services

import (
	"encoding/json"
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
)

type EmailChange struct {
	Purpose string `json:"purpose"`
	UserID  uint   `json:"user_id"`
	Email   string `json:"email"`
	Stamp   string `json:"stamp"`
}

func ConfirmEmailChange(tx orm.Query, payload string) error {
	var change EmailChange
	if err := json.Unmarshal([]byte(payload), &change); err != nil {
		return err
	}
	var user models.User
	err := tx.Where("id = ?", change.UserID).LockForUpdate().First(&user)
	if err := recordError(err, user.ID, "usuario"); err != nil {
		return err
	}
	if !user.IsActive || user.CredentialStamp() != change.Stamp {
		return ErrInvalidVerificationToken
	}
	exists, err := tx.Model(&models.User{}).Where("email = ?", change.Email).Exists()
	if err != nil {
		return err
	}
	if exists {
		return ErrRegistrationUnavailable
	}
	_, err = tx.Model(&models.User{}).Where("id = ?", user.ID).Update("email", change.Email)
	return err
}

func UpdateProfileAtomically(user *models.User, updates map[string]interface{}) error {
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current models.User
	err = tx.Where("id = ?", user.ID).LockForUpdate().First(&current)
	if err := recordError(err, current.ID, "usuario"); err != nil {
		return err
	}
	if current.CredentialStamp() != user.CredentialStamp() || !current.IsActive {
		return ErrConflict
	}
	if _, err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update(updates); err != nil {
		return err
	}
	return tx.Commit()
}
