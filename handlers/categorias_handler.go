package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type categoriaLocalRequest struct {
	// ID de la categoria a asociar con el local
	CategoriaID int `json:"categoria_id" binding:"required" example:"3"`
	// ID del local a asociar con la categoria
	LocalID int `json:"local_id" binding:"required" example:"1"`
}

// GetCategorias godoc
// @Summary Listar categorias
// @Description Devuelve categorias registradas en BD. Sin filtros devuelve todas. Si se envia local o local_id, devuelve solo categorias asociadas a ese local mediante categorias_locales. Response: total (int), categorias ([]CategoriaPG con: id, nombre).
// @Tags Categorias
// @Produce json
// @Param local query string false "Nombre exacto del local" example(SAN MARTIN)
// @Param local_id query int false "ID del local" example(1)
// @Success 200 {object} utils.APIResponse{data=categoriaListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: local_id invalido"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/categorias [get]
func (h *Container) GetCategorias(c *gin.Context) {
	var localID *int
	if raw := strings.TrimSpace(c.Query("local_id")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			utils.RespondError(c, http.StatusBadRequest, "local_id invalido")
			return
		}
		localID = &n
	}

	resultado, err := h.CategoriasPG.GetCategoriasFiltradas(services.FiltroCategorias{
		Local:   c.Query("local"),
		LocalID: localID,
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, categoriaListResponse{
		Total:      len(resultado),
		Categorias: resultado,
	})
}

type crearCategoriaRequest struct {
	// Nombre de la categoria
	Nombre string `json:"nombre"   binding:"required" example:"Depilacion Laser"`
	// ID del local al que se asociara la categoria; si es null u omitido, solo crea la categoria
	LocalID *int `json:"local_id" example:"1"`
}

// CreateCategoria godoc
// @Summary Crear categoria
// @Description Crea una categoria nueva. Requiere token Bearer con rol admin_sys. Body: nombre (requerido) y local_id (opcional, puede ser null). Si local_id viene informado, tambien asocia la categoria al local mediante categorias_locales en la misma transaccion. Response: id (int ID de la categoria creada).
// @Tags Categorias
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearCategoriaRequest true "Datos de la categoria"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: nombre requerido o local_id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Local no encontrado o inactivo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/categorias [post]
func (h *Container) CreateCategoria(c *gin.Context) {
	var req crearCategoriaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	id, err := h.CategoriasPG.CreateCategoria(services.CrearCategoriaInput{
		Nombre:  req.Nombre,
		LocalID: req.LocalID,
	})
	if err != nil {
		utils.RespondError(c, categoriaLocalStatus(err), err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}

// CreateCategoriaLocal godoc
// @Summary Asociar categoria con local
// @Description Crea una relacion en categorias_locales. Requiere token Bearer con rol admin_sys. Body: categoria_id y local_id requeridos. Si la relacion ya existe, la operacion es idempotente.
// @Tags Categorias
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body categoriaLocalRequest true "IDs de categoria y local"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: categoria_id/local_id invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Categoria o local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/categorias/locales [post]
func (h *Container) CreateCategoriaLocal(c *gin.Context) {
	var req categoriaLocalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	err := h.CategoriasPG.CreateCategoriaLocal(services.CategoriaLocalInput{
		CategoriaID: req.CategoriaID,
		LocalID:     req.LocalID,
	})
	if err != nil {
		utils.RespondError(c, categoriaLocalStatus(err), err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "categoria asociada al local correctamente"})
}

// DeleteCategoriaLocal godoc
// @Summary Eliminar asociacion categoria-local
// @Description Elimina una relacion de categorias_locales. Requiere token Bearer con rol admin_sys. Body: categoria_id y local_id requeridos.
// @Tags Categorias
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body categoriaLocalRequest true "IDs de categoria y local"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: categoria_id/local_id invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Asociacion categoria-local no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/categorias/locales [delete]
func (h *Container) DeleteCategoriaLocal(c *gin.Context) {
	var req categoriaLocalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	err := h.CategoriasPG.DeleteCategoriaLocal(services.CategoriaLocalInput{
		CategoriaID: req.CategoriaID,
		LocalID:     req.LocalID,
	})
	if err != nil {
		utils.RespondError(c, categoriaLocalStatus(err), err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "asociacion categoria-local eliminada correctamente"})
}

func categoriaLocalStatus(err error) int {
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "no encontrada") ||
		strings.Contains(lower, "no encontrado") ||
		strings.Contains(lower, "inactivo"):
		return http.StatusNotFound
	case strings.Contains(lower, "positivos") ||
		strings.Contains(lower, "positivo") ||
		strings.Contains(lower, "invalido"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
