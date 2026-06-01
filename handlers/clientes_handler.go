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
	// Nombre del cliente
	Nombre string `json:"nombre" example:"Maria"`
	// Apellido del cliente
	Apellido string `json:"apellido" example:"Lopez"`
	// Numero de telefono del cliente
	NumeroTelefono string `json:"numero_telefono" example:"+59170011223"`
}

type actualizarClienteRequest struct {
	// Nuevo nombre (opcional)
	Nombre *string `json:"nombre" example:"Maria Fernanda"`
	// Nuevo apellido (opcional)
	Apellido *string `json:"apellido" example:"Lopez Aguilar"`
	// Nuevo numero de telefono (opcional)
	NumeroTelefono *string `json:"numero_telefono" example:"+59170011224"`
}

// GetClientes godoc
// @Summary Listar clientes
// @Description Devuelve clientes de BD con filtros. Filtros: nombre busqueda parcial (opcional), apellido busqueda parcial (opcional), numero_telefono busqueda parcial (opcional). Response: total (int), filtros (objeto con nombre, apellido, numero_telefono), clientes ([]ClientePG con: id, nombre, apellido, numero_telefono).
// @Tags Clientes
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre" example(Maria)
// @Param apellido query string false "Busqueda parcial por apellido" example(Lopez)
// @Param numero_telefono query string false "Busqueda parcial por numero de telefono" example(+59170011223)
// @Success 200 {object} utils.APIResponse{data=clienteListResponse}
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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

	utils.Respond(c, http.StatusOK, clienteListResponse{
		Total: len(clientes),
		Filtros: clienteFiltrosResponse{
			Nombre:         strings.TrimSpace(c.Query("nombre")),
			Apellido:       strings.TrimSpace(c.Query("apellido")),
			NumeroTelefono: strings.TrimSpace(c.Query("numero_telefono")),
		},
		Clientes: clientes,
	})
}

// GetClienteByID godoc
// @Summary Obtener cliente por ID
// @Description Devuelve un cliente por su ID. Param: id (requerido, path). Response: cliente (ClientePG con: id, nombre, apellido, numero_telefono).
// @Tags Clientes
// @Produce json
// @Param id path int true "ID del cliente" example(12)
// @Success 200 {object} utils.APIResponse{data=clienteItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Cliente no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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

	utils.Respond(c, http.StatusOK, clienteItemResponse{Cliente: cliente})
}

// CreateCliente godoc
// @Summary Crear cliente
// @Description Crea un cliente en BD. Body: nombre (requerido), apellido (requerido), numero_telefono (requerido). Response: id (int ID del cliente creado).
// @Tags Clientes
// @Accept json
// @Produce json
// @Param payload body crearClienteRequest true "Datos del cliente (nombre, apellido, numero_telefono)"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: nombre/apellido/numero_telefono requeridos"
// @Failure 409 {object} utils.APIResponse "Conflicto: cliente ya existe"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}

// PatchCliente godoc
// @Summary Actualizar cliente
// @Description Actualiza parcialmente un cliente. Param: id (requerido, path). Body: nombre (opcional), apellido (opcional), numero_telefono (opcional). Response: mensaje string.
// @Tags Clientes
// @Accept json
// @Produce json
// @Param id path int true "ID del cliente" example(12)
// @Param payload body actualizarClienteRequest true "Campos a actualizar (todos opcionales, al menos uno requerido)"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, body invalido, campo vacio, sin cambios"
// @Failure 404 {object} utils.APIResponse "Cliente no encontrado"
// @Failure 409 {object} utils.APIResponse "Conflicto: telefono ya existe"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "cliente actualizado correctamente"})
}

// DeleteCliente godoc
// @Summary Eliminar cliente
// @Description Elimina un cliente de BD permanentemente (borrado fisico, no logico). Param: id (requerido, path). Response: mensaje string.
// @Tags Clientes
// @Produce json
// @Param id path int true "ID del cliente" example(12)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Cliente no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
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

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "cliente eliminado correctamente"})
}
