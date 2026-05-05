package handlers

import (
	"net/http"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// GET /bd/categorias
func (h *Container) GetCategorias(c *gin.Context) {
	resultado, err := h.CategoriasPG.GetCategorias()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, gin.H{
		"total":      len(resultado),
		"categorias": resultado,
	})
}

type crearCategoriaRequest struct {
	Nombre string `json:"nombre"   binding:"required"`
}

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

	utils.Respond(c, http.StatusOK, gin.H{"id": id})
}
