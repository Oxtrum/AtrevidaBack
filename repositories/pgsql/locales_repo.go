package pgsql

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var _ repository.LocalesRepository = (*LocalesRepo)(nil)

type LocalesRepo struct {
	db *sqlx.DB
}

func NewLocalesRepo(db *sqlx.DB) *LocalesRepo {
	return &LocalesRepo{db: db}
}

func (r *LocalesRepo) GetAllLocales() ([]models.LocalPG, error) {
	var locales []models.LocalPG
	err := r.db.Select(&locales, `
		SELECT id, nombre, activo
		FROM locales
		WHERE activo = TRUE
		ORDER BY nombre
	`)
	if err != nil {
		return nil, fmt.Errorf("error al consultar locales: %w", err)
	}
	return locales, nil
}
