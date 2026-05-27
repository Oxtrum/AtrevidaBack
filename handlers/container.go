package handlers

import (
	"atrevida-agenda-api/services"
)

type Container struct {
	CategoriasPG      *services.CategoriasService
	ClientesPG        *services.ClientesService
	LocalesHorariosPG *services.LocalesHorariosService
	ServiciosPG       *services.ServiciosPGService
	CombosPG          *services.CombosService
	ComboServiciosPG  *services.ComboServiciosService
	ReservasPG        *services.ReservasPGService
	LocalesPG         *services.LocalesService
}

func NewContainer(
	categoriasPG *services.CategoriasService,
	clientesPG *services.ClientesService,
	localesHorariosPG *services.LocalesHorariosService,
	serviciosPG *services.ServiciosPGService,
	combosPG *services.CombosService,
	comboServiciosPG *services.ComboServiciosService,
	reservasPG *services.ReservasPGService,
	localesPG *services.LocalesService,
) *Container {
	return &Container{
		CategoriasPG:      categoriasPG,
		ClientesPG:        clientesPG,
		LocalesHorariosPG: localesHorariosPG,
		ServiciosPG:       serviciosPG,
		CombosPG:          combosPG,
		ComboServiciosPG:  comboServiciosPG,
		ReservasPG:        reservasPG,
		LocalesPG:         localesPG,
	}
}
