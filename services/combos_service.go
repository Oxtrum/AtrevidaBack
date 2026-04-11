package services

import (
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type FiltroCombos struct {
	Nombre    string
	Categoria string
	Local     string
	Sesiones  int
}

type CombosService struct {
	repo repository.CombosRepository
}

func NewCombosService(repo repository.CombosRepository) *CombosService {
	return &CombosService{repo: repo}
}

func (s *CombosService) GetCombosFiltrados(f FiltroCombos) []models.ComboItem {
	todos := s.repo.GetAllCombos()

	var resultado []models.ComboItem
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
		if f.Sesiones > 0 && item.SesionesTotales != f.Sesiones {
			continue
		}
		resultado = append(resultado, item)
	}

	return resultado
}
