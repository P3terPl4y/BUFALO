package requests

type FacturaStoreRequest struct {
	CargaID          uint    `form:"carga_id"          json:"carga_id"`
	EmisorID         uint    `form:"emisor_id"         json:"emisor_id"`
	ReceptorID       uint    `form:"receptor_id"       json:"receptor_id"`
	ChoferID         *uint   `form:"chofer_id"         json:"chofer_id"`
	NumeroFactura    string  `form:"numero_factura"    json:"numero_factura"`
	FechaEmision     string  `form:"fecha_emision"     json:"fecha_emision"`
	FechaVencimiento string  `form:"fecha_vencimiento" json:"fecha_vencimiento"`
	DistanciaKm      float64 `form:"distancia_km"      json:"distancia_km"`
	TarifaPorKm      float64 `form:"tarifa_por_km"     json:"tarifa_por_km"`
	Subtotal         float64 `form:"subtotal"          json:"subtotal"`
	Impuestos        float64 `form:"impuestos"         json:"impuestos"`
	Total            float64 `form:"total"             json:"total"`
	Moneda           string  `form:"moneda"            json:"moneda"`
	MetodoPago       string  `form:"metodo_pago"       json:"metodo_pago"`
}

type FacturaUpdateRequest = FacturaStoreRequest
