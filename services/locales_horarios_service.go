package services

import (
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type LocalesHorariosService struct {
	repo repository.LocalesHorariosRepository
}

func NewLocalesHorariosService(repo repository.LocalesHorariosRepository) *LocalesHorariosService {
	return &LocalesHorariosService{repo: repo}
}

type FiltroLocalHorarios struct {
	LocalID   int
	DiaSemana *int
}

func (s *LocalesHorariosService) GetHorariosByLocal(filtro FiltroLocalHorarios) ([]models.LocalHorarioPG, error) {
	return s.repo.GetHorariosByLocal(repository.FiltroLocalHorarios{
		LocalID:   filtro.LocalID,
		DiaSemana: filtro.DiaSemana,
	})
}

func (s *LocalesHorariosService) GetHorarioByID(id int) (*models.LocalHorarioPG, error) {
	return s.repo.GetHorarioByID(id)
}

type CrearLocalHorarioInput struct {
	LocalID   int
	DiaSemana int
	HoraDesde string
	HoraHasta string
}

func (s *LocalesHorariosService) CreateHorario(input CrearLocalHorarioInput) (int, error) {
	return s.repo.CreateHorario(repository.CrearLocalHorarioInput{
		LocalID:   input.LocalID,
		DiaSemana: input.DiaSemana,
		HoraDesde: strings.TrimSpace(input.HoraDesde),
		HoraHasta: strings.TrimSpace(input.HoraHasta),
	})
}

type ActualizarLocalHorarioInput struct {
	ID        int
	DiaSemana *int
	HoraDesde *string
	HoraHasta *string
}

func (s *LocalesHorariosService) UpdateHorario(input ActualizarLocalHorarioInput) error {
	var horaDesde *string
	if input.HoraDesde != nil {
		value := strings.TrimSpace(*input.HoraDesde)
		horaDesde = &value
	}

	var horaHasta *string
	if input.HoraHasta != nil {
		value := strings.TrimSpace(*input.HoraHasta)
		horaHasta = &value
	}

	return s.repo.UpdateHorario(repository.ActualizarLocalHorarioInput{
		ID:        input.ID,
		DiaSemana: input.DiaSemana,
		HoraDesde: horaDesde,
		HoraHasta: horaHasta,
	})
}

func (s *LocalesHorariosService) DeleteHorario(id int) error {
	return s.repo.DeleteHorario(id)
}
