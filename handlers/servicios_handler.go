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

// GET /servicios
// Query params:
//
//	nombre    — búsqueda parcial en el nombre del servicio
//	categoria — búsqueda parcial en la categoría (ej: "manual", "combos")
//	local     — "ARANJUEZ" o "CENTRO"
//	sesiones  — número exacto de sesiones (ej: 3, 5, 10)
func (h *Container) GetServicios(c *gin.Context) {
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
	if local != "" && local != "ARANJUEZ" && local != "CENTRO" && local != "SAN MARTIN" {
		utils.RespondError(c, http.StatusBadRequest,
			"local inválido, valores permitidos: ARANJUEZ, CENTRO (o SAN MARTIN)")
		return
	}

	if local == "SAN MARTIN" {
		local = "CENTRO"
	}

	filtro := services.FiltroServicios{
		Nombre:    strings.TrimSpace(c.Query("nombre")),
		Categoria: strings.TrimSpace(c.Query("categoria")),
		Local:     local,
		Sesiones:  sesiones,
	}

	resultado := h.Servicios.GetServiciosFiltrados(filtro)
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
