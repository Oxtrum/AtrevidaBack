package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var (
	ErrPlanInvalido         = errors.New("datos de plan invalidos")
	ErrPlanNoEncontrado     = errors.New("plan no encontrado")
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

type PlanesService struct {
	repo repository.PlanesRepository
}

func NewPlanesService(repo repository.PlanesRepository) *PlanesService {
	return &PlanesService{repo: repo}
}

func (s *PlanesService) ListarPlanes(f FiltroPlanes) ([]models.PlanPG, error) {
	planes, err := s.repo.ListPlanes(repository.FiltroPlanes{
		Cliente:        strings.TrimSpace(f.Cliente),
		ClienteID:      f.ClienteID,
		LocalID:        f.LocalID,
		Local:          strings.TrimSpace(f.Local),
		Estado:         strings.TrimSpace(f.Estado),
		EstadoCobranza: strings.TrimSpace(f.EstadoCobranza),
		FechaDesde:     f.FechaDesde,
		FechaHasta:     f.FechaHasta,
	})
	return planes, traducirErrorRepositorioPlan(err)
}

func (s *PlanesService) ObtenerPlan(id int) (*models.PlanCompletoPG, error) {
	if id < 1 {
		return nil, fmt.Errorf("id debe ser un entero positivo: %w", ErrPlanInvalido)
	}
	plan, err := s.repo.GetPlanByID(id)
	return plan, traducirErrorRepositorioPlan(err)
}

func traducirErrorRepositorioPlan(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrPlanNoEncontrado) {
		return fmt.Errorf("%w: %v", ErrPlanNoEncontrado, err)
	}
	if errors.Is(err, repository.ErrPlanDatosInvalidos) {
		return fmt.Errorf("%w: %v", ErrPlanInvalido, err)
	}
	return err
}
