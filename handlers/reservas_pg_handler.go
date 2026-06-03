package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"atrevida-agenda-api/models"
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
// @Param numero_telefono query string false "Numero de telefono" example(+59170011223)
// @Param servicio_solicitado query string false "Busqueda parcial por servicio solicitado" example(depilacion)
// @Param servicio_confirmado query string false "Busqueda parcial por servicio confirmado" example(depilacion laser piernas)
// @Param estado query string false "Estado de la reserva" Enums(PENDIENTE,RECHAZADO,AGENDADO,COMPLETADO) example(AGENDADO)
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta) example(mesa)
// @Param reservados query bool false "Filtrar por estado reservado" example(true)
// @Success 200 {object} utils.APIResponse{data=reservaCalendarioResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: tipo invalido, estado invalido"
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
	// Estado inicial: PENDIENTE (default), AGENDADO (solo si el servicio no requiere evaluacion)
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
// @Description Crea una reserva en PostgreSQL. local: nombre del local (requerido). fecha: YYYY-MM-DD (requerido, no acepta domingos). hora_desde: HH:MM (requerido). hora_hasta: HH:MM (opcional). Horarios: lunes a viernes 08:00-20:00; sabado SAN MARTIN 08:00-15:00 y PASEO ARANJUEZ 08:00-18:00. tipo: M=mesa o B=bicicleta (opcional). cliente: nombre del cliente (requerido). numero_telefono: telefono del cliente (requerido). estado: PENDIENTE por defecto, AGENDADO solo si servicio no requiere evaluacion (opcional). servicio_id: ID del servicio en BD, recomendado para aplicar requiere_evaluacion sin depender del nombre (opcional). servicio: nombre del servicio principal (opcional). servicio_solicitado: detalle solicitado, se copia de servicio si se omite (opcional). servicio_confirmado: servicio final tras evaluacion, se autocompleta si no requiere evaluacion (opcional). precio: precio de la reserva (opcional). notas: observaciones (opcional). plan_id: ID del plan asociado (opcional).
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
		Local:              strings.TrimSpace(req.Local),
		Fecha:              strings.TrimSpace(req.Fecha),
		HoraDesde:          strings.TrimSpace(req.HoraDesde),
		HoraHasta:          strings.TrimSpace(req.HoraHasta),
		Tipo:               tipoNorm,
		Cliente:            strings.TrimSpace(req.Cliente),
		Telefono:           telefono,
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

type reservaResumenResponse struct {
	// Cantidad de reservas agendadas para el dia consultado
	ReservasAgendadasDia int `json:"reservas_agendadas_dia" example:"15"`
	// Cantidad de servicios completados en el dia consultado
	ServiciosCompletadosDia int `json:"servicios_completados_dia" example:"10"`
	// Resumen por dia de la semana desde el lunes hasta la fecha consultada
	Semana reservaResumenSemanaResponse `json:"semana"`
}

func normalizarTelefono(raw string) (string, error) {
	telefono := strings.TrimSpace(raw)
	telefono = strings.ReplaceAll(telefono, " ", "")
	telefono = strings.ReplaceAll(telefono, "-", "")

	if telefono == "" {
		return "", errors.New("numero_telefono es requerido")
	}
	if len(telefono) > 13 {
		return "", errors.New("numero_telefono no puede exceder 13 caracteres")
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
// @Description Devuelve resumen de reservas del dia. Param: fecha YYYY-MM-DD (requerido, query, no acepta domingos). Response: reservas_agendadas_dia (int), servicios_completados_dia (int), semana (reservaResumenSemanaResponse con: total_reservas int, lunes..sabado int opcionales segun el dia).
// @Tags Reservas BD
// @Produce json
// @Param fecha query string true "Fecha a consultar YYYY-MM-DD (no acepta domingos)" example(2026-05-23)
// @Success 200 {object} utils.APIResponse{data=reservaResumenResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: fecha requerida, formato invalido, domingo no permitido"
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

	resumen, err := h.ReservasPG.GetResumenReservas(fecha)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "domingo") {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, reservaResumenResponse{
		ReservasAgendadasDia:    resumen.ReservasAgendadasDia,
		ServiciosCompletadosDia: resumen.ServiciosCompletadosDia,
		Semana:                  buildReservaResumenSemanaResponse(fecha, resumen.Semana),
	})
}

func buildReservaResumenSemanaResponse(fecha time.Time, semana services.ResumenReservasSemana) reservaResumenSemanaResponse {
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

// GetReservasSimplePG godoc
// @Summary Listar reservas simples
// @Description Devuelve reservas en formato plano (sin agrupar por local). Filtros: local (opcional), fecha YYYY-MM-DD (opcional), fecha_desde/fecha_hasta rango (opcional), cliente (opcional), numero_telefono (opcional), servicio_solicitado busqueda parcial (opcional), servicio_confirmado busqueda parcial (opcional), estado PENDIENTE/RECHAZADO/AGENDADO/COMPLETADO (opcional), tipo mesa/bicicleta (opcional). Response: total (int total de reservas), reservas ([]ReservaSimple con: id, local, tipo M/B, fecha, hora_desde, hora_hasta, cliente, estado, numero_telefono, servicio, servicio_solicitado, servicio_confirmado, precio, notas, notificado, creado_en, actualizado_en).
// @Tags Reservas BD
// @Produce json
// @Param local query string false "Nombre del local" example(SAN MARTIN)
// @Param fecha query string false "Fecha exacta YYYY-MM-DD" example(2026-05-23)
// @Param fecha_desde query string false "Fecha inicio rango YYYY-MM-DD" example(2026-05-19)
// @Param fecha_hasta query string false "Fecha fin rango YYYY-MM-DD" example(2026-05-24)
// @Param cliente query string false "Nombre del cliente" example(Maria Lopez)
// @Param numero_telefono query string false "Numero de telefono" example(+59170011223)
// @Param servicio_solicitado query string false "Busqueda parcial por servicio solicitado" example(depilacion)
// @Param servicio_confirmado query string false "Busqueda parcial por servicio confirmado" example(depilacion laser piernas)
// @Param estado query string false "Estado de la reserva" Enums(PENDIENTE,RECHAZADO,AGENDADO,COMPLETADO) example(COMPLETADO)
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta) example(bicicleta)
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

	filtro := services.FiltroReservasSimple{
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
	}
	if filtro.Estado != "" {
		estado, err := services.NormalizarEstadoReserva(filtro.Estado)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
		filtro.Estado = estado
	}

	resultado, err := h.ReservasPG.GetReservasSimple(filtro)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, reservaSimpleListResponse{
		Total:    len(resultado),
		Reservas: resultado,
	})
}

// GetReservaPGByID godoc
// @Summary Obtener reserva por ID
// @Description Devuelve una reserva por su ID. Param: id (requerido, path). Response: reserva (ReservaSimple con: id, local, tipo M/B, fecha, hora_desde, hora_hasta, cliente, estado, numero_telefono, servicio, servicio_solicitado, servicio_confirmado, precio, notas, notificado, creado_en, actualizado_en).
// @Tags Reservas BD
// @Produce json
// @Param id path int true "ID de la reserva" example(44)
// @Success 200 {object} utils.APIResponse{data=reservaItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
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

	reserva, err := h.ReservasPG.GetReservaByID(id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
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
}

// PatchReservaPG godoc
// @Summary Actualizar reserva en base de datos
// @Description Actualiza datos de una reserva. Solo se actualizan los campos enviados. No cambia estado (usar PATCH /bd/reservas/estado). id: ID de la reserva (requerido). local: nombre del local para validar existencia (requerido). nueva_fecha: nueva fecha YYYY-MM-DD (opcional, no acepta domingos). nueva_hora_desde: nueva hora inicio HH:MM (opcional). nueva_hora_hasta: nueva hora fin HH:MM (opcional). Horarios: lunes a viernes 08:00-20:00; sabado SAN MARTIN 08:00-15:00 y PASEO ARANJUEZ 08:00-18:00. nuevo_tipo: M=mesa o B=bicicleta (opcional). nuevo_numero_telefono: nuevo telefono (opcional). nuevo_servicio: nombre del servicio principal (opcional). nuevo_servicio_solicitado: detalle solicitado por el cliente (opcional). nuevo_servicio_confirmado: servicio final tras evaluacion (opcional). nuevo_precio: nuevo precio (opcional). nuevas_notas: nuevas notas u observaciones (opcional).
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Param payload body actualizarReservaPGRequest true "Datos para actualizar la reserva"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, local requerido, tipo invalido, sin cambios para actualizar u horario fuera de atencion"
// @Failure 404 {object} utils.APIResponse "Reserva no encontrada"
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

	if req.NuevaFecha == "" && req.NuevaHoraDesde == "" && nuevoTipoNorm == "" &&
		nuevoTelefono == "" && req.NuevoServicio == "" && req.NuevoServicioSolicitado == "" &&
		req.NuevoServicioConfirmado == "" && req.NuevoPrecio == nil && req.NuevasNotas == "" {
		utils.RespondError(c, http.StatusBadRequest, "no hay cambios para actualizar")
		return
	}

	err := h.ReservasPG.ActualizarReserva(services.ActualizarReservaPGInput{
		Id:                      id,
		Local:                   req.Local,
		NuevaFecha:              req.NuevaFecha,
		NuevaHoraDesde:          req.NuevaHoraDesde,
		NuevaHoraHasta:          req.NuevaHoraHasta,
		NuevoTipo:               nuevoTipoNorm,
		NuevoNumeroTelefono:     nuevoTelefono,
		NuevoServicio:           req.NuevoServicio,
		NuevoServicioSolicitado: req.NuevoServicioSolicitado,
		NuevoServicioConfirmado: req.NuevoServicioConfirmado,
		NuevoPrecio:             req.NuevoPrecio,
		NuevasNotas:             req.NuevasNotas,
	})

	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "no se pudo encontrar la reserva") {
			utils.RespondError(c, http.StatusNotFound, "No se pudo encontrar la reserva")
			return
		}
		if strings.Contains(errLower, "horario fuera de atenci") ||
			strings.Contains(errLower, "hora_desde") ||
			strings.Contains(errLower, "hora_hasta") ||
			strings.Contains(errLower, "formato de fecha") ||
			strings.Contains(errLower, "hora de inicio") ||
			strings.Contains(errLower, "hora de finalizaci") {
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
// @Description Cambia el estado de una reserva. id: ID de la reserva (requerido). estado: PENDIENTE, AGENDADO, RECHAZADO o COMPLETADO (requerido). causa: motivo del cambio (opcional). servicio_confirmado: servicio final (opcional). precio: precio actualizado (opcional). tipo: M=mesa o B=bicicleta (opcional). Transiciones: PENDIENTE→AGENDADO/RECHAZADO, AGENDADO→COMPLETADO/RECHAZADO. RECHAZADO y COMPLETADO no admiten cambios.
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Param payload body actualizarEstadoReservaPGRequest true "Nuevo estado de la reserva"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: estado invalido, transicion no permitida, campo requerido faltante"
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
// @Description Devuelve reservas activas en estado AGENDADO que aun no fueron marcadas como notificadas/leidas, ordenadas por creado_en descendente (mas recientes primero). Este endpoint esta pensado para polling de la campanita del frontend; se puede consultar cada 5 o 10 minutos. Param: limit cantidad maxima a devolver (opcional, default 20, maximo 100). Response: total (int), reservas ([]ReservaSimple con datos de la reserva agendada pendiente).
// @Tags Notificaciones
// @Produce json
// @Param limit query int false "Cantidad maxima de notificaciones a devolver (default 20, maximo 100)" example(20)
// @Success 200 {object} utils.APIResponse{data=reservaNotificacionListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: limit invalido"
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

	reservas, err := h.ReservasPG.GetReservasAgendadasNoNotificadas(limit)
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
