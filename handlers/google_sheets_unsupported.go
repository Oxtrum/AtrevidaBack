package handlers

import (
	"net/http"

	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

const googleSheetsUnsupportedMessage = "Google Sheets ya no soportado"

func RespondGoogleSheetsUnsupported(c *gin.Context) {
	utils.RespondError(c, http.StatusGone, googleSheetsUnsupportedMessage)
}
