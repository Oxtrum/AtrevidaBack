package handlers

import (
	"net/http"

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
