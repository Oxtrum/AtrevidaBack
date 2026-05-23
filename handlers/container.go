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

	CategoriasPG      *services.CategoriasService
	ClientesPG        *services.ClientesService
	LocalesHorariosPG *services.LocalesHorariosService
	ServiciosPG       *services.ServiciosPGService
	CombosPG          *services.CombosService
	ReservasPG        *services.ReservasPGService
	LocalesPG         *services.LocalesService

	Import *importacion.ImportService
}

func NewContainer(
	reservas *services.ReservasService,
	writer *services.ReservasWriterService,
	//servicios *services.ServiciosService,
	combos *services.CombosService,
	categoriasPG *services.CategoriasService,
	clientesPG *services.ClientesService,
	localesHorariosPG *services.LocalesHorariosService,
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
		Combos:            combos,
		CategoriasPG:      categoriasPG,
		ClientesPG:        clientesPG,
		LocalesHorariosPG: localesHorariosPG,
		ServiciosPG:       serviciosPG,
		CombosPG:          combosPG,
		ReservasPG:        reservasPG,
		LocalesPG:         localesPG,
		Import:            imp,
	}
}
