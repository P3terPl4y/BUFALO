package models

import "time"

// ChoferCalificacion is a rating from the publisher who posted a completed load.
type ChoferCalificacion struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CargaID      uint      `json:"carga_id" gorm:"not null;uniqueIndex"`
	ChoferID     uint      `json:"chofer_id" gorm:"not null;index"`
	PublicadorID uint      `json:"publicador_id" gorm:"not null;index"`
	Puntaje      int       `json:"puntaje" gorm:"not null"`
	Comentario   string    `json:"comentario" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (ChoferCalificacion) TableName() string { return "chofer_calificaciones" }

// RedChofer records a driver's membership in a publisher company's private network.
type RedChofer struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	EmpresaID uint      `json:"empresa_id" gorm:"not null;uniqueIndex:idx_red_chofer_empresa_chofer"`
	ChoferID  uint      `json:"chofer_id" gorm:"not null;uniqueIndex:idx_red_chofer_empresa_chofer"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	Chofer    *Chofer   `json:"chofer,omitempty" gorm:"foreignKey:ChoferID"`
}

func (RedChofer) TableName() string { return "red_choferes" }
