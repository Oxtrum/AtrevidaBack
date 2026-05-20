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
// @Description Devuelve todos los locales registrados en la base de datos.
// @Tags Locales
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/locales [get]
func (h *Container) GetLocales(c *gin.Context) {
	resultado, err := h.LocalesPG.GetLocales()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"total":   len(resultado),
		"locales": resultado,
	})
}

// GetLocalById godoc
// @Summary Obtener local por ID
// @Description Devuelve un local de PostgreSQL por su identificador.
// @Tags Locales
// @Produce json
// @Param id path int true "ID del local"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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

	utils.Respond(c, http.StatusOK, gin.H{
		"total": 1,
		"local": res,
	})
}

type crearLocalRequest struct {
	Nombre   string           `json:"nombre"   binding:"required"`
	Espacios []espacioRequest `json:"espacios"` // opcional
}

type espacioRequest struct {
	TipoEspacio      string `json:"tipo_espacio"       binding:"required"`
	CantidadEspacios int    `json:"cantidad_espacios"  binding:"required,min=1"`
}

// PostLocal godoc
// @Summary Crear local
// @Description Crea un local y opcionalmente sus espacios asociados.
// @Tags Locales
// @Accept json
// @Produce json
// @Param payload body crearLocalRequest true "Datos del local"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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

	utils.Respond(c, http.StatusOK, gin.H{"id": id})
}

// PATCH /admin/locales/:id
type actualizarLocalRequest struct {
	Nombre *string `json:"nombre"`
	Activo *bool   `json:"activo"`
}

// PatchLocal godoc
// @Summary Actualizar local
// @Description Actualiza nombre o estado activo de un local existente.
// @Tags Locales
// @Accept json
// @Produce json
// @Param id path int true "ID del local"
// @Param payload body actualizarLocalRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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

	utils.Respond(c, http.StatusOK, gin.H{"mensaje": "local actualizado correctamente"})
}
