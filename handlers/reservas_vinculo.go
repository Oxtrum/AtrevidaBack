package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type vincularReservaRequest struct {
	// Cliente registrado al que se enlaza la reserva (requerido).
	ClienteID int `json:"cliente_id" example:"12"`
	// Si es true, el telefono del cliente se reemplaza por el de la reserva (solo si son distintos).
	ActualizarTelefono bool `json:"actualizar_telefono" example:"false"`
}

type reservasPorVincularResponse struct {
	// Total de reservas vigentes sin cliente.
	Total    int                           `json:"total" example:"9"`
	Reservas []models.ReservaPorVincularPG `json:"reservas"`
}

func statusVinculoError(err error) int {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "no encontrad"):
		return http.StatusNotFound
	case strings.Contains(msg, "ya tiene un cliente"), strings.Contains(msg, "ya existe un cliente"), strings.Contains(msg, "tiene cargos"), strings.Contains(msg, "usa un paquete"):
		return http.StatusConflict
	case strings.Contains(msg, "requerido"):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// GetReservasPorVincular godoc
// @Summary Reservas por vincular
// @Description Lista las reservas vigentes (no anuladas ni rechazadas) que todavia no tienen un cliente registrado, las mas recientes primero. Vienen del formulario publico cuando el telefono no coincidio exacto con un unico cliente, o del historico. Requiere token Bearer.
// @Tags Reservas BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param limit query int false "Tamano de pagina (1-100, por defecto 50)" example(50)
// @Param offset query int false "Reservas a saltar" example(0)
// @Success 200 {object} utils.APIResponse{data=reservasPorVincularResponse}
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/reservas/por-vincular [get]
func (h *Container) GetReservasPorVincular(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	reservas, total, err := h.ReservasVinculo.PorVincular(limit, offset)
	if err != nil {
		utils.RespondError(c, statusVinculoError(err), err.Error())
		return
	}
	utils.Respond(c, http.StatusOK, reservasPorVincularResponse{Total: total, Reservas: reservas}, "Reservas por vincular")
}

// GetSugerenciasClienteReserva godoc
// @Summary Clientes sugeridos para una reserva
// @Description Propone clientes que podrian ser el dueño de una reserva por vincular: los que comparten telefono (ultimos 8 digitos) o nombre completo, con los de telefono primero. Es solo una pista (por_telefono, por_nombre): nunca vincula; el cajero elige. Requiere token Bearer.
// @Tags Reservas BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la reserva" example(41)
// @Success 200 {object} utils.APIResponse{data=[]models.ClienteSugeridoPG}
// @Failure 400 {object} utils.APIResponse "id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/reservas/{id}/sugerencias-cliente [get]
func (h *Container) GetSugerenciasClienteReserva(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	sugeridos, err := h.ReservasVinculo.Sugerencias(id)
	if err != nil {
		utils.RespondError(c, statusVinculoError(err), err.Error())
		return
	}
	utils.Respond(c, http.StatusOK, sugeridos, "Clientes sugeridos")
}

// PatchReservaCliente godoc
// @Summary Vincular una reserva con un cliente
// @Description Enlaza una reserva con un cliente registrado elegido por el cajero. Solo vincula esa reserva: nada se enlaza por telefono. Si la reserva ya tenia otro cliente (el cajero se equivoco) lo cambia, siempre que no tenga cargos vigentes ni paquete del cliente anterior; si los tiene responde 409 y hay que anularlos primero. Con actualizar_telefono, si el telefono de la reserva es distinto al del cliente, el del cliente se reemplaza por el de la reserva (no se agrega otro). Devuelve cuantas reservas quedaron enlazadas y si se actualizo el telefono. Requiere token Bearer.
// @Tags Reservas BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la reserva" example(41)
// @Param payload body vincularReservaRequest true "Cliente elegido"
// @Success 200 {object} utils.APIResponse{data=models.ResultadoVinculo}
// @Failure 400 {object} utils.APIResponse "cliente_id requerido o id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 404 {object} utils.APIResponse "Reserva o cliente no encontrado"
// @Failure 409 {object} utils.APIResponse "La reserva tiene cargos o paquete del cliente anterior, o el telefono nuevo choca con otro cliente"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/reservas/{id}/cliente [patch]
func (h *Container) PatchReservaCliente(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var req vincularReservaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	resultado, err := h.ReservasVinculo.Vincular(id, req.ClienteID, req.ActualizarTelefono)
	if err != nil {
		utils.RespondError(c, statusVinculoError(err), err.Error())
		return
	}
	utils.Respond(c, http.StatusOK, resultado, "Reserva vinculada")
}
