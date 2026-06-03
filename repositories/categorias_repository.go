package repository

import "atrevida-agenda-api/models"

type CategoriasRepository interface {
	GetAllCategorias() ([]models.CategoriaPG, error)
	GetCategoriasByLocal(localNombre string, localID *int) ([]models.CategoriaPG, error)
	CreateCategoria(nombre string, localID *int) (int, error)
	CreateCategoriaLocal(categoriaID, localID int) error
	DeleteCategoriaLocal(categoriaID, localID int) error
}
