package services

import (
	"context"
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

func (s *CategoriasService) GetCategorias(ctx context.Context) ([]models.CategoriaPG, error) {
	return s.repo.GetAllCategorias(ctx)
}

type FiltroCategorias struct {
	Context context.Context
	Local   string
	LocalID *int
}

func (s *CategoriasService) GetCategoriasFiltradas(filtro FiltroCategorias) ([]models.CategoriaPG, error) {
	local := strings.TrimSpace(filtro.Local)
	if local == "" && filtro.LocalID == nil {
		return s.repo.GetAllCategorias(filtro.Context)
	}

	return s.repo.GetCategoriasByLocal(filtro.Context, local, filtro.LocalID)
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

type ActualizarCategoriaInput struct {
	ID     int
	Nombre string
}

func (s *CategoriasService) UpdateCategoria(input ActualizarCategoriaInput) error {
	if input.ID < 1 {
		return errors.New("id debe ser un entero positivo")
	}
	nombre := strings.TrimSpace(input.Nombre)
	if nombre == "" {
		return errors.New("nombre es requerido")
	}

	return s.repo.UpdateCategoria(input.ID, nombre)
}

func (s *CategoriasService) DeleteCategoria(id int) error {
	if id < 1 {
		return errors.New("id debe ser un entero positivo")
	}

	return s.repo.DeleteCategoria(id)
}

func (s *CategoriasService) GetLocalesByCategoria(ctx context.Context, categoriaID int) ([]models.LocalPG, error) {
	if categoriaID < 1 {
		return nil, errors.New("id debe ser un entero positivo")
	}

	return s.repo.GetLocalesByCategoria(ctx, categoriaID)
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
