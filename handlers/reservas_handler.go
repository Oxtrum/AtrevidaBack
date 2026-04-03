package handlers

import (
	"net/http"
	"strings"

	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

var tiposValidos = map[string]bool{
	"mesa":      true,
	"bicicleta": true,
	"feriado":   true,
}

// GET /reservas
//
// Query params (todos opcionales):
//   local      → nombre exacto del local / sheet (case-insensitive)
//   semana     → substring del título de semana  (case-insensitive)
//   dia        → nombre exacto del día           (case-insensitive)
//   tipo       → "mesa" | "bicicleta" | "feriado"
//   cliente    → substring del nombre de cliente (case-insensitive)
//   reservados → "true" devuelve solo items con cliente asignado
func GetReservas(c *gin.Context) {
	paramTipo := strings.ToLower(strings.TrimSpace(c.Query("tipo")))

	if paramTipo != "" && !tiposValidos[paramTipo] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "tipo inválido",
			"validos": []string{"mesa", "bicicleta", "feriado"},
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_locales": len(resultado),
		"filtros": gin.H{
			"local":      filtro.Local,
			"semana":     filtro.Semana,
			"dia":        filtro.Dia,
			"tipo":       filtro.Tipo,
			"cliente":    filtro.Cliente,
			"reservados": filtro.Reservados,
		},
		"data": resultado,
	})
}
