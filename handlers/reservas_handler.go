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

// GET /reservas
func GetReservas(c *gin.Context) {
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

	resultado, err := services.GetReservasFiltradas(filtro)
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
