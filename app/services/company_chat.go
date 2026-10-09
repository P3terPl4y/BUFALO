package services

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
)

var ErrCompanyChatForbidden = errors.New("no tienes acceso al chat de esta empresa")
var ErrCompanyChatMessage = errors.New("el mensaje debe tener entre 1 y 1000 caracteres")
var ErrCompanyChatMessageMissing = errors.New("mensaje no encontrado")

const companyChatHistoryLimit = 50

type CompanyChatAccess struct {
	Member    bool
	Moderator bool
}

func companyChatAccess(query orm.Query, companyID, userID uint) (CompanyChatAccess, error) {
	if companyID == 0 || userID == 0 {
		return CompanyChatAccess{}, ErrCompanyChatForbidden
	}
	var company models.Empresa
	if err := query.Where("id = ? AND estado = ?", companyID, models.EmpresaActiva).First(&company); err != nil {
		return CompanyChatAccess{}, err
	}
	if company.ID == 0 || company.DeletedAt != nil {
		return CompanyChatAccess{}, ErrCompanyChatForbidden
	}
	var user models.User
	if err := query.Where("id = ? AND is_active = ?", userID, true).First(&user); err != nil {
		return CompanyChatAccess{}, err
	}
	if user.ID == 0 {
		return CompanyChatAccess{}, ErrCompanyChatForbidden
	}
	if company.OwnerID != nil && *company.OwnerID == user.ID {
		return CompanyChatAccess{Member: true, Moderator: true}, nil
	}
	if user.Role != "chofer" && user.Role != "publicador" {
		return CompanyChatAccess{}, ErrCompanyChatForbidden
	}
	if user.EmpresaID != nil && *user.EmpresaID == companyID {
		return CompanyChatAccess{Member: true}, nil
	}
	var profileMember bool
	var err error
	if user.Role == "chofer" {
		profileMember, err = query.Model(&models.Chofer{}).Where("user_id = ? AND empresa_id = ?", userID, companyID).Exists()
	} else {
		profileMember, err = query.Model(&models.Publicador{}).Where("user_id = ? AND empresa_id = ?", userID, companyID).Exists()
	}
	if err != nil {
		return CompanyChatAccess{}, err
	}
	if !profileMember {
		return CompanyChatAccess{}, ErrCompanyChatForbidden
	}
	return CompanyChatAccess{Member: true}, nil
}

func GetCompanyChatAccess(companyID, userID uint) (CompanyChatAccess, error) {
	return companyChatAccess(facades.Orm().Query(), companyID, userID)
}

func ListCompanyChatMessages(companyID, userID uint) ([]models.EmpresaChatMensaje, error) {
	if _, err := GetCompanyChatAccess(companyID, userID); err != nil {
		return nil, err
	}
	var messages []models.EmpresaChatMensaje
	err := facades.Orm().Query().Model(&models.EmpresaChatMensaje{}).
		With("User").Where("empresa_id = ?", companyID).
		OrderBy("id", "desc").Limit(companyChatHistoryLimit).Find(&messages)
	if err != nil {
		return nil, err
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, nil
}

func SendCompanyChatMessage(companyID, userID uint, rawMessage string) error {
	message := strings.TrimSpace(rawMessage)
	if !utf8.ValidString(message) || utf8.RuneCountInString(message) < 1 || utf8.RuneCountInString(message) > 1000 {
		return ErrCompanyChatMessage
	}
	for _, r := range message {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return ErrCompanyChatMessage
		}
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SET LOCAL lock_timeout = '3s'"); err != nil {
		return err
	}
	if _, err := companyChatAccess(tx, companyID, userID); err != nil {
		return err
	}
	row := models.EmpresaChatMensaje{EmpresaID: companyID, UserID: userID, Mensaje: message, CreatedAt: time.Now()}
	if err := tx.Create(&row); err != nil {
		return err
	}
	return tx.Commit()
}

func ModerateCompanyChatMessage(companyID, messageID, moderatorID uint) error {
	if companyID == 0 || messageID == 0 || moderatorID == 0 {
		return ErrCompanyChatForbidden
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SET LOCAL lock_timeout = '3s'"); err != nil {
		return err
	}
	access, err := companyChatAccess(tx, companyID, moderatorID)
	if err != nil {
		return err
	}
	if !access.Moderator {
		return ErrCompanyChatForbidden
	}
	var row models.EmpresaChatMensaje
	if err := tx.Where("id = ? AND empresa_id = ?", messageID, companyID).LockForUpdate().First(&row); err != nil {
		return err
	}
	if row.ID == 0 || row.ModeradoEn != nil {
		return ErrCompanyChatMessageMissing
	}
	now := time.Now()
	if _, err := tx.Model(&models.EmpresaChatMensaje{}).Where("id = ? AND empresa_id = ? AND moderado_en IS NULL", messageID, companyID).
		Update(map[string]any{"moderado_por": moderatorID, "moderado_en": now}); err != nil {
		return err
	}
	return tx.Commit()
}
