package requests

type PublicadorStoreRequest struct {
	// ── Vínculos ──
	UserID    uint `form:"user_id"    json:"user_id"`
	EmpresaID uint `form:"empresa_id" json:"empresa_id"`

	// ── Licencia de broker ──
	NumeroLicenciaBroker     string `form:"numero_licencia_broker"      json:"numero_licencia_broker"`
	PaisEmisionLicencia      string `form:"pais_emision_licencia"       json:"pais_emision_licencia"`
	FechaVencimientoLicencia string `form:"fecha_vencimiento_licencia"  json:"fecha_vencimiento_licencia"`

	// ── Perfil comercial ──
	AniosExperiencia int     `form:"anios_experiencia" json:"anios_experiencia"`
	Especialidad     string  `form:"especialidad"      json:"especialidad"`
	Comision         float64 `form:"comision"          json:"comision"`
	CreditScore      int     `form:"credit_score"      json:"credit_score"`

	// ── Estado ──
	Estado string `form:"estado" json:"estado"`
}

type PublicadorUpdateRequest = PublicadorStoreRequest
