package models

import "time"

type Empresa struct {
	ID              uint          `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	Tipo            TipoEmpresa   `json:"tipo" db:"tipo" gorm:"type:empresa_tipo;not null;index"`
	NombreLegal     string        `json:"nombre_legal" db:"nombre_legal" gorm:"size:255;not null"`
	NombreComercial *string       `json:"nombre_comercial,omitempty" db:"nombre_comercial" gorm:"size:255"`
	TaxID           *string       `json:"tax_id,omitempty" db:"tax_id" gorm:"size:50;uniqueIndex"`
	MCNumber        *string       `json:"mc_number,omitempty" db:"mc_number" gorm:"size:20;index"`
	DOTNumber       *string       `json:"dot_number,omitempty" db:"dot_number" gorm:"size:20"`
	DireccionID     *uint         `json:"direccion_id,omitempty" db:"direccion_id" gorm:"index"`
	Telefono        *string       `json:"telefono,omitempty" db:"telefono" gorm:"size:30"`
	Email           *string       `json:"email,omitempty" db:"email" gorm:"size:255"`
	SitioWeb        *string       `json:"sitio_web,omitempty" db:"sitio_web" gorm:"size:255"`
	CreditScore     *int          `json:"credit_score,omitempty" db:"credit_score"`
	DaysToPay       *float64      `json:"days_to_pay,omitempty" db:"days_to_pay" gorm:"type:numeric(5,2)"`

	// ── Propietario: el user que creó la empresa ──
	OwnerID *uint `json:"owner_id,omitempty" db:"owner_id" gorm:"index"`
	Owner   *User `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`

	Estado    EstadoEmpresa `json:"estado" db:"estado" gorm:"type:empresa_estado;not null;default:'activo';index"`
	CreatedAt time.Time     `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt *time.Time    `json:"updated_at,omitempty" db:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time    `json:"deleted_at,omitempty" db:"deleted_at" gorm:"index"`

	Direccion *Direccion `json:"direccion,omitempty" gorm:"foreignKey:DireccionID"`
	Choferes  []Chofer   `json:"choferes,omitempty" gorm:"foreignKey:EmpresaID"`
}
