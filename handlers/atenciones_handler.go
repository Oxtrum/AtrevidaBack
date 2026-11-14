package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/pagination"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type registrarAtencionRequest struct {
	EmpleadoID                 *int    `json:"empleado_id,omitempty"`
	EmpleadoNombre             string  `json:"empleado_nombre" binding:"required"`
	FechaAtencion              string  `json:"fecha_atencion" binding:"required"`
	HoraInicio                 *string `json:"hora_inicio,omitempty"`
	HoraFin                    *string `json:"hora_fin,omitempty"`
	Lugar                      string  `json:"lugar" binding:"required"`
	Observaciones              *string `json:"observaciones,omitempty"`
	ConfirmarStockInsuficiente bool    `json:"confirmar_stock_insuficiente,omitempty"`
	Insumos                    []struct {
		ItemID   int     `json:"item_id"`
		Cantidad float64 `json:"cantidad"`
	} `json:"insumos"`
}

// PostAtencion godoc
// @Summary Registrar atención y cerrar reserva
// @Description Registra la atención real y cambia una reserva AGENDADA a COMPLETADO en una sola transacción. Los insumos sin stock suficiente quedan en PENDIENTE_STOCK sin descuento parcial. Con inventario.politica_sin_stock=BLOQUEAR responde 409 salvo que el usuario confirme expresamente el faltante mediante confirmar_stock_insuficiente.
// @Tags Atenciones
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de reserva"
// @Param payload body registrarAtencionRequest true "Datos de atención"
// @Success 201 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse "Error de validación"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 404 {object} utils.APIResponse "Reserva no encontrada"
// @Failure 409 {object} utils.APIResponse "Reserva no atendible o stock insuficiente"
// @Router /bd/reservas/{id}/atencion [post]
func (h *Container) PostAtencion(c *gin.Context) {
	id, e := strconv.Atoi(c.Param("id"))
	if e != nil || id <= 0 {
		utils.RespondError(c, 400, "id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "el usuario no tiene un local asignado")
		return
	}
	var req registrarAtencionRequest
	if e = c.ShouldBindJSON(&req); e != nil {
		utils.RespondError(c, 400, e.Error())
		return
	}
	ins := make([]pgsqlrepo.AtencionInsumoInput, 0, len(req.Insumos))
	for _, x := range req.Insumos {
		ins = append(ins, pgsqlrepo.AtencionInsumoInput{ItemID: x.ItemID, Cantidad: x.Cantidad})
	}
	a, e := h.Atenciones.Registrar(services.RegistrarAtencionInput{ReservaID: id, LocalID: scope.LocalID, EmpleadoID: req.EmpleadoID, EmpleadoNombre: req.EmpleadoNombre, FechaAtencion: req.FechaAtencion, HoraInicio: req.HoraInicio, HoraFin: req.HoraFin, Lugar: req.Lugar, Observaciones: req.Observaciones, ConfirmarStockInsuficiente: req.ConfirmarStockInsuficiente, Actor: consumoActor(c), Insumos: ins})
	if e != nil {
		m := strings.ToLower(e.Error())
		status := http.StatusBadRequest
		evento := "cierre_fallido"
		if strings.Contains(m, "no encontrada") {
			status = 404
			evento = "reserva_no_encontrada"
		} else if strings.Contains(m, "solo se puede") || strings.Contains(m, "stock insuficiente") {
			status = 409
			evento = "conflicto_cierre"
			if strings.Contains(m, "stock insuficiente") {
				evento = "stock_insuficiente_bloqueo"
			}
		} else if strings.Contains(m, "acceso") {
			status = 403
			evento = "acceso_denegado"
		} else if strings.Contains(m, "fallo_consumo_sesion") {
			status = 500
			evento = "fallo_consumo_sesion"
		} else if strings.Contains(m, "fallo_descuento_inventario") {
			status = 500
			evento = "fallo_inventario"
		}
		log.Printf("[atenciones] event=%s reserva_id=%d local_scope=%v status=%d error=%q", evento, id, scope.LocalID, status, e.Error())
		utils.RespondError(c, status, e.Error())
		return
	}
	log.Printf("[atenciones] event=cierre_registrado reserva_id=%d atencion_id=%d local_id=%v insumos=%d", id, a.ID, a.LocalID, len(req.Insumos))
	utils.Respond(c, 201, a)
}

// GetJornadaAtenciones godoc
// @Summary Jornada de atenciones
// @Description Devuelve una fila por reserva del dia con su estado operativo derivado. Incluye reservas AGENDADAS, CANCELADAS y cualquier reserva con una atencion activa. El estado operativo se calcula con la zona horaria configurada. La lista admite cursor; el resumen independiente se consulta en /jornada/resumen para que no cambie al filtrar estado o texto.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param fecha query string false "Fecha (YYYY-MM-DD); por defecto hoy en la zona horaria configurada" example(2026-11-09)
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Param estado query string false "Estado operativo" Enums(PROGRAMADA,PENDIENTE_CIERRE,ATENDIDA,CANCELADA) example(PROGRAMADA)
// @Param q query string false "Busqueda por cliente, reserva, servicio, personal o local"
// @Param limit query int false "Tamano de pagina (1-100)"
// @Param cursor query string false "Cursor opaco de la pagina siguiente"
// @Param include_total query bool false "Incluye total de registros filtrados"
// @Success 200 {object} utils.APIResponse{data=models.JornadaResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: fecha invalida, local_id invalido o estado fuera de la lista"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al consultar la jornada"
// @Router /bd/atenciones/jornada [get]
func (h *Container) GetJornadaAtenciones(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	fecha, err := services.ParseFechaOpcional(c.Query("fecha"), "fecha")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	estado := strings.ToUpper(strings.TrimSpace(c.Query("estado")))
	if estado != "" && !services.EstadoJornadaValido(estado) {
		utils.RespondError(c, http.StatusBadRequest, "estado invalido, use PROGRAMADA, PENDIENTE_CIERRE, ATENDIDA o CANCELADA")
		return
	}
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	filters := jornadaFiltrosCursor{
		Fecha: c.Query("fecha"), LocalID: localID, Estado: estado, Q: strings.TrimSpace(c.Query("q")),
	}
	afterOffset, ok := decodeOffsetCursor(c, page, "atenciones_jornada", filters)
	if !ok {
		return
	}
	res, err := h.Atenciones.Jornada(services.JornadaInput{
		Fecha: fecha, LocalID: localID, Estado: estado, Q: strings.TrimSpace(c.Query("q")),
		PageLimit: page.QueryLimit(), Offset: afterOffset.Offset,
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if includeTotal && page.Enabled {
		total, countErr := h.Atenciones.ContarJornada(services.JornadaInput{Fecha: fecha, LocalID: localID, Estado: estado, Q: strings.TrimSpace(c.Query("q"))})
		if countErr != nil {
			utils.RespondError(c, http.StatusInternalServerError, countErr.Error())
			return
		}
		res.TotalRegistros = &total
	}
	items, metadata, pageErr := pagination.Build(res.Items, page, "atenciones_jornada", filters, func(models.JornadaAtencion) any {
		return pagination.OffsetPosition{Offset: afterOffset.Offset + page.Limit}
	})
	if pageErr != nil {
		utils.RespondError(c, http.StatusInternalServerError, pageErr.Error())
		return
	}
	if res.TotalRegistros != nil {
		pagination.AddTotal(metadata, *res.TotalRegistros)
	}
	res.Items = items
	res.Paginacion = metadata
	utils.Respond(c, http.StatusOK, res)
}

type jornadaFiltrosCursor struct {
	Fecha   string `json:"fecha,omitempty"`
	LocalID *int   `json:"local_id,omitempty"`
	Estado  string `json:"estado,omitempty"`
	Q       string `json:"q,omitempty"`
}

// GetResumenJornadaAtenciones godoc
// @Summary Resumen operativo de Jornada
// @Description Devuelve los contadores completos de una fecha y local. No acepta estado, busqueda ni cursor para que sus cifras sean independientes de la lista visible.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param fecha query string false "Fecha (YYYY-MM-DD); por defecto hoy"
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Success 200 {object} utils.APIResponse{data=models.JornadaResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al consultar el resumen"
// @Router /bd/atenciones/jornada/resumen [get]
func (h *Container) GetResumenJornadaAtenciones(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	fecha, err := services.ParseFechaOpcional(c.Query("fecha"), "fecha")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.Atenciones.ResumenJornada(fecha, localID)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Respond(c, http.StatusOK, res)
}

type calendarioPersonalResponse struct {
	Eventos        []models.CalendarioPersonalEvento `json:"eventos"`
	Paginacion     *pagination.Metadata              `json:"paginacion,omitempty"`
	TotalRegistros *int                              `json:"total_registros,omitempty"`
}

type calendarioConsumoResponse struct {
	Eventos        []models.CalendarioConsumoEvento `json:"eventos"`
	Paginacion     *pagination.Metadata             `json:"paginacion,omitempty"`
	TotalRegistros *int                             `json:"total_registros,omitempty"`
}

type seguimientoResponse struct {
	Items      []models.SeguimientoItem `json:"items"`
	Paginacion *pagination.Metadata     `json:"paginacion,omitempty"`
	Total      *int                     `json:"total_registros,omitempty"`
}

type seguimientoFiltrosCursor struct {
	LocalID    *int   `json:"local_id,omitempty"`
	FechaDesde string `json:"fecha_desde,omitempty"`
	FechaHasta string `json:"fecha_hasta,omitempty"`
	Tipo       string `json:"tipo,omitempty"`
	Q          string `json:"q,omitempty"`
}

// GetSeguimientoAtenciones godoc
// @Summary Bandeja de seguimiento operativo
// @Description Lista casos accionables: deducciones pendientes, inasistencias, cancelaciones y reservas agendadas cuyo horario ya concluyó. No crea estados nuevos de reserva.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param fecha_desde query string false "Desde (YYYY-MM-DD)"
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD)"
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Param tipo query string false "Tipo operativo" Enums(DEDUCCION_PENDIENTE,INASISTENCIA,CANCELACION,PENDIENTE_CIERRE)
// @Param q query string false "Búsqueda por cliente, reserva, servicio, personal o local"
// @Param limit query int false "Tamaño de página (1-100)"
// @Param cursor query string false "Cursor opaco de la página siguiente"
// @Param include_total query bool false "Incluye total filtrado"
// @Success 200 {object} utils.APIResponse{data=seguimientoResponse}
// @Router /bd/atenciones/seguimiento [get]
func (h *Container) GetSeguimientoAtenciones(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	desde, err := services.ParseFechaOpcional(c.Query("fecha_desde"), "fecha_desde")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	hasta, err := services.ParseFechaOpcional(c.Query("fecha_hasta"), "fecha_hasta")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	in := services.SeguimientoInput{LocalID: localID, FechaDesde: desde, FechaHasta: hasta, Tipo: c.Query("tipo"), Q: strings.TrimSpace(c.Query("q"))}
	filtros := seguimientoFiltrosCursor{LocalID: localID, FechaDesde: c.Query("fecha_desde"), FechaHasta: c.Query("fecha_hasta"), Tipo: strings.ToUpper(strings.TrimSpace(in.Tipo)), Q: in.Q}
	after, ok := decodeOffsetCursor(c, page, "atenciones_seguimiento", filtros)
	if !ok {
		return
	}
	in.PageLimit, in.Offset = page.QueryLimit(), after.Offset
	f, err := h.Atenciones.PrepararSeguimiento(in)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.Atenciones.Seguimiento(f)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	var total *int
	if includeTotal && page.Enabled {
		n, countErr := h.Atenciones.ContarSeguimiento(f)
		if countErr != nil {
			utils.RespondError(c, http.StatusInternalServerError, countErr.Error())
			return
		}
		total = &n
	}
	items, metadata, pageErr := pagination.Build(items, page, "atenciones_seguimiento", filtros, func(models.SeguimientoItem) any { return pagination.OffsetPosition{Offset: after.Offset + page.Limit} })
	if pageErr != nil {
		utils.RespondError(c, http.StatusInternalServerError, pageErr.Error())
		return
	}
	if total != nil {
		pagination.AddTotal(metadata, *total)
	}
	utils.Respond(c, http.StatusOK, seguimientoResponse{Items: items, Paginacion: metadata, Total: total})
}

// GetResumenSeguimientoAtenciones godoc
// @Summary Resumen de seguimiento operativo
// @Description Devuelve contadores completos por periodo y local; no acepta tipo, texto ni paginación, para mantenerlos independientes de la lista visible.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param fecha_desde query string false "Desde (YYYY-MM-DD)"
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD)"
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Success 200 {object} utils.APIResponse{data=models.SeguimientoResumen}
// @Router /bd/atenciones/seguimiento/resumen [get]
func (h *Container) GetResumenSeguimientoAtenciones(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	desde, err := services.ParseFechaOpcional(c.Query("fecha_desde"), "fecha_desde")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	hasta, err := services.ParseFechaOpcional(c.Query("fecha_hasta"), "fecha_hasta")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	f, err := h.Atenciones.PrepararSeguimiento(services.SeguimientoInput{LocalID: localID, FechaDesde: desde, FechaHasta: hasta})
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	resumen, err := h.Atenciones.ResumenSeguimiento(f)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Respond(c, http.StatusOK, resumen)
}

type calendarioPersonalFiltrosCursor struct {
	LocalID    *int   `json:"local_id,omitempty"`
	FechaDesde string `json:"fecha_desde,omitempty"`
	FechaHasta string `json:"fecha_hasta,omitempty"`
	Cliente    string `json:"cliente,omitempty"`
	Servicio   string `json:"servicio,omitempty"`
	Empleado   string `json:"empleado,omitempty"`
	Q          string `json:"q,omitempty"`
}

// GetCalendarioPersonalAtenciones godoc
// @Summary Calendario semanal de personal atendiente
// @Description Lista atenciones activas por fecha y hora para priorizar quién atendió. No es agenda de disponibilidad ni reporte de inventario. Los usuarios con local asignado quedan limitados a su local.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param fecha_desde query string false "Desde (YYYY-MM-DD); por defecto hoy"
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD); por defecto fecha_desde"
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Param cliente query string false "Busqueda por cliente"
// @Param servicio query string false "Busqueda por servicio"
// @Param empleado query string false "Busqueda por quien atendió"
// @Param q query string false "Busqueda general"
// @Param limit query int false "Tamano de pagina (1-100)"
// @Param cursor query string false "Cursor opaco de la pagina siguiente"
// @Param include_total query bool false "Incluye total de registros filtrados"
// @Success 200 {object} utils.APIResponse{data=calendarioPersonalResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al consultar el calendario"
// @Router /bd/atenciones/calendario-personal [get]
func (h *Container) GetCalendarioPersonalAtenciones(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	desde, err := services.ParseFechaOpcional(c.Query("fecha_desde"), "fecha_desde")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	hasta, err := services.ParseFechaOpcional(c.Query("fecha_hasta"), "fecha_hasta")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	input := services.CalendarioPersonalInput{
		LocalID: localID, FechaDesde: desde, FechaHasta: hasta,
		Cliente: strings.TrimSpace(c.Query("cliente")), Servicio: strings.TrimSpace(c.Query("servicio")),
		Empleado: strings.TrimSpace(c.Query("empleado")), Q: strings.TrimSpace(c.Query("q")),
	}
	filtros := calendarioPersonalFiltrosCursor{
		LocalID: localID, FechaDesde: c.Query("fecha_desde"), FechaHasta: c.Query("fecha_hasta"),
		Cliente: input.Cliente, Servicio: input.Servicio, Empleado: input.Empleado, Q: input.Q,
	}
	afterOffset, ok := decodeOffsetCursor(c, page, "atenciones_calendario_personal", filtros)
	if !ok {
		return
	}
	input.PageLimit, input.Offset = page.QueryLimit(), afterOffset.Offset
	f, err := h.Atenciones.PrepararCalendarioPersonal(input)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	eventos, err := h.Atenciones.CalendarioPersonal(f)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	var total *int
	if includeTotal && page.Enabled {
		n, countErr := h.Atenciones.ContarCalendarioPersonal(f)
		if countErr != nil {
			utils.RespondError(c, http.StatusInternalServerError, countErr.Error())
			return
		}
		total = &n
	}
	eventos, metadata, pageErr := pagination.Build(eventos, page, "atenciones_calendario_personal", filtros, func(models.CalendarioPersonalEvento) any {
		return pagination.OffsetPosition{Offset: afterOffset.Offset + page.Limit}
	})
	if pageErr != nil {
		utils.RespondError(c, http.StatusInternalServerError, pageErr.Error())
		return
	}
	if total != nil {
		pagination.AddTotal(metadata, *total)
	}
	utils.Respond(c, http.StatusOK, calendarioPersonalResponse{Eventos: eventos, Paginacion: metadata, TotalRegistros: total})
}

// GetCalendarioConsumoAtenciones godoc
// @Summary Calendario semanal de insumos usados
// @Description Lista atenciones activas y sus insumos declarados para priorizar qué se usó en cada atención. No es Kardex ni reporte agregado. Los usuarios con local asignado quedan limitados a su local.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param fecha_desde query string false "Desde (YYYY-MM-DD); por defecto hoy"
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD); por defecto fecha_desde"
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Param item_id query int false "Filtrar por insumo usado"
// @Param cliente query string false "Busqueda por cliente"
// @Param servicio query string false "Busqueda por servicio"
// @Param empleado query string false "Busqueda por quien atendió"
// @Param estado_descuento query string false "Estado de deducción" Enums(DESCONTADO,PENDIENTE)
// @Param q query string false "Busqueda general"
// @Param limit query int false "Tamano de pagina (1-100)"
// @Param cursor query string false "Cursor opaco de la pagina siguiente"
// @Param include_total query bool false "Incluye total de registros filtrados"
// @Success 200 {object} utils.APIResponse{data=calendarioConsumoResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al consultar el calendario"
// @Router /bd/atenciones/calendario-consumo [get]
func (h *Container) GetCalendarioConsumoAtenciones(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	desde, err := services.ParseFechaOpcional(c.Query("fecha_desde"), "fecha_desde")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	hasta, err := services.ParseFechaOpcional(c.Query("fecha_hasta"), "fecha_hasta")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	itemID, err := queryInt(c, "item_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	input := services.ConsumosInput{
		LocalID: localID, FechaDesde: desde, FechaHasta: hasta, ItemID: itemID,
		Cliente: c.Query("cliente"), Servicio: c.Query("servicio"), Empleado: c.Query("empleado"),
		EstadoDescuento: c.Query("estado_descuento"), Q: c.Query("q"),
	}
	filtros := consumosFiltrosCursor{
		LocalID: localID, FechaDesde: c.Query("fecha_desde"), FechaHasta: c.Query("fecha_hasta"), ItemID: itemID,
		Cliente: strings.TrimSpace(input.Cliente), Servicio: strings.TrimSpace(input.Servicio), Empleado: strings.TrimSpace(input.Empleado),
		EstadoDescuento: strings.ToUpper(strings.TrimSpace(input.EstadoDescuento)), Q: strings.TrimSpace(input.Q),
	}
	afterOffset, ok := decodeOffsetCursor(c, page, "atenciones_calendario_consumo", filtros)
	if !ok {
		return
	}
	input.PageLimit, input.Offset = page.QueryLimit(), afterOffset.Offset
	f, err := h.Inventario.PrepararConsumos(input)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	consumos, err := h.Inventario.ListarConsumos(f)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	var total *int
	if includeTotal && page.Enabled {
		n, countErr := h.Inventario.ContarConsumos(f)
		if countErr != nil {
			respondTrastiendaError(c, countErr)
			return
		}
		total = &n
	}
	eventos := make([]models.CalendarioConsumoEvento, 0, len(consumos))
	for _, consumo := range consumos {
		var inicio, fin *string
		if consumo.HoraInicio != "" {
			inicio = &consumo.HoraInicio
		}
		if consumo.HoraFin != nil && *consumo.HoraFin != "" {
			fin = consumo.HoraFin
		}
		eventos = append(eventos, models.CalendarioConsumoEvento{
			AtencionID: consumo.AtencionID, ReservaID: consumo.ReservaID, Fecha: consumo.Fecha,
			HoraInicio: inicio, HoraFin: fin, Cliente: consumo.Cliente, Servicio: consumo.Servicio,
			LocalID: consumo.LocalID, LocalNombre: consumo.LocalNombre, EmpleadoNombre: consumo.EmpleadoNombre,
			Insumos: consumo.Insumos, InsumosRegistrados: consumo.InsumosRegistrados, InsumosPendientes: consumo.InsumosPendientes,
		})
	}
	eventos, metadata, pageErr := pagination.Build(eventos, page, "atenciones_calendario_consumo", filtros, func(models.CalendarioConsumoEvento) any {
		return pagination.OffsetPosition{Offset: afterOffset.Offset + page.Limit}
	})
	if pageErr != nil {
		utils.RespondError(c, http.StatusInternalServerError, pageErr.Error())
		return
	}
	if total != nil {
		pagination.AddTotal(metadata, *total)
	}
	utils.Respond(c, http.StatusOK, calendarioConsumoResponse{Eventos: eventos, Paginacion: metadata, TotalRegistros: total})
}

type atencionesResponse struct {
	// Atenciones activas que cumplen los filtros
	Atenciones []models.Atencion `json:"atenciones"`
	// Paginacion por cursor (solo si se envia limit o cursor)
	Paginacion *pagination.Metadata `json:"paginacion,omitempty"`
	// Total de atenciones con los filtros (solo con include_total=true)
	TotalRegistros *int `json:"total_registros,omitempty"`
}

// atencionesFiltrosCursor ata el cursor a los filtros: cambiar un filtro invalida el cursor.
type atencionesFiltrosCursor struct {
	LocalID         *int   `json:"local_id,omitempty"`
	FechaDesde      string `json:"fecha_desde,omitempty"`
	FechaHasta      string `json:"fecha_hasta,omitempty"`
	Cliente         string `json:"cliente,omitempty"`
	Servicio        string `json:"servicio,omitempty"`
	Empleado        string `json:"empleado,omitempty"`
	EmpleadoID      *int   `json:"empleado_id,omitempty"`
	ItemID          *int   `json:"item_id,omitempty"`
	PendientesStock *bool  `json:"pendientes_stock,omitempty"`
	Resultado       string `json:"resultado,omitempty"`
	Q               string `json:"q,omitempty"`
	// Orden pedido; vacio con el orden por defecto. Va en el cursor para rechazar uno generado con otro orden.
	OrdenarPor string `json:"ordenar_por,omitempty"`
	Sentido    string `json:"sentido,omitempty"`
}

type atencionCursor struct {
	Fecha time.Time `json:"f"`
	ID    int       `json:"i"`
}

// GetAtenciones godoc
// @Summary Listar historial de atenciones
// @Description Lista atenciones activas y reservas canceladas con filtros de servidor. Filtros: fecha_desde y fecha_hasta (YYYY-MM-DD, inclusivos), local_id, resultado, cliente, servicio, empleado (texto), empleado_id, item_id, pendientes_stock y q (cliente, reserva, servicio, personal, lugar u observaciones). Los usuarios con local asignado quedan limitados a ese local. Paginacion por cursor (fecha_atencion descendente) u orden por columna con ordenar_por/sentido.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param fecha_desde query string false "Desde (YYYY-MM-DD)"
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD)"
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Param resultado query string false "Resultado operativo" Enums(ATENDIDA,CANCELADA,INASISTENCIA)
// @Param cliente query string false "Busqueda por cliente"
// @Param servicio query string false "Busqueda por servicio"
// @Param empleado query string false "Busqueda por nombre de personal"
// @Param empleado_id query int false "Filtrar por personal"
// @Param item_id query int false "Filtrar por insumo utilizado"
// @Param pendientes_stock query bool false "Solo atenciones con deducciones pendientes (true) o sin ellas (false)"
// @Param q query string false "Busqueda general por cliente, reserva, servicio, personal, lugar u observaciones"
// @Param ordenar_por query string false "Campo por el que ordenar; sin el, fecha_atencion descendente" Enums(fecha_atencion,hora_inicio,cliente_nombre,servicio_reserva,empleado_nombre,local_nombre,reserva_id,insumos_pendientes,resultado) example(cliente_nombre)
// @Param sentido query string false "Sentido del orden; solo con ordenar_por, por defecto asc" Enums(asc,desc) example(desc)
// @Param limit query int false "Tamano de pagina (1-100); sin limit ni cursor devuelve 50 registros"
// @Param cursor query string false "Cursor opaco devuelto en paginacion.next_cursor; solo vale con los mismos filtros y orden"
// @Param include_total query bool false "Incluye total_registros y total_paginas"
// @Success 200 {object} utils.APIResponse{data=atencionesResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: fecha invalida, parametro invalido, ordenar_por fuera de la lista, sentido distinto de asc o desc o cursor de otros filtros"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al listar atenciones"
// @Router /bd/atenciones [get]
func (h *Container) GetAtenciones(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	sort, ok := parseSort(c, pgsqlrepo.AtencionesOrdenables)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	desde, err := services.ParseFechaOpcional(c.Query("fecha_desde"), "fecha_desde")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	hasta, err := services.ParseFechaOpcional(c.Query("fecha_hasta"), "fecha_hasta")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	empleadoID, err := queryInt(c, "empleado_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	itemID, err := queryInt(c, "item_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	pendientesStock, err := queryBool(c, "pendientes_stock")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	resultado := strings.ToUpper(strings.TrimSpace(c.Query("resultado")))
	if resultado != "" && resultado != "ATENDIDA" && resultado != "CANCELADA" && resultado != "INASISTENCIA" {
		utils.RespondError(c, http.StatusBadRequest, "resultado invalido, use ATENDIDA, CANCELADA o INASISTENCIA")
		return
	}
	f := repository.FiltroAtenciones{
		LocalID: localID, FechaDesde: desde, FechaHasta: hasta,
		Cliente: strings.TrimSpace(c.Query("cliente")), Servicio: strings.TrimSpace(c.Query("servicio")),
		Empleado: strings.TrimSpace(c.Query("empleado")), EmpleadoID: empleadoID, ItemID: itemID,
		PendientesStock: pendientesStock, Resultado: resultado, Q: strings.TrimSpace(c.Query("q")),
	}
	filters := atencionesFiltrosCursor{
		LocalID: localID, FechaDesde: c.Query("fecha_desde"), FechaHasta: c.Query("fecha_hasta"),
		Cliente: f.Cliente, Servicio: f.Servicio, Empleado: f.Empleado, EmpleadoID: empleadoID,
		ItemID: itemID, PendientesStock: pendientesStock, Resultado: resultado, Q: f.Q,
	}
	var after atencionCursor
	var afterOffset pagination.OffsetPosition
	if sort.Active() {
		// Con orden pedido el cursor cuenta filas entregadas; el orden va en los filtros del cursor.
		filters.OrdenarPor, filters.Sentido = sort.Key, sort.Direction()
		f.OrdenarPor, f.OrdenDesc = sort.Key, sort.Desc
		if afterOffset, ok = decodeOffsetCursor(c, page, "atenciones", filters); !ok {
			return
		}
		f.Offset = afterOffset.Offset
	} else {
		if !decodePaginationCursor(c, page, "atenciones", filters, &after) {
			return
		}
		if page.Cursor != "" && after.ID < 1 {
			utils.RespondError(c, http.StatusBadRequest, "paginacion invalida: cursor incompleto")
			return
		}
		f.CursorSet = page.Cursor != ""
		f.CursorFecha = after.Fecha
		f.CursorID = after.ID
	}
	f.PageLimit = page.QueryLimit()

	xs, e := h.Atenciones.Listar(f)
	if e != nil {
		utils.RespondError(c, http.StatusInternalServerError, e.Error())
		return
	}
	var total *int
	if includeTotal && page.Enabled {
		n, e := h.Atenciones.Contar(f)
		if e != nil {
			utils.RespondError(c, http.StatusInternalServerError, e.Error())
			return
		}
		total = &n
	}
	xs, metadata, e := pagination.Build(xs, page, "atenciones", filters, func(a models.Atencion) any {
		if sort.Active() {
			return pagination.OffsetPosition{Offset: afterOffset.Offset + page.Limit}
		}
		return atencionCursor{Fecha: a.FechaAtencion, ID: a.ID}
	})
	if e != nil {
		utils.RespondError(c, http.StatusInternalServerError, e.Error())
		return
	}
	if total != nil {
		pagination.AddTotal(metadata, *total)
	}
	utils.Respond(c, http.StatusOK, atencionesResponse{Atenciones: xs, Paginacion: metadata, TotalRegistros: total})
}

// PutAtencion actualiza una atencion mientras tenga insumos pendientes de deduccion.
func (h *Container) PutAtencion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, 400, "id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	var req registrarAtencionRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, 400, err.Error())
		return
	}
	ins := make([]pgsqlrepo.AtencionInsumoInput, 0, len(req.Insumos))
	for _, item := range req.Insumos {
		ins = append(ins, pgsqlrepo.AtencionInsumoInput{ItemID: item.ItemID, Cantidad: item.Cantidad})
	}
	a, err := h.Atenciones.Actualizar(services.ActualizarAtencionInput{
		AtencionID: id, LocalID: scope.LocalID, EmpleadoID: req.EmpleadoID,
		EmpleadoNombre: req.EmpleadoNombre, FechaAtencion: req.FechaAtencion,
		HoraInicio: req.HoraInicio, HoraFin: req.HoraFin, Lugar: req.Lugar,
		Observaciones: req.Observaciones, ConfirmarStockInsuficiente: req.ConfirmarStockInsuficiente,
		Actor: consumoActor(c), Insumos: ins,
	})
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "no encontrada") {
			status = http.StatusNotFound
		} else if strings.Contains(strings.ToLower(err.Error()), "solo se pueden editar") || strings.Contains(strings.ToLower(err.Error()), "stock insuficiente") {
			status = http.StatusConflict
		} else if strings.Contains(strings.ToLower(err.Error()), "acceso") {
			status = http.StatusForbidden
		}
		utils.RespondError(c, status, err.Error())
		return
	}
	utils.Respond(c, 200, a)
}

type desactivarAtencionRequest struct {
	RestituirStock bool `json:"restituir_stock"`
}

func (h *Container) PostDesactivarAtencion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, 400, "id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	var req desactivarAtencionRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, 400, err.Error())
		return
	}
	if err = h.Atenciones.Desactivar(id, scope.LocalID, req.RestituirStock, consumoActor(c)); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "no encontrada") {
			status = http.StatusNotFound
		} else if strings.Contains(strings.ToLower(err.Error()), "acceso") {
			status = http.StatusForbidden
		}
		utils.RespondError(c, status, err.Error())
		return
	}
	utils.Respond(c, 200, gin.H{"id": id, "activo": false, "stock_restituido": req.RestituirStock})
}

// GetAtencion godoc
// @Summary Obtener atención
// @Description Devuelve una atención con sus insumos registrados.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de atención"
// @Success 200 {object} utils.APIResponse
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 404 {object} utils.APIResponse "Atención no encontrada"
// @Router /bd/atenciones/{id} [get]
func (h *Container) GetAtencion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, 400, "id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	a, err := h.Atenciones.Obtener(id)
	if err != nil {
		utils.RespondError(c, 404, "atencion no encontrada")
		return
	}
	if scope.LocalID != nil && (a.LocalID == nil || *a.LocalID != *scope.LocalID) {
		utils.RespondError(c, 403, "no tienes acceso a ese local")
		return
	}
	utils.Respond(c, 200, a)
}

// GetPersonalSugerencias godoc
// @Summary Sugerir personal
// @Description Busca nombres coincidentes entre los empleados activos accesibles en el local.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param q query string true "Texto de búsqueda (mínimo 2 caracteres)"
// @Success 200 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al buscar personal"
// @Router /bd/personal/sugerencias [get]
func (h *Container) GetPersonalSugerencias(c *gin.Context) {
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	xs, e := h.Atenciones.SugerenciasPersonal(c.Query("q"), scope.LocalID)
	if e != nil {
		utils.RespondError(c, 500, e.Error())
		return
	}
	utils.Respond(c, 200, gin.H{"personal": xs})
}

// @Summary Listar insumos sugeridos de una reserva
// @Description Devuelve los insumos configurados para el servicio de la reserva.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param reserva_id query int true "ID de reserva"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse "reserva_id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al consultar insumos"
// @Router /bd/atenciones/insumos-sugeridos [get]
func (h *Container) GetInsumosSugeridos(c *gin.Context) {
	id, e := strconv.Atoi(c.Query("reserva_id"))
	if e != nil || id <= 0 {
		utils.RespondError(c, 400, "reserva_id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	xs, e := h.Atenciones.InsumosSugeridos(id, scope.LocalID)
	if e != nil {
		utils.RespondError(c, 500, e.Error())
		return
	}
	utils.Respond(c, 200, gin.H{"insumos": xs})
}
