package handlers

import (
	"net/http"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type configuracionResponse struct {
	// Parametros del sistema (clave -> valor tipado): inventario.politica_sin_stock, inventario.descontar_en_estado, inventario.permitir_stock_negativo, inventario.considerar_comprometido, finanzas.estado_gasto_fijo_generado, finanzas.dias_aviso_gasto_fijo, finanzas.zona_horaria, finanzas.moneda
	Configuracion map[string]interface{} `json:"configuracion"`
}

// GetConfiguracion godoc
// @Summary Leer parametros del sistema
// @Description Devuelve los parametros de inventario y finanzas (politica ante falta de stock, momento del descuento, stock negativo, aviso de gastos fijos, zona horaria, moneda). Requiere token Bearer.
// @Tags Configuracion
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Success 200 {object} utils.APIResponse{data=configuracionResponse}
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/configuracion [get]
func (h *Container) GetConfiguracion(c *gin.Context) {
	cfg, err := h.Configuracion.Get()
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, configuracionResponse{Configuracion: cfg})
}

type configuracionPublicaResponse struct {
	// true si la pagina principal muestra la seccion de paquetes.
	MostrarPaquetes bool `json:"mostrar_paquetes" example:"true"`
}

// GetConfiguracionPublica godoc
// @Summary Parametros publicos de la pagina principal
// @Description Devuelve solo lo que la pagina principal necesita para decidir que secciones mostrar (hoy: mostrar_paquetes). No requiere token.
// @Tags Configuracion
// @Produce json
// @Success 200 {object} utils.APIResponse{data=configuracionPublicaResponse}
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/configuracion/publica [get]
func (h *Container) GetConfiguracionPublica(c *gin.Context) {
	cfg, err := h.Configuracion.Snapshot()
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, configuracionPublicaResponse{MostrarPaquetes: cfg.MostrarPaquetes})
}

// PatchConfiguracion godoc
// @Summary Actualizar parametros del sistema
// @Description Actualiza solo las claves enviadas. Valores: inventario.politica_sin_stock ADVERTIR|BLOQUEAR; inventario.descontar_en_estado COMPLETADO|AGENDADO; inventario.permitir_stock_negativo y inventario.considerar_comprometido booleanos; finanzas.estado_gasto_fijo_generado PENDIENTE|PAGADO; finanzas.dias_aviso_gasto_fijo entero 0-31; finanzas.zona_horaria zona IANA; finanzas.moneda texto de 1 a 5 caracteres. Requiere token Bearer con rol admin_sys.
// @Tags Configuracion
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body map[string]interface{} true "Parametros a actualizar"
// @Success 200 {object} utils.APIResponse{data=configuracionResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: parametro desconocido o valor invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/configuracion [patch]
func (h *Container) PatchConfiguracion(c *gin.Context) {
	var patch map[string]interface{}
	if err := c.ShouldBindJSON(&patch); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := h.Configuracion.Update(patch)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, configuracionResponse{Configuracion: cfg})
}

// GetReporteFinanciero godoc
// @Summary Reporte financiero (ingresos vs egresos)
// @Description Resume el periodo: ingresos (ventas pagadas + otros ingresos) vs egresos (fijos y variables, incluidas compras de inventario) y utilidad neta. Solo cuentan movimientos PAGADO; los pendientes se informan aparte y los anulados nunca suman. Las ventas usan la misma regla que /bd/pagos/resumen. Con local se filtra por local y los gastos generales (sin local) se informan aparte sin prorratear. Antes de calcular genera los gastos fijos atrasados. Requiere token Bearer con rol admin_sys.
// @Tags Reportes
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param fecha_desde query string true "Desde (YYYY-MM-DD)"
// @Param fecha_hasta query string true "Hasta (YYYY-MM-DD)"
// @Param local query string false "Nombre del local"
// @Param agrupar query string false "dia o mes (default mes)"
// @Success 200 {object} utils.APIResponse{data=models.ReporteFinanciero}
// @Failure 400 {object} utils.APIResponse "Error de validacion: fechas invalidas o rango invertido, agrupar invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/reportes/financiero [get]
func (h *Container) GetReporteFinanciero(c *gin.Context) {
	var reporte *models.ReporteFinanciero
	reporte, err := h.ReporteFinanciero.Generar(c.Query("fecha_desde"), c.Query("fecha_hasta"), c.Query("local"), c.Query("agrupar"))
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, reporte)
}
