package main

import (
	"atrevida-agenda-api/app"
)

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs

// @title Atrevida Fit - Agenda API
// @version v0.1
// @description API REST para reservas, servicios, combos y administracion de Atrevida Fit.
// @BasePath /
// @schemes http https
func main() {
	routerAtrevida, err := app.Build()
	if err != nil {
		panic(err)
	}

	println(app.Name + "(" + app.Version + ")\n" + "Running")

	if err := routerAtrevida.Run(":8080"); err != nil {
		panic(err)
	}
}
