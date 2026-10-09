package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"goravel/app/facades"
	"goravel/app/models"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	frameworkerrors "github.com/goravel/framework/errors"
)

const EmailVerificationTTL = 24 * time.Hour

var ErrInvalidVerificationToken = errors.New("verification token is invalid or expired")

// ErrRegistrationUnavailable evita reemplazar solicitudes pendientes o cuentas existentes.
var ErrRegistrationUnavailable = errors.New("registration already exists or is pending")

type EmailVerificationService struct{}

func NewEmailVerificationService() *EmailVerificationService {
	return &EmailVerificationService{}
}

// Start stores an encrypted payload and returns a one-time opaque token. The
// token itself is never persisted; only its SHA-256 digest is stored.
func (s *EmailVerificationService) Start(email, payload string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || payload == "" {
		return "", errors.New("email and payload are required")
	}

	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return "", fmt.Errorf("generate verification token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(rawToken)
	encryptedPayload, err := facades.Crypt().EncryptString(payload)
	if err != nil {
		return "", fmt.Errorf("encrypt registration payload: %w", err)
	}
	encryptedToken, err := facades.Crypt().EncryptString(token)
	if err != nil {
		return "", err
	}
	row := models.PendingRegistration{
		TokenCiphertext: &encryptedToken, LastSentAt: time.Now(),
		Email:     email,
		TokenHash: tokenDigest(token),
		Payload:   encryptedPayload,
		ExpiresAt: time.Now().Add(EmailVerificationTTL),
	}

	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("SET LOCAL lock_timeout = '3s'"); err != nil {
		return "", err
	}
	// Serializa también el caso en que todavía no existe una fila que bloquear.
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", email); err != nil {
		return "", err
	}
	var pending models.PendingRegistration
	err = tx.Model(&models.PendingRegistration{}).Where("email = ?", email).LockForUpdate().First(&pending)
	if err != nil && !errors.Is(err, frameworkerrors.OrmRecordNotFound) {
		return "", err
	}
	if pending.ID != 0 && pending.ExpiresAt.After(time.Now()) {
		return "", ErrRegistrationUnavailable
	}
	exists, err := tx.Model(&models.User{}).Where("email = ?", email).Exists()
	if err != nil {
		return "", err
	}
	if exists {
		return "", ErrRegistrationUnavailable
	}
	if pending.ID != 0 {
		if _, err := tx.Where("id = ?", pending.ID).Delete(&models.PendingRegistration{}); err != nil {
			return "", err
		}
	}
	if err := tx.Create(&row); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return token, nil
}

// Confirm executes account creation and consumes the token in one database
// transaction, so concurrent clicks cannot create multiple profiles.
func (s *EmailVerificationService) Confirm(token string, createAccount func(orm.Query, string) error) error {
	if len(token) != 43 { // 32 random bytes encoded as unpadded base64url
		return ErrInvalidVerificationToken
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("SET LOCAL lock_timeout = '3s'"); err != nil {
		return err
	}

	var pending models.PendingRegistration
	err = tx.Model(&models.PendingRegistration{}).
		Where("token_hash = ?", tokenDigest(token)).
		LockForUpdate().First(&pending)
	if err != nil && !errors.Is(err, frameworkerrors.OrmRecordNotFound) {
		return err
	}
	// La hora se comprueba después de adquirir el bloqueo, no antes de esperar.
	if pending.ID == 0 || !pending.ExpiresAt.After(time.Now()) {
		return ErrInvalidVerificationToken
	}
	payload, err := facades.Crypt().DecryptString(pending.Payload)
	if err != nil {
		return ErrInvalidVerificationToken
	}
	if createAccount == nil {
		return errors.New("account creation callback is required")
	}
	if err := createAccount(tx, payload); err != nil {
		return err
	}
	if _, err := tx.Where("id = ?", pending.ID).Delete(&models.PendingRegistration{}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *EmailVerificationService) Cancel(token string) error {
	if len(token) != 43 {
		return nil
	}
	_, err := facades.Orm().Query().
		Where("token_hash = ?", tokenDigest(token)).
		Delete(&models.PendingRegistration{})
	return err
}

func tokenDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// IssueResendTicket creates a distinct capability. It cannot activate an account
// and never exposes the confirmation token to the registering browser.
func (s *EmailVerificationService) IssueResendTicket(token string) (string, error) {
	if len(token) != 43 {
		return "", ErrInvalidVerificationToken
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", err
	}
	ticket := base64.RawURLEncoding.EncodeToString(secret[:])
	result, err := facades.Orm().Query().Model(&models.PendingRegistration{}).Where("token_hash = ? AND expires_at > ?", tokenDigest(token), time.Now()).Update("resend_hash", tokenDigest(ticket))
	if err != nil {
		return "", err
	}
	if result.RowsAffected != 1 {
		return "", ErrInvalidVerificationToken
	}
	return ticket, nil
}

// Resend retains the original token and expiry; it cannot prolong registrations
// or invalidate a legitimate link. Failed SMTP attempts consume their quota.
func (s *EmailVerificationService) Resend(ticket string, send func(string, string) error) error {
	if len(ticket) != 43 || send == nil {
		return ErrInvalidVerificationToken
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SET LOCAL lock_timeout = '3s'"); err != nil {
		return err
	}
	var row models.PendingRegistration
	err = tx.Where("resend_hash = ?", tokenDigest(ticket)).LockForUpdate().First(&row)
	if err != nil && !errors.Is(err, frameworkerrors.OrmRecordNotFound) {
		return err
	}
	if row.ID == 0 || !row.ExpiresAt.After(time.Now()) || row.ResendCount >= 5 || time.Since(row.LastSentAt) < time.Minute || row.TokenCiphertext == nil {
		return ErrInvalidVerificationToken
	}
	token, err := facades.Crypt().DecryptString(*row.TokenCiphertext)
	if err != nil {
		return err
	}
	if _, err := tx.Model(&models.PendingRegistration{}).Where("id = ?", row.ID).Update(map[string]any{"last_sent_at": time.Now(), "resend_count": row.ResendCount + 1}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return send(row.Email, token)
}
