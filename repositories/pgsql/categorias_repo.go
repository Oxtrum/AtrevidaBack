package pgsql

import (
	"fmt"
	"strings"

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

func (r *CategoriasRepo) GetCategoriasByLocal(localNombre string, localID *int) ([]models.CategoriaPG, error) {
	conditions := []string{"l.activo = TRUE"}
	args := []interface{}{}
	idx := 1

	if localID != nil {
		conditions = append(conditions, fmt.Sprintf("l.id = $%d", idx))
		args = append(args, *localID)
		idx++
	}
	if strings.TrimSpace(localNombre) != "" {
		conditions = append(conditions, fmt.Sprintf("UPPER(l.nombre) = UPPER($%d)", idx))
		args = append(args, strings.TrimSpace(localNombre))
		idx++
	}

	var categorias []models.CategoriaPG
	err := r.db.Select(&categorias, fmt.Sprintf(`
		SELECT c.id, c.nombre
		FROM categorias c
		JOIN categorias_locales cl ON cl.categoria_id = c.id
		JOIN locales l ON l.id = cl.local_id
		WHERE %s
		ORDER BY c.nombre
	`, strings.Join(conditions, " AND ")), args...)
	if err != nil {
		return nil, fmt.Errorf("error al consultar categorias por local: %w", err)
	}
	if categorias == nil {
		categorias = []models.CategoriaPG{}
	}

	return categorias, nil
}

func (r *CategoriasRepo) CreateCategoria(nombre string, localID *int) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if localID != nil {
		if err := validarLocalActivoTx(tx, *localID); err != nil {
			return 0, err
		}
	}

	var categoriaID int
	err = tx.QueryRowx(`
		INSERT INTO categorias (nombre)
		VALUES ($1)
		RETURNING id
	`, nombre).Scan(&categoriaID)
	if err != nil {
		return 0, fmt.Errorf("error al crear categoria: %w", err)
	}

	if localID != nil {
		_, err = tx.Exec(`
			INSERT INTO categorias_locales (categoria_id, local_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, categoriaID, *localID)
		if err != nil {
			return 0, fmt.Errorf("error al asociar categoria con local: %w", err)
		}
	}

	return categoriaID, tx.Commit()
}

func (r *CategoriasRepo) CreateCategoriaLocal(categoriaID, localID int) error {
	if err := r.validarCategoriaYLocal(categoriaID, localID); err != nil {
		return err
	}

	_, err := r.db.Exec(`
		INSERT INTO categorias_locales (categoria_id, local_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, categoriaID, localID)
	if err != nil {
		return fmt.Errorf("error al asociar categoria con local: %w", err)
	}

	return nil
}

func (r *CategoriasRepo) DeleteCategoriaLocal(categoriaID, localID int) error {
	res, err := r.db.Exec(`
		DELETE FROM categorias_locales
		WHERE categoria_id = $1
		  AND local_id = $2
	`, categoriaID, localID)
	if err != nil {
		return fmt.Errorf("error al eliminar asociacion categoria-local: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("asociacion categoria-local no encontrada")
	}

	return nil
}

func (r *CategoriasRepo) validarCategoriaYLocal(categoriaID, localID int) error {
	var categoriaExiste bool
	if err := r.db.Get(&categoriaExiste, `
		SELECT EXISTS(SELECT 1 FROM categorias WHERE id = $1)
	`, categoriaID); err != nil {
		return fmt.Errorf("error al validar categoria: %w", err)
	}
	if !categoriaExiste {
		return fmt.Errorf("categoria con id %d no encontrada", categoriaID)
	}

	var localExiste bool
	if err := r.db.Get(&localExiste, `
		SELECT EXISTS(SELECT 1 FROM locales WHERE id = $1 AND activo = TRUE)
	`, localID); err != nil {
		return fmt.Errorf("error al validar local: %w", err)
	}
	if !localExiste {
		return fmt.Errorf("local con id %d no encontrado o inactivo", localID)
	}

	return nil
}

func validarLocalActivoTx(tx *sqlx.Tx, localID int) error {
	var localExiste bool
	if err := tx.Get(&localExiste, `
		SELECT EXISTS(SELECT 1 FROM locales WHERE id = $1 AND activo = TRUE)
	`, localID); err != nil {
		return fmt.Errorf("error al validar local: %w", err)
	}
	if !localExiste {
		return fmt.Errorf("local con id %d no encontrado o inactivo", localID)
	}

	return nil
}
