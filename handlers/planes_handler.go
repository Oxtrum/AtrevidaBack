package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type planFiltrosResponse struct {
	Cliente        string `json:"cliente" example:"Maria"`
	Local          string `json:"local" example:"SAN MARTIN"`
	LocalID        *int   `json:"local_id,omitempty" example:"1"`
	Estado         string `json:"estado" example:"ACTIVO"`
	EstadoCobranza string `json:"estado_cobranza" example:"PENDIENTE"`
	FechaDesde     string `json:"fecha_desde" example:"2026-07-01"`
	FechaHasta     string `json:"fecha_hasta" example:"2026-07-31"`
}

type planListResponse struct {
	Total   int            `json:"total" example:"5"`
	Filtros planFiltrosResponse `json:"filtros"`
	Planes  []models.PlanPG `json:"planes"`
}

type planItemResponse struct {
	Plan *models.PlanCompletoPG `json:"plan"`
}

// GetPlanes godoc
// @Summary Listar planes
// @Description Devuelve todos los planes que cumplan los filtros aplicados. Un plan es un contrato adquirido por un cliente; no es un combo de catalogo. Requiere token Bearer. Los usuarios no admin_sys solo ven planes de su local asignado.
// @Tags Planes BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param cliente query string false "Busqueda parcial por nombre del cliente" example(Maria)
// @Param local query string false "Busqueda parcial por nombre del local" example(SAN MARTIN)
// @Param local_id query int false "ID exacto del local" example(1)
// @Param estado query string false "Filtrar por estado contractual: BORRADOR, ACTIVO, COMPLETADO, VENCIDO, CANCELADO" example(ACTIVO)
// @Param estado_cobranza query string false "Filtrar por estado de cobranza: PENDIENTE, PARCIAL, PAGADO, VENCIDO" example(PENDIENTE)
// @Param fecha_desde query string false "Fecha de creacion desde (YYYY-MM-DD)" example(2026-07-01)
// @Param fecha_hasta query string false "Fecha de creacion hasta (YYYY-MM-DD)" example(2026-07-31)
// @Success 200 {object} utils.APIResponse{data=planListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: parametros invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/planes [get]
func (h *Container) GetPlanes(c *gin.Context) {
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "token invalido o sin datos de usuario")
		return
	}

	localID, err := optionalPositiveInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	filtroLocalID := localID
	if scope.LocalID != nil {
		filtroLocalID = scope.LocalID
	}

	var fechaDesde, fechaHasta *time.Time
	if raw := strings.TrimSpace(c.Query("fecha_desde")); raw != "" {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "fecha_desde debe tener formato YYYY-MM-DD")
			return
		}
		fechaDesde = &t
	}
	if raw := strings.TrimSpace(c.Query("fecha_hasta")); raw != "" {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "fecha_hasta debe tener formato YYYY-MM-DD")
			return
		}
		fechaHasta = &t
	}

	planes, err := h.PlanesPG.ListarPlanes(services.FiltroPlanes{
		Cliente:        strings.TrimSpace(c.Query("cliente")),
		Local:          strings.TrimSpace(c.Query("local")),
		LocalID:        filtroLocalID,
		Estado:         strings.TrimSpace(c.Query("estado")),
		EstadoCobranza: strings.TrimSpace(c.Query("estado_cobranza")),
		FechaDesde:     fechaDesde,
		FechaHasta:     fechaHasta,
	})
	if err != nil {
		responderErrorPlan(c, err)
		return
	}

	utils.Respond(c, http.StatusOK, planListResponse{
		Total: len(planes),
		Filtros: planFiltrosResponse{
			Cliente:        strings.TrimSpace(c.Query("cliente")),
			Local:          strings.TrimSpace(c.Query("local")),
			LocalID:        filtroLocalID,
			Estado:         strings.TrimSpace(c.Query("estado")),
			EstadoCobranza: strings.TrimSpace(c.Query("estado_cobranza")),
			FechaDesde:     c.Query("fecha_desde"),
			FechaHasta:     c.Query("fecha_hasta"),
		},
		Planes: planes,
	})
}

// GetPlanByID godoc
// @Summary Obtener plan por ID
// @Description Devuelve el detalle completo de un plan: cabecera, servicios contratados (snapshots), cuotas y pagos aplicados. Requiere token Bearer. Los usuarios no admin_sys solo pueden acceder a planes de su local asignado.
// @Tags Planes BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del plan" example(21)
// @Success 200 {object} utils.APIResponse{data=planItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado para ver este plan"
// @Failure 404 {object} utils.APIResponse "Plan no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/planes/{id} [get]
func (h *Container) GetPlanByID(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	plan, err := h.PlanesPG.ObtenerPlan(id)
	if err != nil {
		responderErrorPlan(c, err)
		return
	}

	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "token invalido")
		return
	}

	if scope.LocalID != nil && (plan.LocalID == nil || *plan.LocalID != *scope.LocalID) {
		utils.RespondError(c, http.StatusForbidden, "no autorizado para ver este plan")
		return
	}

	utils.Respond(c, http.StatusOK, planItemResponse{Plan: plan})
}

func responderErrorPlan(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, services.ErrPlanInvalido):
		status = http.StatusBadRequest
	case errors.Is(err, services.ErrPlanNoEncontrado):
		status = http.StatusNotFound
	}
	utils.RespondError(c, status, err.Error())
}
