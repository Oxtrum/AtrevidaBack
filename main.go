package main

import (
	"atrevida-agenda-api/config"
	"atrevida-agenda-api/db"
	"atrevida-agenda-api/docs"
	"atrevida-agenda-api/handlers"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
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

	categoriasPGRepo := pgsqlrepo.NewCategoriasRepo(pgDB)
	clientesPGRepo := pgsqlrepo.NewClientesRepo(pgDB)
	localesHorariosPGRepo := pgsqlrepo.NewLocalesHorariosRepo(pgDB)
	serviciosPGRepo := pgsqlrepo.NewServiciosRepo(pgDB)
	combosPGRepo := pgsqlrepo.NewCombosRepo(pgDB)
	comboServiciosPGRepo := pgsqlrepo.NewComboServiciosRepo(pgDB)
	reservasPGRepo := pgsqlrepo.NewReservasRepo(pgDB)
	localesPGRepo := pgsqlrepo.NewLocalesRepo(pgDB)

	categoriasPGService := services.NewCategoriasService(categoriasPGRepo)
	clientesPGService := services.NewClientesService(clientesPGRepo)
	localesHorariosPGService := services.NewLocalesHorariosService(localesHorariosPGRepo)
	serviciosPGService := services.NewServiciosPGService(serviciosPGRepo)
	combosPGService := services.NewCombosService(combosPGRepo)
	comboServiciosPGService := services.NewComboServiciosService(comboServiciosPGRepo)
	reservasPGService := services.NewReservasPGService(reservasPGRepo, serviciosPGRepo)
	localesPGService := services.NewLocalesService(localesPGRepo)

	h := handlers.NewContainer(
		categoriasPGService,
		clientesPGService,
		localesHorariosPGService,
		serviciosPGService,
		combosPGService,
		comboServiciosPGService,
		reservasPGService,
		localesPGService,
	)

	println(apiName + "\n" + "Running")

	routerAtrevida := router.Setup(h)
	routerAtrevida.Run(":8080")
}
