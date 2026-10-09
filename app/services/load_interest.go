package services

import (
	"encoding/json"
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
	"strings"
	"time"
	"unicode/utf8"
)

type InterestSender func(driver, publisher *models.User, load *models.Carga, comment string) error
type interestError string

func (e interestError) Error() string { return string(e) }
func IsPublicInterestError(err error) bool {
	var public interestError
	return errors.As(err, &public) || errors.Is(err, ErrNotFound)
}

type interestReservation struct {
	ID        uint
	CargaID   uint
	ChoferID  uint
	CreatedAt time.Time
}

func (interestReservation) TableName() string { return "load_interests" }

func SendLoadInterest(userID, loadID uint, comment string, sender InterestSender) error {
	comment = strings.TrimSpace(comment)
	if !utf8.ValidString(comment) || utf8.RuneCountInString(comment) < 1 || utf8.RuneCountInString(comment) > 1000 {
		return interestError("escribe un comentario de 1 a 1000 caracteres")
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var driver models.Chofer
	err = tx.Where("user_id = ?", userID).LockForUpdate().First(&driver)
	if err := recordError(err, driver.ID, "chofer"); err != nil {
		return err
	}
	var user models.User
	err = tx.Where("id = ? AND role = ? AND is_active = ?", userID, "chofer", true).First(&user)
	if err := recordError(err, user.ID, "usuario"); err != nil {
		return err
	}
	var load models.Carga
	err = tx.Where("id = ? AND estado = ? AND chofer_id IS NULL", loadID, models.CargaPublicada).
		Where("(audiencia = ? OR (audiencia = ? AND empresa_id IN (SELECT empresa_id FROM red_choferes WHERE chofer_id = ?)))", models.AudienciaLoadBoard, models.AudienciaRedPrivada, driver.ID).LockForUpdate().First(&load)
	if err := recordError(err, load.ID, "carga"); err != nil {
		return err
	}
	var publisher models.Publicador
	err = tx.Where("id = ?", load.PublicadorID).First(&publisher)
	if err := recordError(err, publisher.ID, "publicador"); err != nil {
		return err
	}
	var recipient models.User
	err = tx.Where("id = ? AND role = ? AND is_active = ?", publisher.UserID, "publicador", true).First(&recipient)
	if err := recordError(err, recipient.ID, "publicador"); err != nil {
		return err
	}
	duplicate, err := tx.Model(&interestReservation{}).Where("carga_id = ? AND chofer_id = ?", load.ID, driver.ID).Exists()
	if err != nil {
		return err
	}
	if duplicate {
		return interestError("ya enviaste tu interés en esta carga")
	}
	count, err := tx.Model(&interestReservation{}).Where("chofer_id = ? AND created_at >= ?", driver.ID, time.Now().Add(-24*time.Hour)).Count()
	if err != nil {
		return err
	}
	if count >= 10 {
		return interestError("alcanzaste el límite diario de notificaciones")
	}
	reservation := interestReservation{CargaID: load.ID, ChoferID: driver.ID, CreatedAt: time.Now()}
	if err := tx.Create(&reservation); err != nil {
		return err
	}
	driverSnapshot := models.User{Name: user.Name, Email: user.Email}
	driverSnapshot.ID = user.ID
	publisherSnapshot := models.User{Email: recipient.Email}
	publisherSnapshot.ID = recipient.ID
	payload, err := json.Marshal(interestDelivery{Driver: driverSnapshot, Publisher: publisherSnapshot, Load: load, Comment: comment})
	if err != nil {
		return err
	}
	encrypted, err := facades.Crypt().EncryptString(string(payload))
	if err != nil {
		return err
	}
	notification := interestOutbox{InterestID: reservation.ID, Payload: encrypted, AvailableAt: time.Now(), CreatedAt: time.Now()}
	if err := tx.Create(&notification); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// An inline attempt keeps the existing interaction fast. Failure is durable;
	// the background worker will retry without requiring another user request.
	if sender != nil {
		_ = DeliverInterest(notification.ID, sender)
	}
	return nil
}
