package router

import (
	"net/http"
	"time"

	"atrevida-agenda-api/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(h *handlers.Container) *gin.Engine {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
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
		MaxAge: 12 * time.Hour,
	}))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Atrevida Fit - Agenda API"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html>
<html>
<head>
  <title>Atrevida Fit - API Docs</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
  <script id="api-reference" data-url="/swagger/doc.json" data-configuration='{"theme":"purple","defaultHttpClient":{"targetKey":"shell","clientKey":"curl"},"defaultFonts":false}'></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`))
	})

	// Auth
	auth := r.Group("/auth")
	{
		auth.GET("/usuarios", h.AuthRequired, h.AdminSysRequired, h.GetUsuarios)
		auth.POST("/register", h.AuthRequired, h.AdminSysRequired, h.RegisterUsuario)
		auth.POST("/login", h.Login)
		auth.PATCH("/change-password", h.AuthRequired, h.CambiarPassword)
		auth.PATCH("/deactivate", h.AuthRequired, h.AdminSysRequired, h.ActualizarUsuarioActivo)
		auth.PATCH("/usuarios/local", h.AuthRequired, h.AdminSysRequired, h.ActualizarUsuarioLocal)
	}

	// Debug - Sheets
	debug := r.Group("/reservas")
	{
		debug.GET("/unfiltered", func(c *gin.Context) {
			handlers.RespondGoogleSheetsUnsupported(c)
		})
		debug.GET("/raw", func(c *gin.Context) {
			handlers.RespondGoogleSheetsUnsupported(c)
		})
		debug.GET("/celda-raw", func(c *gin.Context) {
			handlers.RespondGoogleSheetsUnsupported(c)
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
		bd.POST("/categorias", h.AuthRequired, h.AdminSysRequired, h.CreateCategoria)
		bd.PUT("/categorias/:id", h.AuthRequired, h.AdminSysRequired, h.UpdateCategoria)
		bd.DELETE("/categorias/:id", h.AuthRequired, h.AdminSysRequired, h.DeleteCategoria)
		bd.GET("/categorias/:id/locales", h.GetCategoriaLocales)
		bd.POST("/categorias/locales", h.AuthRequired, h.AdminSysRequired, h.CreateCategoriaLocal)
		bd.DELETE("/categorias/locales", h.AuthRequired, h.AdminSysRequired, h.DeleteCategoriaLocal)

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
		bd.PATCH("/servicios/:id/local/:local_id/paciente-nuevo", h.AuthRequired, h.PatchServicioVisiblePacienteNuevoLocal)

		bd.GET("/combos", h.GetCombosPG)
		bd.GET("/combos/:id", h.GetComboPGByID)
		bd.POST("/combos", h.AuthRequired, h.AdminSysRequired, h.CreateComboPG)
		bd.PATCH("/combos/:id", h.AuthRequired, h.AdminSysRequired, h.PatchComboPG)
		bd.DELETE("/combos/:id", h.AuthRequired, h.AdminSysRequired, h.DeleteComboPG)
		bd.PUT("/combos/:id/locales", h.AuthRequired, h.AdminSysRequired, h.PutComboLocalesPG)
		bd.PUT("/combos/:id/servicios", h.AuthRequired, h.AdminSysRequired, h.PutComboServiciosPG)
		bd.POST("/combos/:id/imagen/upload-url", h.AuthRequired, h.AdminSysRequired, h.PostComboImagenUploadURL)
		bd.PUT("/combos/:id/imagen", h.AuthRequired, h.AdminSysRequired, h.PutComboImagen)
		bd.DELETE("/combos/:id/imagen", h.AuthRequired, h.AdminSysRequired, h.DeleteComboImagen)

		bd.GET("/paquetes", h.GetPaquetes)
		bd.GET("/paquetes/:id", h.GetPaqueteByID)
		bd.POST("/paquetes", h.AuthRequired, h.AdminSysRequired, h.CreatePaquete)
		bd.PATCH("/paquetes/:id", h.AuthRequired, h.AdminSysRequired, h.PatchPaquete)
		bd.DELETE("/paquetes/:id", h.AuthRequired, h.AdminSysRequired, h.DeletePaquete)
		bd.POST("/paquetes/:id/imagen/upload-url", h.AuthRequired, h.AdminSysRequired, h.PostPaqueteImagenUploadURL)
		bd.PUT("/paquetes/:id/imagen", h.AuthRequired, h.AdminSysRequired, h.PutPaqueteImagen)
		bd.DELETE("/paquetes/:id/imagen", h.AuthRequired, h.AdminSysRequired, h.DeletePaqueteImagen)

		bd.GET("/locales", h.GetLocales)
		bd.GET("/locales/:id", h.GetLocalById)
		bd.GET("/locales/horarios", h.GetHorariosByLocal)
		bd.POST("/locales/horarios", h.CreateHorarioByLocal)
		bd.GET("/locales/horarios/:id", h.GetHorarioByID)
		bd.PATCH("/locales/horarios/:id", h.PatchHorario)
		bd.DELETE("/locales/horarios/:id", h.DeleteHorario)
		bd.POST("/locales", h.PostLocal)
		bd.PATCH("/locales/:id", h.PatchLocal)
		bd.DELETE("/locales/:id", h.DeleteLocal)

		bd.GET("/notificaciones/reservas", h.AuthRequired, h.GetNotificacionesReservasPG)
		bd.PATCH("/notificaciones/reservas/leer", h.PatchNotificacionesReservasLeidasPG)
		bd.PATCH("/notificaciones/reservas/:id/leer", h.PatchNotificacionReservaLeidaPG)

		bd.GET("/reservas", h.GetReservasSimplePG)
		bd.GET("/reservas/resumen", h.AuthRequired, h.GetReservasResumenPG)
		bd.GET("/reservas/:id", h.AuthRequired, h.GetReservaPGByID)
		bd.PATCH("/reservas/notificar", h.AuthRequired, h.PatchReservaNotificadoPG)
		bd.DELETE("/reservas/:id", h.AuthRequired, h.DeleteReservaPG)
		bd.GET("/reservas/calendario", h.GetReservasPG)
		// POST y GET quedan publicos: el formulario de reserva del sitio los usa sin token.
		bd.POST("/reservas", h.PostReservaPG)
		bd.PATCH("/reservas", h.AuthRequired, h.PatchReservaPG)
		bd.PATCH("/reservas/estado", h.AuthRequired, h.PatchReservaEstadoPG)

		bd.GET("/pagos", h.AuthRequired, h.GetPagos)
		bd.GET("/pagos/resumen", h.AuthRequired, h.GetResumenPagos)
		bd.GET("/pagos/:codigo_pago", h.AuthRequired, h.GetPagoByCodigo)
		bd.POST("/pagos", h.AuthRequired, h.CreatePago)
		bd.PATCH("/pagos/:codigo_pago", h.AuthRequired, h.PatchPago)
		bd.DELETE("/pagos/:codigo_pago", h.AuthRequired, h.DeletePago)

		bd.GET("/planes", h.AuthRequired, h.GetPlanes)
		bd.GET("/planes/:id", h.AuthRequired, h.GetPlanByID)
		bd.POST("/planes", h.AuthRequired, h.GerenciaRequired, h.CreatePlan)
		bd.PATCH("/planes/:id", h.AuthRequired, h.GerenciaRequired, h.PatchPlan)
		bd.PATCH("/planes/:id/estado", h.AuthRequired, h.GerenciaRequired, h.PatchPlanEstado)
		bd.PATCH("/planes/:id/sesiones/:numero", h.AuthRequired, h.GerenciaRequired, h.MarcarSesionPlan)
		bd.POST("/planes/:id/cobrar", h.AuthRequired, h.GerenciaRequired, h.CobrarPlan)

		// Trastienda: inventario
		bd.GET("/inventario/items", h.AuthRequired, h.GetInventarioItems)
		bd.GET("/inventario/items/:id", h.AuthRequired, h.GetInventarioItemByID)
		bd.POST("/inventario/items", h.AuthRequired, h.AdminSysRequired, h.CreateInventarioItem)
		bd.PATCH("/inventario/items/:id", h.AuthRequired, h.AdminSysRequired, h.PatchInventarioItem)
		bd.DELETE("/inventario/items/:id", h.AuthRequired, h.AdminSysRequired, h.DeleteInventarioItem)
		bd.POST("/inventario/items/:id/imagen/upload-url", h.AuthRequired, h.AdminSysRequired, h.PostInventarioImagenUploadURL)
		bd.PUT("/inventario/items/:id/imagen", h.AuthRequired, h.AdminSysRequired, h.PutInventarioImagen)
		bd.DELETE("/inventario/items/:id/imagen", h.AuthRequired, h.AdminSysRequired, h.DeleteInventarioImagen)
		bd.GET("/inventario/movimientos", h.AuthRequired, h.GetInventarioMovimientos)
		bd.POST("/inventario/ajustes", h.AuthRequired, h.GerenciaRequired, h.PostInventarioAjuste)
		bd.GET("/inventario/alertas", h.AuthRequired, h.GetInventarioAlertas)
		bd.GET("/inventario/categorias", h.AuthRequired, h.GetInventarioCategorias)
		bd.POST("/inventario/categorias", h.AuthRequired, h.AdminSysRequired, h.PostInventarioCategoria)
		bd.PATCH("/inventario/categorias/:id", h.AuthRequired, h.AdminSysRequired, h.PatchInventarioCategoria)
		bd.GET("/servicios/:id/insumos", h.AuthRequired, h.GetServicioInsumos)
		bd.PUT("/servicios/:id/insumos", h.AuthRequired, h.PutServicioInsumos)

		// Trastienda: compras
		bd.GET("/compras", h.AuthRequired, h.GerenciaRequired, h.GetCompras)
		bd.POST("/compras", h.AuthRequired, h.GerenciaRequired, h.PostCompra)
		bd.GET("/compras/:id", h.AuthRequired, h.GerenciaRequired, h.GetCompraByID)
		bd.PATCH("/compras/:id/anular", h.AuthRequired, h.AdminSysRequired, h.PatchCompraAnular)

		// Trastienda: finanzas (solo admin_sys)
		bd.GET("/finanzas/movimientos", h.AuthRequired, h.AdminSysRequired, h.GetMovimientosFinancieros)
		bd.POST("/finanzas/movimientos", h.AuthRequired, h.AdminSysRequired, h.PostMovimientoFinanciero)
		bd.PATCH("/finanzas/movimientos/:id", h.AuthRequired, h.AdminSysRequired, h.PatchMovimientoFinanciero)
		bd.PATCH("/finanzas/movimientos/:id/estado", h.AuthRequired, h.AdminSysRequired, h.PatchMovimientoFinancieroEstado)
		bd.GET("/finanzas/categorias", h.AuthRequired, h.AdminSysRequired, h.GetCategoriasFinancieras)
		bd.POST("/finanzas/categorias", h.AuthRequired, h.AdminSysRequired, h.PostCategoriaFinanciera)
		bd.PATCH("/finanzas/categorias/:id", h.AuthRequired, h.AdminSysRequired, h.PatchCategoriaFinanciera)
		bd.GET("/finanzas/gastos-fijos", h.AuthRequired, h.AdminSysRequired, h.GetGastosFijos)
		bd.POST("/finanzas/gastos-fijos", h.AuthRequired, h.AdminSysRequired, h.PostGastoFijo)
		bd.GET("/finanzas/gastos-fijos/proximos", h.AuthRequired, h.AdminSysRequired, h.GetGastosFijosProximos)
		bd.GET("/finanzas/gastos-fijos/pendientes", h.AuthRequired, h.AdminSysRequired, h.GetGastosFijosPendientes)
		bd.POST("/finanzas/gastos-fijos/generar", h.AdminSysOrJobSecret, h.PostGenerarGastosFijos)
		bd.PATCH("/finanzas/gastos-fijos/:id", h.AuthRequired, h.AdminSysRequired, h.PatchGastoFijo)
		bd.DELETE("/finanzas/gastos-fijos/:id", h.AuthRequired, h.AdminSysRequired, h.DeleteGastoFijo)
		bd.GET("/reportes/financiero", h.AuthRequired, h.AdminSysRequired, h.GetReporteFinanciero)

		// Parametros del sistema: lectura para cualquier usuario autenticado, escritura solo admin_sys.
		bd.GET("/configuracion", h.AuthRequired, h.GetConfiguracion)
		bd.PATCH("/configuracion", h.AuthRequired, h.AdminSysRequired, h.PatchConfiguracion)
	}

	// Admin
	admin := r.Group("/admin")
	{
		admin.POST("/importar", h.ImportarCatalogo)
	}

	return r
}
