package router

// RootStatus godoc
// @Summary Estado de la API
// @Description Devuelve un mensaje simple para confirmar que la API esta activa.
// @Tags Base
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Router / [get]
func RootStatus() {}

// SwaggerUI godoc
// @Summary Abrir Swagger UI
// @Description Expone la interfaz web de Swagger para navegar la documentacion de la API.
// @Tags Base
// @Produce html
// @Success 200 {string} string "Swagger UI"
// @Router /swagger/{any} [get]
func SwaggerUI() {}

// DebugReservasUnfiltered godoc
// @Summary Ver reservas sin filtrar
// @Description Devuelve todas las reservas procesadas desde Google Sheets sin aplicar filtros.
// @Tags Debug Sheets
// @Produce json
// @Success 200 {array} interface{}
// @Router /reservas/unfiltered [get]
func DebugReservasUnfiltered() {}

// DebugReservasRaw godoc
// @Summary Ver hoja cruda de reservas
// @Description Devuelve el contenido crudo de la hoja SAN MARTIN desde Google Sheets.
// @Tags Debug Sheets
// @Produce json
// @Success 200 {array} interface{}
// @Router /reservas/raw [get]
func DebugReservasRaw() {}

// DebugCeldaRaw godoc
// @Summary Ver contenido crudo de una celda
// @Description Obtiene la coordenada resuelta, el valor crudo y el parseo de una celda especifica.
// @Tags Debug Sheets
// @Produce json
// @Param local query string true "Local"
// @Param semana query string true "Semana"
// @Param dia query string true "Dia"
// @Param hora query string true "Hora"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /reservas/celda-raw [get]
func DebugCeldaRaw() {}
