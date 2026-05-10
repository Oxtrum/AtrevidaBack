package importacion

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type migracion struct {
	db *sqlx.DB
}

func newMigracion(db *sqlx.DB) *migracion {
	return &migracion{db: db}
}

type statsImport struct {
	Categorias      int
	Servicios       int
	ServicioLocales int
	Combos          int
	ComboLocales    int
	ComboServicios  int
}

// Ejecutar trunca las tablas no parametricas y migra desde las temporales.
func (m *migracion) Ejecutar() (statsImport, error) {
	var stats statsImport

	tx, err := m.db.Beginx()
	if err != nil {
		return stats, fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback()

	// Limpieza previa
	truncates := []string{
		`TRUNCATE TABLE combo_servicios  RESTART IDENTITY CASCADE`,
		`TRUNCATE TABLE combo_local      RESTART IDENTITY CASCADE`,
		`TRUNCATE TABLE combos           RESTART IDENTITY CASCADE`,
		`TRUNCATE TABLE servicio_local   RESTART IDENTITY CASCADE`,
		`TRUNCATE TABLE servicios        RESTART IDENTITY CASCADE`,
		`TRUNCATE TABLE categorias       RESTART IDENTITY CASCADE`,
	}
	for _, q := range truncates {
		if _, err := tx.Exec(q); err != nil {
			return stats, fmt.Errorf("error al truncar: %w", err)
		}
	}

	// Migrar categorías
	res, err := tx.Exec(`
		INSERT INTO categorias (nombre)
		SELECT DISTINCT nombre FROM categorias_temp
	`)
	if err != nil {
		return stats, fmt.Errorf("error al migrar categorías: %w", err)
	}
	n, _ := res.RowsAffected()
	stats.Categorias = int(n)

	// Migrar servicios
	// costo ya viene como NUMERIC desde servicios_temp
	res, err = tx.Exec(`
		INSERT INTO servicios (nombre, categoria_id, tiempo, costo, sesiones, tipo_espacio_requerido)
		SELECT
			st.nombre,
			c.id,
			NULLIF(st.tiempo, ''),
			st.costo,
			st.sesiones,
			CASE
				WHEN LOWER(st.nombre) LIKE '%bike%'
					OR LOWER(st.nombre) LIKE '%bici%'
				THEN 'B'
				ELSE 'M'
			END AS tipo_espacio_requerido
		FROM servicios_temp st
		LEFT JOIN categorias c ON UPPER(c.nombre) = UPPER(st.categoria_nombre)
	`)
	if err != nil {
		return stats, fmt.Errorf("error al migrar servicios: %w", err)
	}
	n, _ = res.RowsAffected()
	stats.Servicios = int(n)

	// Migrar servicio_local
	// join por nombre + tiempo + costo para identificar el registro exacto
	res, err = tx.Exec(`
		INSERT INTO servicio_local (servicio_id, local_id)
		SELECT DISTINCT s.id, l.id
		FROM servicio_local_temp slt
		JOIN servicios s ON
			LOWER(s.nombre) = LOWER(slt.servicio_nombre)
			AND COALESCE(s.tiempo, '') = COALESCE(slt.servicio_tiempo, '')
			AND (s.costo IS NOT DISTINCT FROM slt.servicio_costo)
		JOIN locales l ON UPPER(l.nombre) = UPPER(slt.local_nombre)
	`)
	if err != nil {
		return stats, fmt.Errorf("error al migrar servicio_local: %w", err)
	}
	n, _ = res.RowsAffected()
	stats.ServicioLocales = int(n)

	// Migrar combos
	res, err = tx.Exec(`
		INSERT INTO combos (nombre, categoria_id, costo_total, sesiones_totales)
		SELECT
			ct.nombre,
			c.id,
			ct.costo_total,
			ct.sesiones_totales
		FROM combos_temp ct
		LEFT JOIN categorias c ON UPPER(c.nombre) = UPPER(ct.categoria_nombre)
	`)
	if err != nil {
		return stats, fmt.Errorf("error al migrar combos: %w", err)
	}
	n, _ = res.RowsAffected()
	stats.Combos = int(n)

	// Migrar combo_local
	res, err = tx.Exec(`
		INSERT INTO combo_local (combo_id, local_id)
		SELECT DISTINCT cb.id, l.id
		FROM combo_local_temp clt
		JOIN combos  cb ON LOWER(cb.nombre) = LOWER(clt.combo_nombre)
		JOIN locales l  ON UPPER(l.nombre)  = UPPER(clt.local_nombre)
	`)
	if err != nil {
		return stats, fmt.Errorf("error al migrar combo_local: %w", err)
	}
	n, _ = res.RowsAffected()
	stats.ComboLocales = int(n)

	// Migrar combo_servicios
	// join por nombre + tiempo + costo para apuntar al servicio correcto
	/*res, err = tx.Exec(`
		INSERT INTO combo_servicios (combo_id, servicio_id, tiempo, costo, sesiones, orden)
		SELECT
			cb.id,
			s.id,
			NULLIF(cst.tiempo, ''),
			cst.costo,
			cst.sesiones,
			cst.orden
		FROM combo_servicios_temp cst
		JOIN combos    cb ON LOWER(cb.nombre) = LOWER(cst.combo_nombre)
		JOIN servicios s  ON
			LOWER(s.nombre) = LOWER(cst.servicio_nombre)
			AND COALESCE(s.tiempo, '') = COALESCE(cst.tiempo, '')
			AND (s.costo IS NOT DISTINCT FROM cst.costo)
	`)*/
	// Migrar combo_servicios
	res, err = tx.Exec(`
		INSERT INTO combo_servicios (combo_id, servicio_id, servicio_texto, tiempo, costo, sesiones, orden)
		SELECT
			cb.id,
			NULL,           -- servicio_id nullable, sin join por ahora
			cst.servicio_nombre,
			NULLIF(cst.tiempo, ''),
			cst.costo,
			cst.sesiones,
			cst.orden
		FROM combo_servicios_temp cst
		JOIN combos cb ON LOWER(cb.nombre) = LOWER(cst.combo_nombre)
	`)
	if err != nil {
		return stats, fmt.Errorf("error al migrar combo_servicios: %w", err)
	}
	n, _ = res.RowsAffected()
	stats.ComboServicios = int(n)

	return stats, tx.Commit()
}
