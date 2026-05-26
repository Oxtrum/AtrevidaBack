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
	ComboID       int      `json:"combo_id" binding:"required" example:"12"`
	ServicioID    *int     `json:"servicio_id" example:"8"`
	ServicioTexto string   `json:"servicio_texto" example:"Masaje relajante personalizado"`
	Tiempo        string   `json:"tiempo" example:"01:00"`
	Costo         *float64 `json:"costo" example:"250"`
	Sesiones      int      `json:"sesiones" example:"2"`
	Orden         int      `json:"orden" example:"1"`
}

type actualizarComboServicioRequest struct {
	ServicioID    *int     `json:"servicio_id" example:"8"`
	ServicioTexto *string  `json:"servicio_texto" example:"Masaje relajante personalizado"`
	Tiempo        *string  `json:"tiempo" example:"01:00"`
	Costo         *float64 `json:"costo" example:"250"`
	Sesiones      *int     `json:"sesiones" example:"2"`
	Orden         *int     `json:"orden" example:"1"`
}

// GetComboServicioByID godoc
// @Summary Obtener servicio de combo por ID
// @Description Devuelve un item de combo_servicios por su identificador.
// @Tags Combo Servicios BD
// @Produce json
// @Param id path int true "ID del item combo_servicios" example(15)
// @Success 200 {object} utils.APIResponse{data=comboServicioItemResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
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
// @Description Devuelve los items de combo_servicios asociados a un combo activo.
// @Tags Combo Servicios BD
// @Produce json
// @Param combo_id path int true "ID del combo" example(12)
// @Success 200 {object} utils.APIResponse{data=comboServicioListResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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
// @Description Crea un item dentro de combo_servicios validando que el combo padre enviado en el body exista y este activo.
// @Tags Combo Servicios BD
// @Accept json
// @Produce json
// @Param payload body crearComboServicioRequest true "Datos del servicio del combo"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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
// @Description Actualiza campos de un item de combo_servicios. No permite cambiar el combo padre.
// @Tags Combo Servicios BD
// @Accept json
// @Produce json
// @Param id path int true "ID del item combo_servicios" example(15)
// @Param payload body actualizarComboServicioRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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
// @Description Elimina un item de combo_servicios por su identificador.
// @Tags Combo Servicios BD
// @Produce json
// @Param id path int true "ID del item combo_servicios" example(15)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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
