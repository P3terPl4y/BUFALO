package requests

// ChoferStoreRequest es el formulario de datos profesionales del conductor.
// NO incluye nombre, email, teléfono ni ubicación: esos están en User.
type ChoferStoreRequest struct {
	// ── Vínculos (solo en creación admin) ──
	UserID    uint `form:"user_id"    json:"user_id"`
	EmpresaID uint `form:"empresa_id" json:"empresa_id"`

	// ── Licencia ──
	NumeroLicencia           string `form:"numero_licencia"             json:"numero_licencia"`
	TipoLicencia             string `form:"tipo_licencia"               json:"tipo_licencia"`
	PaisEmisionLicencia      string `form:"pais_emision_licencia"       json:"pais_emision_licencia"`
	FechaVencimientoLicencia string `form:"fecha_vencimiento_licencia"  json:"fecha_vencimiento_licencia"`

	// ── Experiencia y capacidad ──
	AniosExperiencia      int    `form:"anios_experiencia"        json:"anios_experiencia"`
	TiposEquipoPermitidos string `form:"tipos_equipo_permitidos"  json:"tipos_equipo_permitidos"`
	Certificaciones       string `form:"certificaciones"          json:"certificaciones"`

	// ── Seguro ──
	NumeroSeguro           string `form:"numero_seguro"             json:"numero_seguro"`
	FechaVencimientoSeguro string `form:"fecha_vencimiento_seguro"  json:"fecha_vencimiento_seguro"`

	// ── Estado ──
	Estado string `form:"estado" json:"estado"`
}

// ChoferUpdateRequest: todos los campos son opcionales en edición.
// Si EmpresaID = 0, no se cambia la empresa. Si UserID se envía, se reasigna.
type ChoferUpdateRequest = ChoferStoreRequest
