package handlers

import "github.com/gin-gonic/gin"

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
	// Nombre del servicio (ej: "Depilacion piernas completas"). Texto libre.
	Servicio string `json:"servicio" example:"Depilacion piernas completas"`
}

// PostReserva godoc
// @Summary Endpoint legacy para crear reservas en Google Sheets
// @Description Endpoint legacy deshabilitado. Google Sheets ya no esta soportado; use POST /bd/reservas.
// @Tags Reservas Sheets
// @Accept json
// @Produce json
// @Param payload body crearReservaRequest true "Datos de la reserva"
// @Failure 410 {object} utils.APIResponse "Google Sheets ya no soportado"
// @Router /reservas [post]
func (h *Container) PostReserva(c *gin.Context) {
	RespondGoogleSheetsUnsupported(c)
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
	// Nuevo nombre del servicio (opcional)
	NuevoServicio string `json:"nuevo_servicio" example:"Evaluacion corporal"`
}

// PatchReserva godoc
// @Summary Endpoint legacy para actualizar reservas en Google Sheets
// @Description Endpoint legacy deshabilitado. Google Sheets ya no esta soportado; use PATCH /bd/reservas.
// @Tags Reservas Sheets
// @Accept json
// @Produce json
// @Param payload body actualizarReservaRequest true "Datos para actualizar la reserva"
// @Failure 410 {object} utils.APIResponse "Google Sheets ya no soportado"
// @Router /reservas [patch]
func (h *Container) PatchReserva(c *gin.Context) {
	RespondGoogleSheetsUnsupported(c)
}
