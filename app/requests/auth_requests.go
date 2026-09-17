package requests

type RegisterRequest struct {
	Name     string `form:"name"`
	Email    string `form:"email"`
	Password string `form:"password"`
	Role     string `form:"role"` // broker | carrier

	// Modo de empresa: "existing" | "new" | "none"
	EmpresaMode string `form:"empresa_mode"`

	// Modo "existing"
	EmpresaID *uint `form:"empresa_id"`

	// Modo "new"
	NewEmpresaNombre string `form:"new_empresa_nombre"`
	NewEmpresaTaxID  string `form:"new_empresa_tax_id"`
	NewEmpresaTel    string `form:"new_empresa_tel"`
	NewEmpresaEmail  string `form:"new_empresa_email"`
}
