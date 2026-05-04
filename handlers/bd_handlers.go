package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

/*
// GET /bd/servicios
func (h *Container) GetServiciosPG(c *gin.Context) {
	sesiones := 0
	if raw := strings.TrimSpace(c.Query("sesiones")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			utils.RespondError(c, http.StatusBadRequest,
				"sesiones debe ser un número entero positivo")
			return
		}
		sesiones = n
	}

	local := strings.ToUpper(strings.TrimSpace(c.Query("local")))
	if local != "" && local != "SAN MARTIN" && local != "PASEO ARANJUEZ" {
		utils.RespondError(c, http.StatusBadRequest,
			"local inválido, valores permitidos: SAN MARTIN, PASEO ARANJUEZ")
		return
	}

	filtro := services.FiltroServicios{
		Nombre:    strings.TrimSpace(c.Query("nombre")),
		Categoria: strings.TrimSpace(c.Query("categoria")),
		Local:     local,
		Sesiones:  sesiones,
	}

	resultado := h.ServiciosPG.GetServiciosFiltrados(filtro)
	if resultado == nil {
		resultado = []models.ServicioItem{}
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"total": len(resultado),
		"filtros": gin.H{
			"nombre":    filtro.Nombre,
			"categoria": filtro.Categoria,
			"local":     filtro.Local,
			"sesiones":  filtro.Sesiones,
		},
		"servicios": resultado,
	})
}
*/

// GET /bd/combos
func (h *Container) GetCombosPG(c *gin.Context) {
	sesiones := 0
	if raw := strings.TrimSpace(c.Query("sesiones")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			utils.RespondError(c, http.StatusBadRequest,
				"sesiones debe ser un número entero positivo")
			return
		}
		sesiones = n
	}

	local := strings.ToUpper(strings.TrimSpace(c.Query("local")))
	if local != "" && local != "SAN MARTIN" && local != "PASEO ARANJUEZ" {
		utils.RespondError(c, http.StatusBadRequest,
			"local inválido, valores permitidos: SAN MARTIN, PASEO ARANJUEZ")
		return
	}

	filtro := services.FiltroCombos{
		Nombre:    strings.TrimSpace(c.Query("nombre")),
		Categoria: strings.TrimSpace(c.Query("categoria")),
		Local:     local,
		Sesiones:  sesiones,
	}

	resultado := h.CombosPG.GetCombosFiltrados(filtro)
	if resultado == nil {
		resultado = []models.ComboItem{}
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"total": len(resultado),
		"filtros": gin.H{
			"nombre":    filtro.Nombre,
			"categoria": filtro.Categoria,
			"local":     filtro.Local,
			"sesiones":  filtro.Sesiones,
		},
		"combos": resultado,
	})
}
