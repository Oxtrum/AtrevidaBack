package services

import (
	"context"
	"errors"
	"fmt"
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

func (s *LocalesService) GetLocales(ctx context.Context) ([]models.LocalConEspacios, error) {
	return s.repo.GetAllLocales(ctx)
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

// UpdateEspacios fija la cantidad de mesas (M) y bicicletas (B) del local. La cantidad minima
// es 0: un tipo puede quedar sin espacios (mantenimiento, cierre temporal).
func (s *LocalesService) UpdateEspacios(localID int, espacios []EspacioInput) error {
	if len(espacios) == 0 {
		return errors.New("debe enviarse al menos un tipo de espacio")
	}
	vistos := map[string]bool{}
	normalizados := make([]repository.TipoEspacioInput, 0, len(espacios))
	for _, e := range espacios {
		tipo := strings.ToUpper(strings.TrimSpace(e.TipoEspacio))
		if tipo != "M" && tipo != "B" {
			return fmt.Errorf("tipo_espacio inválido '%s', valores permitidos: M, B", e.TipoEspacio)
		}
		if e.CantidadEspacios < 0 {
			return fmt.Errorf("cantidad_espacios de '%s' no puede ser negativa", tipo)
		}
		if vistos[tipo] {
			return fmt.Errorf("tipo_espacio '%s' repetido", tipo)
		}
		vistos[tipo] = true
		normalizados = append(normalizados, repository.TipoEspacioInput{TipoEspacio: tipo, CantidadEspacios: e.CantidadEspacios})
	}
	return s.repo.SetEspacios(localID, normalizados)
}

func (s *LocalesService) DeleteLocal(id int) error {
	return s.repo.DeleteLocal(id)
}
