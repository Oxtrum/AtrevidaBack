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
// @Description Devuelve servicios desde PostgreSQL con filtros. Filtros: nombre busqueda parcial (opcional), categoria busqueda parcial (opcional), local SAN MARTIN/PASEO ARANJUEZ (opcional), sesiones numero exacto (opcional), requiere_evaluacion true/false (opcional). Response: total (int), filtros (objeto con nombre, categoria, local, sesiones, requiere_evaluacion), servicios ([]ServicioItem con: id, nombre, categoria, local, tiempo HH:MM, costo, sesiones, tipoEspacio M/B, requiere_evaluacion).
// @Tags Servicios BD
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre" example(depila)
// @Param categoria query string false "Busqueda parcial por categoria" example(Corporal)
// @Param local query string false "Local" Enums(SAN MARTIN,PASEO ARANJUEZ) example(SAN MARTIN)
// @Param sesiones query int false "Numero exacto de sesiones" example(6)
// @Param requiere_evaluacion query bool false "Filtrar por servicios que requieren evaluacion" example(true)
// @Success 200 {object} utils.APIResponse{data=servicioListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: sesiones debe ser entero positivo, local invalido, requiere_evaluacion debe ser true/false"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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
// @Description Devuelve un servicio por su ID. Param: id (requerido, path). Response: servicio (ServicioItem con: id, nombre, categoria, local, tiempo HH:MM, costo, sesiones, tipoEspacio M/B, requiere_evaluacion).
// @Tags Servicios BD
// @Produce json
// @Param id path int true "ID del servicio" example(8)
// @Success 200 {object} utils.APIResponse{data=servicioItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Servicio no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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
	// Nombre del servicio
	Nombre string `json:"nombre"                binding:"required" example:"Depilacion Laser Piernas"`
	// Nombre de la categoría
	CategoriaNombre string `json:"categoria"             binding:"required" example:"Corporal"`
	// Duración del servicio (HH:MM)
	Tiempo string `json:"tiempo" example:"01:00"`
	// Costo del servicio
	Costo *float64 `json:"costo" example:"350"`
	// Cantidad de sesiones (default 1)
	Sesiones int `json:"sesiones" example:"6"`
	// Tipo de espacio requerido: M (mesa) o B (bicicleta)
	TipoEspacioRequerido *string `json:"tipo_espacio_requerido" example:"M"`
	// Indica si requiere evaluación previa (default true)
	RequiereEvaluacion *bool `json:"requiere_evaluacion" example:"true"`
	// Nombre del local donde activar el servicio (opcional)
	LocalNombre string `json:"local" example:"SAN MARTIN"`
}

// CreateServicio godoc
// @Summary Crear servicio
// @Description Crea un servicio en PostgreSQL y opcionalmente lo asocia a un local. Si se envia local, la categoria del servicio debe estar asociada a ese local en categorias_locales. Body: nombre (requerido), categoria (requerido), tiempo HH:MM (opcional), costo (opcional), sesiones entero positivo default 1 (opcional), tipo_espacio_requerido M/B (opcional), requiere_evaluacion true/false default true (opcional), local para activar (opcional). Response: id (int ID del servicio creado).
// @Tags Servicios BD
// @Accept json
// @Produce json
// @Param payload body crearServicioRequest true "Datos del servicio"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: nombre/categoria requerido, tipo_espacio_requerido invalido, local no encontrado, categoria no disponible para el local, local sin espacios"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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
			strings.Contains(err.Error(), "no tiene espacios") ||
			strings.Contains(err.Error(), "no disponible") {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}

// PATCH /admin/servicios/:id
type actualizarServicioRequest struct {
	// Nuevo nombre (opcional)
	Nombre *string `json:"nombre" example:"Depilacion Laser Piernas Premium"`
	// Nueva categoría (opcional)
	CategoriaNombre *string `json:"categoria" example:"Corporal"`
	// Nueva duración HH:MM (opcional)
	Tiempo *string `json:"tiempo" example:"01:15"`
	// Nuevo costo (opcional)
	Costo *float64 `json:"costo" example:"420"`
	// Nueva cantidad de sesiones (opcional)
	Sesiones *int `json:"sesiones" example:"8"`
	// Nuevo tipo de espacio requerido M/B (opcional)
	TipoEspacioRequerido *string `json:"tipo_espacio_requerido" example:"M"`
	// Nuevo estado de requiere evaluación (opcional)
	RequiereEvaluacion *bool `json:"requiere_evaluacion" example:"false"`
	// Nuevo estado activo/inactivo (opcional)
	Activo *bool `json:"activo" example:"true"`
}

// UpdateServicio godoc
// @Summary Actualizar servicio
// @Description Actualiza un servicio existente. Solo se actualizan los campos enviados. Si se cambia categoria y el servicio ya esta asociado a locales, la nueva categoria debe estar asociada a esos locales en categorias_locales. Param: id (requerido, path). Body: nombre (opcional), categoria (opcional), tiempo HH:MM (opcional), costo (opcional), sesiones (opcional), tipo_espacio_requerido M/B (opcional), requiere_evaluacion true/false (opcional), activo true/false (opcional). Response: mensaje string.
// @Tags Servicios BD
// @Accept json
// @Produce json
// @Param id path int true "ID del servicio" example(8)
// @Param payload body actualizarServicioRequest true "Campos a actualizar (todos opcionales, al menos uno requerido)"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, tipo_espacio_requerido invalido, categoria no disponible para locales del servicio, sin campos a modificar"
// @Failure 404 {object} utils.APIResponse "Servicio no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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
		if strings.Contains(err.Error(), "no disponible") {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "servicio actualizado correctamente"})
}

// POST /admin/servicios/:id/activar
type activarServicioRequest struct {
	// Nombre del local donde activar el servicio
	Local string `json:"local" binding:"required" example:"PASEO ARANJUEZ"`
}

// ActivarServicioEnLocal godoc
// @Summary Activar servicio en local
// @Description Asocia un servicio existente a un local. La categoria del servicio debe estar asociada a ese local en categorias_locales. Param: id del servicio (requerido, path). Body: local (requerido). Response: mensaje string.
// @Tags Servicios BD
// @Accept json
// @Produce json
// @Param id path int true "ID del servicio" example(8)
// @Param payload body activarServicioRequest true "Local donde activar el servicio"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, local requerido, servicio/local no encontrado, categoria no disponible para el local, local sin espacios"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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
			strings.Contains(err.Error(), "no tiene espacios") ||
			strings.Contains(err.Error(), "no disponible") {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "servicio activado en local correctamente"})
}

// DeleteServicio godoc
// @Summary Eliminar servicio
// @Description Realiza borrado logico de un servicio (activo=false). Param: id (requerido, path). Response: mensaje string.
// @Tags Servicios BD
// @Produce json
// @Param id path int true "ID del servicio" example(8)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Servicio no encontrado o ya inactivo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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
