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
// @Summary Endpoint debug legacy de reservas en Google Sheets
// @Description Endpoint debug legacy deshabilitado. Google Sheets ya no esta soportado.
// @Tags Debug Sheets
// @Produce json
// @Failure 410 {object} utils.APIResponse "Google Sheets ya no soportado"
// @Router /reservas/unfiltered [get]
func DebugReservasUnfiltered() {}

// DebugReservasRaw godoc
// @Summary Endpoint debug legacy de hoja cruda en Google Sheets
// @Description Endpoint debug legacy deshabilitado. Google Sheets ya no esta soportado.
// @Tags Debug Sheets
// @Produce json
// @Failure 410 {object} utils.APIResponse "Google Sheets ya no soportado"
// @Router /reservas/raw [get]
func DebugReservasRaw() {}

// DebugCeldaRaw godoc
// @Summary Endpoint debug legacy de celda en Google Sheets
// @Description Endpoint debug legacy deshabilitado. Google Sheets ya no esta soportado.
// @Tags Debug Sheets
// @Produce json
// @Param local query string false "Nombre del local" example(SAN MARTIN)
// @Param semana query string false "Semana YYYY-MM-DD" example(2026-05-25)
// @Param dia query string false "Dia" example(lunes)
// @Param hora query string false "Hora HH:MM" example(15:00)
// @Failure 410 {object} utils.APIResponse "Google Sheets ya no soportado"
// @Router /reservas/celda-raw [get]
func DebugCeldaRaw() {}
