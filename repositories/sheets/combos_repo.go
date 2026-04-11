package sheets

import (
	"atrevida-agenda-api/models"
	"atrevida-agenda-api/utils"
)

func (r *ReservasRepo) GetAllCombos() []models.ComboItem {
	data := r.GetSheetData(serviciosSheetName)
	return utils.ParseCombosSheet(data, utils.MinFilaCombos, utils.MaxFilaCombos)
}
