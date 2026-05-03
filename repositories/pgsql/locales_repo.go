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

func (r *LocalesRepo) GetAllLocales() ([]models.LocalConEspacios, error) {
	var locales []models.LocalConEspacios
	err := r.db.Select(&locales, `
		SELECT id, nombre, activo
		FROM locales
		WHERE activo = TRUE
		ORDER BY nombre
	`)
	if err != nil {
		return nil, fmt.Errorf("error al consultar locales: %w", err)
	}

	for i := range locales {
		espacios, err := r.getEspacios(locales[i].ID)
		if err != nil {
			return nil, err
		}
		locales[i].Espacios = espacios
	}

	return locales, nil
}

func (r *LocalesRepo) GetLocalById(id int) (*models.LocalConEspacios, error) {
	if id == 0 {
		return nil, fmt.Errorf("debe consultarse con un id valido")
	}

	var local models.LocalConEspacios

	err := r.db.Get(&local, `
		SELECT id, nombre, activo
		FROM locales
		WHERE id = $1 AND activo = TRUE
	`, id)
	if err != nil {
		return nil, fmt.Errorf("error al consultar local: %w", err)
	}

	espacios, err := r.getEspacios(local.ID)
	if err != nil {
		return nil, err
	}
	local.Espacios = espacios

	return &local, nil
}

func (r *LocalesRepo) getEspacios(localID int) ([]models.TipoEspacioLocal, error) {
	var espacios []models.TipoEspacioLocal
	err := r.db.Select(&espacios, `
		SELECT tipo_espacio, cantidad_espacios
		FROM tipos_espacio_locales
		WHERE local_id = $1
		ORDER BY tipo_espacio
	`, localID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar espacios del local %d: %w", localID, err)
	}
	return espacios, nil
}

func (r *LocalesRepo) CreateLocal(nombre string, espacios []repository.TipoEspacioInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var localID int
	err = tx.QueryRowx(`
		INSERT INTO locales (nombre, activo)
		VALUES ($1, TRUE)
		RETURNING id
	`, nombre).Scan(&localID)
	if err != nil {
		return 0, fmt.Errorf("error al crear local: %w", err)
	}

	for _, e := range espacios {
		_, err = tx.Exec(`
			INSERT INTO tipos_espacio_locales (tipo_espacio, cantidad_espacios, local_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (tipo_espacio, local_id) DO UPDATE
				SET cantidad_espacios = EXCLUDED.cantidad_espacios
		`, e.TipoEspacio, e.CantidadEspacios, localID)
		if err != nil {
			return 0, fmt.Errorf("error al insertar espacio '%s': %w", e.TipoEspacio, err)
		}
	}

	return localID, tx.Commit()
}

func (r *LocalesRepo) UpdateLocal(id int, nombre *string, activo *bool) error {
	if nombre == nil && activo == nil {
		return fmt.Errorf("debe especificarse al menos un campo a modificar")
	}

	sets := []string{}
	args := []interface{}{}
	idx := 1

	if nombre != nil {
		sets = append(sets, fmt.Sprintf("nombre = $%d", idx))
		args = append(args, *nombre)
		idx++
	}
	if activo != nil {
		sets = append(sets, fmt.Sprintf("activo = $%d", idx))
		args = append(args, *activo)
		idx++
	}

	args = append(args, id)
	query := fmt.Sprintf(
		"UPDATE locales SET %s WHERE id = $%d",
		joinSets(sets), idx,
	)

	res, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error al actualizar local: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("local con id %d no encontrado", id)
	}
	return nil
}

func joinSets(sets []string) string {
	result := ""
	for i, s := range sets {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
