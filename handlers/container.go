package handlers

import (
	"atrevida-agenda-api/importacion"
	"atrevida-agenda-api/services"
)

type Container struct {
	Reservas *services.ReservasService
	Writer   *services.ReservasWriterService
	//Servicios *services.ServiciosService
	Combos *services.CombosService

	ServiciosPG *services.ServiciosPGService
	CombosPG    *services.CombosService
	ReservasPG  *services.ReservasPGService
	LocalesPG   *services.LocalesService

	Import *importacion.ImportService
}

func NewContainer(
	reservas *services.ReservasService,
	writer *services.ReservasWriterService,
	//servicios *services.ServiciosService,
	combos *services.CombosService,
	serviciosPG *services.ServiciosPGService,
	combosPG *services.CombosService,
	reservasPG *services.ReservasPGService,
	localesPG *services.LocalesService,
	imp *importacion.ImportService,
) *Container {
	return &Container{
		Reservas: reservas,
		Writer:   writer,
		//Servicios:   servicios,
		Combos:      combos,
		ServiciosPG: serviciosPG,
		CombosPG:    combosPG,
		ReservasPG:  reservasPG,
		LocalesPG:   localesPG,
		Import:      imp,
	}
}
