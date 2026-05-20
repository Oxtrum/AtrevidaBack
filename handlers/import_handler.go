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
// @Description Ejecuta la importacion de categorias, servicios y combos desde Google Sheets hacia PostgreSQL.
// @Tags Admin
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /admin/importar [post]
func (h *Container) ImportarCatalogo(c *gin.Context) {
	resultado, err := h.Import.Ejecutar()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError,
			"error durante la importación: "+err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"categorias":       resultado.Categorias,
		"servicios":        resultado.Servicios,
		"servicio_locales": resultado.ServicioLocales,
		"combos":           resultado.Combos,
		"combo_locales":    resultado.ComboLocales,
		"combo_servicios":  resultado.ComboServicios,
	})
}
