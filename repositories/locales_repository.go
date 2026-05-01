package repository

import "atrevida-agenda-api/models"

type LocalesRepository interface {
	GetAllLocales() ([]models.LocalPG, error)
}
