package repository

import "atrevida-agenda-api/models"

type TipoEspacioInput struct {
	TipoEspacio      string
	CantidadEspacios int
}

type LocalesRepository interface {
	GetAllLocales() ([]models.LocalConEspacios, error)
	GetLocalById(id int) (*models.LocalConEspacios, error)
	CreateLocal(nombre string, espacios []TipoEspacioInput) (int, error)
	UpdateLocal(id int, nombre *string, activo *bool) error
}
