package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// PendingRegistration stores an encrypted, short-lived signup payload. It is
// separate from users and role profiles, which are created only after the
// email owner submits the confirmation form.
type PendingRegistration struct {
	orm.Model
	Email           string `gorm:"column:email;type:varchar(100);uniqueIndex;not null"`
	TokenHash       string `gorm:"column:token_hash;type:char(64);uniqueIndex;not null"`
	Payload         string `gorm:"column:payload;type:text;not null"`
	ResendHash      *string
	TokenCiphertext *string
	LastSentAt      time.Time
	ResendCount     int
	ExpiresAt       time.Time `gorm:"column:expires_at;index;not null"`
}

func (PendingRegistration) TableName() string { return "pending_registrations" }
