package handlers

import (
	"net/http"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// GetCategorias godoc
// @Summary Listar categorias
// @Description Devuelve todas las categorias registradas en la base de datos.
// @Tags Categorias
// @Produce json
// @Success 200 {object} utils.APIResponse{data=categoriaListResponse}
// @Failure 500 {object} utils.APIResponse
// @Router /bd/categorias [get]
func (h *Container) GetCategorias(c *gin.Context) {
	resultado, err := h.CategoriasPG.GetCategorias()
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
	Nombre string `json:"nombre"   binding:"required" example:"Depilacion Laser"`
}

// CreateCategoria godoc
// @Summary Crear categoria
// @Description Crea una nueva categoria en la base de datos.
// @Tags Categorias
// @Accept json
// @Produce json
// @Param payload body crearCategoriaRequest true "Datos de la categoria"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /bd/categorias [post]
func (h *Container) CreateCategoria(c *gin.Context) {
	var req crearCategoriaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	id, err := h.CategoriasPG.CreateCategoria(services.CrearCategoriaInput{
		Nombre: req.Nombre,
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}
