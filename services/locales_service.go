package services

import (
	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type LocalesService struct {
	repo repository.LocalesRepository
}

func NewLocalesService(repo repository.LocalesRepository) *LocalesService {
	return &LocalesService{repo: repo}
}

func (s *LocalesService) GetLocales() ([]models.LocalPG, error) {
	return s.repo.GetAllLocales()
}
