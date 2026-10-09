package models

import (
	"time"
)

// Direccion normaliza las ubicaciones geográficas usadas por empresas,
// choferes y cargas. Evita duplicidad y permite cálculos de distancia.
type Direccion struct {
	ID              uint       `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	OwnerID         *uint      `json:"owner_id,omitempty" db:"owner_id" gorm:"index"`
	Owner           *User      `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
	Calle           *string    `json:"calle,omitempty" db:"calle" gorm:"size:255"`
	Ciudad          string     `json:"ciudad" db:"ciudad" gorm:"size:100;not null;index"`
	EstadoProvincia string     `json:"estado_provincia" db:"estado_provincia" gorm:"size:100;not null;index"`
	CodigoPostal    *string    `json:"codigo_postal,omitempty" db:"codigo_postal" gorm:"size:20;index"`
	Pais            string     `json:"pais" db:"pais" gorm:"size:100;not null;default:'Cuba'"`
	Latitud         *float64   `json:"latitud,omitempty" db:"latitud" gorm:"type:numeric(10,7)"`
	Longitud        *float64   `json:"longitud,omitempty" db:"longitud" gorm:"type:numeric(10,7)"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty" db:"updated_at" gorm:"autoUpdateTime"`
}
