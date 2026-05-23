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
// @Param local query string false "Nombre del local" example(SAN MARTIN)
// @Param semana query string false "Semana a consultar" example(2026-05-25)
// @Param dia query string false "Dia a consultar" example(lunes)
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta,feriado) example(mesa)
// @Param cliente query string false "Nombre del cliente" example(Maria)
// @Param reservados query bool false "Filtrar solo reservados" example(true)
// @Success 200 {object} utils.APIResponse{data=reservaListResponse}
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

	utils.Respond(c, http.StatusOK, reservaListResponse{
		TotalLocales: len(resultado),
		Filtros: reservaFiltrosResponse{
			Local:      filtro.Local,
			Semana:     filtro.Semana,
			Dia:        filtro.Dia,
			Tipo:       filtro.Tipo,
			Cliente:    filtro.Cliente,
			Reservados: filtro.Reservados,
		},
		Reservas: resultado,
	})
}
