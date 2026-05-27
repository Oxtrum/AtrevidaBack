package handlers

import "github.com/gin-gonic/gin"

var tiposValidos = map[string]bool{
	"mesa":      true,
	"bicicleta": true,
	"feriado":   true,
}

// GetReservas godoc
// @Summary Endpoint legacy de reservas en Google Sheets
// @Description Endpoint legacy deshabilitado. Google Sheets ya no esta soportado; use los endpoints de reservas bajo /bd.
// @Tags Reservas Sheets
// @Produce json
// @Param local query string false "Nombre del local" example(SAN MARTIN)
// @Param semana query string false "Semana a consultar YYYY-MM-DD" example(2026-05-25)
// @Param dia query string false "Dia a consultar" example(lunes)
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta,feriado) example(mesa)
// @Param cliente query string false "Nombre del cliente" example(Maria)
// @Param reservados query bool false "Filtrar solo reservados" example(true)
// @Failure 410 {object} utils.APIResponse "Google Sheets ya no soportado"
// @Router /reservas [get]
func (h *Container) GetReservas(c *gin.Context) {
	RespondGoogleSheetsUnsupported(c)
}
