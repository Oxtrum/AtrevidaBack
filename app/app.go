package app

import (
	"context"
	"log"

	"atrevida-agenda-api/config"
	"atrevida-agenda-api/db"
	"atrevida-agenda-api/docs"
	"atrevida-agenda-api/handlers"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
	"atrevida-agenda-api/router"
	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

const (
	Name    = "Atrevida Fit - Agenda API"
	Version = "v0.1"
)

// Build initializes config, dependencies, handlers and the HTTP router.
func Build() (*gin.Engine, error) {
	config.Load()
	configureSwagger()

	pgDB, err := db.Connect(config.App)
	if err != nil {
		return nil, err
	}

	authRepo := pgsqlrepo.NewAuthRepo(pgDB)
	categoriasPGRepo := pgsqlrepo.NewCategoriasRepo(pgDB)
	clientesPGRepo := pgsqlrepo.NewClientesRepo(pgDB)
	localesHorariosPGRepo := pgsqlrepo.NewLocalesHorariosRepo(pgDB)
	serviciosPGRepo := pgsqlrepo.NewServiciosRepo(pgDB)
	combosPGRepo := pgsqlrepo.NewCombosRepo(pgDB)
	paquetesPGRepo := pgsqlrepo.NewPaquetesRepo(pgDB)
	reservasPGRepo := pgsqlrepo.NewReservasRepo(pgDB)
	localesPGRepo := pgsqlrepo.NewLocalesRepo(pgDB)
	pagosPGRepo := pgsqlrepo.NewPagosRepo(pgDB)
	planesPGRepo := pgsqlrepo.NewPlanesRepo(pgDB)
	configuracionRepo := pgsqlrepo.NewConfiguracionRepo(pgDB)
	inventarioRepo := pgsqlrepo.NewInventarioRepo(pgDB)
	finanzasRepo := pgsqlrepo.NewFinanzasRepo(pgDB)

	authService := services.NewAuthService(authRepo, config.App.Auth.TokenSecret, config.App.Auth.TokenTTL)
	categoriasPGService := services.NewCategoriasService(categoriasPGRepo)
	clientesPGService := services.NewClientesService(clientesPGRepo)
	localesHorariosPGService := services.NewLocalesHorariosService(localesHorariosPGRepo)
	serviciosPGService := services.NewServiciosPGService(serviciosPGRepo)
	combosPGService := services.NewCombosService(combosPGRepo)
	combosPGService.Storage = services.NewSupabaseStorage(
		config.App.Storage.SupabaseURL, config.App.Storage.SecretKey, config.App.Storage.Bucket,
	)
	paquetesPGService := services.NewPaquetesService(paquetesPGRepo)
	paquetesPGService.Storage = services.NewSupabaseStorage(
		config.App.Storage.SupabaseURL, config.App.Storage.SecretKey, config.App.Storage.Bucket,
	)
	configuracionService := services.NewConfiguracionService(configuracionRepo)
	inventarioService := services.NewInventarioService(inventarioRepo, configuracionService)
	// Imagenes en S3 (inventario y fotos del personal): sin S3_BUCKET el Storage queda nil y los endpoints de imagen responden 503.
	s3Storage, err := services.NewS3Storage(
		context.Background(), config.App.S3.Bucket, config.App.S3.Region, config.App.S3.PublicBaseURL,
	)
	if err != nil {
		log.Printf("[S3] almacenamiento de imagenes deshabilitado: %v", err)
		s3Storage = nil
	}
	if s3Storage != nil {
		inventarioService.Storage = s3Storage
	}
	inventarioConsumo := services.NewInventarioConsumo(inventarioRepo, configuracionService)
	finanzasService := services.NewFinanzasService(finanzasRepo, configuracionService)
	finanzasService.AlCambiarStock = inventarioService.RefrescarAlertas
	configuracionService.AlCambiarDisponibilidad = func() { inventarioService.RefrescarAlertas(nil) }
	gastosFijosService := services.NewGastosFijosService(finanzasRepo, configuracionService, finanzasService)
	reporteFinancieroService := services.NewReporteFinancieroService(finanzasRepo, inventarioRepo, gastosFijosService, configuracionService)
	rrhhService := services.NewRRHHService(pgsqlrepo.NewRRHHRepo(pgDB), configuracionService)
	if s3Storage != nil {
		rrhhService.Storage = s3Storage
	}
	reporteFinancieroService.Planillas = rrhhService

	// Sesiones de plan marcadas a mano descuentan insumos en la misma transaccion (RN-18).
	planesPGRepo.ConsumoSesion = func(tx *sqlx.Tx, servicioID, localID, planMovimientoID int) error {
		return inventarioConsumo.ConsumirSesionPlan(tx, servicioID, localID, planMovimientoID, repository.ConsumoActor{})
	}

	reservasPGService := services.NewReservasPGService(reservasPGRepo, serviciosPGRepo)
	reservasPGService.SetInventarioConsumo(inventarioConsumo)
	reservasPGService.SetConfiguracion(configuracionService)
	reservasPGService.SetPlanes(planesPGRepo)
	reservasVinculoRepo := pgsqlrepo.NewReservasVinculoRepo(pgDB)
	reservasPGService.SetClientes(reservasVinculoRepo)
	localesPGService := services.NewLocalesService(localesPGRepo)
	pagosPGService := services.NewPagosService(pagosPGRepo)
	pagosPGService.Config = configuracionService
	cuentaService := services.NewCuentaService(pgsqlrepo.NewCuentaRepo(pgDB))
	cuentaService.Config = configuracionService
	planesPGService := services.NewPlanesService(planesPGRepo, combosPGRepo)

	h := handlers.NewContainer(
		authService,
		categoriasPGService,
		clientesPGService,
		localesHorariosPGService,
		serviciosPGService,
		combosPGService,
		paquetesPGService,
		reservasPGService,
		localesPGService,
		pagosPGService,
		planesPGService,
	)
	h.Cuenta = cuentaService
	h.ReservasVinculo = services.NewReservasVinculoService(reservasVinculoRepo)
	h.Configuracion = configuracionService
	h.Inventario = inventarioService
	h.Finanzas = finanzasService
	h.GastosFijos = gastosFijosService
	h.ReporteFinanciero = reporteFinancieroService
	h.RRHH = rrhhService
	h.JobSecret = config.App.Jobs.Secret

	return router.Setup(h), nil
}

func configureSwagger() {
	docs.SwaggerInfo.Title = Name
	docs.SwaggerInfo.Version = Version
	docs.SwaggerInfo.Description = "API REST para reservas, servicios, combos y administracion de Atrevida Fit."
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
}
