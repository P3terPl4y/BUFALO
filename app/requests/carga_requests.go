package requests

type CargaStoreRequest struct {
	NumeroReferencia   string   `form:"numero_referencia"    json:"numero_referencia"`
	OrigenDireccionID  uint     `form:"origen_direccion_id"  json:"origen_direccion_id"`
	DestinoDireccionID uint     `form:"destino_direccion_id" json:"destino_direccion_id"`
	FechaRecogida      string   `form:"fecha_recogida"       json:"fecha_recogida"`
	FechaEntrega       string   `form:"fecha_entrega"        json:"fecha_entrega"`
	TipoCarga          string   `form:"tipo_carga"           json:"tipo_carga"`
	TipoEquipo         string   `form:"tipo_equipo"          json:"tipo_equipo"`
	PesoKg             *float64 `form:"peso_kg"              json:"peso_kg"`
	Commodity          string   `form:"commodity"            json:"commodity"`
	DistanciaKm        float64  `form:"distancia_km"         json:"distancia_km"`
	DistanciaRealKm    *float64 `form:"distancia_real_km"    json:"distancia_real_km"`
	TarifaTotal        *float64 `form:"tarifa_total"         json:"tarifa_total"`
	TarifaPorKm        *float64 `form:"tarifa_por_km"        json:"tarifa_por_km"`
	Moneda             string   `form:"moneda"               json:"moneda"`
	Audiencia          string   `form:"audiencia"            json:"audiencia"`

	// Solo para admin: permiten asignar explícitamente.
	// El controller ignora estos campos si el usuario es publicador normal.
	PublicadorID *uint `form:"publicador_id" json:"publicador_id"`
	EmpresaID    *uint `form:"empresa_id"    json:"empresa_id"`
	ChoferID     *uint `form:"chofer_id"     json:"chofer_id"`
}

type CargaUpdateRequest = CargaStoreRequest
