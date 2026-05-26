package handlers

import (
	"net/http"
	"strings"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// POST /reservas
type crearReservaRequest struct {
	// Nombre del local
	Local string `json:"local" binding:"required" example:"SAN MARTIN"`
	// Semana en formato YYYY-MM-DD (primer dia de la semana, lunes)
	Semana string `json:"semana" binding:"required" example:"2026-05-25"`
	// Dia de la semana (lunes, martes, miercoles, jueves, viernes, sabado)
	Dia string `json:"dia" binding:"required" example:"lunes"`
	// Hora de inicio (HH:MM)
	HoraDesde string `json:"hora_desde" binding:"required" example:"09:00"`
	// Hora de fin (HH:MM), opcional
	HoraHasta string `json:"hora_hasta" example:"10:00"`
	// Tipo de espacio: M (mesa) o B (bicicleta)
	Tipo string `json:"tipo" binding:"required" example:"M"`
	// Nombre del cliente
	Cliente string `json:"cliente" binding:"required" example:"Maria Lopez"`
	// Servicio solicitado (texto libre)
	Servicio string `json:"servicio" example:"Depilacion piernas completas"`
}

// PostReserva godoc
// @Summary Crear reserva en Google Sheets
// @Description Crea una reserva en uno o varios slots dentro de Google Sheets. Si hay conflictos de horario, algunos slots pueden fallar y otros crearse (207 Multi-Status).
// @Tags Reservas Sheets
// @Accept json
// @Produce json
// @Param payload body crearReservaRequest true "Datos de la reserva"
// @Success 200 {object} utils.APIResponse{data=slotsResponse} "Todos los slots reservados exitosamente"
// @Success 207 {object} utils.APIResponse{data=slotsResponse} "Algunos slots no pudieron reservarse"
// @Failure 400 {object} utils.APIResponse "Error de validacion: campo requerido faltante o tipo invalido"
// @Failure 409 {object} utils.APIResponse "Conflicto: ningun slot pudo reservarse"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /reservas [post]
func (h *Container) PostReserva(c *gin.Context) {
	var req crearReservaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	tipoNorm := strings.ToUpper(strings.TrimSpace(req.Tipo))
	if tipoNorm != "M" && tipoNorm != "B" {
		utils.RespondError(c, http.StatusBadRequest, "tipo inválido, valores permitidos: M, B")
		return
	}

	resultado, err := h.Writer.CrearReserva(services.CrearReservaInput{
		Local:     strings.TrimSpace(req.Local),
		Semana:    strings.TrimSpace(req.Semana),
		Dia:       strings.TrimSpace(req.Dia),
		HoraDesde: strings.TrimSpace(req.HoraDesde),
		HoraHasta: strings.TrimSpace(req.HoraHasta),
		Tipo:      tipoNorm,
		Cliente:   strings.TrimSpace(req.Cliente),
		Servicio:  strings.TrimSpace(req.Servicio),
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	data := slotsResponse{
		SlotsOk:    resultado.Exitosos,
		SlotsError: resultado.Errores,
	}

	switch {
	case len(resultado.Exitosos) == 0:
		utils.RespondError(c, http.StatusConflict,
			"no se pudo reservar ningún slot: "+strings.Join(resultado.Errores, "; "))
	case len(resultado.Errores) > 0:
		utils.RespondMultiStatus(c, data, "algunos slots no pudieron reservarse")
	default:
		utils.Respond(c, http.StatusOK, data)
	}
}

// PATCH /reservas
type actualizarReservaRequest struct {
	// Nombre del local
	Local string `json:"local" binding:"required" example:"SAN MARTIN"`
	// Semana en formato YYYY-MM-DD
	Semana string `json:"semana" binding:"required" example:"2026-05-25"`
	// Dia actual de la reserva
	Dia string `json:"dia" binding:"required" example:"lunes"`
	// Hora actual de la reserva
	Hora string `json:"hora" binding:"required" example:"09:00"`
	// Tipo actual de espacio: M (mesa) o B (bicicleta)
	Tipo string `json:"tipo" binding:"required" example:"M"`
	// Nombre del cliente
	Cliente string `json:"cliente" binding:"required" example:"Maria Lopez"`

	// Nuevo dia (opcional)
	NuevoDia string `json:"nuevo_dia" example:"martes"`
	// Nueva hora de inicio (opcional)
	NuevaHoraDesde string `json:"nueva_hora_desde" example:"11:00"`
	// Nueva hora de fin (opcional)
	NuevaHoraHasta string `json:"nueva_hora_hasta" example:"12:00"`
	// Nuevo tipo de espacio: M o B (opcional)
	NuevoTipo string `json:"nuevo_tipo" example:"B"`
	// Nuevo servicio (opcional)
	NuevoServicio string `json:"nuevo_servicio" example:"Evaluacion corporal"`
}

// PatchReserva godoc
// @Summary Actualizar reserva en Google Sheets
// @Description Modifica una reserva existente en Google Sheets. Solo se actualizan los campos nuevos enviados. Si hay conflictos, algunos slots pueden fallar (207 Multi-Status).
// @Tags Reservas Sheets
// @Accept json
// @Produce json
// @Param payload body actualizarReservaRequest true "Datos para actualizar la reserva"
// @Success 200 {object} utils.APIResponse{data=slotsResponse} "Todos los slots actualizados exitosamente"
// @Success 207 {object} utils.APIResponse{data=slotsResponse} "Algunos slots no pudieron actualizarse"
// @Failure 400 {object} utils.APIResponse "Error de validacion: campo requerido faltante, tipo invalido, sin cambios para actualizar"
// @Failure 409 {object} utils.APIResponse "Conflicto: ningun slot pudo actualizarse"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /reservas [patch]
func (h *Container) PatchReserva(c *gin.Context) {
	var req actualizarReservaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	tipoNorm := strings.ToUpper(strings.TrimSpace(req.Tipo))
	if tipoNorm != "M" && tipoNorm != "B" {
		utils.RespondError(c, http.StatusBadRequest, "tipo inválido, valores permitidos: M, B")
		return
	}

	nuevoTipoNorm := strings.ToUpper(strings.TrimSpace(req.NuevoTipo))
	if nuevoTipoNorm != "" && nuevoTipoNorm != "M" && nuevoTipoNorm != "B" {
		utils.RespondError(c, http.StatusBadRequest, "nuevo_tipo inválido, valores permitidos: M, B")
		return
	}

	if req.NuevoDia == "" && req.NuevaHoraDesde == "" && nuevoTipoNorm == "" && req.NuevoServicio == "" {
		utils.RespondError(c, http.StatusBadRequest,
			"debe especificarse al menos un campo a modificar: nuevo_dia, nueva_hora_desde, nuevo_tipo, nuevo_servicio")
		return
	}

	resultado, err := h.Writer.ActualizarReserva(services.ActualizarReservaInput{
		Local:          strings.TrimSpace(req.Local),
		Semana:         strings.TrimSpace(req.Semana),
		Dia:            strings.TrimSpace(req.Dia),
		Hora:           strings.TrimSpace(req.Hora),
		Tipo:           tipoNorm,
		Cliente:        strings.TrimSpace(req.Cliente),
		NuevoDia:       strings.TrimSpace(req.NuevoDia),
		NuevaHoraDesde: strings.TrimSpace(req.NuevaHoraDesde),
		NuevaHoraHasta: strings.TrimSpace(req.NuevaHoraHasta),
		NuevoTipo:      nuevoTipoNorm,
		NuevoServicio:  strings.TrimSpace(req.NuevoServicio),
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	data := slotsResponse{
		SlotsOk:    resultado.Exitosos,
		SlotsError: resultado.Errores,
	}

	switch {
	case len(resultado.Exitosos) == 0:
		utils.RespondError(c, http.StatusConflict,
			"no se pudo actualizar ningún slot: "+strings.Join(resultado.Errores, "; "))
	case len(resultado.Errores) > 0:
		utils.RespondMultiStatus(c, data, "algunos slots no pudieron actualizarse")
	default:
		utils.Respond(c, http.StatusOK, data)
	}
}
