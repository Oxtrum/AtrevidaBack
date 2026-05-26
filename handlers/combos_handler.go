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

// GET /combos
// Query params:
//
//	nombre    — búsqueda parcial en el nombre del combo
//	categoria — búsqueda parcial en la categoría
//	local     — "ARANJUEZ", "CENTRO" o "ARANJUEZ + CENTRO"
//	sesiones  — sesiones del combo
//
// GetCombos godoc
// @Summary Listar combos
// @Description Devuelve combos desde catalogo Sheets con filtros. Filtros: nombre busqueda parcial (opcional), categoria busqueda parcial (opcional), local ARANJUEZ/CENTRO/SAN MARTIN (opcional), sesiones numero exacto (opcional). Response: total (int), filtros (objeto con nombre, categoria, local, sesiones), combos ([]ComboItem con: nombre, categoria, local, costo_total, sesiones_totales, servicios_incluidos []ServicioIncluido con nombre, tiempo HH:MM, costo, sesiones).
// @Tags Catalogo
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre" example(relax)
// @Param categoria query string false "Busqueda parcial por categoria" example(Corporal)
// @Param local query string false "Local" Enums(ARANJUEZ,CENTRO,SAN MARTIN) example(ARANJUEZ)
// @Param sesiones query int false "Numero exacto de sesiones" example(4)
// @Success 200 {object} utils.APIResponse{data=comboListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: sesiones debe ser entero positivo, local invalido"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /combos [get]
func (h *Container) GetCombos(c *gin.Context) {
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
	if local != "" && local != "ARANJUEZ" && local != "CENTRO" && local != "SAN MARTIN" {
		utils.RespondError(c, http.StatusBadRequest,
			"local inválido, valores permitidos: ARANJUEZ, CENTRO (o SAN MARTIN)")
		return
	}

	if local == "SAN MARTIN" {
		local = "CENTRO"
	}

	filtro := services.FiltroCombos{
		Nombre:    strings.TrimSpace(c.Query("nombre")),
		Categoria: strings.TrimSpace(c.Query("categoria")),
		Local:     local,
		Sesiones:  sesiones,
	}

	resultado := h.Combos.GetCombosFiltrados(filtro)
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
