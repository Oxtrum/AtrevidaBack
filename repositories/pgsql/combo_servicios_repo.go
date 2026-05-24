package pgsql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type ComboServiciosRepo struct {
	db *sqlx.DB
}

func NewComboServiciosRepo(db *sqlx.DB) *ComboServiciosRepo {
	return &ComboServiciosRepo{db: db}
}

func (r *ComboServiciosRepo) GetComboServicioByID(id int) (*models.ComboServicioDetallePG, error) {
	var item models.ComboServicioDetallePG
	err := r.db.Get(&item, comboServicioSelectQuery()+`
		WHERE cs.id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("combo_servicio con id %d no encontrado", id)
	}

	return &item, nil
}

func (r *ComboServiciosRepo) GetComboServiciosByComboID(comboID int) ([]models.ComboServicioDetallePG, error) {
	var existe bool
	if err := r.db.Get(&existe, `
		SELECT EXISTS(SELECT 1 FROM combos WHERE id = $1 AND activo = TRUE)
	`, comboID); err != nil {
		return nil, fmt.Errorf("error al validar combo: %w", err)
	}
	if !existe {
		return nil, fmt.Errorf("combo con id %d no encontrado o inactivo", comboID)
	}

	var items []models.ComboServicioDetallePG
	err := r.db.Select(&items, comboServicioSelectQuery()+`
		WHERE cs.combo_id = $1
		ORDER BY cs.orden, cs.id
	`, comboID)
	if err != nil {
		return nil, fmt.Errorf("error al listar servicios del combo: %w", err)
	}
	if items == nil {
		items = []models.ComboServicioDetallePG{}
	}

	return items, nil
}

func (r *ComboServiciosRepo) CreateComboServicio(input repository.CrearComboServicioInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if err := validarComboActivo(tx, input.ComboID); err != nil {
		return 0, err
	}
	if err := validarComboServicioInput(tx, input.ServicioID, input.ServicioTexto, input.Sesiones); err != nil {
		return 0, err
	}

	var id int
	err = tx.QueryRowx(`
		INSERT INTO combo_servicios
			(combo_id, servicio_id, servicio_texto, tiempo, costo, sesiones, orden)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		input.ComboID,
		input.ServicioID,
		nullStr(input.ServicioTexto),
		nullStr(input.Tiempo),
		input.Costo,
		input.Sesiones,
		input.Orden,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error al crear servicio del combo: %w", err)
	}

	return id, tx.Commit()
}

func (r *ComboServiciosRepo) UpdateComboServicio(input repository.ActualizarComboServicioInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	sets := []string{}
	args := []interface{}{}
	idx := 1

	if input.ServicioID != nil {
		if err := validarServicioActivo(tx, *input.ServicioID); err != nil {
			return err
		}
		sets = append(sets, fmt.Sprintf("servicio_id = $%d", idx))
		args = append(args, *input.ServicioID)
		idx++
	}
	if input.ServicioTexto != nil {
		sets = append(sets, fmt.Sprintf("servicio_texto = $%d", idx))
		args = append(args, nullStr(*input.ServicioTexto))
		idx++
	}
	if input.Tiempo != nil {
		sets = append(sets, fmt.Sprintf("tiempo = $%d", idx))
		args = append(args, nullStr(*input.Tiempo))
		idx++
	}
	if input.Costo != nil {
		sets = append(sets, fmt.Sprintf("costo = $%d", idx))
		args = append(args, *input.Costo)
		idx++
	}
	if input.Sesiones != nil {
		if *input.Sesiones < 1 {
			return fmt.Errorf("sesiones debe ser un numero entero positivo")
		}
		sets = append(sets, fmt.Sprintf("sesiones = $%d", idx))
		args = append(args, *input.Sesiones)
		idx++
	}
	if input.Orden != nil {
		sets = append(sets, fmt.Sprintf("orden = $%d", idx))
		args = append(args, *input.Orden)
		idx++
	}
	if len(sets) == 0 {
		return fmt.Errorf("debe especificarse al menos un campo a modificar")
	}

	args = append(args, input.ID)
	query := fmt.Sprintf(`
		UPDATE combo_servicios
		SET %s
		WHERE id = $%d
	`, strings.Join(sets, ", "), idx)

	res, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error al actualizar servicio del combo: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("combo_servicio con id %d no encontrado", input.ID)
	}

	return tx.Commit()
}

func (r *ComboServiciosRepo) DeleteComboServicio(id int) error {
	res, err := r.db.Exec(`DELETE FROM combo_servicios WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("error al eliminar servicio del combo: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("combo_servicio con id %d no encontrado", id)
	}

	return nil
}

func comboServicioSelectQuery() string {
	return `
		SELECT
			cs.id,
			cs.combo_id,
			cb.nombre AS combo_nombre,
			cs.servicio_id,
			cs.servicio_texto,
			COALESCE(NULLIF(BTRIM(cs.servicio_texto), ''), s.nombre, '') AS servicio_nombre,
			cs.tiempo,
			cs.costo,
			cs.sesiones,
			cs.orden
		FROM combo_servicios cs
		JOIN combos cb ON cb.id = cs.combo_id
		LEFT JOIN servicios s ON s.id = cs.servicio_id
	`
}

func validarComboActivo(tx *sqlx.Tx, comboID int) error {
	if comboID < 1 {
		return fmt.Errorf("combo_id debe ser un numero entero positivo")
	}

	var existe bool
	err := tx.QueryRowx(`
		SELECT EXISTS(SELECT 1 FROM combos WHERE id = $1 AND activo = TRUE)
	`, comboID).Scan(&existe)
	if err != nil {
		return fmt.Errorf("error al validar combo: %w", err)
	}
	if !existe {
		return fmt.Errorf("combo con id %d no encontrado o inactivo", comboID)
	}

	return nil
}

func validarComboServicioInput(tx *sqlx.Tx, servicioID *int, servicioTexto string, sesiones int) error {
	if sesiones < 1 {
		return fmt.Errorf("sesiones debe ser un numero entero positivo")
	}
	if servicioID == nil && strings.TrimSpace(servicioTexto) == "" {
		return fmt.Errorf("debe enviarse servicio_id o servicio_texto")
	}
	if servicioID != nil {
		return validarServicioActivo(tx, *servicioID)
	}

	return nil
}

func validarServicioActivo(tx *sqlx.Tx, servicioID int) error {
	if servicioID < 1 {
		return fmt.Errorf("servicio_id debe ser un numero entero positivo")
	}

	var existe bool
	err := tx.QueryRowx(`
		SELECT EXISTS(SELECT 1 FROM servicios WHERE id = $1 AND activo = TRUE)
	`, servicioID).Scan(&existe)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error al validar servicio: %w", err)
	}
	if !existe {
		return fmt.Errorf("servicio con id %d no encontrado o inactivo", servicioID)
	}

	return nil
}
