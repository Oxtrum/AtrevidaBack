package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type registrarAtencionRequest struct {
	EmpleadoID                 *int    `json:"empleado_id,omitempty"`
	EmpleadoNombre             string  `json:"empleado_nombre" binding:"required"`
	FechaAtencion              string  `json:"fecha_atencion" binding:"required"`
	HoraInicio                 *string `json:"hora_inicio,omitempty"`
	HoraFin                    *string `json:"hora_fin,omitempty"`
	Lugar                      string  `json:"lugar" binding:"required"`
	Observaciones              *string `json:"observaciones,omitempty"`
	ConfirmarStockInsuficiente bool    `json:"confirmar_stock_insuficiente,omitempty"`
	Insumos                    []struct {
		ItemID   int     `json:"item_id"`
		Cantidad float64 `json:"cantidad"`
	} `json:"insumos"`
}

// PostAtencion godoc
// @Summary Registrar atención y cerrar reserva
// @Description Registra la atención real y cambia una reserva AGENDADA a COMPLETADO en una sola transacción. Los insumos sin stock suficiente quedan en PENDIENTE_STOCK sin descuento parcial. Con inventario.politica_sin_stock=BLOQUEAR responde 409 salvo que el usuario confirme expresamente el faltante mediante confirmar_stock_insuficiente.
// @Tags Atenciones
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de reserva"
// @Param payload body registrarAtencionRequest true "Datos de atención"
// @Success 201 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse "Error de validación"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 404 {object} utils.APIResponse "Reserva no encontrada"
// @Failure 409 {object} utils.APIResponse "Reserva no atendible o stock insuficiente"
// @Router /bd/reservas/{id}/atencion [post]
func (h *Container) PostAtencion(c *gin.Context) {
	id, e := strconv.Atoi(c.Param("id"))
	if e != nil || id <= 0 {
		utils.RespondError(c, 400, "id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "el usuario no tiene un local asignado")
		return
	}
	var req registrarAtencionRequest
	if e = c.ShouldBindJSON(&req); e != nil {
		utils.RespondError(c, 400, e.Error())
		return
	}
	ins := make([]pgsqlrepo.AtencionInsumoInput, 0, len(req.Insumos))
	for _, x := range req.Insumos {
		ins = append(ins, pgsqlrepo.AtencionInsumoInput{ItemID: x.ItemID, Cantidad: x.Cantidad})
	}
	a, e := h.Atenciones.Registrar(services.RegistrarAtencionInput{ReservaID: id, LocalID: scope.LocalID, EmpleadoID: req.EmpleadoID, EmpleadoNombre: req.EmpleadoNombre, FechaAtencion: req.FechaAtencion, HoraInicio: req.HoraInicio, HoraFin: req.HoraFin, Lugar: req.Lugar, Observaciones: req.Observaciones, ConfirmarStockInsuficiente: req.ConfirmarStockInsuficiente, Actor: consumoActor(c), Insumos: ins})
	if e != nil {
		m := strings.ToLower(e.Error())
		status := http.StatusBadRequest
		evento := "cierre_fallido"
		if strings.Contains(m, "no encontrada") {
			status = 404
			evento = "reserva_no_encontrada"
		} else if strings.Contains(m, "solo se puede") || strings.Contains(m, "stock insuficiente") {
			status = 409
			evento = "conflicto_cierre"
			if strings.Contains(m, "stock insuficiente") {
				evento = "stock_insuficiente_bloqueo"
			}
		} else if strings.Contains(m, "acceso") {
			status = 403
			evento = "acceso_denegado"
		} else if strings.Contains(m, "fallo_consumo_sesion") {
			status = 500
			evento = "fallo_consumo_sesion"
		} else if strings.Contains(m, "fallo_descuento_inventario") {
			status = 500
			evento = "fallo_inventario"
		}
		log.Printf("[atenciones] event=%s reserva_id=%d local_scope=%v status=%d error=%q", evento, id, scope.LocalID, status, e.Error())
		utils.RespondError(c, status, e.Error())
		return
	}
	log.Printf("[atenciones] event=cierre_registrado reserva_id=%d atencion_id=%d local_id=%v insumos=%d", id, a.ID, a.LocalID, len(req.Insumos))
	utils.Respond(c, 201, a)
}

// GetAtenciones godoc
// @Summary Listar atenciones
// @Description Lista las atenciones activas del local autorizado, de la más reciente a la más antigua, con paginación por desplazamiento. fecha_desde y fecha_hasta acotan por fecha de atención (incluidas); para un solo día se envía el mismo valor en ambas.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Cantidad de resultados" default(50)
// @Param offset query int false "Resultados que se omiten" default(0)
// @Param local_id query int false "Filtra por local (solo usuarios sin local asignado)"
// @Param fecha_desde query string false "Primer día a incluir, YYYY-MM-DD"
// @Param fecha_hasta query string false "Último día a incluir, YYYY-MM-DD"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse "Error de validación: local_id o fechas inválidas"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al listar atenciones"
// @Router /bd/atenciones [get]
func (h *Container) GetAtenciones(c *gin.Context) {
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	localID := scope.LocalID
	if localID == nil && strings.TrimSpace(c.Query("local_id")) != "" {
		id, err := strconv.Atoi(c.Query("local_id"))
		if err != nil || id <= 0 {
			utils.RespondError(c, 400, "local_id invalido")
			return
		}
		localID = &id
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	desde, hasta, e := services.FiltroFechasAtencion(c.Query("fecha_desde"), c.Query("fecha_hasta"))
	if e != nil {
		utils.RespondError(c, 400, e.Error())
		return
	}
	xs, e := h.Atenciones.Listar(localID, desde, hasta, limit, offset)
	if e != nil {
		utils.RespondError(c, 500, e.Error())
		return
	}
	total, e := h.Atenciones.Contar(localID, desde, hasta)
	if e != nil {
		utils.RespondError(c, 500, e.Error())
		return
	}
	utils.Respond(c, 200, gin.H{"atenciones": xs, "total": total, "limit": limit, "offset": offset})
}

// PutAtencion actualiza una atencion mientras tenga insumos pendientes de deduccion.
func (h *Container) PutAtencion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, 400, "id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	var req registrarAtencionRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, 400, err.Error())
		return
	}
	ins := make([]pgsqlrepo.AtencionInsumoInput, 0, len(req.Insumos))
	for _, item := range req.Insumos {
		ins = append(ins, pgsqlrepo.AtencionInsumoInput{ItemID: item.ItemID, Cantidad: item.Cantidad})
	}
	a, err := h.Atenciones.Actualizar(services.ActualizarAtencionInput{
		AtencionID: id, LocalID: scope.LocalID, EmpleadoID: req.EmpleadoID,
		EmpleadoNombre: req.EmpleadoNombre, FechaAtencion: req.FechaAtencion,
		HoraInicio: req.HoraInicio, HoraFin: req.HoraFin, Lugar: req.Lugar,
		Observaciones: req.Observaciones, ConfirmarStockInsuficiente: req.ConfirmarStockInsuficiente,
		Actor: consumoActor(c), Insumos: ins,
	})
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "no encontrada") {
			status = http.StatusNotFound
		} else if strings.Contains(strings.ToLower(err.Error()), "solo se pueden editar") || strings.Contains(strings.ToLower(err.Error()), "stock insuficiente") {
			status = http.StatusConflict
		} else if strings.Contains(strings.ToLower(err.Error()), "acceso") {
			status = http.StatusForbidden
		}
		utils.RespondError(c, status, err.Error())
		return
	}
	utils.Respond(c, 200, a)
}

type desactivarAtencionRequest struct {
	RestituirStock bool `json:"restituir_stock"`
}

func (h *Container) PostDesactivarAtencion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, 400, "id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	var req desactivarAtencionRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, 400, err.Error())
		return
	}
	if err = h.Atenciones.Desactivar(id, scope.LocalID, req.RestituirStock, consumoActor(c)); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "no encontrada") {
			status = http.StatusNotFound
		} else if strings.Contains(strings.ToLower(err.Error()), "acceso") {
			status = http.StatusForbidden
		}
		utils.RespondError(c, status, err.Error())
		return
	}
	utils.Respond(c, 200, gin.H{"id": id, "activo": false, "stock_restituido": req.RestituirStock})
}

// GetAtencion godoc
// @Summary Obtener atención
// @Description Devuelve una atención con sus insumos registrados.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de atención"
// @Success 200 {object} utils.APIResponse
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 404 {object} utils.APIResponse "Atención no encontrada"
// @Router /bd/atenciones/{id} [get]
func (h *Container) GetAtencion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, 400, "id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	a, err := h.Atenciones.Obtener(id)
	if err != nil {
		utils.RespondError(c, 404, "atencion no encontrada")
		return
	}
	if scope.LocalID != nil && (a.LocalID == nil || *a.LocalID != *scope.LocalID) {
		utils.RespondError(c, 403, "no tienes acceso a ese local")
		return
	}
	utils.Respond(c, 200, a)
}

// GetPersonalSugerencias godoc
// @Summary Sugerir personal
// @Description Busca nombres coincidentes entre los empleados activos accesibles en el local.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param q query string true "Texto de búsqueda (mínimo 2 caracteres)"
// @Success 200 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al buscar personal"
// @Router /bd/personal/sugerencias [get]
func (h *Container) GetPersonalSugerencias(c *gin.Context) {
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	xs, e := h.Atenciones.SugerenciasPersonal(c.Query("q"), scope.LocalID)
	if e != nil {
		utils.RespondError(c, 500, e.Error())
		return
	}
	utils.Respond(c, 200, gin.H{"personal": xs})
}

// @Summary Listar insumos sugeridos de una reserva
// @Description Devuelve los insumos configurados para el servicio de la reserva.
// @Tags Atenciones
// @Produce json
// @Security BearerAuth
// @Param reserva_id query int true "ID de reserva"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse "reserva_id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al consultar insumos"
// @Router /bd/atenciones/insumos-sugeridos [get]
func (h *Container) GetInsumosSugeridos(c *gin.Context) {
	id, e := strconv.Atoi(c.Query("reserva_id"))
	if e != nil || id <= 0 {
		utils.RespondError(c, 400, "reserva_id invalido")
		return
	}
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, 403, "no autorizado")
		return
	}
	xs, e := h.Atenciones.InsumosSugeridos(id, scope.LocalID)
	if e != nil {
		utils.RespondError(c, 500, e.Error())
		return
	}
	utils.Respond(c, 200, gin.H{"insumos": xs})
}
