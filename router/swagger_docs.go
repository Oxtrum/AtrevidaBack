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
// @Description Devuelve todas las reservas procesadas desde Google Sheets sin aplicar filtros. Solo uso en depuracion.
// @Tags Debug Sheets
// @Produce json
// @Success 200 {array} interface{} "Lista completa de reservas procesadas"
// @Router /reservas/unfiltered [get]
func DebugReservasUnfiltered() {}

// DebugReservasRaw godoc
// @Summary Ver hoja cruda de reservas
// @Description Devuelve el contenido crudo de la hoja SAN MARTIN desde Google Sheets. Solo uso en depuracion.
// @Tags Debug Sheets
// @Produce json
// @Success 200 {array} interface{} "Datos crudos de la hoja"
// @Router /reservas/raw [get]
func DebugReservasRaw() {}

// DebugCeldaRaw godoc
// @Summary Ver contenido crudo de una celda
// @Description Obtiene la coordenada A1, el valor crudo y el parseo de una celda especifica. local: nombre del local (requerido). semana: YYYY-MM-DD (requerido). dia: lunes a sabado (requerido). hora: HH:MM (requerido). Solo uso en depuracion.
// @Tags Debug Sheets
// @Produce json
// @Param local query string true "Nombre del local" example(SAN MARTIN)
// @Param semana query string true "Semana YYYY-MM-DD" example(2026-05-25)
// @Param dia query string true "Dia" example(lunes)
// @Param hora query string true "Hora HH:MM" example(15:00)
// @Success 200 {object} utils.APIResponse "Coordenada A1, raw, parsed"
// @Failure 500 {object} utils.APIResponse "Error al resolver o leer la celda"
// @Router /reservas/celda-raw [get]
func DebugCeldaRaw() {}
