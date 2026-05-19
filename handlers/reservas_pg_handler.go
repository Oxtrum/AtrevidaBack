package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

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
			"tipo inválido, valores permitidos: mesa, bicicleta")
		return
	}

	var reservados *bool
	if raw := strings.TrimSpace(c.Query("reservados")); raw != "" {
		v := strings.ToLower(raw) == "true"
		reservados = &v
	}

	filtro := services.FiltroReservasPG{
		Local:      strings.TrimSpace(c.Query("local")),
		Fecha:      strings.TrimSpace(c.Query("fecha")),
		FechaDesde: strings.TrimSpace(c.Query("fecha_desde")),
		FechaHasta: strings.TrimSpace(c.Query("fecha_hasta")),
		Cliente:    strings.TrimSpace(c.Query("cliente")),
		Tipo:       paramTipo,
		Reservados: reservados,
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
			"local":       filtro.Local,
			"fecha":       filtro.Fecha,
			"fecha_desde": filtro.FechaDesde,
			"fecha_hasta": filtro.FechaHasta,
			"tipo":        filtro.Tipo,
			"cliente":     filtro.Cliente,
			"reservados":  reservados,
		},
		"reservas": resultado,
	})
}

// POST /bd/reservas
type crearReservaPGRequest struct {
	Local     string   `json:"local"      binding:"required"`
	Fecha     string   `json:"fecha"      binding:"required"` // "2025-04-04"
	HoraDesde string   `json:"hora_desde" binding:"required"`
	HoraHasta string   `json:"hora_hasta"` // opcional
	Tipo      string   `json:"tipo"`
	Cliente   string   `json:"cliente"    binding:"required"`
	Servicio  string   `json:"servicio"`
	Precio    *float64 `json:"precio"`
	Notas     string   `json:"notas"`
	PlanID    *int     `json:"plan_id"`
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
	/*
		if tipoNorm != "M" && tipoNorm != "B" {
			utils.RespondError(c, http.StatusBadRequest, "tipo inválido, valores permitidos: M, B")
			return
		}
	*/

	id, err := h.ReservasPG.CrearReserva(services.CrearReservaPGInput{
		Local:     strings.TrimSpace(req.Local),
		Fecha:     strings.TrimSpace(req.Fecha),
		HoraDesde: strings.TrimSpace(req.HoraDesde),
		HoraHasta: strings.TrimSpace(req.HoraHasta),
		Tipo:      tipoNorm,
		Cliente:   strings.TrimSpace(req.Cliente),
		Servicio:  strings.TrimSpace(req.Servicio),
		Precio:    req.Precio,
		Notas:     strings.TrimSpace(req.Notas),
		PlanID:    req.PlanID,
	})

	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no hay espacios") ||
			strings.Contains(err.Error(), "no está disponible") {
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
// @Param tipo query string false "Tipo de reserva" Enums(mesa,bicicleta)
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/reservas [get]
func (h *Container) GetReservasSimplePG(c *gin.Context) {
	paramTipo := strings.ToLower(strings.TrimSpace(c.Query("tipo")))
	if paramTipo != "" && paramTipo != "mesa" && paramTipo != "bicicleta" {
		utils.RespondError(c, http.StatusBadRequest,
			"tipo inválido, valores permitidos: mesa, bicicleta")
		return
	}

	filtro := services.FiltroReservasSimple{
		Local:      strings.TrimSpace(c.Query("local")),
		Fecha:      strings.TrimSpace(c.Query("fecha")),
		FechaDesde: strings.TrimSpace(c.Query("fecha_desde")),
		FechaHasta: strings.TrimSpace(c.Query("fecha_hasta")),
		Cliente:    strings.TrimSpace(c.Query("cliente")),
		Tipo:       paramTipo,
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
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
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
	Id    int    `json:"id"         binding:"required"`
	Local string `json:"local"      binding:"required"`

	NuevaFecha     string   `json:"nueva_fecha"`
	NuevaHoraDesde string   `json:"nueva_hora_desde"`
	NuevaHoraHasta string   `json:"nueva_hora_hasta"`
	NuevoTipo      string   `json:"nuevo_tipo"`
	NuevoServicio  string   `json:"nuevo_servicio"`
	NuevoPrecio    *float64 `json:"nuevo_precio"`
	NuevasNotas    string   `json:"nuevas_notas"`
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
		utils.RespondError(c, http.StatusBadRequest, "nuevo_tipo inválido, valores permitidos: M, B")
		return
	}

	if req.NuevaFecha == "" && req.NuevaHoraDesde == "" && nuevoTipoNorm == "" &&
		req.NuevoServicio == "" && req.NuevoPrecio == nil && req.NuevasNotas == "" {
		utils.RespondError(c, http.StatusBadRequest, "no hay cambios para actualizar")
		return
	}

	err := h.ReservasPG.ActualizarReserva(services.ActualizarReservaPGInput{
		Id:             id,
		Local:          req.Local,
		NuevaFecha:     req.NuevaFecha,
		NuevaHoraDesde: req.NuevaHoraDesde,
		NuevaHoraHasta: req.NuevaHoraHasta,
		NuevoTipo:      nuevoTipoNorm,
		NuevoServicio:  req.NuevoServicio,
		NuevoPrecio:    req.NuevoPrecio,
		NuevasNotas:    req.NuevasNotas,
	})

	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"mensaje": "Reserva actualizada correctamente",
	})
}
