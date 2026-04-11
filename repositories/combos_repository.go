package repository

import "atrevida-agenda-api/models"

type CombosRepository interface {
	GetAllCombos() []models.ComboItem
}
