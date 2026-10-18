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
	PaquetesPG        *services.PaquetesService
	ReservasPG        *services.ReservasPGService
	LocalesPG         *services.LocalesService
	PagosPG           *services.PagosService
	PlanesPG          *services.PlanesService

	// Trastienda: inventario y finanzas
	Configuracion     *services.ConfiguracionService
	Inventario        *services.InventarioService
	Finanzas          *services.FinanzasService
	GastosFijos       *services.GastosFijosService
	ReporteFinanciero *services.ReporteFinancieroService
	// JobSecret permite al job diario generar gastos fijos sin token de usuario.
	JobSecret string
}

func NewContainer(
	auth *services.AuthService,
	categoriasPG *services.CategoriasService,
	clientesPG *services.ClientesService,
	localesHorariosPG *services.LocalesHorariosService,
	serviciosPG *services.ServiciosPGService,
	combosPG *services.CombosService,
	paquetesPG *services.PaquetesService,
	reservasPG *services.ReservasPGService,
	localesPG *services.LocalesService,
	pagosPG *services.PagosService,
	planesPG *services.PlanesService,
) *Container {
	return &Container{
		Auth:              auth,
		CategoriasPG:      categoriasPG,
		ClientesPG:        clientesPG,
		LocalesHorariosPG: localesHorariosPG,
		ServiciosPG:       serviciosPG,
		CombosPG:          combosPG,
		PaquetesPG:        paquetesPG,
		ReservasPG:        reservasPG,
		LocalesPG:         localesPG,
		PagosPG:           pagosPG,
		PlanesPG:          planesPG,
	}
}
