package handlers

import "atrevida-agenda-api/services"

// Container agrupa los servicios que necesitan los handlers.
type Container struct {
	Reservas  *services.ReservasService
	Writer    *services.ReservasWriterService
	Servicios *services.ServiciosService
}

func NewContainer(
	reservas *services.ReservasService,
	writer *services.ReservasWriterService,
	servicios *services.ServiciosService,
) *Container {
	return &Container{
		Reservas:  reservas,
		Writer:    writer,
		Servicios: servicios,
	}
}
