package services

import (
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type FiltroServicios struct {
	Nombre             string // búsqueda parcial, case-insensitive
	Categoria          string // búsqueda parcial, case-insensitive
	Local              string // "ARANJUEZ", "CENTRO" — exacto, case-insensitive
	Sesiones           int    // 0 = sin filtro; >0 = exacto
	RequiereEvaluacion *bool  // nil = sin filtro
}

type ServiciosService struct {
	repo repository.ServiciosRepository
}

func NewServiciosService(repo repository.ServiciosRepository) *ServiciosService {
	return &ServiciosService{repo: repo}
}

func (s *ServiciosService) GetServiciosFiltrados(f FiltroServicios) []models.ServicioItem {
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
