package handlers

import (
	"errors"
	"net/http"
	"strconv"
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
	Total   int                 `json:"total" example:"5"`
	Filtros planFiltrosResponse `json:"filtros"`
	Planes  []models.PlanPG     `json:"planes"`
}

type planItemResponse struct {
	Plan *models.PlanCompletoPG `json:"plan"`
}

type planServicioManualRequest struct {
	ServicioIDOrigen       *int     `json:"servicio_id_origen,omitempty" example:"8"`
	NombreTexto         string   `json:"nombre_snapshot" example:"Masaje relajante"`
	TiempoTexto         *string  `json:"tiempo_snapshot,omitempty" example:"01:00"`
	PrecioUnitarioTexto *float64 `json:"precio_unitario_snapshot,omitempty" example:"200"`
	SesionesContratadas    int      `json:"sesiones_contratadas" example:"2"`
	Orden                  int      `json:"orden" example:"0"`
	// Numero de sesion dentro del servicio (para seguimiento por sesion); default 1 si no se especifica.
	SesionNumero int `json:"sesion_numero,omitempty" example:"1"`
}

type crearPlanRequest struct {
	ClienteID      int                         `json:"cliente_id" example:"12"`
	LocalID        int                         `json:"local_id" example:"1"`
	ComboID        *int                        `json:"combo_id,omitempty" example:"12"`
	Servicios      []planServicioManualRequest `json:"servicios,omitempty"`
	FechaInicio    *string                     `json:"fecha_inicio,omitempty" example:"2026-07-15"`
	FechaFin       *string                     `json:"fecha_fin,omitempty" example:"2026-08-14"`
	TipoPago       string                      `json:"tipo_pago" example:"UNICO"`
	CantidadCuotas int                         `json:"cantidad_cuotas,omitempty" example:"3"`
	Descuento      *float64                    `json:"descuento,omitempty" example:"50"`
	Notas          *string                     `json:"notas,omitempty" example:"Pago contado"`
	// Codigo del pago de caja a aplicar a la cuota (UNICO); marca la cuota PAGADO.
	PagoCodigo *string `json:"pago_codigo,omitempty" example:"PAG-000123"`
}

type actualizarPlanRequest struct {
	Notas       *string `json:"notas,omitempty" example:"Actualizar notas del plan"`
	FechaInicio *string `json:"fecha_inicio,omitempty" example:"2026-07-20"`
	FechaFin    *string `json:"fecha_fin,omitempty" example:"2026-08-19"`
}

type cambiarEstadoPlanRequest struct {
	Estado string `json:"estado" example:"ACTIVO"`
}

type marcarSesionRequest struct {
	// TRUE marca la sesión como realizada; FALSE la vuelve a pendiente.
	Realizado bool `json:"realizado" example:"true"`
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

// CreatePlan godoc
// @Summary Crear plan desde combo o manual
// @Description Crea un plan contractual para un cliente. Dos origenes mutuamente excluyentes: combo_id (copia snapshot del catalogo) o servicios (composicion manual). Calcula subtotal, descuento y precio total en backend. Crea cuotas segun tipo_pago. Requiere token Bearer con rol gerencia o admin_sys.
// @Tags Planes BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearPlanRequest true "Datos del plan a crear"
// @Success 201 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: datos del plan invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Cliente, local o combo no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/planes [post]
func (h *Container) CreatePlan(c *gin.Context) {
	var req crearPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	userID, _ := authenticatedUserID(c)

	var fechaInicio, fechaFin *time.Time
	if req.FechaInicio != nil {
		t, err := time.Parse("2006-01-02", *req.FechaInicio)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "fecha_inicio debe tener formato YYYY-MM-DD")
			return
		}
		fechaInicio = &t
	}
	if req.FechaFin != nil {
		t, err := time.Parse("2006-01-02", *req.FechaFin)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "fecha_fin debe tener formato YYYY-MM-DD")
			return
		}
		fechaFin = &t
	}

	descuento := 0.0
	if req.Descuento != nil {
		descuento = *req.Descuento
	}

	var servicios []services.PlanServicioInput
	if req.Servicios != nil {
		for _, s := range req.Servicios {
			servicios = append(servicios, services.PlanServicioInput{
				ServicioIDOrigen:       s.ServicioIDOrigen,
				NombreTexto:         s.NombreTexto,
				TiempoTexto:         s.TiempoTexto,
				PrecioUnitarioTexto: s.PrecioUnitarioTexto,
				SesionesContratadas:    s.SesionesContratadas,
				Orden:                  s.Orden,
				SesionNumero:           s.SesionNumero,
			})
		}
	}

	id, err := h.PlanesPG.CrearPlan(services.CrearPlanInput{
		ClienteID:      req.ClienteID,
		LocalID:        req.LocalID,
		ComboID:        req.ComboID,
		Servicios:      servicios,
		FechaInicio:    fechaInicio,
		FechaFin:       fechaFin,
		TipoPago:       strings.ToUpper(strings.TrimSpace(req.TipoPago)),
		CantidadCuotas: req.CantidadCuotas,
		Descuento:      descuento,
		Notas:          req.Notas,
		CreadoPor:      &userID,
		PagoCodigo:     req.PagoCodigo,
	})
	if err != nil {
		responderErrorPlan(c, err)
		return
	}

	utils.Respond(c, http.StatusCreated, idResponse{ID: id})
}

// PatchPlan godoc
// @Summary Actualizar campos editables de un plan
// @Description Actualiza notas, fecha_inicio o fecha_fin de un plan. Solo permitido cuando el plan esta en estado BORRADOR. Requiere token Bearer con rol gerencia o admin_sys.
// @Tags Planes BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del plan" example(21)
// @Param payload body actualizarPlanRequest true "Campos del plan a modificar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id o campos invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Plan no encontrado"
// @Failure 409 {object} utils.APIResponse "El plan no permite modificaciones en su estado actual"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/planes/{id} [patch]
func (h *Container) PatchPlan(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var req actualizarPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	var fechaInicio, fechaFin *time.Time
	if req.FechaInicio != nil {
		t, err := time.Parse("2006-01-02", *req.FechaInicio)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "fecha_inicio debe tener formato YYYY-MM-DD")
			return
		}
		fechaInicio = &t
	}
	if req.FechaFin != nil {
		t, err := time.Parse("2006-01-02", *req.FechaFin)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "fecha_fin debe tener formato YYYY-MM-DD")
			return
		}
		fechaFin = &t
	}

	err = h.PlanesPG.ActualizarPlan(services.ActualizarPlanInput{
		ID: id, Notas: req.Notas,
		FechaInicio: fechaInicio, FechaFin: fechaFin,
	})
	if err != nil {
		responderErrorPlan(c, err)
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "plan actualizado correctamente"})
}

// PatchPlanEstado godoc
// @Summary Cambiar estado de un plan
// @Description Transicion de estado del plan. Transiciones validas: BORRADOR -> ACTIVO, ACTIVO -> COMPLETADO, ACTIVO -> CANCELADO. Requiere token Bearer con rol gerencia o admin_sys.
// @Tags Planes BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del plan" example(21)
// @Param payload body cambiarEstadoPlanRequest true "Nuevo estado del plan"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: estado invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Plan no encontrado"
// @Failure 409 {object} utils.APIResponse "Transicion de estado no permitida"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/planes/{id}/estado [patch]
func (h *Container) PatchPlanEstado(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var req cambiarEstadoPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	userID, _ := authenticatedUserID(c)

	err = h.PlanesPG.CambiarEstado(services.CambiarEstadoInput{
		ID: id, Estado: req.Estado, UsuarioID: &userID,
	})
	if err != nil {
		responderErrorPlan(c, err)
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "estado del plan actualizado correctamente"})
}

// MarcarSesionPlan godoc
// @Summary Marcar una sesión del plan como realizada o pendiente
// @Description Actualiza el estado realizado de todas las líneas de una sesión del plan. Requiere token Bearer con rol gerencia o admin_sys.
// @Tags Planes BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del plan"
// @Param numero path int true "Número de sesión"
// @Param request body marcarSesionRequest true "Estado de la sesión"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /bd/planes/{id}/sesiones/{numero} [patch]
func (h *Container) MarcarSesionPlan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}
	numero, err := strconv.Atoi(c.Param("numero"))
	if err != nil || numero < 1 {
		utils.RespondError(c, http.StatusBadRequest, "numero invalido")
		return
	}
	var req marcarSesionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	if err := h.PlanesPG.MarcarSesion(id, numero, req.Realizado); err != nil {
		responderErrorPlan(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, gin.H{"ok": true})
}

func responderErrorPlan(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, services.ErrPlanInvalido) || errors.Is(err, services.ErrPlanOrigenInvalido):
		status = http.StatusBadRequest
	case errors.Is(err, services.ErrPlanNoEncontrado):
		status = http.StatusNotFound
	case errors.Is(err, services.ErrPlanTransicionInvalida):
		status = http.StatusConflict
	}
	utils.RespondError(c, status, err.Error())
}
