package services

import (
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type ComboServiciosService struct {
	repo repository.ComboServiciosRepository
}

func NewComboServiciosService(repo repository.ComboServiciosRepository) *ComboServiciosService {
	return &ComboServiciosService{repo: repo}
}

type CrearComboServicioPGInput struct {
	ComboID       int
	ServicioID    *int
	ServicioTexto string
	Tiempo        string
	Costo         *float64
	Sesiones      int
	Orden         int
}

type ActualizarComboServicioPGInput struct {
	ID            int
	ServicioID    *int
	ServicioTexto *string
	Tiempo        *string
	Costo         *float64
	Sesiones      *int
	Orden         *int
}

func (s *ComboServiciosService) GetByID(id int) (*models.ComboServicioDetallePG, error) {
	return s.repo.GetComboServicioByID(id)
}

func (s *ComboServiciosService) GetByComboID(comboID int) ([]models.ComboServicioDetallePG, error) {
	return s.repo.GetComboServiciosByComboID(comboID)
}

func (s *ComboServiciosService) Create(input CrearComboServicioPGInput) (int, error) {
	return s.repo.CreateComboServicio(repository.CrearComboServicioInput{
		ComboID:       input.ComboID,
		ServicioID:    input.ServicioID,
		ServicioTexto: strings.TrimSpace(input.ServicioTexto),
		Tiempo:        strings.TrimSpace(input.Tiempo),
		Costo:         input.Costo,
		Sesiones:      input.Sesiones,
		Orden:         input.Orden,
	})
}

func (s *ComboServiciosService) Update(input ActualizarComboServicioPGInput) error {
	if input.ServicioTexto != nil {
		v := strings.TrimSpace(*input.ServicioTexto)
		input.ServicioTexto = &v
	}
	if input.Tiempo != nil {
		v := strings.TrimSpace(*input.Tiempo)
		input.Tiempo = &v
	}

	return s.repo.UpdateComboServicio(repository.ActualizarComboServicioInput{
		ID:            input.ID,
		ServicioID:    input.ServicioID,
		ServicioTexto: input.ServicioTexto,
		Tiempo:        input.Tiempo,
		Costo:         input.Costo,
		Sesiones:      input.Sesiones,
		Orden:         input.Orden,
	})
}

func (s *ComboServiciosService) Delete(id int) error {
	return s.repo.DeleteComboServicio(id)
}
