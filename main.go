package main

import (
	"atrevida-agenda-api/config"
	"atrevida-agenda-api/handlers"
	"atrevida-agenda-api/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()
	versionString := "v0.1"
	apiName := "Atrevida Fit - Agenda API" + "(" + versionString + ")"

	r := gin.Default()

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

	r.GET("/reservas", handlers.GetReservas)

	r.Run(":8080")
}
