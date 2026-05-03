package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// GET /bd/locales
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

// GET /bd/locales/:id
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
