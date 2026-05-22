package services

import (
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type LocalesService struct {
	repo repository.LocalesRepository
}

func NewLocalesService(repo repository.LocalesRepository) *LocalesService {
	return &LocalesService{repo: repo}
}

func (s *LocalesService) GetLocales() ([]models.LocalConEspacios, error) {
	return s.repo.GetAllLocales()
}

func (s *LocalesService) GetLocalById(id int) (*models.LocalConEspacios, error) {
	return s.repo.GetLocalById(id)
}

type CrearLocalInput struct {
	Nombre   string
	Espacios []EspacioInput
}

type EspacioInput struct {
	TipoEspacio      string
	CantidadEspacios int
}

func (s *LocalesService) CreateLocal(input CrearLocalInput) (int, error) {
	espacios := make([]repository.TipoEspacioInput, 0, len(input.Espacios))
	for _, e := range input.Espacios {
		espacios = append(espacios, repository.TipoEspacioInput{
			TipoEspacio:      strings.ToUpper(strings.TrimSpace(e.TipoEspacio)),
			CantidadEspacios: e.CantidadEspacios,
		})
	}
	return s.repo.CreateLocal(strings.TrimSpace(input.Nombre), espacios)
}

type ActualizarLocalInput struct {
	ID     int
	Nombre *string
	Activo *bool
}

func (s *LocalesService) UpdateLocal(input ActualizarLocalInput) error {
	return s.repo.UpdateLocal(input.ID, input.Nombre, input.Activo)
}

func (s *LocalesService) DeleteLocal(id int) error {
	return s.repo.DeleteLocal(id)
}
