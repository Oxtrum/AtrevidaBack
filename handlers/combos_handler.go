package handlers

import "github.com/gin-gonic/gin"

// GET /combos
// Query params:
//
//	nombre    - busqueda parcial en el nombre del combo
//	categoria - busqueda parcial en la categoria
//	local     - "ARANJUEZ", "CENTRO" o "ARANJUEZ + CENTRO"
//	sesiones  - sesiones del combo
//
// GetCombos godoc
// @Summary Endpoint legacy de combos en Google Sheets
// @Description Endpoint legacy deshabilitado. Google Sheets ya no esta soportado; use GET /bd/combos.
// @Tags Catalogo
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre" example(relax)
// @Param categoria query string false "Busqueda parcial por categoria" example(Corporal)
// @Param local query string false "Local" Enums(ARANJUEZ,CENTRO,SAN MARTIN) example(ARANJUEZ)
// @Param sesiones query int false "Numero exacto de sesiones" example(4)
// @Failure 410 {object} utils.APIResponse "Google Sheets ya no soportado"
// @Router /combos [get]
func (h *Container) GetCombos(c *gin.Context) {
	RespondGoogleSheetsUnsupported(c)
}
