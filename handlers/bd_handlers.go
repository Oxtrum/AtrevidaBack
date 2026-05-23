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

// GetCombosPG godoc
// @Summary Listar combos desde base de datos
// @Description Devuelve combos persistidos en PostgreSQL con filtros opcionales.
// @Tags Combos BD
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre" example(relax)
// @Param categoria query string false "Busqueda parcial por categoria" example(Corporal)
// @Param local query string false "Local" Enums(SAN MARTIN,PASEO ARANJUEZ) example(PASEO ARANJUEZ)
// @Param sesiones query int false "Numero exacto de sesiones" example(4)
// @Success 200 {object} utils.APIResponse{data=comboListResponse}
// @Failure 400 {object} utils.APIResponse
// @Router /bd/combos [get]
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

	utils.Respond(c, http.StatusOK, comboListResponse{
		Total: len(resultado),
		Filtros: comboFiltrosResponse{
			Nombre:    filtro.Nombre,
			Categoria: filtro.Categoria,
			Local:     filtro.Local,
			Sesiones:  filtro.Sesiones,
		},
		Combos: resultado,
	})
}
