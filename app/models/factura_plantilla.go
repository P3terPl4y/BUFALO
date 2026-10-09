package models

import "time"

// FacturaPlantilla is an invoice layout owned by a broker and reusable by its invoices.
type FacturaPlantilla struct {
	ID             uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PublicadorID   uint      `json:"publicador_id" gorm:"not null;index"`
	Nombre         string    `json:"nombre" gorm:"size:48;not null"`
	Preset         string    `json:"preset" gorm:"size:20;not null;default:''"`
	Formato        string    `json:"formato" gorm:"size:12;not null;default:'a4'"`
	Color          string    `json:"color" gorm:"size:7;not null;default:'#253746'"`
	BloquesJSON    string    `json:"-" gorm:"type:jsonb;not null"`
	Predeterminada bool      `json:"predeterminada" gorm:"not null;default:false"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (FacturaPlantilla) TableName() string { return "factura_plantillas" }
