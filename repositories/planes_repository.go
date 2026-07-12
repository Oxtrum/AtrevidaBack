package repository

import (
	"errors"
	"time"

	"atrevida-agenda-api/models"
)

var (
	ErrPlanNoEncontrado       = errors.New("plan no encontrado")
	ErrPlanDatosInvalidos     = errors.New("datos de plan invalidos")
	ErrPlanTransicionInvalida = errors.New("transicion de estado no permitida")
	ErrPlanEstadoBloqueado    = errors.New("el plan no permite modificaciones en su estado actual")
)

type FiltroPlanes struct {
	Cliente        string
	ClienteID      *int
	LocalID        *int
	Local          string
	Estado         string
	EstadoCobranza string
	FechaDesde     *time.Time
	FechaHasta     *time.Time
}

type CrearPlanInput struct {
	ClienteID           int
	LocalID             int
	ComboIDOrigen       *int
	ComboNombreTexto *string
	FechaInicio         *time.Time
	FechaFin            *time.Time
	Estado              string
	TipoPago            string
	Subtotal            float64
	Descuento           float64
	PrecioTotal         float64
	Moneda              string
	Notas               *string
	CreadoPor           *int
	Servicios           []CrearPlanServicioInput
	Cuotas              []CrearPlanCuotaInput
	PagoCodigo          *string
}

type CrearPlanServicioInput struct {
	ServicioIDOrigen       *int
	NombreTexto         string
	TiempoTexto         *string
	PrecioUnitarioTexto *float64
	SesionesContratadas    int
	Orden                  int
	SesionNumero           int
}

type CrearPlanCuotaInput struct {
	Numero      int
	Vencimiento *string
	Monto       float64
}

type ActualizarPlanInput struct {
	ID          int
	Notas       *string
	FechaInicio *time.Time
	FechaFin    *time.Time
}

type PlanesRepository interface {
	ListPlanes(filtro FiltroPlanes) ([]models.PlanPG, error)
	GetPlanByID(id int) (*models.PlanCompletoPG, error)
	CreatePlan(input CrearPlanInput) (int, error)
	UpdatePlan(input ActualizarPlanInput) error
	UpdatePlanEstado(id int, estado string, usuarioID *int) error
	MarcarSesion(planID, numero int, realizado bool) (int, error)
}
