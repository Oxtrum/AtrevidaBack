package app

import (
	"atrevida-agenda-api/config"
	"atrevida-agenda-api/db"
	"atrevida-agenda-api/docs"
	"atrevida-agenda-api/handlers"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
	"atrevida-agenda-api/router"
	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
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

	if err := db.RunMigrations(pgDB, "file://migrations"); err != nil {
		return nil, err
	}

	authRepo := pgsqlrepo.NewAuthRepo(pgDB)
	categoriasPGRepo := pgsqlrepo.NewCategoriasRepo(pgDB)
	clientesPGRepo := pgsqlrepo.NewClientesRepo(pgDB)
	localesHorariosPGRepo := pgsqlrepo.NewLocalesHorariosRepo(pgDB)
	serviciosPGRepo := pgsqlrepo.NewServiciosRepo(pgDB)
	combosPGRepo := pgsqlrepo.NewCombosRepo(pgDB)
	comboServiciosPGRepo := pgsqlrepo.NewComboServiciosRepo(pgDB)
	reservasPGRepo := pgsqlrepo.NewReservasRepo(pgDB)
	localesPGRepo := pgsqlrepo.NewLocalesRepo(pgDB)

	authService := services.NewAuthService(authRepo, config.App.Auth.TokenSecret, config.App.Auth.TokenTTL)
	categoriasPGService := services.NewCategoriasService(categoriasPGRepo)
	clientesPGService := services.NewClientesService(clientesPGRepo)
	localesHorariosPGService := services.NewLocalesHorariosService(localesHorariosPGRepo)
	serviciosPGService := services.NewServiciosPGService(serviciosPGRepo)
	combosPGService := services.NewCombosService(combosPGRepo)
	comboServiciosPGService := services.NewComboServiciosService(comboServiciosPGRepo)
	reservasPGService := services.NewReservasPGService(reservasPGRepo, serviciosPGRepo)
	localesPGService := services.NewLocalesService(localesPGRepo)

	h := handlers.NewContainer(
		authService,
		categoriasPGService,
		clientesPGService,
		localesHorariosPGService,
		serviciosPGService,
		combosPGService,
		comboServiciosPGService,
		reservasPGService,
		localesPGService,
	)

	return router.Setup(h), nil
}

func configureSwagger() {
	docs.SwaggerInfo.Title = Name
	docs.SwaggerInfo.Version = Version
	docs.SwaggerInfo.Description = "API REST para reservas, servicios, combos y administracion de Atrevida Fit."
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
}
