package main

import (
	"atrevida-agenda-api/config"
	"atrevida-agenda-api/handlers"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()
	versionString := "v0.1"
	apiName := "Atrevida Fit - Agenda API" + "(" + versionString + ")"

	r := gin.Default()

	r.Use(cors.Default())

	println(apiName + "\n" + "Running")

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": apiName,
		})
	})

	r.GET("/reservas/unfiltered", func(c *gin.Context) {
		reservas := services.GetAllReservas()

		c.JSON(http.StatusOK, reservas)
	})

	r.GET("/reservas/raw", func(c *gin.Context) {
		data := services.GetSheetData("SAN MARTIN")

		c.JSON(http.StatusOK, data)
	})

	r.GET("/reservas/celda-raw", func(c *gin.Context) {
		local := c.Query("local")
		semana := c.Query("semana")
		dia := c.Query("dia")
		hora := c.Query("hora")

		a1, err := services.ResolverCoordenada(local, semana, dia, hora)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		raw, err := services.GetCeldaRaw(local, a1)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"a1":     a1,
			"raw":    raw,
			"len":    len(raw),
			"bytes":  []byte(raw),
			"parsed": utils.ParseCelda(raw),
		})
	})

	r.GET("/reservas", handlers.GetReservas)
	r.POST("/reservas", handlers.PostReserva)
	r.PATCH("/reservas", handlers.PatchReserva)

	r.Run(":8080")
}
