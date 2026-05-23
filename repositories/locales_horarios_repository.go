package repository

import "atrevida-agenda-api/models"

type FiltroLocalHorarios struct {
	LocalID   int
	DiaSemana *int
}

type CrearLocalHorarioInput struct {
	LocalID   int
	DiaSemana int
	HoraDesde string
	HoraHasta string
}

type ActualizarLocalHorarioInput struct {
	ID        int
	DiaSemana *int
	HoraDesde *string
	HoraHasta *string
}

type LocalesHorariosRepository interface {
	GetHorarioByID(id int) (*models.LocalHorarioPG, error)
	GetHorariosByLocal(filtro FiltroLocalHorarios) ([]models.LocalHorarioPG, error)
	CreateHorario(input CrearLocalHorarioInput) (int, error)
	UpdateHorario(input ActualizarLocalHorarioInput) error
	DeleteHorario(id int) error
}
