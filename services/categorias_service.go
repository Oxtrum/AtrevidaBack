package services

import (
	"errors"
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

type FiltroCategorias struct {
	Local   string
	LocalID *int
}

func (s *CategoriasService) GetCategoriasFiltradas(filtro FiltroCategorias) ([]models.CategoriaPG, error) {
	local := strings.TrimSpace(filtro.Local)
	if local == "" && filtro.LocalID == nil {
		return s.repo.GetAllCategorias()
	}

	return s.repo.GetCategoriasByLocal(local, filtro.LocalID)
}

type CrearCategoriaInput struct {
	Nombre  string
	LocalID *int
}

func (s *CategoriasService) CreateCategoria(input CrearCategoriaInput) (int, error) {
	if input.LocalID != nil && *input.LocalID < 1 {
		return 0, errors.New("local_id debe ser un entero positivo")
	}

	return s.repo.CreateCategoria(strings.TrimSpace(input.Nombre), input.LocalID)
}

type CategoriaLocalInput struct {
	CategoriaID int
	LocalID     int
}

func (s *CategoriasService) CreateCategoriaLocal(input CategoriaLocalInput) error {
	if input.CategoriaID < 1 || input.LocalID < 1 {
		return errors.New("categoria_id y local_id deben ser enteros positivos")
	}
	return s.repo.CreateCategoriaLocal(input.CategoriaID, input.LocalID)
}

func (s *CategoriasService) DeleteCategoriaLocal(input CategoriaLocalInput) error {
	if input.CategoriaID < 1 || input.LocalID < 1 {
		return errors.New("categoria_id y local_id deben ser enteros positivos")
	}
	return s.repo.DeleteCategoriaLocal(input.CategoriaID, input.LocalID)
}
