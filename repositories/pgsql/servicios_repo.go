package pgsql

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
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
			COALESCE(l.nombre, '')      AS local
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
			new(int),
			&item.Nombre,
			&item.Categoria,
			&item.Tiempo,
			&item.Costo,
			&item.Sesiones,
			&item.Local,
		); err != nil {
			continue
		}
		resultado = append(resultado, item)
	}

	return resultado
}

// Escritura

type CreateServicioInput struct {
	Nombre      string
	Categoria   string // find-create
	LocalNombre string // find-create
	Tiempo      string
	Costo       float64
	Sesiones    int
}

func (r *ServiciosRepo) CreateServicio(input CreateServicioInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// find-create
	catID, err := findOrCreateCategoria(tx, input.Categoria)
	if err != nil {
		return 0, fmt.Errorf("error al resolver categoría: %w", err)
	}

	// insert servicio
	var servicioID int
	err = tx.QueryRowx(`
		INSERT INTO servicios (nombre, categoria_id, tiempo, costo, sesiones)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, input.Nombre, catID, nullStr(input.Tiempo), nullFloat(input.Costo), input.Sesiones).
		Scan(&servicioID)
	if err != nil {
		return 0, fmt.Errorf("error al insertar servicio: %w", err)
	}

	// relate local
	if input.LocalNombre != "" {
		localID, err := findLocal(tx, input.LocalNombre)
		if err != nil {
			return 0, fmt.Errorf("local '%s' no encontrado: %w", input.LocalNombre, err)
		}
		_, err = tx.Exec(`
			INSERT INTO servicio_local (servicio_id, local_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, servicioID, localID)
		if err != nil {
			return 0, err
		}
	}

	return servicioID, tx.Commit()
}

// ── Helpers compartidos ───────────────────────────────────────────────────────

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
