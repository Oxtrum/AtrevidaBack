package importacion

import "github.com/jmoiron/sqlx"

type staging struct {
	db *sqlx.DB
}

func newStaging(db *sqlx.DB) *staging {
	return &staging{db: db}
}

func (s *staging) Drop() error {
	queries := []string{
		`DROP TABLE IF EXISTS combo_servicios_temp`,
		`DROP TABLE IF EXISTS combo_local_temp`,
		`DROP TABLE IF EXISTS combos_temp`,
		`DROP TABLE IF EXISTS servicio_local_temp`,
		`DROP TABLE IF EXISTS servicios_temp`,
		`DROP TABLE IF EXISTS categorias_temp`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *staging) Create() error {
	queries := []string{
		`CREATE TABLE categorias_temp (
			nombre VARCHAR(100) NOT NULL,
			UNIQUE(nombre)
		)`,

		// unique compuesto: mismo nombre con distinto tiempo/costo = registro diferente
		`CREATE TABLE servicios_temp (
			nombre           VARCHAR(200) NOT NULL,
			categoria_nombre VARCHAR(100),
			tiempo           VARCHAR(50),
			costo            NUMERIC(10,2),
			sesiones         INT NOT NULL DEFAULT 1,
			UNIQUE(nombre, tiempo, costo)
		);
		CREATE INDEX ix_servicios_temp_nombre ON servicios_temp(nombre)`,

		`CREATE TABLE servicio_local_temp (
			servicio_nombre VARCHAR(200) NOT NULL,
			servicio_tiempo VARCHAR(50),
			servicio_costo  NUMERIC(10,2),
			local_nombre    VARCHAR(100) NOT NULL,
			UNIQUE(servicio_nombre, servicio_tiempo, servicio_costo, local_nombre)
		);
		CREATE INDEX ix_servicio_local_temp ON servicio_local_temp(servicio_nombre)`,

		// unique compuesto para combos: nombre + tiempo + costo + sesiones
		`CREATE TABLE combos_temp (
			nombre           VARCHAR(200) NOT NULL,
			categoria_nombre VARCHAR(100),
			costo_total      NUMERIC(10,2),
			sesiones_totales INT NOT NULL DEFAULT 1,
			UNIQUE(nombre, costo_total, sesiones_totales)
		);
		CREATE INDEX ix_combos_temp_nombre ON combos_temp(nombre)`,

		`CREATE TABLE combo_local_temp (
			combo_nombre VARCHAR(200) NOT NULL,
			local_nombre VARCHAR(100) NOT NULL,
			UNIQUE(combo_nombre, local_nombre)
		);
		CREATE INDEX ix_combo_local_temp ON combo_local_temp(combo_nombre)`,

		`CREATE TABLE combo_servicios_temp (
			combo_nombre    VARCHAR(200) NOT NULL,
			servicio_nombre VARCHAR(200) NOT NULL,
			servicio_texto  VARCHAR(500),
			tiempo          VARCHAR(50),
			costo           NUMERIC(10,2),
			sesiones        INT NOT NULL DEFAULT 1,
			orden           INT NOT NULL DEFAULT 0,
			UNIQUE(combo_nombre, servicio_nombre, tiempo, costo)
		);
		CREATE INDEX ix_combo_servicios_temp ON combo_servicios_temp(combo_nombre)`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *staging) Truncate() error {
	queries := []string{
		`TRUNCATE TABLE combo_servicios_temp`,
		`TRUNCATE TABLE combo_local_temp`,
		`TRUNCATE TABLE combos_temp`,
		`TRUNCATE TABLE servicio_local_temp`,
		`TRUNCATE TABLE servicios_temp`,
		`TRUNCATE TABLE categorias_temp`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
