package models

import "time"

// EmpresaChatMensaje es un mensaje persistente, visible únicamente a los
// miembros autorizados de una empresa.
type EmpresaChatMensaje struct {
	ID          uint       `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	EmpresaID   uint       `json:"empresa_id" db:"empresa_id" gorm:"not null;index:idx_empresa_chat_messages"`
	UserID      uint       `json:"user_id" db:"user_id" gorm:"not null;index"`
	Mensaje     string     `json:"mensaje" db:"mensaje" gorm:"type:text;not null"`
	ModeradoPor *uint      `json:"moderado_por,omitempty" db:"moderado_por"`
	ModeradoEn  *time.Time `json:"moderado_en,omitempty" db:"moderado_en"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	User        *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (EmpresaChatMensaje) TableName() string { return "empresa_chat_mensajes" }
