package pgsql

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
)

type CombosRepo struct {
	db *sqlx.DB
}

func NewCombosRepo(db *sqlx.DB) *CombosRepo {
	return &CombosRepo{db: db}
}

// GetAllCombos
func (r *CombosRepo) GetAllCombos() []models.ComboItem {
	// Get combos
	type comboRow struct {
		ID              int    `db:"id"`
		Nombre          string `db:"nombre"`
		Categoria       string `db:"categoria"`
		Local           string `db:"local"`
		CostoTotal      string `db:"costo_total"`
		SesionesTotales int    `db:"sesiones_totales"`
	}

	combosQuery := `
		SELECT
			cb.id,
			cb.nombre,
			COALESCE(c.nombre, '')               AS categoria,
			COALESCE(
				STRING_AGG(l.nombre, ' + ' ORDER BY l.nombre), ''
			)                                    AS local,
			COALESCE(cb.costo_total::text, '')   AS costo_total,
			cb.sesiones_totales
		FROM combos cb
		LEFT JOIN categorias c   ON c.id = cb.categoria_id
		LEFT JOIN combo_local cl ON cl.combo_id = cb.id
		LEFT JOIN locales l      ON l.id = cl.local_id
		WHERE cb.activo = TRUE
		GROUP BY cb.id, cb.nombre, c.nombre, cb.costo_total, cb.sesiones_totales
		ORDER BY c.nombre, cb.nombre
	`

	var combosRows []comboRow
	if err := r.db.Select(&combosRows, combosQuery); err != nil {
		return nil
	}

	if len(combosRows) == 0 {
		return []models.ComboItem{}
	}

	// Ger servicios
	type servicioRow struct {
		ComboID  int    `db:"combo_id"`
		Nombre   string `db:"nombre"`
		Tiempo   string `db:"tiempo"`
		Costo    string `db:"costo"`
		Sesiones int    `db:"sesiones"`
	}

	serviciosQuery := `
		SELECT
			cs.combo_id,
			s.nombre,
			COALESCE(cs.tiempo, s.tiempo, '')  AS tiempo,
			COALESCE(cs.costo::text, s.costo::text, '') AS costo,
			cs.sesiones
		FROM combo_servicios cs
		JOIN servicios s ON s.id = cs.servicio_id
		ORDER BY cs.combo_id, cs.orden
	`

	var serviciosRows []servicioRow
	if err := r.db.Select(&serviciosRows, serviciosQuery); err != nil {
		return nil
	}

	// indexar by combo_id
	serviciosPorCombo := map[int][]models.ServicioIncluido{}
	for _, sr := range serviciosRows {
		serviciosPorCombo[sr.ComboID] = append(serviciosPorCombo[sr.ComboID],
			models.ServicioIncluido{
				Nombre:   sr.Nombre,
				Tiempo:   sr.Tiempo,
				Costo:    sr.Costo,
				Sesiones: sr.Sesiones,
			},
		)
	}

	// cons dto
	resultado := make([]models.ComboItem, 0, len(combosRows))
	for _, cr := range combosRows {
		servicios := serviciosPorCombo[cr.ID]
		if servicios == nil {
			servicios = []models.ServicioIncluido{}
		}
		resultado = append(resultado, models.ComboItem{
			Nombre:             cr.Nombre,
			Categoria:          cr.Categoria,
			Local:              cr.Local,
			CostoTotal:         cr.CostoTotal,
			SesionesTotales:    cr.SesionesTotales,
			ServiciosIncluidos: servicios,
		})
	}

	return resultado
}

// Escritura

type ServicioIncluidoInput struct {
	ServicioID int
	Tiempo     string
	Costo      float64
	Sesiones   int
	Orden      int
}

type CreateComboInput struct {
	Nombre      string
	Categoria   string
	LocalNombre string
	CostoTotal  float64
	Servicios   []ServicioIncluidoInput
}

func (r *CombosRepo) CreateCombo(input CreateComboInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	catID, err := findOrCreateCategoria(tx, input.Categoria)
	if err != nil {
		return 0, fmt.Errorf("error al find categoría: %w", err)
	}

	sesionesTotales := 0
	for _, s := range input.Servicios {
		sesionesTotales += s.Sesiones
	}

	var comboID int
	err = tx.QueryRowx(`
		INSERT INTO combos (nombre, categoria_id, costo_total, sesiones_totales)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, strings.TrimSpace(input.Nombre), catID, nullFloat(input.CostoTotal), sesionesTotales).
		Scan(&comboID)
	if err != nil {
		return 0, fmt.Errorf("error al insertar combo: %w", err)
	}

	if input.LocalNombre != "" {
		localID, err := findLocal(tx, input.LocalNombre)
		if err != nil {
			return 0, fmt.Errorf("local '%s' no encontrado: %w", input.LocalNombre, err)
		}
		_, err = tx.Exec(`
			INSERT INTO combo_local (combo_id, local_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, comboID, localID)
		if err != nil {
			return 0, err
		}
	}

	for _, s := range input.Servicios {
		_, err = tx.Exec(`
			INSERT INTO combo_servicios (combo_id, servicio_id, tiempo, costo, sesiones, orden)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, comboID, s.ServicioID, nullStr(s.Tiempo), nullFloat(s.Costo), s.Sesiones, s.Orden)
		if err != nil {
			return 0, fmt.Errorf("error al insertar servicio del combo: %w", err)
		}
	}

	return comboID, tx.Commit()
}
