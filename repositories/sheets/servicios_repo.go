package sheets

import "context"

import "atrevida-agenda-api/models"

func (r *ReservasRepo) GetAllServicios(_ context.Context) ([]models.ServicioItem, error) {
	return []models.ServicioItem{}, nil
}
