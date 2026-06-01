package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type crearComboServicioRequest struct {
	// ID del combo padre
	ComboID int `json:"combo_id" binding:"required" example:"12"`
	// ID del servicio en BD (opcional si se envía servicio_texto)
	ServicioID *int `json:"servicio_id" example:"8"`
	// Nombre del servicio personalizado (opcional si se envía servicio_id)
	ServicioTexto string `json:"servicio_texto" example:"Masaje relajante personalizado"`
	// Duración del servicio (HH:MM)
	Tiempo string `json:"tiempo" example:"01:00"`
	// Costo del servicio
	Costo *float64 `json:"costo" example:"250"`
	// Cantidad de sesiones
	Sesiones int `json:"sesiones" example:"2"`
	// Orden de aparición dentro del combo
	Orden int `json:"orden" example:"1"`
}

type actualizarComboServicioRequest struct {
	// ID del servicio en BD (opcional)
	ServicioID *int `json:"servicio_id" example:"8"`
	// Nombre del servicio personalizado (opcional)
	ServicioTexto *string `json:"servicio_texto" example:"Masaje relajante personalizado"`
	// Duración del servicio HH:MM (opcional)
	Tiempo *string `json:"tiempo" example:"01:00"`
	// Costo del servicio (opcional)
	Costo *float64 `json:"costo" example:"250"`
	// Cantidad de sesiones (opcional)
	Sesiones *int `json:"sesiones" example:"2"`
	// Orden de aparición (opcional)
	Orden *int `json:"orden" example:"1"`
}

// GetComboServicioByID godoc
// @Summary Obtener servicio de combo por ID
// @Description Devuelve un item de combo_servicios por su ID. Param: id (requerido, path). Response: servicio (ComboServicioDetallePG con: id, combo_id, combo_nombre, servicio_id, servicio_texto, servicio_nombre, tiempo HH:MM, costo, sesiones, orden).
// @Tags Combo Servicios BD
// @Produce json
// @Param id path int true "ID del item combo_servicios" example(15)
// @Success 200 {object} utils.APIResponse{data=comboServicioItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/servicios/{id} [get]
func (h *Container) GetComboServicioByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	item, err := h.ComboServiciosPG.GetByID(id)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, comboServicioItemResponse{Servicio: item})
}

// GetComboServiciosByCombo godoc
// @Summary Listar servicios de un combo
// @Description Devuelve los items de combo_servicios asociados a un combo activo. Param: combo_id (requerido, path). Response: total (int), combo_id (int), servicios ([]ComboServicioDetallePG con: id, combo_id, combo_nombre, servicio_id, servicio_texto, servicio_nombre, tiempo HH:MM, costo, sesiones, orden).
// @Tags Combo Servicios BD
// @Produce json
// @Param combo_id path int true "ID del combo" example(12)
// @Success 200 {object} utils.APIResponse{data=comboServicioListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: combo_id invalido"
// @Failure 404 {object} utils.APIResponse "Combo no encontrado o inactivo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/{combo_id}/servicios [get]
func (h *Container) GetComboServiciosByCombo(c *gin.Context) {
	comboID, err := strconv.Atoi(c.Param("combo_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "combo_id invalido")
		return
	}

	items, err := h.ComboServiciosPG.GetByComboID(comboID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrado") || strings.Contains(err.Error(), "inactivo") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, comboServicioListResponse{
		Total:     len(items),
		ComboID:   comboID,
		Servicios: items,
	})
}

// CreateComboServicio godoc
// @Summary Crear servicio de combo
// @Description Crea un item dentro de combo_servicios. Body: combo_id ID del combo padre (requerido), servicio_id (opcional si se envia servicio_texto), servicio_texto (opcional si se envia servicio_id), tiempo HH:MM (opcional), costo (opcional), sesiones entero positivo default 1 (opcional), orden posicion (opcional). Response: id (int ID del item creado).
// @Tags Combo Servicios BD
// @Accept json
// @Produce json
// @Param payload body crearComboServicioRequest true "Datos del servicio del combo"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: combo_id requerido, sesiones debe ser positivo, debe enviar servicio_id o servicio_texto"
// @Failure 404 {object} utils.APIResponse "Combo no encontrado o inactivo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/servicios [post]
func (h *Container) CreateComboServicio(c *gin.Context) {
	var req crearComboServicioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	id, err := h.ComboServiciosPG.Create(services.CrearComboServicioPGInput{
		ComboID:       req.ComboID,
		ServicioID:    req.ServicioID,
		ServicioTexto: req.ServicioTexto,
		Tiempo:        req.Tiempo,
		Costo:         req.Costo,
		Sesiones:      req.Sesiones,
		Orden:         req.Orden,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrado") || strings.Contains(err.Error(), "inactivo") {
			status = http.StatusNotFound
		} else if strings.Contains(err.Error(), "debe") || strings.Contains(err.Error(), "positivo") {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}

// PatchComboServicio godoc
// @Summary Actualizar servicio de combo
// @Description Actualiza campos de un item de combo_servicios. No permite cambiar el combo padre. Param: id (requerido, path). Body: servicio_id (opcional), servicio_texto (opcional), tiempo HH:MM (opcional), costo (opcional), sesiones (opcional), orden (opcional). Response: mensaje string.
// @Tags Combo Servicios BD
// @Accept json
// @Produce json
// @Param id path int true "ID del item combo_servicios" example(15)
// @Param payload body actualizarComboServicioRequest true "Campos a actualizar (todos opcionales, al menos uno requerido)"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, sin campos a modificar, sesiones debe ser positivo"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/servicios/{id} [patch]
func (h *Container) PatchComboServicio(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	var req actualizarComboServicioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.ServicioID == nil && req.ServicioTexto == nil && req.Tiempo == nil &&
		req.Costo == nil && req.Sesiones == nil && req.Orden == nil {
		utils.RespondError(c, http.StatusBadRequest,
			"debe especificarse al menos un campo a modificar")
		return
	}

	err = h.ComboServiciosPG.Update(services.ActualizarComboServicioPGInput{
		ID:            id,
		ServicioID:    req.ServicioID,
		ServicioTexto: req.ServicioTexto,
		Tiempo:        req.Tiempo,
		Costo:         req.Costo,
		Sesiones:      req.Sesiones,
		Orden:         req.Orden,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrado") || strings.Contains(err.Error(), "inactivo") {
			status = http.StatusNotFound
		} else if strings.Contains(err.Error(), "debe") || strings.Contains(err.Error(), "positivo") {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "servicio del combo actualizado correctamente"})
}

// DeleteComboServicio godoc
// @Summary Eliminar servicio de combo
// @Description Elimina un item de combo_servicios por su ID. Param: id (requerido, path). Response: mensaje string.
// @Tags Combo Servicios BD
// @Produce json
// @Param id path int true "ID del item combo_servicios" example(15)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/servicios/{id} [delete]
func (h *Container) DeleteComboServicio(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	err = h.ComboServiciosPG.Delete(id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no encontrado") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "servicio del combo eliminado correctamente"})
}
