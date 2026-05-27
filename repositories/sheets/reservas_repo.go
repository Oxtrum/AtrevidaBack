package sheets

import (
	"fmt"

	"atrevida-agenda-api/config"
	"atrevida-agenda-api/models"
)

const unsupportedMessage = "Google Sheets ya no soportado"

type ReservasRepo struct{}

func NewReservasRepo(_ *config.Config) *ReservasRepo {
	return &ReservasRepo{}
}

func (r *ReservasRepo) GetAllReservas() []models.LocalReservas {
	return []models.LocalReservas{}
}

func (r *ReservasRepo) GetSheetData(_ string) [][]interface{} {
	return [][]interface{}{}
}

func (r *ReservasRepo) GetCeldaRaw(_, _ string) (string, error) {
	return "", fmt.Errorf(unsupportedMessage)
}

func (r *ReservasRepo) WriteCelda(_, _, _ string) error {
	return fmt.Errorf(unsupportedMessage)
}

func (r *ReservasRepo) ResolverCoordenada(_, _, _, _ string) (string, error) {
	return "", fmt.Errorf(unsupportedMessage)
}
