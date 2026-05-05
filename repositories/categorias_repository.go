package repository

import "atrevida-agenda-api/models"

type CategoriasRepository interface {
	GetAllCategorias() ([]models.CategoriaPG, error)
	CreateCategoria(nombre string) (int, error)
}
