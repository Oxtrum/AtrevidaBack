package handlers

import (
	"net/http"

	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// POST /admin/importar
// Triggerea pipelinede importación desde Google Sheets a PostgreSQL.
// ImportarCatalogo godoc
// @Summary Importar catalogo desde Google Sheets
// @Description Ejecuta la importacion de categorias, servicios y combos desde Google Sheets hacia PostgreSQL. Response: categorias (int importadas), servicios (int), servicio_locales (int relaciones), combos (int), combo_locales (int relaciones), combo_servicios (int relaciones).
// @Tags Admin
// @Produce json
// @Success 200 {object} utils.APIResponse{data=importResponse}
// @Failure 500 {object} utils.APIResponse "Error durante la importacion"
// @Router /admin/importar [post]
func (h *Container) ImportarCatalogo(c *gin.Context) {
	resultado, err := h.Import.Ejecutar()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError,
			"error durante la importación: "+err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, importResponse{
		Categorias:      resultado.Categorias,
		Servicios:       resultado.Servicios,
		ServicioLocales: resultado.ServicioLocales,
		Combos:          resultado.Combos,
		ComboLocales:    resultado.ComboLocales,
		ComboServicios:  resultado.ComboServicios,
	})
}
