package pgsql

import (
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type ServiciosRepo struct {
	db *sqlx.DB
}

func NewServiciosRepo(db *sqlx.DB) *ServiciosRepo {
	return &ServiciosRepo{db: db}
}

// GetAllServicios (activos)
func (r *ServiciosRepo) GetAllServicios() []models.ServicioItem {
	query := `
		SELECT
			s.id,
			s.nombre,
			COALESCE(c.nombre, '')      AS categoria,
			COALESCE(s.tiempo, '')      AS tiempo,
			COALESCE(s.costo::text, '') AS costo,
			s.sesiones,
			COALESCE(l.nombre, '')      AS local,
			COALESCE(s.tipo_espacio_requerido, '') AS tipoEspacios,
			s.requiere_evaluacion
		FROM servicios s
		LEFT JOIN categorias c      ON c.id = s.categoria_id
		LEFT JOIN servicio_local sl ON sl.servicio_id = s.id
		LEFT JOIN locales l         ON l.id = sl.local_id
		WHERE s.activo = TRUE
		ORDER BY c.nombre, s.nombre, l.nombre
	`
	rows, err := r.db.Queryx(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var resultado []models.ServicioItem
	for rows.Next() {
		var item models.ServicioItem
		if err := rows.Scan(
			&item.Id,
			&item.Nombre,
			&item.Categoria,
			&item.Tiempo,
			&item.Costo,
			&item.Sesiones,
			&item.Local,
			&item.TipoEspacio,
			&item.RequiereEvaluacion,
		); err != nil {
			log.Println("SCAN ERROR:", err)
			continue
		}
		resultado = append(resultado, item)
		//log.Println("Obtenidas:", len(resultado))
	}
	return resultado
}

func (r *ServiciosRepo) GetServicioByID(id int) (*models.ServicioItem, error) {
	query := `
		SELECT
			s.id,
			s.nombre,
			COALESCE(c.nombre, '')      AS categoria,
			COALESCE(s.tiempo, '')      AS tiempo,
			COALESCE(s.costo::text, '') AS costo,
			s.sesiones,
			COALESCE(l.nombre, '')      AS local,
			COALESCE(s.tipo_espacio_requerido, '') AS tipoEspacios,
			s.requiere_evaluacion
		FROM servicios s
		LEFT JOIN categorias c      ON c.id = s.categoria_id
		LEFT JOIN servicio_local sl ON sl.servicio_id = s.id
		LEFT JOIN locales l         ON l.id = sl.local_id
		WHERE s.id = $1
		LIMIT 1
	`

	var item models.ServicioItem

	err := r.db.QueryRowx(query, id).Scan(
		&item.Id,
		&item.Nombre,
		&item.Categoria,
		&item.Tiempo,
		&item.Costo,
		&item.Sesiones,
		&item.Local,
		&item.TipoEspacio,
		&item.RequiereEvaluacion,
	)

	if err != nil {
		return nil, fmt.Errorf("servicio con id %d no encontrado: %v", id, err)
	}

	return &item, nil
}

func (r *ServiciosRepo) GetServicioByNombre(nombre string) (*models.ServicioItem, error) {

	query := `
		SELECT
			s.nombre,
			COALESCE(s.costo::text, '') AS costo,
			COALESCE(s.tipo_espacio_requerido, '') AS tipoEspacio,
			s.requiere_evaluacion
		FROM servicios s
		WHERE LOWER(translate(s.nombre, 'áéíóúÁÉÍÓÚ', 'aeiouAEIOU')) =
			LOWER(translate($1, 'áéíóúÁÉÍÓÚ', 'aeiouAEIOU'))
		AND s.activo = TRUE
		LIMIT 1
	`

	var item models.ServicioItem

	err := r.db.QueryRowx(query, strings.TrimSpace(nombre)).Scan(
		&item.Nombre,
		&item.Costo,
		&item.TipoEspacio,
		&item.RequiereEvaluacion,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"servicio '%s' no encontrado o inactivo",
			nombre,
		)
	}

	return &item, nil
}

type CreateServicioInput struct {
	Nombre      string
	Categoria   string // find
	LocalNombre string // find
	Tiempo      string
	Costo       float64
	Sesiones    int
}

func (r *ServiciosRepo) CreateServicio(input repository.CrearServicioInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var catID int
	err = tx.QueryRowx(
		`SELECT id FROM categorias WHERE UPPER(nombre) = UPPER($1)`,
		strings.TrimSpace(input.CategoriaNombre),
	).Scan(&catID)
	if err != nil {
		return 0, fmt.Errorf("categoría '%s' no encontrada", input.CategoriaNombre)
	}

	if strings.TrimSpace(input.LocalNombre) != "" {
		if err := validarCategoriaDisponibleEnLocal(tx, catID, input.LocalNombre); err != nil {
			return 0, err
		}
	}

	// NOTA: Temporal hasta que existan mas espacios
	if input.TipoEspacioRequerido != nil {
		t := strings.ToUpper(*input.TipoEspacioRequerido)
		if t != "M" && t != "B" {
			return 0, fmt.Errorf("tipo_espacio_requerido inválido, valores permitidos: M, B")
		}
	}

	var servicioID int
	err = tx.QueryRowx(`
		INSERT INTO servicios (nombre, categoria_id, tiempo, costo, sesiones, tipo_espacio_requerido, requiere_evaluacion)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		strings.TrimSpace(input.Nombre),
		catID,
		nullStr(input.Tiempo),
		input.Costo,
		input.Sesiones,
		input.TipoEspacioRequerido,
		input.RequiereEvaluacion,
	).Scan(&servicioID)
	if err != nil {
		return 0, fmt.Errorf("error al crear servicio: %w", err)
	}

	if strings.TrimSpace(input.LocalNombre) != "" {
		if err := activarEnLocal(tx, servicioID, input.LocalNombre, input.TipoEspacioRequerido, catID); err != nil {
			return 0, err
		}
	}

	return servicioID, tx.Commit()
}

func (r *ServiciosRepo) UpdateServicio(input repository.ActualizarServicioInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	sets := []string{}
	args := []interface{}{}
	idx := 1

	if input.Nombre != nil {
		sets = append(sets, fmt.Sprintf("nombre = $%d", idx))
		args = append(args, strings.TrimSpace(*input.Nombre))
		idx++
	}
	if input.CategoriaNombre != nil {
		var catID int
		err := tx.QueryRowx(
			`SELECT id FROM categorias WHERE UPPER(nombre) = UPPER($1)`,
			strings.TrimSpace(*input.CategoriaNombre),
		).Scan(&catID)
		if err != nil {
			return fmt.Errorf("categoría '%s' no encontrada", *input.CategoriaNombre)
		}
		if err := validarCategoriaParaLocalesDelServicio(tx, input.ID, catID); err != nil {
			return err
		}
		sets = append(sets, fmt.Sprintf("categoria_id = $%d", idx))
		args = append(args, catID)
		idx++
	}
	if input.Tiempo != nil {
		sets = append(sets, fmt.Sprintf("tiempo = $%d", idx))
		args = append(args, *input.Tiempo)
		idx++
	}
	if input.Costo != nil {
		sets = append(sets, fmt.Sprintf("costo = $%d", idx))
		args = append(args, *input.Costo)
		idx++
	}
	if input.Sesiones != nil {
		sets = append(sets, fmt.Sprintf("sesiones = $%d", idx))
		args = append(args, *input.Sesiones)
		idx++
	}
	if input.TipoEspacioRequerido != nil {
		t := strings.ToUpper(*input.TipoEspacioRequerido)
		if t != "M" && t != "B" {
			return fmt.Errorf("tipo_espacio_requerido inválido, valores permitidos: M, B")
		}
		sets = append(sets, fmt.Sprintf("tipo_espacio_requerido = $%d", idx))
		args = append(args, t)
		idx++
	}
	if input.RequiereEvaluacion != nil {
		sets = append(sets, fmt.Sprintf("requiere_evaluacion = $%d", idx))
		args = append(args, *input.RequiereEvaluacion)
		idx++
	}
	if input.Activo != nil {
		sets = append(sets, fmt.Sprintf("activo = $%d", idx))
		args = append(args, *input.Activo)
		idx++
	}

	if len(sets) == 0 {
		return fmt.Errorf("debe especificarse al menos un campo a modificar")
	}

	args = append(args, input.ID)
	query := fmt.Sprintf(
		"UPDATE servicios SET %s WHERE id = $%d",
		strings.Join(sets, ", "), idx,
	)

	res, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error al actualizar servicio: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("servicio con id %d no encontrado", input.ID)
	}

	return tx.Commit()
}

func (r *ServiciosRepo) AddServicioInLocal(servicioID int, localNombre string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var tipoEspacio *string
	var categoriaID *int
	err = tx.QueryRowx(
		`SELECT tipo_espacio_requerido, categoria_id FROM servicios WHERE id = $1 AND activo = TRUE`,
		servicioID,
	).Scan(&tipoEspacio, &categoriaID)
	if err != nil {
		return fmt.Errorf("servicio con id %d no encontrado o inactivo", servicioID)
	}

	if categoriaID != nil {
		if err := validarCategoriaDisponibleEnLocal(tx, *categoriaID, localNombre); err != nil {
			return err
		}
	}

	if err := activarEnLocal(tx, servicioID, localNombre, tipoEspacio, categoriaIDValue(categoriaID)); err != nil {
		return err
	}

	return tx.Commit()
}

// ── Helpers compartidos ───────────────────────────────────────────────────────

// NOTA: Evaluar si se queda
func findOrCreateCategoria(tx *sqlx.Tx, nombre string) (int, error) {
	if strings.TrimSpace(nombre) == "" {
		return 0, fmt.Errorf("nombre de categoría vacío")
	}
	var id int
	err := tx.QueryRowx(`
		INSERT INTO categorias (nombre)
		VALUES ($1)
		ON CONFLICT (nombre) DO UPDATE SET nombre = EXCLUDED.nombre
		RETURNING id
	`, strings.ToUpper(strings.TrimSpace(nombre))).Scan(&id)
	return id, err
}

func findLocal(tx *sqlx.Tx, nombre string) (int, error) {
	var id int
	err := tx.QueryRowx(
		`SELECT id FROM locales WHERE UPPER(nombre) = UPPER($1)`, nombre,
	).Scan(&id)
	return id, err
}

func activarEnLocal(tx *sqlx.Tx, servicioID int, localNombre string, tipoEspacioRequerido *string, categoriaID int) error {
	// resolver local
	var localID int
	err := tx.QueryRowx(
		`SELECT id FROM locales WHERE UPPER(nombre) = UPPER($1) AND activo = TRUE`,
		strings.TrimSpace(localNombre),
	).Scan(&localID)
	if err != nil {
		return fmt.Errorf("local '%s' no encontrado o inactivo", localNombre)
	}

	if categoriaID > 0 {
		if err := validarCategoriaDisponibleEnLocalID(tx, categoriaID, localID, localNombre); err != nil {
			return err
		}
	}

	// validar que el local tenga el tipo de espacio requerido
	if tipoEspacioRequerido != nil {
		var existe bool
		err = tx.QueryRowx(`
			SELECT EXISTS(
				SELECT 1 FROM tipos_espacio_locales
				WHERE local_id = $1 AND tipo_espacio = $2
			)
		`, localID, strings.ToUpper(*tipoEspacioRequerido)).Scan(&existe)
		if err != nil || !existe {
			return fmt.Errorf(
				"el local '%s' no tiene espacios de tipo '%s' requeridos por este servicio",
				localNombre, *tipoEspacioRequerido,
			)
		}
	}

	// crear relación
	_, err = tx.Exec(`
		INSERT INTO servicio_local (servicio_id, local_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, servicioID, localID)
	if err != nil {
		return fmt.Errorf("error al activar servicio en local: %w", err)
	}

	return nil
}

func categoriaIDValue(categoriaID *int) int {
	if categoriaID == nil {
		return 0
	}
	return *categoriaID
}

func validarCategoriaDisponibleEnLocal(tx *sqlx.Tx, categoriaID int, localNombre string) error {
	var localID int
	err := tx.QueryRowx(
		`SELECT id FROM locales WHERE UPPER(nombre) = UPPER($1) AND activo = TRUE`,
		strings.TrimSpace(localNombre),
	).Scan(&localID)
	if err != nil {
		return fmt.Errorf("local '%s' no encontrado o inactivo", localNombre)
	}

	return validarCategoriaDisponibleEnLocalID(tx, categoriaID, localID, localNombre)
}

func validarCategoriaDisponibleEnLocalID(tx *sqlx.Tx, categoriaID, localID int, localNombre string) error {
	var existe bool
	err := tx.QueryRowx(`
		SELECT EXISTS(
			SELECT 1
			FROM categorias_locales
			WHERE categoria_id = $1
			  AND local_id = $2
		)
	`, categoriaID, localID).Scan(&existe)
	if err != nil {
		return fmt.Errorf("error al validar categoria en local: %w", err)
	}
	if !existe {
		return fmt.Errorf("categoria no disponible para el local '%s'", localNombre)
	}

	return nil
}

func validarCategoriaParaLocalesDelServicio(tx *sqlx.Tx, servicioID, categoriaID int) error {
	type localServicio struct {
		ID     int    `db:"id"`
		Nombre string `db:"nombre"`
	}

	var locales []localServicio
	err := tx.Select(&locales, `
		SELECT l.id, l.nombre
		FROM servicio_local sl
		JOIN locales l ON l.id = sl.local_id
		WHERE sl.servicio_id = $1
		  AND l.activo = TRUE
	`, servicioID)
	if err != nil {
		return fmt.Errorf("error al validar locales del servicio: %w", err)
	}

	for _, local := range locales {
		if err := validarCategoriaDisponibleEnLocalID(tx, categoriaID, local.ID, local.Nombre); err != nil {
			return err
		}
	}

	return nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullFloat(f float64) interface{} {
	if f == 0 {
		return nil
	}
	return f
}

func (r *ServiciosRepo) DeleteServicio(id int) error {
	res, err := r.db.Exec(
		`UPDATE servicios SET activo = FALSE WHERE id = $1 AND activo = TRUE`,
		id,
	)
	if err != nil {
		return fmt.Errorf("error al eliminar servicio: %w", err)
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("servicio con id %d no encontrado o inactivo", id)
	}

	return nil
}
