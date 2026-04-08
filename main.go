package main

import (
	"net/http"

	"atrevida-agenda-api/config"
	"atrevida-agenda-api/handlers"
	sheetsrepo "atrevida-agenda-api/repositories/sheets"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()

	versionString := "v0.1"
	apiName := "Atrevida Fit - Agenda API" + "(" + versionString + ")"

	// Dependencias
	repo := sheetsrepo.NewReservasRepo(config.App)
	reservasService := services.NewReservasService(repo)
	writerService := services.NewReservasWriterService(repo)
	serviciosService := services.NewServiciosService(repo)
	h := handlers.NewContainer(reservasService, writerService, serviciosService)

	// Router
	r := gin.Default()

	println(apiName + "\n" + "Running")

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": apiName})
	})

	// Debug
	r.GET("/reservas/unfiltered", func(c *gin.Context) {
		c.JSON(http.StatusOK, repo.GetAllReservas())
	})
	r.GET("/reservas/raw", func(c *gin.Context) {
		c.JSON(http.StatusOK, repo.GetSheetData("SAN MARTIN"))
	})
	r.GET("/reservas/celda-raw", func(c *gin.Context) {
		local := c.Query("local")
		semana := c.Query("semana")
		dia := c.Query("dia")
		hora := c.Query("hora")

		a1, err := repo.ResolverCoordenada(local, semana, dia, hora)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		raw, err := repo.GetCeldaRaw(local, a1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"a1":     a1,
			"raw":    raw,
			"len":    len(raw),
			"bytes":  []byte(raw),
			"parsed": utils.ParseCelda(raw),
		})
	})

	// Reservas
	r.GET("/reservas", h.GetReservas)
	r.POST("/reservas", h.PostReserva)
	r.PATCH("/reservas", h.PatchReserva)

	// Catálogo de servicios
	r.GET("/servicios", h.GetServicios)

	r.Run(":8080")
}
