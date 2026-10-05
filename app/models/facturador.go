package models

import "time"

type TipoEmisorFactura string

const (
	EmisorFacturaPublicador TipoEmisorFactura = "publicador"
	EmisorFacturaChofer     TipoEmisorFactura = "chofer"
)

// Factura documenta el cobro de una carga y vincula el emisor con su perfil.
type Factura struct {
	ID               uint              `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	CargaID          uint              `json:"carga_id" db:"carga_id" gorm:"uniqueIndex;not null"`
	EmisorID         *uint             `json:"emisor_id,omitempty" db:"emisor_id" gorm:"index"`
	ReceptorID       *uint             `json:"receptor_id,omitempty" db:"receptor_id" gorm:"index"`
	EmisorTipo       TipoEmisorFactura `json:"emisor_tipo" db:"emisor_tipo" gorm:"size:20;not null;default:publicador;index"`
	PublicadorID     *uint             `json:"publicador_id,omitempty" db:"publicador_id" gorm:"index"`
	ChoferID         *uint             `json:"chofer_id,omitempty" db:"chofer_id" gorm:"index"`
	NumeroFactura    string            `json:"numero_factura" db:"numero_factura" gorm:"size:50;uniqueIndex;not null"`
	FechaEmision     time.Time         `json:"fecha_emision" db:"fecha_emision" gorm:"not null"`
	FechaVencimiento *time.Time        `json:"fecha_vencimiento,omitempty" db:"fecha_vencimiento"`
	DistanciaKm      float64           `json:"distancia_km" db:"distancia_km" gorm:"type:numeric(10,2);not null;default:0;index"`
	TarifaPorKm      float64           `json:"tarifa_por_km" db:"tarifa_por_km" gorm:"type:numeric(12,4);not null;default:0"`
	Subtotal         float64           `json:"subtotal" db:"subtotal" gorm:"type:numeric(12,2);not null"`
	Impuestos        float64           `json:"impuestos" db:"impuestos" gorm:"type:numeric(12,2);default:0"`
	Total            float64           `json:"total" db:"total" gorm:"type:numeric(12,2);not null"`
	Moneda           Moneda            `json:"moneda" db:"moneda" gorm:"type:char(3);default:'CUP'"`
	Estado           EstadoFactura     `json:"estado" db:"estado" gorm:"type:factura_estado;not null;default:'borrador';index"`
	MetodoPago       *string           `json:"metodo_pago,omitempty" db:"metodo_pago" gorm:"size:50"`
	FechaPago        *time.Time        `json:"fecha_pago,omitempty" db:"fecha_pago"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        *time.Time        `json:"updated_at,omitempty" db:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt        *time.Time        `json:"deleted_at,omitempty" db:"deleted_at" gorm:"index"`

	// Relaciones
	Carga      *Carga      `json:"carga,omitempty" gorm:"foreignKey:CargaID"`
	Emisor     *Empresa    `json:"emisor,omitempty" gorm:"foreignKey:EmisorID"`
	Receptor   *Empresa    `json:"receptor,omitempty" gorm:"foreignKey:ReceptorID"`
	Publicador *Publicador `json:"publicador,omitempty" gorm:"foreignKey:PublicadorID"`
	Chofer     *Chofer     `json:"chofer,omitempty" gorm:"foreignKey:ChoferID"`
}
