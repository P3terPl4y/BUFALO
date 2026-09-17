package requests

type UserRegisterRequest struct {
	// ── Identidad y acceso ──
	Name     string `form:"name"`
	Email    string `form:"email"`
	Password string `form:"password"`
	Role     string `form:"role"` // publicador | chofer

	// ── Empresa ──
	EmpresaMode            string `form:"empresa_mode"` // existing | new
	EmpresaID              uint   `form:"empresa_id"`
	EmpresaNombreLegal     string `form:"empresa_nombre_legal"`
	EmpresaNombreComercial string `form:"empresa_nombre_comercial"`
	EmpresaTaxID           string `form:"empresa_tax_id"`
	EmpresaMCNumber        string `form:"empresa_mc_number"`
	EmpresaDOTNumber       string `form:"empresa_dot_number"`
	EmpresaTelefono        string `form:"empresa_telefono"`
	EmpresaEmail           string `form:"empresa_email"`
	EmpresaSitioWeb        string `form:"empresa_sitio_web"`
	// Contacto
	Phone          string `form:"phone"`
	PhoneAlt       string `form:"phone_alt"`
	WhatsApp       string `form:"whatsapp"`
	Telegram       string `form:"telegram"`
	EmergencyName  string `form:"emergency_name"`
	EmergencyPhone string `form:"emergency_phone"`
	// ── Ubicación ──
	Address    string  `form:"address"`
	City       string  `form:"city"`
	State      string  `form:"state"`
	Country    string  `form:"country"`
	PostalCode string  `form:"postal_code"`
	Latitude   float64 `form:"latitude"`
	Longitude  float64 `form:"longitude"`
	Radius     int     `form:"radius"`

	// ── Preferencias de carga ──
	PreferredEquipmentTypes string  `form:"preferred_equipment_types"`
	PreferredCargoTypes     string  `form:"preferred_cargo_types"`
	MaxWeight               float64 `form:"max_weight"`
	MaxDistance             float64 `form:"max_distance"`
	PreferredRoutes         string  `form:"preferred_routes"`

	// ── Disponibilidad ──
	AvailableFrom string `form:"available_from"`
	AvailableTo   string `form:"available_to"`
	Notes         string `form:"notes"`

	// ══════════════════════════════════════════════════════
	// PERFIL DE CHOFER (solo si Role == "chofer")
	// ══════════════════════════════════════════════════════
	ChoferNumeroLicencia           string `form:"chofer_numero_licencia"`
	ChoferTipoLicencia             string `form:"chofer_tipo_licencia"`
	ChoferPaisEmisionLicencia      string `form:"chofer_pais_emision_licencia"`
	ChoferFechaVencimientoLicencia string `form:"chofer_fecha_vencimiento_licencia"`
	ChoferAniosExperiencia         int    `form:"chofer_anios_experiencia"`
	ChoferTiposEquipoPermitidos    string `form:"chofer_tipos_equipo_permitidos"`
	ChoferCertificaciones          string `form:"chofer_certificaciones"`
	ChoferNumeroSeguro             string `form:"chofer_numero_seguro"`
	ChoferFechaVencimientoSeguro   string `form:"chofer_fecha_vencimiento_seguro"`

	// ══════════════════════════════════════════════════════
	// PERFIL DE PUBLICADOR (solo si Role == "publicador")
	// ══════════════════════════════════════════════════════
	PublicadorNumeroLicenciaBroker     string  `form:"publicador_numero_licencia_broker"`
	PublicadorPaisEmisionLicencia      string  `form:"publicador_pais_emision_licencia"`
	PublicadorFechaVencimientoLicencia string  `form:"publicador_fecha_vencimiento_licencia"`
	PublicadorAniosExperiencia         int     `form:"publicador_anios_experiencia"`
	PublicadorEspecialidad             string  `form:"publicador_especialidad"`
	PublicadorComision                 float64 `form:"publicador_comision"`
	PublicadorCreditScore              int     `form:"publicador_credit_score"`
}

type UserUpdateRequest = UserRegisterRequest

type AdminUpdateRequestByUser struct {
	// Identidad
	Name     string `form:"name"`
	Email    string `form:"email"`
	Password string `form:"password"`
	Role     string `form:"role"`
	IsActive string `form:"is_active"`
	
	Phone          string `form:"phone"`
	PhoneAlt       string `form:"phone_alt"`
	WhatsApp       string `form:"whatsapp"`
	Telegram       string `form:"telegram"`
	EmergencyName  string `form:"emergency_name"`
	EmergencyPhone string `form:"emergency_phone"`
	// Ubicación
	Address    string  `form:"address"`
	City       string  `form:"city"`
	State      string  `form:"state"`
	Country    string  `form:"country"`
	PostalCode string  `form:"postal_code"`
	Latitude   float64 `form:"latitude"`
	Longitude  float64 `form:"longitude"`
	Radius     int     `form:"radius"`

	// Preferencias
	PreferredEquipmentTypes string  `form:"preferred_equipment_types"`
	PreferredCargoTypes     string  `form:"preferred_cargo_types"`
	MaxWeight               float64 `form:"max_weight"`
	MaxDistance             float64 `form:"max_distance"`
	PreferredRoutes         string  `form:"preferred_routes"`

	// Disponibilidad
	AvailableFrom string `form:"available_from"`
	AvailableTo   string `form:"available_to"`
	Notes         string `form:"notes"`
}
