package pgsql

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var _ repository.CategoriasRepository = (*CategoriasRepo)(nil)

type CategoriasRepo struct {
	db *sqlx.DB
}

func NewCategoriasRepo(db *sqlx.DB) *CategoriasRepo {
	return &CategoriasRepo{db: db}
}

func (r *CategoriasRepo) GetAllCategorias() ([]models.CategoriaPG, error) {
	var categorias []models.CategoriaPG
	err := r.db.Select(&categorias, `
		SELECT id, nombre 
		FROM categorias
		ORDER BY nombre
	`)
	if err != nil {
		return nil, fmt.Errorf("error al consultar categorias: %w", err)
	}

	return categorias, nil
}

func (r *CategoriasRepo) CreateCategoria(nombre string) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var categoriaID int
	err = tx.QueryRowx(`
		INSERT INTO categorias (nombre)
		VALUES ($1)
		RETURNING id
	`, nombre).Scan(&categoriaID)
	if err != nil {
		return 0, fmt.Errorf("error al crear categoria: %w", err)
	}

	return categoriaID, tx.Commit()
}
