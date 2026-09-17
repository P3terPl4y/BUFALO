package requests

type EmpresaStoreRequest struct {
	Tipo            string  `form:"tipo"             json:"tipo"`
	NombreLegal     string  `form:"nombre_legal"     json:"nombre_legal"`
	NombreComercial string  `form:"nombre_comercial" json:"nombre_comercial"`
	TaxID           string  `form:"tax_id"           json:"tax_id"`
	MCNumber        string  `form:"mc_number"        json:"mc_number"`
	DOTNumber       string  `form:"dot_number"       json:"dot_number"`
	DireccionID     *uint   `form:"direccion_id"     json:"direccion_id"`
	Telefono        string  `form:"telefono"         json:"telefono"`
	Email           string  `form:"email"            json:"email"`
	SitioWeb        string  `form:"sitio_web"        json:"sitio_web"`
	CreditScore     *int    `form:"credit_score"     json:"credit_score"`
	DaysToPay       float64 `form:"days_to_pay"      json:"days_to_pay"`
	Estado          string  `form:"estado"           json:"estado"`
}

type EmpresaUpdateRequest = EmpresaStoreRequest
