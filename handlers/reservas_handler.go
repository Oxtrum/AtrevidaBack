package handlers

import (
	"net/http"
	"strings"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

var tiposValidos = map[string]bool{
	"mesa":      true,
	"bicicleta": true,
	"feriado":   true,
}

// GetReservas godoc
// @Summary Listar reservas desde Google Sheets
// @Description Devuelve reservas filtradas por local, semana, dia, tipo, cliente y si estan reservadas.
// @Tags Reservas Sheets
// @Produce json
// @Param local query string false "Nombre del local"
// @Param semana query string false "Semana a consultar"
// @Param dia query string false "Dia a consultar"
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta,feriado)
// @Param cliente query string false "Nombre del cliente"
// @Param reservados query bool false "Filtrar solo reservados"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /reservas [get]
func (h *Container) GetReservas(c *gin.Context) {
	paramTipo := strings.ToLower(strings.TrimSpace(c.Query("tipo")))

	if paramTipo != "" && !tiposValidos[paramTipo] {
		utils.RespondError(c, http.StatusBadRequest,
			"tipo inválido, valores permitidos: mesa, bicicleta, feriado")
		return
	}

	filtro := services.FiltroReservas{
		Local:      strings.TrimSpace(c.Query("local")),
		Semana:     strings.TrimSpace(c.Query("semana")),
		Dia:        strings.TrimSpace(c.Query("dia")),
		Tipo:       paramTipo,
		Cliente:    strings.TrimSpace(c.Query("cliente")),
		Reservados: strings.ToLower(c.Query("reservados")) == "true",
	}

	resultado, err := h.Reservas.GetReservasFiltradas(filtro)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"total_locales": len(resultado),
		"filtros": gin.H{
			"local":      filtro.Local,
			"semana":     filtro.Semana,
			"dia":        filtro.Dia,
			"tipo":       filtro.Tipo,
			"cliente":    filtro.Cliente,
			"reservados": filtro.Reservados,
		},
		"reservas": resultado,
	})
}
