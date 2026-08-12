package repository

import (
	"context"
	"time"

	"atrevida-agenda-api/models"
)

type FiltroReservasPG struct {
	Context            context.Context
	LocalID            *int
	LocalNombre        string
	Fecha              *time.Time
	FechaDesde         *time.Time
	FechaHasta         *time.Time
	Cliente            string
	NumeroTelefono     string
	ServicioSolicitado string
	ServicioConfirmado string
	Estado             string
	TipoEspacio        string
	PlanID             *int
	SoloActivas        bool
	PageLimit          int
	CursorSet          bool
	CursorLocal        string
	CursorFecha        time.Time
	CursorHora         string
	CursorID           int
}

type CreateReservaInput struct {
	LocalNombre        string
	TipoEspacio        string
	Fecha              time.Time
	HoraDesde          string
	HoraHasta          string
	Cliente            string
	Estado             string
	NumeroTelefono     string
	PlanID             *int
	ServicioNombre     string
	ServicioSolicitado string
	ServicioConfirmado *string
	Precio             *float64
	Notas              string
	Detalle            []CrearDetalleInput
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

	NuevaFecha              *time.Time
	NuevaHoraDesde          *string
	NuevaHoraHasta          *string
	NuevoTipo               *string
	NuevoCliente            *string
	NuevoNumeroTelefono     *string
	NuevoServicio           *string
	NuevoServicioSolicitado *string
	NuevoServicioConfirmado *string
	NuevoPrecio             *float64
	NuevasNotas             *string
	NuevoLocal              *string
	NuevoPlanID             *int
	// LimpiarPlanID desvincula la reserva de su plan (plan_id = NULL).
	LimpiarPlanID bool
	// ResetNotificado marca la reserva como no notificada para reavisar al cliente.
	ResetNotificado bool
}

type UpdateReservaEstadoInput struct {
	ID                 int
	Estado             string
	ServicioConfirmado *string
	Precio             *float64
	TipoEspacio        *string
}

type CapacidadLocal struct {
	LocalNombre string
	TipoEspacio string
	Capacidad   int
}

type FiltroResumenPagosReservas struct {
	Context     context.Context
	LocalID     *int
	LocalNombre string
	Fecha       time.Time
	FechaDesde  time.Time
	FechaHasta  time.Time
}

type ResumenPagosReservas struct {
	IngresosDia       float64 `db:"ingresos_dia"`
	IngresosSemana    float64 `db:"ingresos_semana"`
	IngresosLunes     float64 `db:"ingresos_lunes"`
	IngresosMartes    float64 `db:"ingresos_martes"`
	IngresosMiercoles float64 `db:"ingresos_miercoles"`
	IngresosJueves    float64 `db:"ingresos_jueves"`
	IngresosViernes   float64 `db:"ingresos_viernes"`
	IngresosSabado    float64 `db:"ingresos_sabado"`
	CancelacionesDia  int     `db:"cancelaciones_dia"`
}

type ReservasPGRepository interface {
	GetReservas(f FiltroReservasPG) ([]models.ReservaPGCompleta, error)
	CountReservas(f FiltroReservasPG) (int, error)
	GetReservasAgendadasNoNotificadas(ctx context.Context, localNombre string, limit int) ([]models.ReservaPGCompleta, error)
	GetReservaByID(id int) (*models.ReservaPGCompleta, error)
	GetLocalIDByNombre(nombre string) (int, error)
	GetCapacidades(localNombre string) ([]CapacidadLocal, error)
	GetResumenPagosReservas(f FiltroResumenPagosReservas) (ResumenPagosReservas, error)
	CreateReserva(input CreateReservaInput) (int, error)
	UpdateReserva(input UpdateReservaInput) error
	UpdateReservaEstado(input UpdateReservaEstadoInput) error
	UpdateReservaNotificado(id int, notificado bool) error
	UpdateReservasNotificado(ids []int, notificado bool) (int, error)
	AnularReserva(id int) error
}
