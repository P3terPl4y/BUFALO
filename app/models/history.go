package models
import (
"time"
	)
// CargaHistorial registra cada cambio de estado o modificación relevante
// de una carga para fines de auditoría y trazabilidad.
type CargaHistorial struct {
	ID        uint      `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	CargaID   string      `json:"carga_id" db:"carga_id" gorm:"not null;index"`
	Estado    EstadoCarga `json:"estado" db:"estado" gorm:"type:carga_estado;not null"`
	Comentario *string    `json:"comentario,omitempty" db:"comentario" gorm:"size:500"`
	UsuarioID *string     `json:"usuario_id,omitempty" db:"usuario_id" gorm:"index"`
	CreatedAt time.Time   `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
}
