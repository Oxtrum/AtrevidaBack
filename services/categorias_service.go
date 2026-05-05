package services

import (
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type CategoriasService struct {
	repo repository.CategoriasRepository
}

func NewCategoriasService(repo repository.CategoriasRepository) *CategoriasService {
	return &CategoriasService{repo: repo}
}

func (s *CategoriasService) GetCategorias() ([]models.CategoriaPG, error) {
	return s.repo.GetAllCategorias()
}

type CrearCategoriaInput struct {
	Nombre string
}

func (s *CategoriasService) CreateCategoria(input CrearCategoriaInput) (int, error) {

	return s.repo.CreateCategoria(strings.TrimSpace(input.Nombre))
}
