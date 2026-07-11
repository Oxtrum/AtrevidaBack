package repository

import (
	"errors"
	"time"

	"atrevida-agenda-api/models"
)

var (
	ErrPlanNoEncontrado         = errors.New("plan no encontrado")
	ErrPlanDatosInvalidos       = errors.New("datos de plan invalidos")
	ErrPlanTransicionInvalida   = errors.New("transicion de estado no permitida")
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

type PlanesRepository interface {
	ListPlanes(filtro FiltroPlanes) ([]models.PlanPG, error)
	GetPlanByID(id int) (*models.PlanCompletoPG, error)
}
