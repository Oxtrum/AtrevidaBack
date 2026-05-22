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
// @Description Devuelve reservas agrupadas por local con filtros opcionales.
// @Tags Reservas BD
// @Produce json
// @Param local query string false "Nombre del local"
// @Param fecha query string false "Fecha exacta"
// @Param fecha_desde query string false "Fecha desde"
// @Param fecha_hasta query string false "Fecha hasta"
// @Param cliente query string false "Nombre del cliente"
// @Param numero_telefono query string false "Numero de telefono"
// @Param servicio_solicitado query string false "Busqueda parcial por servicio solicitado"
// @Param servicio_confirmado query string false "Busqueda parcial por servicio confirmado"
// @Param estado query string false "Estado de la reserva" Enums(PENDIENTE,RECHAZADO,AGENDADO,COMPLETADO)
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta)
// @Param reservados query bool false "Filtrar por estado reservado"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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

	utils.Respond(c, http.StatusOK, gin.H{
		"total_locales": len(resultado),
		"filtros": gin.H{
			"local":               filtro.Local,
			"fecha":               filtro.Fecha,
			"fecha_desde":         filtro.FechaDesde,
			"fecha_hasta":         filtro.FechaHasta,
			"tipo":                filtro.Tipo,
			"cliente":             filtro.Cliente,
			"numero_telefono":     filtro.NumeroTelefono,
			"servicio_solicitado": filtro.ServicioSolicitado,
			"servicio_confirmado": filtro.ServicioConfirmado,
			"estado":              filtro.Estado,
			"reservados":          reservados,
		},
		"reservas": resultado,
	})
}

// POST /bd/reservas
type crearReservaPGRequest struct {
	Local              string   `json:"local" binding:"required"`
	Fecha              string   `json:"fecha" binding:"required"`
	HoraDesde          string   `json:"hora_desde" binding:"required"`
	HoraHasta          string   `json:"hora_hasta"`
	Tipo               string   `json:"tipo"`
	Cliente            string   `json:"cliente" binding:"required"`
	NumeroTelefono     string   `json:"numero_telefono" binding:"required"`
	Estado             string   `json:"estado"`
	Servicio           string   `json:"servicio"`
	ServicioSolicitado string   `json:"servicio_solicitado"`
	ServicioConfirmado *string  `json:"servicio_confirmado"`
	Precio             *float64 `json:"precio"`
	Notas              string   `json:"notas"`
	PlanID             *int     `json:"plan_id"`
}

// PostReservaPG godoc
// @Summary Crear reserva en base de datos
// @Description Crea una reserva persistida en PostgreSQL.
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Param payload body crearReservaPGRequest true "Datos de la reserva"
// @Success 201 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 409 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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
		Servicio:           strings.TrimSpace(req.Servicio),
		ServicioSolicitado: strings.TrimSpace(req.ServicioSolicitado),
		ServicioConfirmado: req.ServicioConfirmado,
		Precio:             req.Precio,
		Notas:              strings.TrimSpace(req.Notas),
		PlanID:             req.PlanID,
	})

	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no hay espacios") ||
			strings.Contains(err.Error(), "no estÃ¡ disponible") {
			status = http.StatusConflict
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusCreated, gin.H{
		"id":      id,
		"mensaje": "Reserva creada correctamente",
	})
}

type actualizarEstadoReservaPGRequest struct {
	Id                 int      `json:"id" binding:"required"`
	Estado             string   `json:"estado" binding:"required"`
	Causa              string   `json:"causa"`
	ServicioConfirmado *string  `json:"servicio_confirmado"`
	Precio             *float64 `json:"precio"`
	Tipo               string   `json:"tipo"`
}

type reservaResumenSemanaResponse struct {
	TotalReservas int  `json:"total_reservas"`
	Lunes         *int `json:"lunes,omitempty"`
	Martes        *int `json:"martes,omitempty"`
	Miercoles     *int `json:"miercoles,omitempty"`
	Jueves        *int `json:"jueves,omitempty"`
	Viernes       *int `json:"viernes,omitempty"`
	Sabado        *int `json:"sabado,omitempty"`
}

type reservaResumenResponse struct {
	ReservasAgendadasDia    int                          `json:"reservas_agendadas_dia"`
	ServiciosCompletadosDia int                          `json:"servicios_completados_dia"`
	Semana                  reservaResumenSemanaResponse `json:"semana"`
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
// @Description Devuelve el resumen numerico de reservas agendadas del dia, servicios completados del dia y acumulado semanal desde el lunes hasta la fecha indicada.
// @Tags Reservas BD
// @Produce json
// @Param fecha query string true "Fecha a consultar en formato YYYY-MM-DD"
// @Success 200 {object} utils.APIResponse{data=reservaResumenResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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
// @Description Devuelve reservas desde PostgreSQL en formato simple y sin agrupacion por local.
// @Tags Reservas BD
// @Produce json
// @Param local query string false "Nombre del local"
// @Param fecha query string false "Fecha exacta"
// @Param fecha_desde query string false "Fecha desde"
// @Param fecha_hasta query string false "Fecha hasta"
// @Param cliente query string false "Nombre del cliente"
// @Param numero_telefono query string false "Numero de telefono"
// @Param servicio_solicitado query string false "Busqueda parcial por servicio solicitado"
// @Param servicio_confirmado query string false "Busqueda parcial por servicio confirmado"
// @Param estado query string false "Estado de la reserva" Enums(PENDIENTE,RECHAZADO,AGENDADO,COMPLETADO)
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta)
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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

	utils.Respond(c, http.StatusOK, gin.H{
		"total":    len(resultado),
		"reservas": resultado,
	})
}

// GetReservaPGByID godoc
// @Summary Obtener reserva por ID
// @Description Devuelve una reserva de PostgreSQL por su identificador.
// @Tags Reservas BD
// @Produce json
// @Param id path int true "ID de la reserva"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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

	utils.Respond(c, http.StatusOK, gin.H{
		"reserva": reserva,
	})
}

type actualizarReservaPGRequest struct {
	Id                      int      `json:"id" binding:"required"`
	Local                   string   `json:"local" binding:"required"`
	NuevaFecha              string   `json:"nueva_fecha"`
	NuevaHoraDesde          string   `json:"nueva_hora_desde"`
	NuevaHoraHasta          string   `json:"nueva_hora_hasta"`
	NuevoTipo               string   `json:"nuevo_tipo"`
	NuevoNumeroTelefono     string   `json:"nuevo_numero_telefono"`
	NuevoServicio           string   `json:"nuevo_servicio"`
	NuevoServicioSolicitado string   `json:"nuevo_servicio_solicitado"`
	NuevoServicioConfirmado string   `json:"nuevo_servicio_confirmado"`
	NuevoPrecio             *float64 `json:"nuevo_precio"`
	NuevasNotas             string   `json:"nuevas_notas"`
}

// PatchReservaPG godoc
// @Summary Actualizar reserva en base de datos
// @Description Actualiza una reserva existente en PostgreSQL.
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Param payload body actualizarReservaPGRequest true "Datos para actualizar la reserva"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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
		if strings.Contains(strings.ToLower(err.Error()), "no se pudo encontrar la reserva") {
			utils.RespondError(c, http.StatusNotFound, "No se pudo encontrar la reserva")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"mensaje": "Reserva actualizada correctamente",
	})
}

// PatchReservaEstadoPG godoc
// @Summary Actualizar estado de reserva
// @Description Cambia el estado de una reserva segun las reglas de negocio.
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Param payload body actualizarEstadoReservaPGRequest true "Nuevo estado de la reserva"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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

	utils.Respond(c, http.StatusOK, gin.H{
		"mensaje": "Estado de reserva actualizado correctamente",
	})
}

// DeleteReservaPG godoc
// @Summary Eliminar reserva
// @Description Realiza el borrado logico de una reserva estableciendo activo en false.
// @Tags Reservas BD
// @Produce json
// @Param id path int true "ID de la reserva"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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

	utils.Respond(c, http.StatusOK, gin.H{
		"mensaje": "reserva eliminada correctamente",
	})
}
