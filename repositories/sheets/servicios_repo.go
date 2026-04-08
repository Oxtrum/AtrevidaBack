package sheets

import (
	"atrevida-agenda-api/models"
	"atrevida-agenda-api/utils"
)

const serviciosSheetName = "SERVICIOS"

func (r *ReservasRepo) GetAllServicios() []models.ServicioItem {
	data := r.GetSheetData(serviciosSheetName)
	return utils.ParseServiciosSheet(data, utils.MaxFilaServicios)
}
