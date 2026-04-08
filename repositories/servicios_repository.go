package repository

import "atrevida-agenda-api/models"

type ServiciosRepository interface {
	GetAllServicios() []models.ServicioItem
}
