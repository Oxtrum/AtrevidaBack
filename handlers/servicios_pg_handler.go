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

// GET /bd/servicios
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

	local := strings.ToUpper(strings.TrimSpace(c.Query("local")))
	if local != "" && local != "SAN MARTIN" && local != "PASEO ARANJUEZ" {
		utils.RespondError(c, http.StatusBadRequest,
			"local inválido, valores permitidos: SAN MARTIN, PASEO ARANJUEZ")
		return
	}

	filtro := services.FiltroServicios{
		Nombre:    strings.TrimSpace(c.Query("nombre")),
		Categoria: strings.TrimSpace(c.Query("categoria")),
		Local:     local,
		Sesiones:  sesiones,
	}

	resultado := h.ServiciosPG.GetServiciosFiltrados(filtro)
	if resultado == nil {
		resultado = []models.ServicioItem{}
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"total": len(resultado),
		"filtros": gin.H{
			"nombre":    filtro.Nombre,
			"categoria": filtro.Categoria,
			"local":     filtro.Local,
			"sesiones":  filtro.Sesiones,
		},
		"servicios": resultado,
	})
}

// GET /bd/servicios/:id
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

	utils.Respond(c, http.StatusOK, gin.H{"servicio": resultado})
}

// POST /admin/servicios
type crearServicioRequest struct {
	Nombre               string   `json:"nombre"                binding:"required"`
	CategoriaNombre      string   `json:"categoria"             binding:"required"`
	Tiempo               string   `json:"tiempo"`
	Costo                *float64 `json:"costo"`
	Sesiones             int      `json:"sesiones"`
	TipoEspacioRequerido *string  `json:"tipo_espacio_requerido"` // "M" | "B" | nil
	LocalNombre          string   `json:"local"`                  // opcional
}

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

	id, err := h.ServiciosPG.CreateServicio(services.CrearServicioPGInput{
		Nombre:               req.Nombre,
		CategoriaNombre:      req.CategoriaNombre,
		Tiempo:               req.Tiempo,
		Costo:                req.Costo,
		Sesiones:             sesiones,
		TipoEspacioRequerido: req.TipoEspacioRequerido,
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

	utils.Respond(c, http.StatusOK, gin.H{"id": id})
}

// PATCH /admin/servicios/:id
type actualizarServicioRequest struct {
	Nombre               *string  `json:"nombre"`
	CategoriaNombre      *string  `json:"categoria"`
	Tiempo               *string  `json:"tiempo"`
	Costo                *float64 `json:"costo"`
	Sesiones             *int     `json:"sesiones"`
	TipoEspacioRequerido *string  `json:"tipo_espacio_requerido"`
	Activo               *bool    `json:"activo"`
}

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
		req.TipoEspacioRequerido == nil && req.Activo == nil {
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

	utils.Respond(c, http.StatusOK, gin.H{"mensaje": "servicio actualizado correctamente"})
}

// POST /admin/servicios/:id/activar
type activarServicioRequest struct {
	Local string `json:"local" binding:"required"`
}

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

	utils.Respond(c, http.StatusOK, gin.H{"mensaje": "servicio activado en local correctamente"})
}
