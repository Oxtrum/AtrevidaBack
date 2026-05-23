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

// GET /servicios
// Query params:
//
//	nombre    — búsqueda parcial en el nombre del servicio
//	categoria — búsqueda parcial en la categoría (ej: "manual", "combos")
//	local     — "ARANJUEZ" o "CENTRO"
//	sesiones  — número exacto de sesiones (ej: 3, 5, 10)
//
// GetServicios godoc
// @Summary Listar servicios
// @Description Devuelve servicios filtrados por nombre, categoria, local y sesiones.
// @Tags Catalogo
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre" example(depila)
// @Param categoria query string false "Busqueda parcial por categoria" example(Facial)
// @Param local query string false "Local" Enums(ARANJUEZ,CENTRO,SAN MARTIN) example(CENTRO)
// @Param sesiones query int false "Numero exacto de sesiones" example(10)
// @Param requiere_evaluacion query bool false "Filtrar por servicios que requieren evaluacion" example(false)
// @Success 200 {object} utils.APIResponse{data=servicioListResponse}
// @Failure 400 {object} utils.APIResponse
// @Router /servicios [get]
func (h *Container) GetServicios(c *gin.Context) {
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

	var requiereEvaluacion *bool
	if raw := strings.TrimSpace(c.Query("requiere_evaluacion")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest,
				"requiere_evaluacion debe ser true o false")
			return
		}
		requiereEvaluacion = &v
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

	filtro := services.FiltroServicios{
		Nombre:             strings.TrimSpace(c.Query("nombre")),
		Categoria:          strings.TrimSpace(c.Query("categoria")),
		Local:              local,
		Sesiones:           sesiones,
		RequiereEvaluacion: requiereEvaluacion,
	}

	resultado := h.ServiciosPG.GetServiciosFiltrados(filtro)
	if resultado == nil {
		resultado = []models.ServicioItem{}
	}

	utils.Respond(c, http.StatusOK, servicioListResponse{
		Total: len(resultado),
		Filtros: servicioFiltrosResponse{
			Nombre:             filtro.Nombre,
			Categoria:          filtro.Categoria,
			Local:              filtro.Local,
			Sesiones:           filtro.Sesiones,
			RequiereEvaluacion: filtro.RequiereEvaluacion,
		},
		Servicios: resultado,
	})
}
