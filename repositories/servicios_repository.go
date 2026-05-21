package repository

import "atrevida-agenda-api/models"

type CrearServicioInput struct {
	Nombre               string
	CategoriaNombre      string
	Tiempo               string
	Costo                *float64
	Sesiones             int
	TipoEspacioRequerido *string // "M" | "B" | nil
	RequiereEvaluacion   bool
	LocalNombre          string
}

type ActualizarServicioInput struct {
	ID                   int
	Nombre               *string
	CategoriaNombre      *string
	Tiempo               *string
	Costo                *float64
	Sesiones             *int
	TipoEspacioRequerido *string
	RequiereEvaluacion   *bool
	Activo               *bool
}

type ServiciosRepository interface {
	GetAllServicios() []models.ServicioItem
	GetServicioByID(id int) (*models.ServicioItem, error)
	GetServicioByNombre(nombre string) (*models.ServicioItem, error)
	CreateServicio(input CrearServicioInput) (int, error)
	UpdateServicio(input ActualizarServicioInput) error
	AddServicioInLocal(servicioID int, localNombre string) error
}
