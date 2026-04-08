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
	Local     string `json:"local"      binding:"required"`
	Semana    string `json:"semana"     binding:"required"`
	Dia       string `json:"dia"        binding:"required"`
	HoraDesde string `json:"hora_desde" binding:"required"`
	HoraHasta string `json:"hora_hasta"`
	Tipo      string `json:"tipo"       binding:"required"`
	Cliente   string `json:"cliente"    binding:"required"`
	Servicio  string `json:"servicio"`
}

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

	data := gin.H{
		"slots_ok":    resultado.Exitosos,
		"slots_error": resultado.Errores,
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
	Local   string `json:"local"   binding:"required"`
	Semana  string `json:"semana"  binding:"required"`
	Dia     string `json:"dia"     binding:"required"`
	Hora    string `json:"hora"    binding:"required"`
	Tipo    string `json:"tipo"    binding:"required"`
	Cliente string `json:"cliente" binding:"required"`

	NuevoDia       string `json:"nuevo_dia"`
	NuevaHoraDesde string `json:"nueva_hora_desde"`
	NuevaHoraHasta string `json:"nueva_hora_hasta"`
	NuevoTipo      string `json:"nuevo_tipo"`
	NuevoServicio  string `json:"nuevo_servicio"`
}

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

	data := gin.H{
		"slots_ok":    resultado.Exitosos,
		"slots_error": resultado.Errores,
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
