package repository

import (
	"context"

	"atrevida-agenda-api/models"
)

type CategoriasRepository interface {
	GetAllCategorias(ctx context.Context) ([]models.CategoriaPG, error)
	GetCategoriasByLocal(ctx context.Context, localNombre string, localID *int) ([]models.CategoriaPG, error)
	CreateCategoria(nombre string, localID *int) (int, error)
	UpdateCategoria(id int, nombre string) error
	DeleteCategoria(id int) error
	GetLocalesByCategoria(ctx context.Context, categoriaID int) ([]models.LocalPG, error)
	CreateCategoriaLocal(categoriaID, localID int) error
	DeleteCategoriaLocal(categoriaID, localID int) error
}
