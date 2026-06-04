package handlers

import (
	"atrevida-agenda-api/services"
)

type Container struct {
	Auth              *services.AuthService
	CategoriasPG      *services.CategoriasService
	ClientesPG        *services.ClientesService
	LocalesHorariosPG *services.LocalesHorariosService
	ServiciosPG       *services.ServiciosPGService
	CombosPG          *services.CombosService
	ComboServiciosPG  *services.ComboServiciosService
	ReservasPG        *services.ReservasPGService
	LocalesPG         *services.LocalesService
	PagosPG           *services.PagosService
}

func NewContainer(
	auth *services.AuthService,
	categoriasPG *services.CategoriasService,
	clientesPG *services.ClientesService,
	localesHorariosPG *services.LocalesHorariosService,
	serviciosPG *services.ServiciosPGService,
	combosPG *services.CombosService,
	comboServiciosPG *services.ComboServiciosService,
	reservasPG *services.ReservasPGService,
	localesPG *services.LocalesService,
	pagosPG *services.PagosService,
) *Container {
	return &Container{
		Auth:              auth,
		CategoriasPG:      categoriasPG,
		ClientesPG:        clientesPG,
		LocalesHorariosPG: localesHorariosPG,
		ServiciosPG:       serviciosPG,
		CombosPG:          combosPG,
		ComboServiciosPG:  comboServiciosPG,
		ReservasPG:        reservasPG,
		LocalesPG:         localesPG,
		PagosPG:           pagosPG,
	}
}
