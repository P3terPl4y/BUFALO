package requests

// DireccionStoreRequest mapea el formulario de direcciones.
// Los punteros del modelo (Calle, CodigoPostal, Latitud, Longitud) llegan
// aquí como string / *float64 y el controller los convierte.
type DireccionStoreRequest struct {
	Calle           string   `form:"calle"            json:"calle"`
	Ciudad          string   `form:"ciudad"           json:"ciudad"`
	EstadoProvincia string   `form:"estado_provincia" json:"estado_provincia"`
	CodigoPostal    string   `form:"codigo_postal"    json:"codigo_postal"`
	Pais            string   `form:"pais"             json:"pais"`
	Latitud         *float64 `form:"latitud"          json:"latitud"`
	Longitud        *float64 `form:"longitud"         json:"longitud"`
}

type DireccionUpdateRequest = DireccionStoreRequest
