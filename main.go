package main

import (
	"atrevida-agenda-api/config"
	"atrevida-agenda-api/db"
	"atrevida-agenda-api/handlers"
	"atrevida-agenda-api/importacion"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
	sheetsrepo "atrevida-agenda-api/repositories/sheets"
	"atrevida-agenda-api/router"
	"atrevida-agenda-api/services"
)

func main() {
	config.Load()

	versionString := "v0.1"
	apiName := "Atrevida Fit - Agenda API" + "(" + versionString + ")"

	pgDB, err := db.Connect(config.App)
	if err != nil {
		panic(err)
	}
	if err := db.RunMigrations(pgDB, "file://migrations"); err != nil {
		panic(err)
	}

	// Repos
	repo := sheetsrepo.NewReservasRepo(config.App)

	serviciosPGRepo := pgsqlrepo.NewServiciosRepo(pgDB)
	combosPGRepo := pgsqlrepo.NewCombosRepo(pgDB)
	reservasPGRepo := pgsqlrepo.NewReservasRepo(pgDB)
	localesPGRepo := pgsqlrepo.NewLocalesRepo(pgDB)

	// Services
	reservasService := services.NewReservasService(repo)
	writerService := services.NewReservasWriterService(repo)
	serviciosService := services.NewServiciosService(repo)
	combosService := services.NewCombosService(repo)

	serviciosPGService := services.NewServiciosService(serviciosPGRepo)
	combosPGService := services.NewCombosService(combosPGRepo)
	reservasPGService := services.NewReservasPGService(reservasPGRepo)
	localesPGService := services.NewLocalesService(localesPGRepo)

	importService := importacion.NewImportService(pgDB, repo)

	h := handlers.NewContainer(
		reservasService,
		writerService,
		serviciosService,
		combosService,
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
