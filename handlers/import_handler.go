package handlers

import "github.com/gin-gonic/gin"

// POST /admin/importar
// Triggerea pipeline de importacion desde Google Sheets a PostgreSQL.
// ImportarCatalogo godoc
// @Summary Endpoint legacy para importar catalogo desde Google Sheets
// @Description Endpoint legacy deshabilitado. Google Sheets ya no esta soportado; el codigo anterior queda comentado para una posible recuperacion futura.
// @Tags Admin
// @Produce json
// @Failure 410 {object} utils.APIResponse "Google Sheets ya no soportado"
// @Router /admin/importar [post]
func (h *Container) ImportarCatalogo(c *gin.Context) {
	// resultado, err := h.Import.Ejecutar()
	// if err != nil {
	// 	utils.RespondError(c, http.StatusInternalServerError,
	// 		"error durante la importacion: "+err.Error())
	// 	return
	// }
	//
	// utils.Respond(c, http.StatusOK, importResponse{
	// 	Categorias:      resultado.Categorias,
	// 	Servicios:       resultado.Servicios,
	// 	ServicioLocales: resultado.ServicioLocales,
	// 	Combos:          resultado.Combos,
	// 	ComboLocales:    resultado.ComboLocales,
	// 	ComboServicios:  resultado.ComboServicios,
	// })

	RespondGoogleSheetsUnsupported(c)
}
