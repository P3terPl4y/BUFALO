package models

import "time"

type Carga struct {
	ID uint `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`

	// ── Referencia ──
	NumeroReferencia string `json:"numero_referencia" db:"numero_referencia" gorm:"size:50;uniqueIndex;not null"`

	// ── Actores ──
	PublicadorID uint  `json:"publicador_id" db:"publicador_id" gorm:"not null;index"` // quién publica
	EmpresaID    uint  `json:"empresa_id"    db:"empresa_id"    gorm:"not null;index"` // empresa del publicador (broker)
	ChoferID     *uint `json:"chofer_id"     db:"chofer_id"     gorm:"index"`          // asignado al aceptar

	// ── Ruta ──
	OrigenDireccionID  uint `json:"origen_direccion_id"  db:"origen_direccion_id"  gorm:"not null;index"`
	DestinoDireccionID uint `json:"destino_direccion_id" db:"destino_direccion_id" gorm:"not null;index"`

	// ── Fechas ──
	FechaRecogida time.Time  `json:"fecha_recogida" db:"fecha_recogida" gorm:"not null;index"`
	FechaEntrega  *time.Time `json:"fecha_entrega"  db:"fecha_entrega"`

	// ── Carga ──
	TipoCarga   TipoCarga   `json:"tipo_carga"   db:"tipo_carga"   gorm:"type:carga_tipo;not null"`
	TipoEquipo  TipoEquipo  `json:"tipo_equipo"  db:"tipo_equipo"  gorm:"type:equipo_tipo;not null"`
	PesoKg      *float64    `json:"peso_kg"      db:"peso_kg"      gorm:"type:numeric(10,2)"`
	Commodity   *string     `json:"commodity"    db:"commodity"    gorm:"size:255"`

	// ── Distancias ──
	DistanciaKm     float64  `json:"distancia_km"      db:"distancia_km"      gorm:"type:numeric(10,2);not null;default:0;index"`
	DistanciaRealKm *float64 `json:"distancia_real_km" db:"distancia_real_km" gorm:"type:numeric(10,2)"`

	// ── Tarifa ──
	TarifaTotal *float64 `json:"tarifa_total"  db:"tarifa_total"  gorm:"type:numeric(12,2)"`
	TarifaPorKm *float64 `json:"tarifa_por_km" db:"tarifa_por_km" gorm:"type:numeric(12,4);index"`
	Moneda      Moneda   `json:"moneda"        db:"moneda"        gorm:"type:char(3);default:'CUP'"`

	// ── Estado y audiencia ──
	Estado    EstadoCarga `json:"estado"    db:"estado"    gorm:"type:carga_estado;not null;default:'publicada';index"`
	Audiencia Audiencia   `json:"audiencia" db:"audiencia" gorm:"type:audiencia_tipo;default:'load_board'"`

	// ── Auditoría ──
	CreatedAt time.Time  `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at" gorm:"index"`

	// ── Relaciones ──
	Publicador       *Publicador `json:"publicador,omitempty"        gorm:"foreignKey:PublicadorID"`
	Empresa          *Empresa    `json:"empresa,omitempty"           gorm:"foreignKey:EmpresaID"`
	Chofer           *Chofer     `json:"chofer,omitempty"            gorm:"foreignKey:ChoferID"`
	OrigenDireccion  *Direccion  `json:"origen_direccion,omitempty"  gorm:"foreignKey:OrigenDireccionID"`
	DestinoDireccion *Direccion  `json:"destino_direccion,omitempty" gorm:"foreignKey:DestinoDireccionID"`
	Factura          *Factura    `json:"factura,omitempty"           gorm:"foreignKey:CargaID"`
}
