package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/pagination"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

var telefonoRegex = regexp.MustCompile(`^\+?\d+$`)

const allowEstadoOverrideTemporal = true

// GetReservasPG godoc
// @Summary Listar reservas desde base de datos
// @Description Devuelve reservas agrupadas por local con filtros opcionales. Filtros: local (opcional), fecha YYYY-MM-DD (opcional), fecha_desde/fecha_hasta rango (opcional), cliente (opcional), numero_telefono (opcional), servicio_solicitado busqueda parcial (opcional), servicio_confirmado busqueda parcial (opcional), estado PENDIENTE/RECHAZADO/AGENDADO/COMPLETADO (opcional), tipo mesa/bicicleta (opcional), reservados true/false (opcional). Response: total_locales (int), filtros (objeto con los filtros aplicados), reservas ([]LocalReservas cada uno con: local string, semanas []Semana con titulo y slots []ReservaSlot con hora y map dia->[]ReservaItem con tipo M/B, cliente, servicio, servicio_solicitado, servicio_confirmado, estado, numero_telefono, notificado, creado_en, actualizado_en).
// @Tags Reservas BD
// @Produce json
// @Param local query string false "Nombre del local" example(SAN MARTIN)
// @Param fecha query string false "Fecha exacta YYYY-MM-DD" example(2026-05-23)
// @Param fecha_desde query string false "Fecha inicio rango YYYY-MM-DD" example(2026-05-19)
// @Param fecha_hasta query string false "Fecha fin rango YYYY-MM-DD" example(2026-05-24)
// @Param cliente query string false "Nombre del cliente" example(Maria Lopez)
// @Param numero_telefono query string false "Numero de telefono; acepta 70011223, 59170011223 o +591 700-11223" example(+59170011223)
// @Param servicio_solicitado query string false "Busqueda parcial por servicio solicitado" example(depilacion)
// @Param servicio_confirmado query string false "Busqueda parcial por servicio confirmado" example(depilacion laser piernas)
// @Param estado query string false "Estado de la reserva" Enums(PENDIENTE,RECHAZADO,AGENDADO,COMPLETADO) example(AGENDADO)
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta) example(mesa)
// @Param reservados query bool false "Filtrar por estado reservado" example(true)
// @Success 200 {object} utils.APIResponse{data=reservaCalendarioResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: tipo, estado, orden, vigencia, paginacion o cursor invalido"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/reservas/calendario [get]
func (h *Container) GetReservasPG(c *gin.Context) {
	paramTipo := strings.ToLower(strings.TrimSpace(c.Query("tipo")))
	if paramTipo != "" && paramTipo != "mesa" && paramTipo != "bicicleta" {
		utils.RespondError(c, http.StatusBadRequest,
			"tipo invÃ¡lido, valores permitidos: mesa, bicicleta")
		return
	}

	var reservados *bool
	if raw := strings.TrimSpace(c.Query("reservados")); raw != "" {
		v := strings.ToLower(raw) == "true"
		reservados = &v
	}

	filtro := services.FiltroReservasPG{
		Context:            c.Request.Context(),
		Local:              strings.TrimSpace(c.Query("local")),
		Fecha:              strings.TrimSpace(c.Query("fecha")),
		FechaDesde:         strings.TrimSpace(c.Query("fecha_desde")),
		FechaHasta:         strings.TrimSpace(c.Query("fecha_hasta")),
		Cliente:            strings.TrimSpace(c.Query("cliente")),
		NumeroTelefono:     strings.TrimSpace(c.Query("numero_telefono")),
		ServicioSolicitado: strings.TrimSpace(c.Query("servicio_solicitado")),
		ServicioConfirmado: strings.TrimSpace(c.Query("servicio_confirmado")),
		Estado:             strings.TrimSpace(c.Query("estado")),
		Tipo:               paramTipo,
		Reservados:         reservados,
	}
	if filtro.Estado != "" {
		estado, err := services.NormalizarEstadoReserva(filtro.Estado)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
		filtro.Estado = estado
	}
	resultado, err := h.ReservasPG.GetReservasFiltradas(filtro)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if resultado == nil {
		resultado = []models.LocalReservas{}
	}

	utils.Respond(c, http.StatusOK, reservaCalendarioResponse{
		TotalLocales: len(resultado),
		Filtros: reservaPGFiltrosResponse{
			Local:              filtro.Local,
			Fecha:              filtro.Fecha,
			FechaDesde:         filtro.FechaDesde,
			FechaHasta:         filtro.FechaHasta,
			Tipo:               filtro.Tipo,
			Cliente:            filtro.Cliente,
			NumeroTelefono:     filtro.NumeroTelefono,
			ServicioSolicitado: filtro.ServicioSolicitado,
			ServicioConfirmado: filtro.ServicioConfirmado,
			Estado:             filtro.Estado,
			Reservados:         reservados,
		},
		Reservas: resultado,
	})
}

// POST /bd/reservas
type crearReservaPGRequest struct {
	// Nombre del local
	Local string `json:"local" binding:"required" example:"SAN MARTIN"`
	// Fecha de la reserva (YYYY-MM-DD)
	Fecha string `json:"fecha" binding:"required" example:"2026-05-23"`
	// Hora de inicio (HH:MM)
	HoraDesde string `json:"hora_desde" binding:"required" example:"15:00"`
	// Hora de fin (HH:MM)
	HoraHasta string `json:"hora_hasta" example:"16:00"`
	// Tipo de espacio: M (mesa) o B (bicicleta)
	Tipo string `json:"tipo" example:"M"`
	// Nombre del cliente
	Cliente string `json:"cliente" binding:"required" example:"Maria Lopez"`
	// Numero de telefono del cliente
	NumeroTelefono string `json:"numero_telefono" binding:"required" example:"+59170011223"`
	// Forma internacional E.164 opcional; no reemplaza el campo legacy.
	TelefonoE164 *string `json:"telefono_e164,omitempty" example:"+59170011223"`
	// Estado inicial: PENDIENTE (default), AGENDADO
	Estado string `json:"estado" example:"PENDIENTE"`
	// ID del servicio seleccionado en BD. Si se envia, se usa para validar si requiere evaluacion.
	ServicioID *int `json:"servicio_id" example:"8"`
	// Nombre del servicio principal (ej: "Depilacion laser"). Se usa como identificador general.
	Servicio string `json:"servicio" example:"Depilacion laser"`
	// Detalle de lo que solicito el cliente (ej: "Piernas completas"). Si se omite, se copia de "servicio".
	ServicioSolicitado string `json:"servicio_solicitado" example:"Piernas completas"`
	// Servicio final confirmado tras evaluacion (ej: "Depilacion Laser Piernas"). Si se omite y el servicio no requiere evaluacion, se autocompleta con el nombre del servicio encontrado en BD.
	ServicioConfirmado *string `json:"servicio_confirmado" example:"Depilacion Laser Piernas"`
	// Precio de la reserva
	Precio *float64 `json:"precio" example:"350"`
	// Notas u observaciones
	Notas string `json:"notas" example:"Primera sesion del plan"`
	// ID del plan asociado (opcional)
	PlanID *int `json:"plan_id" example:"21"`
}

// PostReservaPG godoc
// @Summary Crear reserva en base de datos
// @Description Crea una reserva en PostgreSQL. local: nombre del local (requerido). fecha: YYYY-MM-DD (requerido, no acepta domingos). hora_desde: HH:MM (requerido). hora_hasta: HH:MM (opcional). Horarios: lunes a viernes 08:00-20:00; sabado SAN MARTIN 08:00-15:00 y PASEO ARANJUEZ 08:00-18:00. tipo: M=mesa o B=bicicleta (opcional). cliente: nombre del cliente (requerido). numero_telefono: telefono del cliente (requerido). estado: PENDIENTE por defecto, AGENDADO (opcional). servicio_id: ID del servicio en BD, recomendado para aplicar requiere_evaluacion sin depender del nombre (opcional). servicio: nombre del servicio principal (opcional). servicio_solicitado: detalle solicitado, se copia de servicio si se omite (opcional). servicio_confirmado: servicio final tras evaluacion, se autocompleta si no requiere evaluacion (opcional). precio: precio de la reserva (opcional). notas: observaciones (opcional). plan_id: ID del plan asociado (opcional).
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Param payload body crearReservaPGRequest true "Datos de la reserva"
// @Success 201 {object} utils.APIResponse{data=reservaCreatedResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: campo requerido faltante, estado invalido, servicio requiere evaluacion, formato incorrecto u horario fuera de atencion"
// @Failure 409 {object} utils.APIResponse "Conflicto: no hay espacios disponibles o el horario no esta libre"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/reservas [post]
func (h *Container) PostReservaPG(c *gin.Context) {
	var req crearReservaPGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	tipoNorm := strings.ToUpper(strings.TrimSpace(req.Tipo))
	telefono, err := normalizarTelefono(req.NumeroTelefono)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	estadoFinal := "PENDIENTE"
	if allowEstadoOverrideTemporal && strings.TrimSpace(req.Estado) != "" {
		estadoFinal, err = services.NormalizarEstadoReserva(req.Estado)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
	}

	id, err := h.ReservasPG.CrearReserva(services.CrearReservaPGInput{
		Local:     strings.TrimSpace(req.Local),
		Fecha:     strings.TrimSpace(req.Fecha),
		HoraDesde: strings.TrimSpace(req.HoraDesde),
		HoraHasta: strings.TrimSpace(req.HoraHasta),
		Tipo:      tipoNorm,
		Cliente:   strings.TrimSpace(req.Cliente),
		Telefono:  telefono, TelefonoE164: req.TelefonoE164,
		Estado:             estadoFinal,
		ServicioID:         req.ServicioID,
		Servicio:           strings.TrimSpace(req.Servicio),
		ServicioSolicitado: strings.TrimSpace(req.ServicioSolicitado),
		ServicioConfirmado: req.ServicioConfirmado,
		Precio:             req.Precio,
		Notas:              strings.TrimSpace(req.Notas),
		PlanID:             req.PlanID,
	})

	if err != nil {
		status := http.StatusInternalServerError
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "horario fuera de atenci") ||
			strings.Contains(errLower, "telefono_e164") ||
			strings.Contains(errLower, "hora_desde") ||
			strings.Contains(errLower, "hora_hasta") ||
			strings.Contains(errLower, "formato de fecha") ||
			strings.Contains(errLower, "estado inicial") ||
			strings.Contains(errLower, "requieren evaluación") ||
			strings.Contains(errLower, "requieren evaluaci") ||
			strings.Contains(errLower, "requiere evaluación") ||
			strings.Contains(errLower, "requiere evaluaci") {
			status = http.StatusBadRequest
		} else if strings.Contains(errLower, "no hay espacios") ||
			strings.Contains(errLower, "no hay ambientes") ||
			strings.Contains(errLower, "no está disponible") ||
			strings.Contains(errLower, "no estÃ¡ disponible") {
			status = http.StatusConflict
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusCreated, reservaCreatedResponse{
		ID:      id,
		Mensaje: "Reserva creada correctamente",
	})
}

type actualizarEstadoReservaPGRequest struct {
	// ID de la reserva
	Id int `json:"id" binding:"required" example:"44"`
	// Nuevo estado: PENDIENTE, AGENDADO, RECHAZADO o COMPLETADO
	Estado string `json:"estado" binding:"required" example:"AGENDADO"`
	// Motivo del cambio de estado
	Causa string `json:"causa" example:"Cliente confirmo por WhatsApp"`
	// Servicio confirmado (opcional, se actualiza si se envia)
	ServicioConfirmado *string `json:"servicio_confirmado" example:"Depilacion Laser Piernas"`
	// Precio actualizado (opcional)
	Precio *float64 `json:"precio" example:"350"`
	// Tipo de espacio: M (mesa) o B (bicicleta)
	Tipo string `json:"tipo" example:"M"`
}

type actualizarNotificadoReservaPGRequest struct {
	// ID de la reserva
	Id int `json:"id" binding:"required" example:"44"`
	// Estado de notificacion: true = notificado, false = no notificado
	Notificado *bool `json:"notificado" binding:"required" example:"true"`
}

type marcarNotificacionesReservasLeidasRequest struct {
	// IDs de reservas a marcar como leidas
	Ids []int `json:"ids" binding:"required" example:"44,45,46"`
}

type reservaResumenSemanaResponse struct {
	// Total de reservas en la semana (lunes a la fecha consultada)
	TotalReservas int `json:"total_reservas" example:"45"`
	// Reservas del dia lunes (incluido si la fecha es lunes o posterior)
	Lunes *int `json:"lunes,omitempty" example:"8"`
	// Reservas del dia martes (incluido si la fecha es martes o posterior)
	Martes *int `json:"martes,omitempty" example:"10"`
	// Reservas del dia miercoles (incluido si la fecha es miercoles o posterior)
	Miercoles *int `json:"miercoles,omitempty" example:"12"`
	// Reservas del dia jueves (incluido si la fecha es jueves o posterior)
	Jueves *int `json:"jueves,omitempty" example:"5"`
	// Reservas del dia viernes (incluido si la fecha es viernes o posterior)
	Viernes *int `json:"viernes,omitempty" example:"7"`
	// Reservas del dia sabado (incluido si la fecha es sabado o posterior)
	Sabado *int `json:"sabado,omitempty" example:"3"`
}

type reservaResumenIngresosSemanaResponse struct {
	// Total de ingresos en la semana (lunes a la fecha consultada)
	TotalIngresos float64 `json:"total_ingresos" example:"8450.75"`
	// Ingresos del dia lunes (incluido si la fecha es lunes o posterior)
	Lunes *float64 `json:"lunes,omitempty" example:"1250.5"`
	// Ingresos del dia martes (incluido si la fecha es martes o posterior)
	Martes *float64 `json:"martes,omitempty" example:"980"`
	// Ingresos del dia miercoles (incluido si la fecha es miercoles o posterior)
	Miercoles *float64 `json:"miercoles,omitempty" example:"1420.25"`
	// Ingresos del dia jueves (incluido si la fecha es jueves o posterior)
	Jueves *float64 `json:"jueves,omitempty" example:"2100"`
	// Ingresos del dia viernes (incluido si la fecha es viernes o posterior)
	Viernes *float64 `json:"viernes,omitempty" example:"1850"`
	// Ingresos del dia sabado (incluido si la fecha es sabado o posterior)
	Sabado *float64 `json:"sabado,omitempty" example:"850"`
}

type reservaResumenResponse struct {
	// Cantidad de reservas agendadas para el dia consultado
	ReservasAgendadasDia int `json:"reservas_agendadas_dia" example:"15"`
	// Cantidad de servicios completados en el dia consultado
	ServiciosCompletadosDia int `json:"servicios_completados_dia" example:"10"`
	// Ingresos de pagos activos y PAGADOS en el dia consultado
	IngresosHoy float64 `json:"ingresos_hoy" example:"1250.5"`
	// Cantidad de pagos activos y PAGADOS en el dia consultado
	CancelacionesHoy int `json:"cancelaciones_hoy" example:"6"`
	// Ingresos de pagos activos y PAGADOS desde el lunes hasta la fecha consultada
	IngresosSemana float64 `json:"ingresos_semana" example:"8450.75"`
	// Resumen por dia de la semana desde el lunes hasta la fecha consultada
	Semana reservaResumenSemanaResponse `json:"semana"`
	// Desglose de ingresos por dia desde el lunes hasta la fecha consultada
	Ingresos reservaResumenIngresosSemanaResponse `json:"ingresos"`
}

func normalizarTelefono(raw string) (string, error) {
	telefono := strings.TrimSpace(raw)
	telefono = strings.ReplaceAll(telefono, " ", "")
	telefono = strings.ReplaceAll(telefono, "-", "")

	if telefono == "" {
		return "", errors.New("numero_telefono es requerido")
	}
	if len(telefono) > 20 {
		return "", errors.New("numero_telefono no puede exceder 20 caracteres")
	}
	if !telefonoRegex.MatchString(telefono) {
		return "", errors.New("numero_telefono solo puede contener digitos y un '+' inicial")
	}

	digitos := telefono
	if strings.HasPrefix(digitos, "+") {
		digitos = digitos[1:]
	}
	if len(digitos) < 7 {
		return "", errors.New("numero_telefono debe tener al menos 7 digitos")
	}

	return telefono, nil
}

// GetReservasResumenPG godoc
// @Summary Obtener resumen numerico de reservas
// @Description Devuelve resumen de reservas del dia y pagos del periodo. Requiere token Bearer. Los usuarios con local asignado solo consultan su local desde el token; si el token no tiene local, puede filtrar por el query local o consultar todos si no lo envia. Param: fecha YYYY-MM-DD (requerido, query). Si fecha es domingo, calcula el resumen con el sabado anterior para devolver la semana que finaliza. Los pagos consideran registros activos con estado PAGADO en la tabla pagos, usando el mismo filtro de local y rango lunes-fecha. Response: reservas_agendadas_dia (int), servicios_completados_dia (int), ingresos_hoy (number), cancelaciones_hoy (int), ingresos_semana (number), semana (reservaResumenSemanaResponse con: total_reservas int, lunes..sabado int opcionales segun el dia efectivo), ingresos (reservaResumenIngresosSemanaResponse con: total_ingresos number, lunes..sabado number opcionales segun el dia efectivo).
// @Tags Reservas BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param fecha query string true "Fecha a consultar YYYY-MM-DD; si es domingo se usa el sabado anterior" example(2026-05-24)
// @Param local query string false "Nombre exacto del local a consultar cuando el token no tiene local; si se omite, consulta todos los locales" example(SAN MARTIN)
// @Success 200 {object} utils.APIResponse{data=reservaResumenResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: fecha requerida o formato invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/reservas/resumen [get]
func (h *Container) GetReservasResumenPG(c *gin.Context) {
	fechaRaw := strings.TrimSpace(c.Query("fecha"))
	if fechaRaw == "" {
		utils.RespondError(c, http.StatusBadRequest, "fecha es requerida")
		return
	}

	fecha, err := time.Parse("2006-01-02", fechaRaw)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "formato de fecha invalido, use YYYY-MM-DD")
		return
	}

	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, http.StatusForbidden, services.ErrNoAutorizado.Error())
		return
	}

	localNombre := scope.NombreLocal
	localID := scope.LocalID
	if strings.TrimSpace(localNombre) == "" {
		localNombre = queryLocalNombre(c)
		localID = nil
	}

	resumen, err := h.ReservasPG.GetResumenReservasContext(c.Request.Context(), fecha, localNombre, localID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "local no encontrado") {
			utils.RespondError(c, http.StatusNotFound, err.Error())
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, reservaResumenResponse{
		ReservasAgendadasDia:    resumen.ReservasAgendadasDia,
		ServiciosCompletadosDia: resumen.ServiciosCompletadosDia,
		IngresosHoy:             resumen.IngresosDia,
		CancelacionesHoy:        resumen.CancelacionesDia,
		IngresosSemana:          resumen.IngresosSemana,
		Semana:                  buildReservaResumenSemanaResponse(fecha, resumen.Semana),
		Ingresos:                buildReservaResumenIngresosSemanaResponse(fecha, resumen.Ingresos),
	})
}

func queryLocalNombre(c *gin.Context) string {
	return strings.TrimSpace(c.Query("local"))
}

func buildReservaResumenSemanaResponse(fecha time.Time, semana services.ResumenReservasSemana) reservaResumenSemanaResponse {
	if fecha.Weekday() == time.Sunday {
		fecha = fecha.AddDate(0, 0, -1)
	}

	resp := reservaResumenSemanaResponse{
		TotalReservas: semana.TotalReservas,
		Lunes:         intPtr(semana.Lunes),
	}

	switch fecha.Weekday() {
	case time.Monday:
		return resp
	case time.Tuesday:
		resp.Martes = intPtr(semana.Martes)
	case time.Wednesday:
		resp.Martes = intPtr(semana.Martes)
		resp.Miercoles = intPtr(semana.Miercoles)
	case time.Thursday:
		resp.Martes = intPtr(semana.Martes)
		resp.Miercoles = intPtr(semana.Miercoles)
		resp.Jueves = intPtr(semana.Jueves)
	case time.Friday:
		resp.Martes = intPtr(semana.Martes)
		resp.Miercoles = intPtr(semana.Miercoles)
		resp.Jueves = intPtr(semana.Jueves)
		resp.Viernes = intPtr(semana.Viernes)
	case time.Saturday:
		resp.Martes = intPtr(semana.Martes)
		resp.Miercoles = intPtr(semana.Miercoles)
		resp.Jueves = intPtr(semana.Jueves)
		resp.Viernes = intPtr(semana.Viernes)
		resp.Sabado = intPtr(semana.Sabado)
	}

	return resp
}

func intPtr(v int) *int {
	return &v
}

func buildReservaResumenIngresosSemanaResponse(fecha time.Time, ingresos services.ResumenIngresosSemana) reservaResumenIngresosSemanaResponse {
	if fecha.Weekday() == time.Sunday {
		fecha = fecha.AddDate(0, 0, -1)
	}

	resp := reservaResumenIngresosSemanaResponse{
		TotalIngresos: ingresos.TotalIngresos,
		Lunes:         float64Ptr(ingresos.Lunes),
	}

	switch fecha.Weekday() {
	case time.Monday:
		return resp
	case time.Tuesday:
		resp.Martes = float64Ptr(ingresos.Martes)
	case time.Wednesday:
		resp.Martes = float64Ptr(ingresos.Martes)
		resp.Miercoles = float64Ptr(ingresos.Miercoles)
	case time.Thursday:
		resp.Martes = float64Ptr(ingresos.Martes)
		resp.Miercoles = float64Ptr(ingresos.Miercoles)
		resp.Jueves = float64Ptr(ingresos.Jueves)
	case time.Friday:
		resp.Martes = float64Ptr(ingresos.Martes)
		resp.Miercoles = float64Ptr(ingresos.Miercoles)
		resp.Jueves = float64Ptr(ingresos.Jueves)
		resp.Viernes = float64Ptr(ingresos.Viernes)
	case time.Saturday:
		resp.Martes = float64Ptr(ingresos.Martes)
		resp.Miercoles = float64Ptr(ingresos.Miercoles)
		resp.Jueves = float64Ptr(ingresos.Jueves)
		resp.Viernes = float64Ptr(ingresos.Viernes)
		resp.Sabado = float64Ptr(ingresos.Sabado)
	}

	return resp
}

func float64Ptr(v float64) *float64 {
	return &v
}

// GetReservasSimplePG godoc
// @Summary Listar reservas simples
// @Description Devuelve reservas en formato plano (sin agrupar por local). La paginacion es opcional. Los filtros de busqueda y vigencia se aplican antes de LIMIT y del conteo. `orden=cronologico` ordena por fecha, hora, local e ID; sin ese parametro conserva el orden legacy por local.
// @Tags Reservas BD
// @Produce json
// @Param local query string false "Nombre del local" example(SAN MARTIN)
// @Param fecha query string false "Fecha exacta YYYY-MM-DD" example(2026-05-23)
// @Param fecha_desde query string false "Fecha inicio rango YYYY-MM-DD" example(2026-05-19)
// @Param fecha_hasta query string false "Fecha fin rango YYYY-MM-DD" example(2026-05-24)
// @Param cliente query string false "Nombre del cliente" example(Maria Lopez)
// @Param numero_telefono query string false "Numero de telefono; acepta 70011223, 59170011223 o +591 700-11223" example(+59170011223)
// @Param servicio_solicitado query string false "Busqueda parcial por servicio solicitado" example(depilacion)
// @Param servicio_confirmado query string false "Busqueda parcial por servicio confirmado" example(depilacion laser piernas)
// @Param busqueda query string false "Busqueda parcial por ID, cliente, telefono, servicio, local o fecha" example(Maria)
// @Param estado query string false "Estado de la reserva" Enums(PENDIENTE,RECHAZADO,AGENDADO,COMPLETADO) example(COMPLETADO)
// @Param excluir_estado query string false "Estado que se excluye del resultado" Enums(PENDIENTE,RECHAZADO,AGENDADO,COMPLETADO) example(COMPLETADO)
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta) example(bicicleta)
// @Param vigente_fecha query string false "Fecha local desde la que una reserva sigue vigente; requiere vigente_hora" example(2026-08-12)
// @Param vigente_hora query string false "Hora local HH:MM desde la que una reserva sigue vigente; requiere vigente_fecha" example(15:30)
// @Param vigencia_solo_pendientes query bool false "Aplica la vigencia solo a reservas PENDIENTE" example(false)
// @Param orden query string false "Orden estable del listado" Enums(cronologico) example(cronologico)
// @Param limit query int false "Tamano de pagina opcional (1-100); sin limit ni cursor conserva modo legacy" example(50)
// @Param cursor query string false "Cursor opaco devuelto en paginacion.next_cursor"
// @Param include_total query bool false "Incluye el total de paginas" example(false)
// @Success 200 {object} utils.APIResponse{data=reservaSimpleListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: tipo invalido, estado invalido"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/reservas [get]
func (h *Container) GetReservasSimplePG(c *gin.Context) {
	paramTipo := strings.ToLower(strings.TrimSpace(c.Query("tipo")))
	if paramTipo != "" && paramTipo != "mesa" && paramTipo != "bicicleta" {
		utils.RespondError(c, http.StatusBadRequest,
			"tipo invÃ¡lido, valores permitidos: mesa, bicicleta")
		return
	}
	orden := strings.ToLower(strings.TrimSpace(c.Query("orden")))
	if orden != "" && orden != "cronologico" {
		utils.RespondError(c, http.StatusBadRequest, "orden invalido, valor permitido: cronologico")
		return
	}
	vigenteFecha := strings.TrimSpace(c.Query("vigente_fecha"))
	vigenteHora := strings.TrimSpace(c.Query("vigente_hora"))
	if (vigenteFecha == "") != (vigenteHora == "") {
		utils.RespondError(c, http.StatusBadRequest, "vigente_fecha y vigente_hora deben enviarse juntos")
		return
	}
	if vigenteFecha != "" {
		if _, err := time.Parse("2006-01-02", vigenteFecha); err != nil {
			utils.RespondError(c, http.StatusBadRequest, "formato de vigente_fecha invalido, use YYYY-MM-DD")
			return
		}
		if _, err := time.Parse("15:04", vigenteHora); err != nil {
			utils.RespondError(c, http.StatusBadRequest, "formato de vigente_hora invalido, use HH:MM")
			return
		}
	}
	vigenciaPendientes := false
	if raw := strings.TrimSpace(c.Query("vigencia_solo_pendientes")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "vigencia_solo_pendientes debe ser true o false")
			return
		}
		vigenciaPendientes = value
	}
	if vigenciaPendientes && vigenteFecha == "" {
		utils.RespondError(c, http.StatusBadRequest, "vigencia_solo_pendientes requiere vigente_fecha y vigente_hora")
		return
	}

	filtro := services.FiltroReservasSimple{
		Context:            c.Request.Context(),
		Local:              strings.TrimSpace(c.Query("local")),
		Fecha:              strings.TrimSpace(c.Query("fecha")),
		FechaDesde:         strings.TrimSpace(c.Query("fecha_desde")),
		FechaHasta:         strings.TrimSpace(c.Query("fecha_hasta")),
		Cliente:            strings.TrimSpace(c.Query("cliente")),
		NumeroTelefono:     strings.TrimSpace(c.Query("numero_telefono")),
		ServicioSolicitado: strings.TrimSpace(c.Query("servicio_solicitado")),
		ServicioConfirmado: strings.TrimSpace(c.Query("servicio_confirmado")),
		Busqueda:           strings.TrimSpace(c.Query("busqueda")),
		Estado:             strings.TrimSpace(c.Query("estado")),
		ExcluirEstado:      strings.TrimSpace(c.Query("excluir_estado")),
		Tipo:               paramTipo,
		VigenteFecha:       vigenteFecha,
		VigenteHora:        vigenteHora,
		VigenciaPendientes: vigenciaPendientes,
		Orden:              orden,
	}
	if filtro.Estado != "" {
		estado, err := services.NormalizarEstadoReserva(filtro.Estado)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
		filtro.Estado = estado
	}
	if filtro.ExcluirEstado != "" {
		estado, err := services.NormalizarEstadoReserva(filtro.ExcluirEstado)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
		filtro.ExcluirEstado = estado
	}
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	filters := map[string]string{
		"local": filtro.Local, "fecha": filtro.Fecha, "fecha_desde": filtro.FechaDesde, "fecha_hasta": filtro.FechaHasta,
		"cliente": filtro.Cliente, "numero_telefono": filtro.NumeroTelefono, "servicio_solicitado": filtro.ServicioSolicitado,
		"servicio_confirmado": filtro.ServicioConfirmado, "busqueda": filtro.Busqueda,
		"estado": filtro.Estado, "excluir_estado": filtro.ExcluirEstado, "tipo": filtro.Tipo,
		"vigente_fecha": filtro.VigenteFecha, "vigente_hora": filtro.VigenteHora,
		"vigencia_solo_pendientes": strconv.FormatBool(filtro.VigenciaPendientes), "orden": filtro.Orden,
	}
	var after reservationCursor
	if !decodePaginationCursor(c, page, "reservas", filters, &after) || (page.Cursor != "" && (after.ID < 1 || after.Date.IsZero() || after.Time == "")) {
		if !c.Writer.Written() {
			utils.RespondError(c, http.StatusBadRequest, "paginacion invalida: cursor incompleto")
		}
		return
	}
	filtro.PageLimit = page.QueryLimit()
	filtro.CursorSet = page.Cursor != ""
	filtro.CursorLocal, filtro.CursorFecha, filtro.CursorHora, filtro.CursorID = after.Local, after.Date, after.Time, after.ID

	resultado, err := h.ReservasPG.GetReservasSimple(filtro)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	resultado, metadata, err := pagination.Build(resultado, page, "reservas", filters, func(item services.ReservaSimple) any {
		date, _ := time.Parse("2006-01-02", item.Fecha)
		return reservationCursor{Local: item.Local, Date: date, Time: item.HoraDesde, ID: item.ID}
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if includeTotal {
		countFilter := filtro
		countFilter.PageLimit = 0
		countFilter.CursorSet = false
		count, countErr := h.ReservasPG.CountReservasSimple(countFilter)
		if countErr != nil {
			utils.RespondError(c, http.StatusInternalServerError, countErr.Error())
			return
		}
		pagination.AddTotal(metadata, count)
	}

	utils.Respond(c, http.StatusOK, reservaSimpleListResponse{
		Total:      len(resultado),
		Reservas:   resultado,
		Paginacion: metadata,
	})
}

// GetReservaPGByID godoc
// @Summary Obtener reserva por ID
// @Description Devuelve una reserva por su ID. Requiere token Bearer. Los usuarios con local asignado solo pueden consultar reservas de su local; admin_sys puede consultar cualquiera. Param: id (requerido, path). Response: reserva (ReservaSimple con: id, local, tipo M/B, fecha, hora_desde, hora_hasta, cliente, estado, numero_telefono, servicio, servicio_solicitado, servicio_confirmado, precio, notas, notificado, creado_en, actualizado_en).
// @Tags Reservas BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la reserva" example(44)
// @Success 200 {object} utils.APIResponse{data=reservaItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Reserva no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/reservas/{id} [get]
func (h *Container) GetReservaPGByID(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invÃ¡lido")
		return
	}

	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, http.StatusForbidden, services.ErrNoAutorizado.Error())
		return
	}

	reserva, err := h.ReservasPG.GetReservaByID(id, scope.LocalID)
	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "no rows") || strings.Contains(errLower, "no encontrad") {
			utils.RespondError(c, http.StatusNotFound, "reserva no encontrada")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, reservaItemResponse{Reserva: reserva})
}

type actualizarReservaPGRequest struct {
	// ID de la reserva a actualizar
	Id int `json:"id" binding:"required" example:"44"`
	// Nombre del local (siempre requerido para validar existencia)
	Local string `json:"local" binding:"required" example:"SAN MARTIN"`
	// Nueva fecha (YYYY-MM-DD), opcional
	NuevaFecha string `json:"nueva_fecha" example:"2026-05-24"`
	// Nueva hora de inicio (HH:MM), opcional
	NuevaHoraDesde string `json:"nueva_hora_desde" example:"16:00"`
	// Nueva hora de fin (HH:MM), opcional
	NuevaHoraHasta string `json:"nueva_hora_hasta" example:"17:00"`
	// Nuevo tipo de espacio: M (mesa) o B (bicicleta), opcional
	NuevoTipo string `json:"nuevo_tipo" example:"B"`
	// Nuevo numero de telefono, opcional
	NuevoNumeroTelefono string `json:"nuevo_numero_telefono" example:"+59170011224"`
	// Nueva forma internacional E.164 opcional.
	NuevoTelefonoE164 *string `json:"nuevo_telefono_e164,omitempty" example:"+59170011224"`
	// Nuevo nombre del servicio principal (opcional)
	NuevoServicio string `json:"nuevo_servicio" example:"Evaluacion corporal"`
	// Nuevo detalle de lo que solicito el cliente (opcional)
	NuevoServicioSolicitado string `json:"nuevo_servicio_solicitado" example:"Evaluacion corporal completa"`
	// Nuevo servicio confirmado tras evaluacion (opcional)
	NuevoServicioConfirmado string `json:"nuevo_servicio_confirmado" example:"Evaluacion corporal"`
	// Nuevo precio, opcional
	NuevoPrecio *float64 `json:"nuevo_precio" example:"180"`
	// Nuevas notas u observaciones, opcional
	NuevasNotas string `json:"nuevas_notas" example:"Reagendada por solicitud del cliente"`
	// Nuevo nombre del cliente, opcional
	NuevoCliente string `json:"nuevo_cliente" example:"Maria Lopez"`
	// Nuevo local al que se mueve la reserva, opcional
	NuevoLocal string `json:"nuevo_local" example:"PASEO ARANJUEZ"`
	// Nuevo plan o paquete al que se imputa la reserva, opcional
	NuevoPlanID *int `json:"nuevo_plan_id" example:"7"`
	// Desvincula la reserva de su plan actual; tiene prioridad sobre nuevo_plan_id, opcional
	LimpiarPlanID bool `json:"limpiar_plan_id" example:"false"`
}

// PatchReservaPG godoc
// @Summary Actualizar reserva en base de datos
// @Description Actualiza datos de una reserva. Solo se actualizan los campos enviados. No cambia estado (usar PATCH /bd/reservas/estado). Las reservas PENDIENTE, RECHAZADO y AGENDADO son editables; las COMPLETADO no. En una reserva AGENDADO, enviar nuevo_servicio arrastra tambien servicio_confirmado y el tipo de espacio, y cambiar fecha, hora o local marca la reserva como no notificada para reavisar al cliente. id: ID de la reserva (requerido). local: nombre del local para validar existencia (requerido). nueva_fecha: nueva fecha YYYY-MM-DD (opcional, no acepta domingos ni fechas pasadas). nueva_hora_desde: nueva hora inicio HH:MM (opcional). nueva_hora_hasta: nueva hora fin HH:MM (opcional). Horarios: lunes a viernes 08:00-20:00; sabado SAN MARTIN 08:00-15:00 y PASEO ARANJUEZ 08:00-18:00. nuevo_tipo: M=mesa o B=bicicleta (opcional). nuevo_cliente: nuevo nombre del cliente (opcional). nuevo_numero_telefono: nuevo telefono (opcional). nuevo_servicio: nombre del servicio principal (opcional). nuevo_servicio_solicitado: detalle solicitado por el cliente (opcional). nuevo_servicio_confirmado: servicio final tras evaluacion (opcional). nuevo_precio: nuevo precio (opcional). nuevas_notas: nuevas notas u observaciones (opcional). nuevo_local: local destino al que se mueve la reserva (opcional). nuevo_plan_id: plan o paquete al que se imputa la reserva (opcional). limpiar_plan_id: desvincula la reserva de su plan actual (opcional).
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body actualizarReservaPGRequest true "Datos para actualizar la reserva"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, local requerido, tipo invalido, sin cambios para actualizar, fecha pasada u horario fuera de atencion"
// @Failure 401 {object} utils.APIResponse "Token ausente o invalido"
// @Failure 404 {object} utils.APIResponse "Reserva no encontrada"
// @Failure 409 {object} utils.APIResponse "Reserva completada o sin espacios disponibles en el horario destino"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/reservas [patch]
func (h *Container) PatchReservaPG(c *gin.Context) {
	var req actualizarReservaPGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	id := req.Id
	if id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id no valido")
		return
	}

	if req.Local == "" {
		utils.RespondError(c, http.StatusBadRequest, "local es requerido")
		return
	}

	nuevoTipoNorm := strings.ToUpper(strings.TrimSpace(req.NuevoTipo))
	if nuevoTipoNorm != "" && nuevoTipoNorm != "M" && nuevoTipoNorm != "B" {
		utils.RespondError(c, http.StatusBadRequest, "nuevo_tipo invÃ¡lido, valores permitidos: M, B")
		return
	}

	nuevoTelefono := ""
	if strings.TrimSpace(req.NuevoNumeroTelefono) != "" {
		nuevoTelefonoNormalizado, err := normalizarTelefono(req.NuevoNumeroTelefono)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
		nuevoTelefono = nuevoTelefonoNormalizado
	}

	if req.NuevaFecha == "" && req.NuevaHoraDesde == "" && req.NuevaHoraHasta == "" && nuevoTipoNorm == "" &&
		nuevoTelefono == "" && req.NuevoTelefonoE164 == nil && req.NuevoServicio == "" && req.NuevoServicioSolicitado == "" &&
		req.NuevoServicioConfirmado == "" && req.NuevoPrecio == nil && req.NuevasNotas == "" &&
		strings.TrimSpace(req.NuevoCliente) == "" && strings.TrimSpace(req.NuevoLocal) == "" &&
		req.NuevoPlanID == nil && !req.LimpiarPlanID {
		utils.RespondError(c, http.StatusBadRequest, "no hay cambios para actualizar")
		return
	}

	err := h.ReservasPG.ActualizarReserva(services.ActualizarReservaPGInput{
		Id:                  id,
		Local:               req.Local,
		NuevaFecha:          req.NuevaFecha,
		NuevaHoraDesde:      req.NuevaHoraDesde,
		NuevaHoraHasta:      req.NuevaHoraHasta,
		NuevoTipo:           nuevoTipoNorm,
		NuevoCliente:        req.NuevoCliente,
		NuevoNumeroTelefono: nuevoTelefono, NuevoTelefonoE164: req.NuevoTelefonoE164,
		NuevoServicio:           req.NuevoServicio,
		NuevoServicioSolicitado: req.NuevoServicioSolicitado,
		NuevoServicioConfirmado: req.NuevoServicioConfirmado,
		NuevoPrecio:             req.NuevoPrecio,
		NuevasNotas:             req.NuevasNotas,
		NuevoLocal:              req.NuevoLocal,
		NuevoPlanID:             req.NuevoPlanID,
		LimpiarPlanID:           req.LimpiarPlanID,
	})

	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "no se pudo encontrar la reserva") ||
			strings.Contains(errLower, "reserva no encontrada") {
			utils.RespondError(c, http.StatusNotFound, "No se pudo encontrar la reserva")
			return
		}
		if strings.Contains(errLower, "no se puede editar una reserva completada") ||
			strings.Contains(errLower, "no hay ambientes disponibles") ||
			strings.Contains(errLower, "no hay espacios disponibles") {
			utils.RespondError(c, http.StatusConflict, err.Error())
			return
		}
		if strings.Contains(errLower, "no está disponible en este local") ||
			strings.Contains(errLower, "telefono_e164") ||
			strings.Contains(errLower, "horario fuera de atenci") ||
			strings.Contains(errLower, "hora_desde") ||
			strings.Contains(errLower, "hora_hasta") ||
			strings.Contains(errLower, "formato de fecha") ||
			strings.Contains(errLower, "hora de inicio") ||
			strings.Contains(errLower, "hora de finalizaci") ||
			strings.Contains(errLower, "fecha pasada") ||
			strings.Contains(errLower, "no encontrado") {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "Reserva actualizada correctamente"})
}

// PatchReservaEstadoPG godoc
// @Summary Actualizar estado de reserva
// @Description Cambia el estado de una reserva. id: ID de la reserva (requerido). estado: PENDIENTE, AGENDADO, RECHAZADO o COMPLETADO (requerido). causa: motivo del cambio, requerido cuando el estado es RECHAZADO. servicio_confirmado: servicio final (opcional). precio: precio actualizado (opcional). tipo: M=mesa o B=bicicleta (opcional). Transiciones: PENDIENTE→AGENDADO/RECHAZADO, AGENDADO→COMPLETADO/RECHAZADO, RECHAZADO→PENDIENTE. COMPLETADO no admite cambios.
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body actualizarEstadoReservaPGRequest true "Nuevo estado de la reserva"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: estado invalido, transicion no permitida, campo requerido faltante"
// @Failure 401 {object} utils.APIResponse "Token ausente o invalido"
// @Failure 404 {object} utils.APIResponse "Reserva no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/reservas/estado [patch]
func (h *Container) PatchReservaEstadoPG(c *gin.Context) {
	var req actualizarEstadoReservaPGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id no valido")
		return
	}

	err := h.ReservasPG.ActualizarEstadoReserva(services.ActualizarEstadoReservaInput{
		Id:                 req.Id,
		Estado:             req.Estado,
		Causa:              req.Causa,
		ServicioConfirmado: req.ServicioConfirmado,
		Precio:             req.Precio,
		Tipo:               req.Tipo,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no se pudo encontrar la reserva") {
			utils.RespondError(c, http.StatusNotFound, "No se pudo encontrar la reserva")
			return
		}
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "Estado de reserva actualizado correctamente"})
}

// PatchReservaNotificadoPG godoc
// @Summary Actualizar notificacion de reserva
// @Description Marca una reserva como notificada o no. id: ID de la reserva (requerido). notificado: true=notificado, false=no notificado (requerido). Se usa para tracking de avisos al cliente.
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Param payload body actualizarNotificadoReservaPGRequest true "Estado de notificacion"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, notificado requerido"
// @Failure 404 {object} utils.APIResponse "Reserva no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/reservas/notificar [patch]
func (h *Container) PatchReservaNotificadoPG(c *gin.Context) {
	var req actualizarNotificadoReservaPGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}
	if req.Notificado == nil {
		utils.RespondError(c, http.StatusBadRequest, "notificado es requerido")
		return
	}

	err := h.ReservasPG.ActualizarNotificacionReserva(req.Id, *req.Notificado)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no se pudo encontrar la reserva") {
			utils.RespondError(c, http.StatusNotFound, "No se pudo encontrar la reserva")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "Notificacion de reserva actualizada correctamente"})
}

// GetNotificacionesReservasPG godoc
// @Summary Listar notificaciones de reservas
// @Description Devuelve los datos minimos de reservas activas en estado AGENDADO que aun no fueron marcadas como notificadas/leidas, ordenadas por creado_en descendente (mas recientes primero). Requiere token Bearer. Los usuarios con local asignado solo ven notificaciones de su local; admin_sys ve todas. Este endpoint esta pensado para polling de la campanita del frontend; se puede consultar cada 5 o 10 minutos. Param: limit cantidad maxima a devolver (opcional, default 20, maximo 100). Response: total (int), reservas con cliente, servicio, local, fecha, horario y telefono.
// @Tags Notificaciones
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param limit query int false "Cantidad maxima de notificaciones a devolver (default 20, maximo 100)" example(20)
// @Success 200 {object} utils.APIResponse{data=reservaNotificacionListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: limit invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/notificaciones/reservas [get]
func (h *Container) GetNotificacionesReservasPG(c *gin.Context) {
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			utils.RespondError(c, http.StatusBadRequest, "limit invalido")
			return
		}
		limit = parsed
	}

	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, http.StatusForbidden, services.ErrNoAutorizado.Error())
		return
	}

	reservas, err := h.ReservasPG.GetReservasAgendadasNoNotificadasContext(c.Request.Context(), limit, scope.NombreLocal)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, reservaNotificacionListResponse{
		Total:    len(reservas),
		Reservas: reservas,
	})
}

// PatchNotificacionesReservasLeidasPG godoc
// @Summary Marcar notificaciones de reservas como leidas
// @Description Marca como notificadas/leidas varias reservas agendadas en una sola operacion. Body: ids lista de IDs de reservas a marcar. Response: actualizadas cantidad de reservas activas actualizadas.
// @Tags Notificaciones
// @Accept json
// @Produce json
// @Param payload body marcarNotificacionesReservasLeidasRequest true "IDs de reservas a marcar como leidas"
// @Success 200 {object} utils.APIResponse{data=actualizadasResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: ids requerido o contiene valores invalidos"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/notificaciones/reservas/leer [patch]
func (h *Container) PatchNotificacionesReservasLeidasPG(c *gin.Context) {
	var req marcarNotificacionesReservasLeidasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	actualizadas, err := h.ReservasPG.ActualizarNotificacionReservas(req.Ids, true)
	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "ids es requerido") || strings.Contains(errLower, "valor invalido") {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, actualizadasResponse{Actualizadas: actualizadas})
}

// PatchNotificacionReservaLeidaPG godoc
// @Summary Marcar notificacion de reserva como leida
// @Description Marca como notificada/leida una reserva agendada para quitarla de la campanita. Param: id de la reserva (requerido, path). Response: mensaje string.
// @Tags Notificaciones
// @Produce json
// @Param id path int true "ID de la reserva" example(44)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Reserva no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/notificaciones/reservas/{id}/leer [patch]
func (h *Container) PatchNotificacionReservaLeidaPG(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	err = h.ReservasPG.ActualizarNotificacionReserva(id, true)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no se pudo encontrar la reserva") {
			utils.RespondError(c, http.StatusNotFound, "No se pudo encontrar la reserva")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "Notificacion de reserva marcada como leida correctamente"})
}

// DeleteReservaPG godoc
// @Summary Eliminar reserva
// @Description Eliminacion logica de una reserva (activo=false). id: ID de la reserva (requerido, path).
// @Tags Reservas BD
// @Produce json
// @Param id path int true "ID de la reserva" example(44)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Reserva no encontrada o ya inactiva"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/reservas/{id} [delete]
func (h *Container) DeleteReservaPG(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invÃ¡lido")
		return
	}

	err = h.ReservasPG.DeleteReserva(id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "no encontrada") ||
			strings.Contains(strings.ToLower(err.Error()), "no encontrado") ||
			strings.Contains(strings.ToLower(err.Error()), "inactiva") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "reserva eliminada correctamente"})
}
