package main

import (
	"atrevida-agenda-api/config"
	"atrevida-agenda-api/db"
	"atrevida-agenda-api/docs"
	"atrevida-agenda-api/handlers"
	"atrevida-agenda-api/importacion"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
	sheetsrepo "atrevida-agenda-api/repositories/sheets"
	"atrevida-agenda-api/router"
	"atrevida-agenda-api/services"
)

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs

// @title Atrevida Fit - Agenda API
// @version v0.1
// @description API REST para reservas, servicios, combos y administracion de Atrevida Fit.
// @BasePath /
// @schemes http https
func main() {
	config.Load()

	versionString := "v0.1"
	apiName := "Atrevida Fit - Agenda API" + "(" + versionString + ")"
	docs.SwaggerInfo.Title = "Atrevida Fit - Agenda API"
	docs.SwaggerInfo.Version = versionString
	docs.SwaggerInfo.Description = "API REST para reservas, servicios, combos y administracion de Atrevida Fit."
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	pgDB, err := db.Connect(config.App)
	if err != nil {
		panic(err)
	}
	if err := db.RunMigrations(pgDB, "file://migrations"); err != nil {
		panic(err)
	}

	// Repos
	repo := sheetsrepo.NewReservasRepo(config.App)

	categoriasPGRepo := pgsqlrepo.NewCategoriasRepo(pgDB)
	clientesPGRepo := pgsqlrepo.NewClientesRepo(pgDB)
	serviciosPGRepo := pgsqlrepo.NewServiciosRepo(pgDB)
	combosPGRepo := pgsqlrepo.NewCombosRepo(pgDB)
	reservasPGRepo := pgsqlrepo.NewReservasRepo(pgDB)
	localesPGRepo := pgsqlrepo.NewLocalesRepo(pgDB)

	// Services
	reservasService := services.NewReservasService(repo)
	writerService := services.NewReservasWriterService(repo)
	//serviciosService := services.NewServiciosService(repo)
	combosService := services.NewCombosService(repo)

	categoriasPGService := services.NewCategoriasService(categoriasPGRepo)
	clientesPGService := services.NewClientesService(clientesPGRepo)
	serviciosPGService := services.NewServiciosPGService(serviciosPGRepo)
	combosPGService := services.NewCombosService(combosPGRepo)
	reservasPGService := services.NewReservasPGService(reservasPGRepo, serviciosPGRepo)
	localesPGService := services.NewLocalesService(localesPGRepo)

	importService := importacion.NewImportService(pgDB, repo)

	h := handlers.NewContainer(
		reservasService,
		writerService,
		//serviciosService,
		combosService,
		categoriasPGService,
		clientesPGService,
		serviciosPGService,
		combosPGService,
		reservasPGService,
		localesPGService,
		importService,
	)

	println(apiName + "\n" + "Running")

	routerAtrevida := router.Setup(h, repo)
	routerAtrevida.Run(":8080")
}
