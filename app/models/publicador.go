package models

import "time"

// EstadoPublicador representa la situación operativa del publicador.

// Publicador es el perfil profesional de quien publica cargas (broker persona).
// NO repite datos personales: esos viven en User.
type Publicador struct {
	ID uint `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`

	// ── Vínculos ──
	UserID    uint `json:"user_id"    db:"user_id"    gorm:"not null;uniqueIndex"` // 1:1 con User
	EmpresaID uint `json:"empresa_id" db:"empresa_id" gorm:"index"`                // empresa tipo "broker"

	// ── Licencia de broker ──
	NumeroLicenciaBroker     string     `json:"numero_licencia_broker"      db:"numero_licencia_broker"      gorm:"size:50;uniqueIndex"` // MC number, etc.
	PaisEmisionLicencia      string     `json:"pais_emision_licencia"       db:"pais_emision_licencia"       gorm:"size:100;default:Cuba"`
	FechaVencimientoLicencia *time.Time `json:"fecha_vencimiento_licencia"  db:"fecha_vencimiento_licencia"`

	// ── Perfil comercial ──
	AniosExperiencia int     `json:"anios_experiencia" db:"anios_experiencia" gorm:"default:0"`
	Especialidad     string  `json:"especialidad"      db:"especialidad"      gorm:"size:100"`                   // "refrigerados", "carga pesada", etc.
	Comision         float64 `json:"comision"         db:"comision"          gorm:"type:numeric(5,2);default:0"` // % sobre tarifa

	// ── Crédito (del publicador, distinto del de la empresa) ──
	CreditScore int `json:"credit_score" db:"credit_score" gorm:"default:0"`

	// ── Estado operativo ──
	Estado EstadoPublicador `json:"estado" db:"estado" gorm:"type:publicador_estado;not null;default:'activo';index"`

	// ── Auditoría ──
	CreatedAt time.Time  `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at" gorm:"index"`

	// ── Relaciones ──
	User    *User    `json:"user,omitempty"    gorm:"foreignKey:UserID"`
	Empresa *Empresa `json:"empresa,omitempty" gorm:"foreignKey:EmpresaID"`
}
