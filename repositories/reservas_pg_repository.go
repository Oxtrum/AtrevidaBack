package repository

import (
	"time"

	"atrevida-agenda-api/models"
)

type FiltroReservasPG struct {
	LocalNombre string
	Fecha       *time.Time
	FechaDesde  *time.Time
	FechaHasta  *time.Time
	Cliente     string
	TipoEspacio string
	PlanID      *int
	SoloActivas bool
}

type CreateReservaInput struct {
	LocalNombre    string
	TipoEspacio    string
	Fecha          time.Time
	HoraDesde      string
	HoraHasta      string
	Cliente        string
	PlanID         *int
	ServicioNombre string
	Precio         *float64
	Notas          string
	Detalle        []CrearDetalleInput
}

type CrearDetalleInput struct {
	ServicioNombre string
	ServicioTiempo string
	Precio         *float64
	Sesiones       int
	Notas          string
}

type UpdateReservaInput struct {
	Id          int
	LocalNombre string

	NuevaFecha     *time.Time
	NuevaHoraDesde *string
	NuevaHoraHasta *string
	NuevoTipo      *string
	NuevoServicio  *string
	NuevoPrecio    *float64
	NuevasNotas    *string
}

type CapacidadLocal struct {
	LocalNombre string
	TipoEspacio string
	Capacidad   int
}

type ReservasPGRepository interface {
	GetReservas(f FiltroReservasPG) ([]models.ReservaPGCompleta, error)
	GetReservaByID(id int) (*models.ReservaPGCompleta, error)
	GetCapacidades(localNombre string) ([]CapacidadLocal, error)
	CreateReserva(input CreateReservaInput) (int, error)
	UpdateReserva(input UpdateReservaInput) error
	AnularReserva(id int) error
}
