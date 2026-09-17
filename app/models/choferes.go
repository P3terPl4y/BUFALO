package models

import "time"

// EstadoChofer representa la disponibilidad operativa del conductor.


// Chofer es el perfil profesional del conductor. NO repite datos personales
// (nombre, email, ubicación): esos están en User.
type Chofer struct {
	ID uint `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`

	// ── Vínculos ──
	UserID    uint `json:"user_id"    db:"user_id"    gorm:"not null;uniqueIndex"` // 1:1 con User
	EmpresaID uint `json:"empresa_id" db:"empresa_id" gorm:"index"`      // empresa tipo "carrier"

	// ── Licencia de conducción ──
	NumeroLicencia           string     `json:"numero_licencia"            db:"numero_licencia"            gorm:"size:50;uniqueIndex;not null"`
	TipoLicencia             string     `json:"tipo_licencia"              db:"tipo_licencia"              gorm:"size:20;not null"`     // A, B, C, etc.
	PaisEmisionLicencia      string     `json:"pais_emision_licencia"      db:"pais_emision_licencia"      gorm:"size:100;default:Cuba"`
	FechaVencimientoLicencia *time.Time `json:"fecha_vencimiento_licencia" db:"fecha_vencimiento_licencia"`

	// ── Experiencia y capacidad ──
	AniosExperiencia       int    `json:"anios_experiencia"        db:"anios_experiencia"        gorm:"default:0"`
	TiposEquipoPermitidos  string `json:"tipos_equipo_permitidos"  db:"tipos_equipo_permitidos"  gorm:"type:text"` // CSV: "dry_van,reefer"
	Certificaciones        string `json:"certificaciones"          db:"certificaciones"          gorm:"type:text"` // CSV: "hazmat,tanker"

	// ── Seguro del conductor ──
	NumeroSeguro           string     `json:"numero_seguro"            db:"numero_seguro"            gorm:"size:100"`
	FechaVencimientoSeguro *time.Time `json:"fecha_vencimiento_seguro" db:"fecha_vencimiento_seguro"`

	// ── Estado operativo ──
	Estado EstadoChofer `json:"estado" db:"estado" gorm:"type:chofer_estado;not null;default:'disponible';index"`

	// ── Auditoría ──
	CreatedAt time.Time  `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at" gorm:"index"`

	// ── Relaciones ──
	User    *User    `json:"user,omitempty"    gorm:"foreignKey:UserID"`
	Empresa *Empresa `json:"empresa,omitempty" gorm:"foreignKey:EmpresaID"`
}
