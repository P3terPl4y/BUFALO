package models

// ═══════════════════════════════════════════════════════════════════
// TODOS LOS ENUMS DEL DOMINIO EN UN SOLO LUGAR
// ═══════════════════════════════════════════════════════════════════

// ── Empresa ──
type TipoEmpresa string

// Tipos de organización que pueden participar en una operación logística.
const (
	TipoBroker    TipoEmpresa = "broker"
	TipoCarrier   TipoEmpresa = "carrier"
	TipoShipper   TipoEmpresa = "shipper"
	TipoFactoring TipoEmpresa = "factoring"
	TipoMixto     TipoEmpresa = "mixto"
)

type EstadoEmpresa string

const (
	EmpresaActiva     EstadoEmpresa = "activo"
	EmpresaInactiva   EstadoEmpresa = "inactivo"
	EmpresaSuspendida EstadoEmpresa = "suspendido"
)

// ── User ──
// (Role es string libre: "admin", "publicador", "chofer")

// ── Chofer ──
type EstadoChofer string

const (
	ChoferDisponible EstadoChofer = "disponible"
	ChoferEnViaje    EstadoChofer = "en_viaje"
	ChoferInactivo   EstadoChofer = "inactivo"
)

// ── Publicador ──
type EstadoPublicador string

const (
	PublicadorActivo     EstadoPublicador = "activo"
	PublicadorInactivo   EstadoPublicador = "inactivo"
	PublicadorSuspendido EstadoPublicador = "suspendido"
)

// ── Carga ──
type TipoCarga string

const (
	CargaFTL TipoCarga = "FTL"
	CargaLTL TipoCarga = "LTL"
)

type TipoEquipo string

const (
	EquipoDryVan     TipoEquipo = "dry_van"
	EquipoFlatbed    TipoEquipo = "flatbed"
	EquipoReefer     TipoEquipo = "reefer"
	EquipoStepDeck   TipoEquipo = "step_deck"
	EquipoDoubleDrop TipoEquipo = "double_drop"
	EquipoLowboy     TipoEquipo = "lowboy"
	EquipoCargoVan   TipoEquipo = "cargo_van"
	EquipoBoxTruck   TipoEquipo = "box_truck"
	EquipoPowerOnly  TipoEquipo = "power_only"
)

type EstadoCarga string

const (
	CargaPublicada  EstadoCarga = "publicada"
	CargaNegociando EstadoCarga = "negociando"
	CargaAsignada   EstadoCarga = "asignada"
	CargaEnTransito EstadoCarga = "en_transito"
	CargaEntregada  EstadoCarga = "entregada"
	CargaCancelada  EstadoCarga = "cancelada"
)

type Audiencia string

const (
	AudienciaLoadBoard    Audiencia = "load_board"
	AudienciaRedPrivada   Audiencia = "red_privada"
	AudienciaRedExtendida Audiencia = "red_extendida"
)

type Moneda string

const (
	MonedaCUP Moneda = "CUP"
	MonedaMLC Moneda = "MLC"
	MonedaUSD Moneda = "USD"
	MonedaEUR Moneda = "EUR"
)

// ── Factura ──
type EstadoFactura string

const (
	FacturaBorrador  EstadoFactura = "borrador"
	FacturaEmitida   EstadoFactura = "emitida"
	FacturaPagada    EstadoFactura = "pagada"
	FacturaVencida   EstadoFactura = "vencida"
	FacturaCancelada EstadoFactura = "cancelada"
)
