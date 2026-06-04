package services

import (
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type ServiciosPGService struct {
	repo repository.ServiciosRepository
}

func NewServiciosPGService(repo repository.ServiciosRepository) *ServiciosPGService {
	return &ServiciosPGService{repo: repo}
}

func (s *ServiciosPGService) GetServiciosFiltrados(f FiltroServicios) []models.ServicioItem {
	todos := s.repo.GetAllServicios()

	var resultado []models.ServicioItem
	for _, item := range todos {
		if f.Nombre != "" &&
			!strings.Contains(strings.ToLower(item.Nombre), strings.ToLower(f.Nombre)) {
			continue
		}
		if f.Categoria != "" &&
			!strings.Contains(strings.ToLower(item.Categoria), strings.ToLower(f.Categoria)) {
			continue
		}
		if f.Local != "" && !strings.Contains(strings.ToLower(item.Local), strings.ToLower(f.Local)) {
			continue
		}
		if f.Sesiones > 0 && item.Sesiones != f.Sesiones {
			continue
		}
		if f.RequiereEvaluacion != nil && item.RequiereEvaluacion != *f.RequiereEvaluacion {
			continue
		}
		resultado = append(resultado, item)
	}
	return resultado
}

func (s *ServiciosPGService) GetServicioByID(id int) (*models.ServicioItem, error) {
	return s.repo.GetServicioByID(id)
}

type CrearServicioPGInput struct {
	Nombre               string
	CategoriaNombre      string
	Tiempo               string
	Costo                *float64
	Sesiones             int
	TipoEspacioRequerido *string
	RequiereEvaluacion   bool
	LocalNombre          string
}

func (s *ServiciosPGService) CreateServicio(input CrearServicioPGInput) (int, error) {
	var tipoEspacio *string
	if input.TipoEspacioRequerido != nil {
		t := strings.ToUpper(*input.TipoEspacioRequerido)
		tipoEspacio = &t
	}

	return s.repo.CreateServicio(repository.CrearServicioInput{
		Nombre:               strings.TrimSpace(input.Nombre),
		CategoriaNombre:      strings.TrimSpace(input.CategoriaNombre),
		Tiempo:               strings.TrimSpace(input.Tiempo),
		Costo:                input.Costo,
		Sesiones:             input.Sesiones,
		TipoEspacioRequerido: tipoEspacio,
		RequiereEvaluacion:   input.RequiereEvaluacion,
		LocalNombre:          strings.TrimSpace(input.LocalNombre),
	})
}

type ActualizarServicioPGInput struct {
	ID                   int
	Nombre               *string
	CategoriaNombre      *string
	Tiempo               *string
	Costo                *float64
	Sesiones             *int
	TipoEspacioRequerido *string
	RequiereEvaluacion   *bool
	Activo               *bool
}

func (s *ServiciosPGService) UpdateServicio(input ActualizarServicioPGInput) error {
	var tipoEspacio *string
	if input.TipoEspacioRequerido != nil {
		t := strings.ToUpper(*input.TipoEspacioRequerido)
		tipoEspacio = &t
	}

	return s.repo.UpdateServicio(repository.ActualizarServicioInput{
		ID:                   input.ID,
		Nombre:               input.Nombre,
		CategoriaNombre:      input.CategoriaNombre,
		Tiempo:               input.Tiempo,
		Costo:                input.Costo,
		Sesiones:             input.Sesiones,
		TipoEspacioRequerido: tipoEspacio,
		RequiereEvaluacion:   input.RequiereEvaluacion,
		Activo:               input.Activo,
	})
}

func (s *ServiciosPGService) ActivarServicioEnLocal(servicioID int, localNombre string) error {
	return s.repo.AddServicioInLocal(servicioID, strings.TrimSpace(localNombre))
}

func (s *ServiciosPGService) SetVisiblePacienteNuevoEnLocal(servicioID, localID int, visible bool) error {
	return s.repo.SetVisiblePacienteNuevoEnLocal(servicioID, localID, visible)
}

func (s *ServiciosPGService) DeleteServicio(id int) error {
	return s.repo.DeleteServicio(id)
}
