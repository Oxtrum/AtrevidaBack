package router

import (
	"net/http"
	"time"

	"atrevida-agenda-api/handlers"
	sheetsrepo "atrevida-agenda-api/repositories/sheets"
	"atrevida-agenda-api/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(h *handlers.Container, repo *sheetsrepo.ReservasRepo) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:4173",
			"http://127.0.0.1:4173",
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Atrevida Fit - Agenda API"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Debug - Sheets
	debug := r.Group("/reservas")
	{
		debug.GET("/unfiltered", func(c *gin.Context) {
			c.JSON(http.StatusOK, repo.GetAllReservas())
		})
		debug.GET("/raw", func(c *gin.Context) {
			c.JSON(http.StatusOK, repo.GetSheetData("SAN MARTIN"))
		})
		debug.GET("/celda-raw", func(c *gin.Context) {
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
	}

	// Sheets
	r.GET("/reservas", h.GetReservas)
	r.POST("/reservas", h.PostReserva)
	r.PATCH("/reservas", h.PatchReserva)
	r.GET("/servicios", h.GetServicios)
	r.GET("/combos", h.GetCombos)

	// BD
	bd := r.Group("/bd")
	{
		bd.GET("/categorias", h.GetCategorias)
		bd.POST("/categorias", h.CreateCategoria)

		bd.GET("/clientes", h.GetClientes)
		bd.GET("/clientes/:id", h.GetClienteByID)
		bd.POST("/clientes", h.CreateCliente)
		bd.PATCH("/clientes/:id", h.PatchCliente)
		bd.DELETE("/clientes/:id", h.DeleteCliente)

		bd.GET("/servicios", h.GetServiciosPG)
		bd.GET("/servicios/:id", h.GetServicioPGByID)
		bd.POST("/servicios", h.CreateServicio)
		bd.PATCH("/servicios/:id", h.UpdateServicio)
		bd.DELETE("/servicios/:id", h.DeleteServicio)
		bd.POST("/servicios/local/:id", h.ActivarServicioEnLocal)

		bd.GET("/combos", h.GetCombosPG)
		//bd.GET("/combos/:id", h.GetComboById)

		bd.GET("/locales", h.GetLocales)
		bd.GET("/locales/:id", h.GetLocalById)
		bd.POST("/locales", h.PostLocal)
		bd.PATCH("/locales/:id", h.PatchLocal)
		bd.DELETE("/locales/:id", h.DeleteLocal)

		bd.GET("/reservas", h.GetReservasSimplePG)
		bd.GET("/reservas/resumen", h.GetReservasResumenPG)
		bd.GET("/reservas/:id", h.GetReservaPGByID)
		bd.DELETE("/reservas/:id", h.DeleteReservaPG)
		bd.GET("/reservas/calendario", h.GetReservasPG)
		bd.POST("/reservas", h.PostReservaPG)
		bd.PATCH("/reservas", h.PatchReservaPG)
		bd.PATCH("/reservas/estado", h.PatchReservaEstadoPG)

	}

	// Admin
	admin := r.Group("/admin")
	{
		admin.POST("/importar", h.ImportarCatalogo)
	}

	return r
}
