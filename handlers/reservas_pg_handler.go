package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// GET /bd/reservas
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
		Local:          strings.TrimSpace(c.Query("local")),
		Fecha:          strings.TrimSpace(c.Query("fecha")),
		FechaDesde:     strings.TrimSpace(c.Query("fecha_desde")),
		FechaHasta:     strings.TrimSpace(c.Query("fecha_hasta")),
		Cliente:        strings.TrimSpace(c.Query("cliente")),
		NumeroTelefono: strings.TrimSpace(c.Query("numero_telefono")),
		Estado:         strings.TrimSpace(c.Query("estado")),
		Tipo:           paramTipo,
		Reservados:     reservados,
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
			"local":           filtro.Local,
			"fecha":           filtro.Fecha,
			"fecha_desde":     filtro.FechaDesde,
			"fecha_hasta":     filtro.FechaHasta,
			"tipo":            filtro.Tipo,
			"cliente":         filtro.Cliente,
			"numero_telefono": filtro.NumeroTelefono,
			"estado":          filtro.Estado,
			"reservados":      reservados,
		},
		"reservas": resultado,
	})
}

// POST /bd/reservas
type crearReservaPGRequest struct {
	Local          string   `json:"local"      binding:"required"`
	Fecha          string   `json:"fecha"      binding:"required"`
	HoraDesde      string   `json:"hora_desde" binding:"required"`
	HoraHasta      string   `json:"hora_hasta"`
	Tipo           string   `json:"tipo"`
	Cliente        string   `json:"cliente"    binding:"required"`
	NumeroTelefono string   `json:"numero_telefono" binding:"required"`
	Estado         string   `json:"estado"`
	Servicio       string   `json:"servicio"`
	Precio         *float64 `json:"precio"`
	Notas          string   `json:"notas"`
	PlanID         *int     `json:"plan_id"`
}

var telefonoRegex = regexp.MustCompile(`^\+?\d+$`)

const allowEstadoOverrideTemporal = true

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
		Telefono:  telefono,
		Estado:    estadoFinal,
		Servicio:  strings.TrimSpace(req.Servicio),
		Precio:    req.Precio,
		Notas:     strings.TrimSpace(req.Notas),
		PlanID:    req.PlanID,
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
	Id     int    `json:"id" binding:"required"`
	Estado string `json:"estado" binding:"required"`
	Causa  string `json:"causa"`
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

func (h *Container) GetReservasSimplePG(c *gin.Context) {
	paramTipo := strings.ToLower(strings.TrimSpace(c.Query("tipo")))
	if paramTipo != "" && paramTipo != "mesa" && paramTipo != "bicicleta" {
		utils.RespondError(c, http.StatusBadRequest,
			"tipo invÃ¡lido, valores permitidos: mesa, bicicleta")
		return
	}

	filtro := services.FiltroReservasSimple{
		Local:          strings.TrimSpace(c.Query("local")),
		Fecha:          strings.TrimSpace(c.Query("fecha")),
		FechaDesde:     strings.TrimSpace(c.Query("fecha_desde")),
		FechaHasta:     strings.TrimSpace(c.Query("fecha_hasta")),
		Cliente:        strings.TrimSpace(c.Query("cliente")),
		NumeroTelefono: strings.TrimSpace(c.Query("numero_telefono")),
		Estado:         strings.TrimSpace(c.Query("estado")),
		Tipo:           paramTipo,
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

// GET /bd/reservas/:id
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
	Id                  int      `json:"id"         binding:"required"`
	Local               string   `json:"local"      binding:"required"`
	NuevaFecha          string   `json:"nueva_fecha"`
	NuevaHoraDesde      string   `json:"nueva_hora_desde"`
	NuevaHoraHasta      string   `json:"nueva_hora_hasta"`
	NuevoTipo           string   `json:"nuevo_tipo"`
	NuevoNumeroTelefono string   `json:"nuevo_numero_telefono"`
	NuevoServicio       string   `json:"nuevo_servicio"`
	NuevoPrecio         *float64 `json:"nuevo_precio"`
	NuevasNotas         string   `json:"nuevas_notas"`
}

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
		nuevoTelefono == "" && req.NuevoServicio == "" && req.NuevoPrecio == nil && req.NuevasNotas == "" {
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
		NuevoNumeroTelefono: nuevoTelefono,
		NuevoServicio:       req.NuevoServicio,
		NuevoPrecio:         req.NuevoPrecio,
		NuevasNotas:         req.NuevasNotas,
	})

	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"mensaje": "Reserva actualizada correctamente",
	})
}

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
		Id:     req.Id,
		Estado: req.Estado,
		Causa:  req.Causa,
	})
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"mensaje": "Estado de reserva actualizado correctamente",
	})
}
