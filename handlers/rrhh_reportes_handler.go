package handlers

import (
	"net/http"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type reporteAtencionesRRHHResponse struct {
	// Reporte de personal del periodo
	Reporte models.ReporteAtencionesRRHH `json:"reporte"`
}

// GetReporteAtencionesRRHH godoc
// @Summary Reporte de personal que mas atiende
// @Description Ranking del personal segun las atenciones registradas en el periodo (semanal de lunes a domingo, quincenal del 1 al 15 y del 16 a fin de mes, o mensual) que contiene a la fecha indicada. Devuelve un bloque general y uno por local, cada uno con totales, ranking, atenciones por dia y los usuarios que registraron. Compara contra el periodo anterior equivalente. Solo cuenta atenciones activas; quien atendio se agrupa por empleado de RRHH o, si no esta vinculado, por el nombre escrito. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param periodo query string false "semanal, quincenal o mensual" default(mensual)
// @Param fecha query string false "Cualquier dia del periodo, YYYY-MM-DD (por defecto hoy)"
// @Success 200 {object} utils.APIResponse{data=reporteAtencionesRRHHResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: periodo o fecha invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/reportes/atenciones [get]
func (h *Container) GetReporteAtencionesRRHH(c *gin.Context) {
	reporte, err := h.RRHH.ReporteAtenciones(c.Query("periodo"), c.Query("fecha"))
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, reporteAtencionesRRHHResponse{Reporte: *reporte})
}
