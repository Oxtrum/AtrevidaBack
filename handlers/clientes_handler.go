package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type crearClienteRequest struct {
	Nombre         string `json:"nombre"`
	Apellido       string `json:"apellido"`
	NumeroTelefono string `json:"numero_telefono"`
}

type actualizarClienteRequest struct {
	Nombre         *string `json:"nombre"`
	Apellido       *string `json:"apellido"`
	NumeroTelefono *string `json:"numero_telefono"`
}

// GetClientes godoc
// @Summary Listar clientes
// @Description Devuelve clientes de PostgreSQL con filtros opcionales por nombre, apellido y numero de telefono.
// @Tags Clientes
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre"
// @Param apellido query string false "Busqueda parcial por apellido"
// @Param numero_telefono query string false "Busqueda parcial por numero de telefono"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/clientes [get]
func (h *Container) GetClientes(c *gin.Context) {
	clientes, err := h.ClientesPG.GetClientes(services.FiltroClientes{
		Nombre:         c.Query("nombre"),
		Apellido:       c.Query("apellido"),
		NumeroTelefono: c.Query("numero_telefono"),
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"total": len(clientes),
		"filtros": gin.H{
			"nombre":          strings.TrimSpace(c.Query("nombre")),
			"apellido":        strings.TrimSpace(c.Query("apellido")),
			"numero_telefono": strings.TrimSpace(c.Query("numero_telefono")),
		},
		"clientes": clientes,
	})
}

// GetClienteByID godoc
// @Summary Obtener cliente por ID
// @Description Devuelve un cliente de PostgreSQL por su identificador.
// @Tags Clientes
// @Produce json
// @Param id path int true "ID del cliente"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/clientes/{id} [get]
func (h *Container) GetClienteByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	cliente, err := h.ClientesPG.GetClienteByID(id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "no encontrado") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"cliente": cliente,
	})
}

// CreateCliente godoc
// @Summary Crear cliente
// @Description Crea un nuevo cliente en la base de datos.
// @Tags Clientes
// @Accept json
// @Produce json
// @Param payload body crearClienteRequest true "Datos del cliente"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 409 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/clientes [post]
func (h *Container) CreateCliente(c *gin.Context) {
	var req crearClienteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	if strings.TrimSpace(req.Nombre) == "" ||
		strings.TrimSpace(req.Apellido) == "" ||
		strings.TrimSpace(req.NumeroTelefono) == "" {
		utils.RespondError(c, http.StatusBadRequest, "nombre, apellido y numero_telefono son obligatorios")
		return
	}

	id, err := h.ClientesPG.CreateCliente(services.CrearClienteInput{
		Nombre:         req.Nombre,
		Apellido:       req.Apellido,
		NumeroTelefono: req.NumeroTelefono,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "ya existe") {
			status = http.StatusConflict
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{"id": id})
}

// PatchCliente godoc
// @Summary Actualizar cliente
// @Description Actualiza parcialmente un cliente existente en PostgreSQL.
// @Tags Clientes
// @Accept json
// @Produce json
// @Param id path int true "ID del cliente"
// @Param payload body actualizarClienteRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 409 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/clientes/{id} [patch]
func (h *Container) PatchCliente(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	var req actualizarClienteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	if req.Nombre == nil && req.Apellido == nil && req.NumeroTelefono == nil {
		utils.RespondError(c, http.StatusBadRequest, "debe especificarse al menos un campo a modificar")
		return
	}

	if req.Nombre != nil && strings.TrimSpace(*req.Nombre) == "" {
		utils.RespondError(c, http.StatusBadRequest, "nombre no puede estar vacio")
		return
	}
	if req.Apellido != nil && strings.TrimSpace(*req.Apellido) == "" {
		utils.RespondError(c, http.StatusBadRequest, "apellido no puede estar vacio")
		return
	}
	if req.NumeroTelefono != nil && strings.TrimSpace(*req.NumeroTelefono) == "" {
		utils.RespondError(c, http.StatusBadRequest, "numero_telefono no puede estar vacio")
		return
	}

	err = h.ClientesPG.UpdateCliente(services.ActualizarClienteInput{
		ID:             id,
		Nombre:         req.Nombre,
		Apellido:       req.Apellido,
		NumeroTelefono: req.NumeroTelefono,
	})
	if err != nil {
		status := http.StatusInternalServerError
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "no encontrado") {
			status = http.StatusNotFound
		}
		if strings.Contains(lower, "ya existe") {
			status = http.StatusConflict
		}
		if strings.Contains(lower, "debe especificarse") {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{"mensaje": "cliente actualizado correctamente"})
}

// DeleteCliente godoc
// @Summary Eliminar cliente
// @Description Elimina un cliente de la base de datos.
// @Tags Clientes
// @Produce json
// @Param id path int true "ID del cliente"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/clientes/{id} [delete]
func (h *Container) DeleteCliente(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	err = h.ClientesPG.DeleteCliente(id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "no encontrado") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{"mensaje": "cliente eliminado correctamente"})
}
