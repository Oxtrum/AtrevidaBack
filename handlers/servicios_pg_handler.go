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

// GetServiciosPG godoc
// @Summary Listar servicios desde base de datos
// @Description Devuelve servicios persistidos en PostgreSQL con filtros opcionales.
// @Tags Servicios BD
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre" example(depila)
// @Param categoria query string false "Busqueda parcial por categoria" example(Corporal)
// @Param local query string false "Local" Enums(SAN MARTIN,PASEO ARANJUEZ) example(SAN MARTIN)
// @Param sesiones query int false "Numero exacto de sesiones" example(6)
// @Param requiere_evaluacion query bool false "Filtrar por servicios que requieren evaluacion" example(true)
// @Success 200 {object} utils.APIResponse{data=servicioListResponse}
// @Failure 400 {object} utils.APIResponse
// @Router /bd/servicios [get]
func (h *Container) GetServiciosPG(c *gin.Context) {
	sesiones := 0
	if raw := strings.TrimSpace(c.Query("sesiones")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			utils.RespondError(c, http.StatusBadRequest,
				"sesiones debe ser un número entero positivo")
			return
		}
		sesiones = n
	}

	var requiereEvaluacion *bool
	if raw := strings.TrimSpace(c.Query("requiere_evaluacion")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest,
				"requiere_evaluacion debe ser true o false")
			return
		}
		requiereEvaluacion = &v
	}

	local := strings.ToUpper(strings.TrimSpace(c.Query("local")))
	if local != "" && local != "SAN MARTIN" && local != "PASEO ARANJUEZ" {
		utils.RespondError(c, http.StatusBadRequest,
			"local inválido, valores permitidos: SAN MARTIN, PASEO ARANJUEZ")
		return
	}

	filtro := services.FiltroServicios{
		Nombre:             strings.TrimSpace(c.Query("nombre")),
		Categoria:          strings.TrimSpace(c.Query("categoria")),
		Local:              local,
		Sesiones:           sesiones,
		RequiereEvaluacion: requiereEvaluacion,
	}

	resultado := h.ServiciosPG.GetServiciosFiltrados(filtro)
	if resultado == nil {
		resultado = []models.ServicioItem{}
	}

	utils.Respond(c, http.StatusOK, servicioListResponse{
		Total: len(resultado),
		Filtros: servicioFiltrosResponse{
			Nombre:             filtro.Nombre,
			Categoria:          filtro.Categoria,
			Local:              filtro.Local,
			Sesiones:           filtro.Sesiones,
			RequiereEvaluacion: filtro.RequiereEvaluacion,
		},
		Servicios: resultado,
	})
}

// GetServicioPGByID godoc
// @Summary Obtener servicio por ID
// @Description Devuelve un servicio de PostgreSQL por su identificador.
// @Tags Servicios BD
// @Produce json
// @Param id path int true "ID del servicio" example(8)
// @Success 200 {object} utils.APIResponse{data=servicioItemResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Router /bd/servicios/{id} [get]
func (h *Container) GetServicioPGByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	resultado, err := h.ServiciosPG.GetServicioByID(id)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, servicioItemResponse{Servicio: resultado})
}

// POST /admin/servicios
type crearServicioRequest struct {
	Nombre               string   `json:"nombre"                binding:"required" example:"Depilacion Laser Piernas"`
	CategoriaNombre      string   `json:"categoria"             binding:"required" example:"Corporal"`
	Tiempo               string   `json:"tiempo" example:"01:00"`
	Costo                *float64 `json:"costo" example:"350"`
	Sesiones             int      `json:"sesiones" example:"6"`
	TipoEspacioRequerido *string  `json:"tipo_espacio_requerido" example:"M"` // "M" | "B" | nil
	RequiereEvaluacion   *bool    `json:"requiere_evaluacion" example:"true"`
	LocalNombre          string   `json:"local" example:"SAN MARTIN"` // opcional
}

// CreateServicio godoc
// @Summary Crear servicio
// @Description Crea un nuevo servicio en PostgreSQL y opcionalmente lo asocia a un local.
// @Tags Servicios BD
// @Accept json
// @Produce json
// @Param payload body crearServicioRequest true "Datos del servicio"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/servicios [post]
func (h *Container) CreateServicio(c *gin.Context) {
	var req crearServicioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.TipoEspacioRequerido != nil {
		t := strings.ToUpper(*req.TipoEspacioRequerido)
		if t != "M" && t != "B" {
			utils.RespondError(c, http.StatusBadRequest,
				"tipo_espacio_requerido inválido, valores permitidos: M, B")
			return
		}
	}

	sesiones := req.Sesiones
	if sesiones < 1 {
		sesiones = 1
	}
	requiereEvaluacion := true
	if req.RequiereEvaluacion != nil {
		requiereEvaluacion = *req.RequiereEvaluacion
	}

	id, err := h.ServiciosPG.CreateServicio(services.CrearServicioPGInput{
		Nombre:               req.Nombre,
		CategoriaNombre:      req.CategoriaNombre,
		Tiempo:               req.Tiempo,
		Costo:                req.Costo,
		Sesiones:             sesiones,
		TipoEspacioRequerido: req.TipoEspacioRequerido,
		RequiereEvaluacion:   requiereEvaluacion,
		LocalNombre:          req.LocalNombre,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrada") ||
			strings.Contains(err.Error(), "no tiene espacios") {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}

// PATCH /admin/servicios/:id
type actualizarServicioRequest struct {
	Nombre               *string  `json:"nombre" example:"Depilacion Laser Piernas Premium"`
	CategoriaNombre      *string  `json:"categoria" example:"Corporal"`
	Tiempo               *string  `json:"tiempo" example:"01:15"`
	Costo                *float64 `json:"costo" example:"420"`
	Sesiones             *int     `json:"sesiones" example:"8"`
	TipoEspacioRequerido *string  `json:"tipo_espacio_requerido" example:"M"`
	RequiereEvaluacion   *bool    `json:"requiere_evaluacion" example:"false"`
	Activo               *bool    `json:"activo" example:"true"`
}

// UpdateServicio godoc
// @Summary Actualizar servicio
// @Description Actualiza un servicio existente en PostgreSQL.
// @Tags Servicios BD
// @Accept json
// @Produce json
// @Param id path int true "ID del servicio" example(8)
// @Param payload body actualizarServicioRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/servicios/{id} [patch]
func (h *Container) UpdateServicio(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var req actualizarServicioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Nombre == nil && req.CategoriaNombre == nil && req.Tiempo == nil &&
		req.Costo == nil && req.Sesiones == nil &&
		req.TipoEspacioRequerido == nil && req.RequiereEvaluacion == nil && req.Activo == nil {
		utils.RespondError(c, http.StatusBadRequest,
			"debe especificarse al menos un campo a modificar")
		return
	}

	if req.TipoEspacioRequerido != nil {
		t := strings.ToUpper(*req.TipoEspacioRequerido)
		if t != "M" && t != "B" {
			utils.RespondError(c, http.StatusBadRequest,
				"tipo_espacio_requerido inválido, valores permitidos: M, B")
			return
		}
	}

	err = h.ServiciosPG.UpdateServicio(services.ActualizarServicioPGInput{
		ID:                   id,
		Nombre:               req.Nombre,
		CategoriaNombre:      req.CategoriaNombre,
		Tiempo:               req.Tiempo,
		Costo:                req.Costo,
		Sesiones:             req.Sesiones,
		TipoEspacioRequerido: req.TipoEspacioRequerido,
		RequiereEvaluacion:   req.RequiereEvaluacion,
		Activo:               req.Activo,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrad") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "servicio actualizado correctamente"})
}

// POST /admin/servicios/:id/activar
type activarServicioRequest struct {
	Local string `json:"local" binding:"required" example:"PASEO ARANJUEZ"`
}

// ActivarServicioEnLocal godoc
// @Summary Activar servicio en local
// @Description Asocia un servicio existente a un local determinado.
// @Tags Servicios BD
// @Accept json
// @Produce json
// @Param id path int true "ID del servicio" example(8)
// @Param payload body activarServicioRequest true "Local donde activar el servicio"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/servicios/local/{id} [post]
func (h *Container) ActivarServicioEnLocal(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var req activarServicioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.ServiciosPG.ActivarServicioEnLocal(id, req.Local)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrado") ||
			strings.Contains(err.Error(), "no tiene espacios") {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "servicio activado en local correctamente"})
}

// DeleteServicio godoc
// @Summary Eliminar servicio
// @Description Realiza el borrado logico de un servicio estableciendo activo en false.
// @Tags Servicios BD
// @Produce json
// @Param id path int true "ID del servicio" example(8)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/servicios/{id} [delete]
func (h *Container) DeleteServicio(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id invÃ¡lido")
		return
	}

	err = h.ServiciosPG.DeleteServicio(id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrado") || strings.Contains(err.Error(), "inactivo") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "servicio eliminado correctamente"})
}
