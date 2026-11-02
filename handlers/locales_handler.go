package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// GetLocales godoc
// @Summary Listar locales
// @Description Devuelve todos los locales registrados en BD. Sin filtros. Response: total (int), locales ([]LocalConEspacios con: id, nombre, activo, espacios []TipoEspacioLocal con tipo_espacio M/B y cantidad_espacios, horarios []LocalHorarioPG con id, local_id, dia_semana 1-7, hora_desde HH:MM, hora_hasta HH:MM, activo).
// @Tags Locales
// @Produce json
// @Success 200 {object} utils.APIResponse{data=localListResponse}
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales [get]
func (h *Container) GetLocales(c *gin.Context) {
	resultado, err := h.LocalesPG.GetLocales(c.Request.Context())
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, localListResponse{
		Total:   len(resultado),
		Locales: resultado,
	})
}

// GetLocalById godoc
// @Summary Obtener local por ID
// @Description Devuelve un local por su ID. Param: id (requerido, path). Response: total (1), local (LocalConEspacios con: id, nombre, activo, espacios, horarios).
// @Tags Locales
// @Produce json
// @Param id path int true "ID del local" example(3)
// @Success 200 {object} utils.APIResponse{data=localItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales/{id} [get]
func (h *Container) GetLocalById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	res, err := h.LocalesPG.GetLocalById(id)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, localItemResponse{
		Total: 1,
		Local: res,
	})
}

type crearLocalRequest struct {
	// Nombre del local
	Nombre string `json:"nombre"   binding:"required" example:"SAN MARTIN"`
	// Espacios del local (opcional)
	Espacios []espacioRequest `json:"espacios"`
}

type espacioRequest struct {
	// Tipo de espacio: M (mesa) o B (bicicleta)
	TipoEspacio string `json:"tipo_espacio"       binding:"required" example:"M"`
	// Cantidad de espacios de este tipo
	CantidadEspacios int `json:"cantidad_espacios"  binding:"required,min=1" example:"6"`
}

// PostLocal godoc
// @Summary Crear local
// @Description Crea un local y opcionalmente sus espacios. Body: nombre (requerido), espacios (opcional, array con: tipo_espacio M/B requerido, cantidad_espacios entero positivo requerido). Response: id (int ID del local creado).
// @Tags Locales
// @Accept json
// @Produce json
// @Param payload body crearLocalRequest true "Datos del local (nombre + espacios opcionales)"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: nombre requerido, tipo_espacio invalido M/B, cantidad_espacios debe ser >= 1"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales [post]
func (h *Container) PostLocal(c *gin.Context) {
	var req crearLocalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	for _, e := range req.Espacios {
		t := strings.ToUpper(strings.TrimSpace(e.TipoEspacio))
		if t != "M" && t != "B" {
			utils.RespondError(c, http.StatusBadRequest,
				"tipo_espacio inválido '"+e.TipoEspacio+"', valores permitidos: M, B")
			return
		}
	}

	espacios := make([]services.EspacioInput, 0, len(req.Espacios))
	for _, e := range req.Espacios {
		espacios = append(espacios, services.EspacioInput{
			TipoEspacio:      e.TipoEspacio,
			CantidadEspacios: e.CantidadEspacios,
		})
	}

	id, err := h.LocalesPG.CreateLocal(services.CrearLocalInput{
		Nombre:   req.Nombre,
		Espacios: espacios,
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}

// PATCH /admin/locales/:id
type actualizarLocalRequest struct {
	// Nuevo nombre del local (opcional)
	Nombre *string `json:"nombre" example:"PASEO ARANJUEZ"`
	// Estado activo/inactivo (opcional)
	Activo *bool `json:"activo" example:"true"`
}

// PatchLocal godoc
// @Summary Actualizar local
// @Description Actualiza nombre o estado de un local. Param: id (requerido, path). Body: nombre (opcional), activo true/false (opcional). Response: mensaje string.
// @Tags Locales
// @Accept json
// @Produce json
// @Param id path int true "ID del local" example(3)
// @Param payload body actualizarLocalRequest true "Campos a actualizar (nombre y/o activo)"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, body invalido, sin campos a modificar"
// @Failure 404 {object} utils.APIResponse "Local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales/{id} [patch]
func (h *Container) PatchLocal(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var req actualizarLocalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Nombre == nil && req.Activo == nil {
		utils.RespondError(c, http.StatusBadRequest,
			"debe especificarse al menos un campo a modificar: nombre, activo")
		return
	}

	err = h.LocalesPG.UpdateLocal(services.ActualizarLocalInput{
		ID:     id,
		Nombre: req.Nombre,
		Activo: req.Activo,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrado") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "local actualizado correctamente"})
}

type espacioEdicionRequest struct {
	// Tipo de espacio: M (mesa) o B (bicicleta)
	TipoEspacio string `json:"tipo_espacio" binding:"required" example:"M"`
	// Cantidad total de espacios de este tipo; 0 permitido (mantenimiento)
	CantidadEspacios *int `json:"cantidad_espacios" binding:"required,min=0" example:"4"`
}

type actualizarEspaciosRequest struct {
	// Espacios a fijar; los tipos no enviados quedan como estaban
	Espacios []espacioEdicionRequest `json:"espacios" binding:"required,min=1,dive"`
}

// PutLocalEspacios godoc
// @Summary Actualizar mesas y bicicletas de un local
// @Description Fija la cantidad de espacios de un local. Param: id (path). Body: espacios (array con tipo_espacio M/B y cantidad_espacios entero >= 0; 0 deja el tipo sin espacios, por ejemplo por mantenimiento). Los tipos no enviados no se tocan. No cancela reservas existentes: si se reduce por debajo de lo ya reservado, solo se bloquean las reservas nuevas. Requiere rol admin_sys o gerencia. Response: mensaje string.
// @Tags Locales
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID del local" example(3)
// @Param payload body actualizarEspaciosRequest true "Espacios a fijar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, tipo_espacio invalido, cantidad negativa o tipo repetido"
// @Failure 401 {object} utils.APIResponse "Token ausente o invalido"
// @Failure 403 {object} utils.APIResponse "Rol sin permiso"
// @Failure 404 {object} utils.APIResponse "Local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales/{id}/espacios [put]
func (h *Container) PutLocalEspacios(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var req actualizarEspaciosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	espacios := make([]services.EspacioInput, 0, len(req.Espacios))
	for _, e := range req.Espacios {
		espacios = append(espacios, services.EspacioInput{TipoEspacio: e.TipoEspacio, CantidadEspacios: *e.CantidadEspacios})
	}

	if err := h.LocalesPG.UpdateEspacios(id, espacios); err != nil {
		status := http.StatusInternalServerError
		switch {
		case strings.Contains(err.Error(), "no encontrado"):
			status = http.StatusNotFound
		case strings.Contains(err.Error(), "tipo_espacio") ||
			strings.Contains(err.Error(), "cantidad_espacios") ||
			strings.Contains(err.Error(), "al menos un tipo"):
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "espacios actualizados correctamente"})
}

// DeleteLocal godoc
// @Summary Eliminar local
// @Description Realiza borrado logico de un local (activo=false). Param: id (requerido, path). Response: mensaje string.
// @Tags Locales
// @Produce json
// @Param id path int true "ID del local" example(3)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Local no encontrado o ya inactivo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales/{id} [delete]
func (h *Container) DeleteLocal(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id invÃ¡lido")
		return
	}

	err = h.LocalesPG.DeleteLocal(id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrado") || strings.Contains(err.Error(), "inactivo") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "local eliminado correctamente"})
}
