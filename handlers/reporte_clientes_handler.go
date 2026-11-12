package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/pagination"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type reporteClientesResponse struct {
	// Reporte de clientes del periodo
	Reporte models.ReporteClientes `json:"reporte"`
}

// GetReporteClientes godoc
// @Summary Reporte de clientes
// @Description Ranking de clientes del periodo (semanal de lunes a domingo, quincenal del 1 al 15 y del 16 a fin de mes, o mensual) que contiene a la fecha indicada: quien genero mas dinero, quien tuvo mas citas, quien cancelo mas y quien lleva mas periodos seguidos viniendo. Devuelve un bloque general y uno por local, cada uno con totales (clientes activos, nuevos y recurrentes, ticket promedio por cliente, tasa de retorno), los 10 primeros de cada ranking, los 10 mayores deudores y la actividad por dia; la lista completa de clientes se pide paginada en /bd/reportes/clientes/detalle. Compara contra el periodo anterior equivalente. El dinero sale de los pagos PAGADO activos con cliente asignado (misma regla de fechas que /bd/pagos/resumen) y las citas de las reservas activas COMPLETADO vinculadas a un cliente; las ventas y citas sin cliente no entran en los rankings y se informan aparte. Un cliente es activo si pago o completo una cita en el periodo, y es nuevo si su primera actividad en el negocio cae en el periodo. Las deudas son el saldo pendiente de hoy y no dependen del periodo. Requiere token Bearer con rol admin_sys.
// @Tags Reportes
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param periodo query string false "semanal, quincenal o mensual" default(mensual)
// @Param fecha query string false "Cualquier dia del periodo, YYYY-MM-DD (por defecto hoy)"
// @Success 200 {object} utils.APIResponse{data=reporteClientesResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: periodo o fecha invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/reportes/clientes [get]
func (h *Container) GetReporteClientes(c *gin.Context) {
	reporte, err := h.ReporteClientes.Generar(c.Query("periodo"), c.Query("fecha"))
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, reporteClientesResponse{Reporte: *reporte})
}

type reporteClientesDetalleFiltrosCursor struct {
	Periodo    string `json:"periodo,omitempty"`
	Fecha      string `json:"fecha,omitempty"`
	LocalID    int    `json:"local_id,omitempty"`
	Busqueda   string `json:"busqueda,omitempty"`
	OrdenarPor string `json:"ordenar_por,omitempty"`
	Sentido    string `json:"sentido,omitempty"`
}

type reporteClientesDetalleResponse struct {
	// Cantidad de clientes devueltos en esta pagina.
	Total int `json:"total" example:"50"`
	// Clientes con movimiento en el periodo, del que mas dinero genero al que menos salvo que se pida otro orden.
	Clientes []models.ClienteReporte `json:"clientes"`
	// Datos de paginacion; siempre viene.
	Paginacion *pagination.Metadata `json:"paginacion"`
}

// GetReporteClientesDetalle godoc
// @Summary Detalle paginado del reporte de clientes
// @Description Lista paginada de los clientes con movimiento en el periodo (pago, cita completada, reserva agendada o cancelada), con las mismas reglas y columnas que los rankings de /bd/reportes/clientes. Siempre pagina: sin limit usa 50 por pagina y el maximo es 100. Por defecto ordena por dinero generado (posicion); con ordenar_por el orden lo resuelve el servidor sobre todo el resultado y los clientes sin dato en esa columna van al final. La paginacion avanza por desplazamiento y el cursor solo vale con el mismo periodo, fecha, local_id, busqueda, ordenar_por y sentido que lo generaron. Requiere token Bearer con rol admin_sys.
// @Tags Reportes
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param periodo query string false "semanal, quincenal o mensual" default(mensual)
// @Param fecha query string false "Cualquier dia del periodo, YYYY-MM-DD (por defecto hoy)"
// @Param local_id query int false "ID del local; sin el, todos los locales" example(1)
// @Param busqueda query string false "Busqueda parcial por nombre, telefono o servicio favorito" example(Maria)
// @Param ordenar_por query string false "Campo por el que ordenar; sin el, mas dinero generado primero" Enums(posicion,nombre,ingresos,variacion_pct,pagos,citas,canceladas,racha_periodos,servicio_top,ultima_actividad,saldo_pendiente) example(citas)
// @Param sentido query string false "Sentido del orden; solo con ordenar_por, por defecto asc" Enums(asc,desc) example(desc)
// @Param limit query int false "Tamano de pagina (1-100); por defecto 50" example(50)
// @Param cursor query string false "Cursor opaco devuelto en paginacion.next_cursor"
// @Param include_total query bool false "Incluye el total de registros y de paginas" example(false)
// @Success 200 {object} utils.APIResponse{data=reporteClientesDetalleResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: periodo, fecha o local_id invalidos, ordenar_por fuera de la lista, sentido distinto de asc o desc, limit fuera de rango o cursor de otros filtros"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/reportes/clientes/detalle [get]
func (h *Container) GetReporteClientesDetalle(c *gin.Context) {
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	// Este listado nunca se entrega entero: sin limit ni cursor se usa la pagina por defecto.
	if !page.Enabled {
		page = pagination.Request{Enabled: true, Limit: pagination.DefaultLimit}
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	sort, ok := parseSort(c, services.ClientesReporteOrdenables)
	if !ok {
		return
	}
	filtros := reporteClientesDetalleFiltrosCursor{
		Periodo: strings.ToLower(strings.TrimSpace(c.Query("periodo"))), Fecha: strings.TrimSpace(c.Query("fecha")),
		Busqueda: strings.TrimSpace(c.Query("busqueda")),
	}
	var localID *int
	if raw := strings.TrimSpace(c.Query("local_id")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			utils.RespondError(c, http.StatusBadRequest, "local_id invalido")
			return
		}
		localID, filtros.LocalID = &id, id
	}
	if sort.Active() {
		filtros.OrdenarPor, filtros.Sentido = sort.Key, sort.Direction()
	}
	after, ok := decodeOffsetCursor(c, page, "reporte_clientes_detalle", filtros)
	if !ok {
		return
	}
	clientes, total, err := h.ReporteClientes.Detalle(services.FiltroDetalleClientes{
		Periodo: filtros.Periodo, Fecha: filtros.Fecha, LocalID: localID, Busqueda: filtros.Busqueda,
		OrdenarPor: sort.Key, OrdenDesc: sort.Desc, Limit: page.QueryLimit(), Offset: after.Offset,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	clientes, metadata, err := pagination.Build(clientes, page, "reporte_clientes_detalle", filtros, func(models.ClienteReporte) any {
		return pagination.OffsetPosition{Offset: after.Offset + page.Limit}
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if includeTotal {
		pagination.AddTotal(metadata, total)
	}
	utils.Respond(c, http.StatusOK, reporteClientesDetalleResponse{Total: len(clientes), Clientes: clientes, Paginacion: metadata})
}
