package repository

import "context"

import "atrevida-agenda-api/models"

type TipoEspacioInput struct {
	TipoEspacio      string
	CantidadEspacios int
}

type LocalesRepository interface {
	GetAllLocales(ctx context.Context) ([]models.LocalConEspacios, error)
	GetLocalById(id int) (*models.LocalConEspacios, error)
	CreateLocal(nombre string, espacios []TipoEspacioInput) (int, error)
	UpdateLocal(id int, nombre *string, activo *bool) error
	DeleteLocal(id int) error
	// SetEspacios fija la cantidad de cada tipo de espacio del local (crea el tipo si no existia).
	SetEspacios(localID int, espacios []TipoEspacioInput) error
}
