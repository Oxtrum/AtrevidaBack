package importacion

import (
	"database/sql"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
)

const batchSize = 100

// reCosto extrae el número al inicio de strings como "70 Bs", "210  Bs", "150.5 Bs"
var reCosto = regexp.MustCompile(`^(\d+(?:\.\d+)?)`)

type loader struct {
	db *sqlx.DB
}

func newLoader(db *sqlx.DB) *loader {
	return &loader{db: db}
}

// CargarServicios vuelca todos los ServicioItem a las tablas temporales.
func (l *loader) CargarServicios(items []models.ServicioItem) error {
	if len(items) == 0 {
		return nil
	}

	// categorias_temp
	categoriasVistas := map[string]bool{}
	var categorias []string
	for _, item := range items {
		cat := strings.ToUpper(strings.TrimSpace(item.Categoria))
		if cat != "" && !categoriasVistas[cat] {
			categoriasVistas[cat] = true
			categorias = append(categorias, cat)
		}
	}
	for i := 0; i < len(categorias); i += batchSize {
		fin := min(i+batchSize, len(categorias))
		if _, err := l.db.NamedExec(
			`INSERT INTO categorias_temp (nombre) VALUES (:nombre) ON CONFLICT DO NOTHING`,
			nombresAMaps(categorias[i:fin]),
		); err != nil {
			return err
		}
	}

	// servicios_temp
	type servicioRow struct {
		Nombre          string         `db:"nombre"`
		CategoriaNombre string         `db:"categoria_nombre"`
		Tiempo          string         `db:"tiempo"`
		Costo           sql.NullString `db:"costo"`
		Sesiones        int            `db:"sesiones"`
	}

	var servicioRows []servicioRow
	for _, item := range items {
		servicioRows = append(servicioRows, servicioRow{
			Nombre:          strings.TrimSpace(item.Nombre),
			CategoriaNombre: strings.ToUpper(strings.TrimSpace(item.Categoria)),
			Tiempo:          strings.TrimSpace(item.Tiempo),
			Costo:           extraerCosto(item.Costo),
			Sesiones:        item.Sesiones,
		})
	}

	for i := 0; i < len(servicioRows); i += batchSize {
		fin := min(i+batchSize, len(servicioRows))
		if _, err := l.db.NamedExec(`
			INSERT INTO servicios_temp (nombre, categoria_nombre, tiempo, costo, sesiones)
			VALUES (:nombre, :categoria_nombre, :tiempo, :costo, :sesiones)
			ON CONFLICT (nombre, tiempo, costo) DO NOTHING
		`, servicioRows[i:fin]); err != nil {
			return err
		}
	}

	// servicio_local_temp
	type relacionRow struct {
		ServicioNombre string         `db:"servicio_nombre"`
		ServicioTiempo string         `db:"servicio_tiempo"`
		ServicioCosto  sql.NullString `db:"servicio_costo"`
		LocalNombre    string         `db:"local_nombre"`
	}

	var relaciones []relacionRow
	for _, item := range items {
		costo := extraerCosto(item.Costo)
		for _, local := range expandirLocales(item.Local) {
			relaciones = append(relaciones, relacionRow{
				ServicioNombre: strings.TrimSpace(item.Nombre),
				ServicioTiempo: strings.TrimSpace(item.Tiempo),
				ServicioCosto:  costo,
				LocalNombre:    local,
			})
		}
	}

	for i := 0; i < len(relaciones); i += batchSize {
		fin := min(i+batchSize, len(relaciones))
		if _, err := l.db.NamedExec(`
			INSERT INTO servicio_local_temp (servicio_nombre, servicio_tiempo, servicio_costo, local_nombre)
			VALUES (:servicio_nombre, :servicio_tiempo, :servicio_costo, :local_nombre)
			ON CONFLICT DO NOTHING
		`, relaciones[i:fin]); err != nil {
			return err
		}
	}

	return nil
}

// CargarCombos vuelca todos los ComboItem a las tablas temporales.
func (l *loader) CargarCombos(items []models.ComboItem) error {
	if len(items) == 0 {
		return nil
	}

	// categorias_temp
	categoriasVistas := map[string]bool{}
	var categorias []string
	for _, item := range items {
		cat := strings.ToUpper(strings.TrimSpace(item.Categoria))
		if cat != "" && !categoriasVistas[cat] {
			categoriasVistas[cat] = true
			categorias = append(categorias, cat)
		}
	}
	if len(categorias) > 0 {
		if _, err := l.db.NamedExec(
			`INSERT INTO categorias_temp (nombre) VALUES (:nombre) ON CONFLICT DO NOTHING`,
			nombresAMaps(categorias),
		); err != nil {
			return err
		}
	}

	// combos_temp
	type comboRow struct {
		Nombre          string         `db:"nombre"`
		CategoriaNombre string         `db:"categoria_nombre"`
		CostoTotal      sql.NullString `db:"costo_total"`
		SesionesTotales int            `db:"sesiones_totales"`
	}

	var comboRows []comboRow
	for _, item := range items {
		comboRows = append(comboRows, comboRow{
			Nombre:          strings.TrimSpace(item.Nombre),
			CategoriaNombre: strings.ToUpper(strings.TrimSpace(item.Categoria)),
			CostoTotal:      extraerCosto(item.CostoTotal),
			SesionesTotales: item.SesionesTotales,
		})
	}

	for i := 0; i < len(comboRows); i += batchSize {
		fin := min(i+batchSize, len(comboRows))
		if _, err := l.db.NamedExec(`
			INSERT INTO combos_temp (nombre, categoria_nombre, costo_total, sesiones_totales)
			VALUES (:nombre, :categoria_nombre, :costo_total, :sesiones_totales)
			ON CONFLICT (nombre, costo_total, sesiones_totales) DO NOTHING
		`, comboRows[i:fin]); err != nil {
			return err
		}
	}

	// combo_local_temp
	type comboLocalRow struct {
		ComboNombre string `db:"combo_nombre"`
		LocalNombre string `db:"local_nombre"`
	}

	var comboLocales []comboLocalRow
	for _, item := range items {
		for _, local := range expandirLocales(item.Local) {
			comboLocales = append(comboLocales, comboLocalRow{
				ComboNombre: strings.TrimSpace(item.Nombre),
				LocalNombre: local,
			})
		}
	}

	for i := 0; i < len(comboLocales); i += batchSize {
		fin := min(i+batchSize, len(comboLocales))
		if _, err := l.db.NamedExec(`
			INSERT INTO combo_local_temp (combo_nombre, local_nombre)
			VALUES (:combo_nombre, :local_nombre)
			ON CONFLICT DO NOTHING
		`, comboLocales[i:fin]); err != nil {
			return err
		}
	}

	// combo_servicios_temp
	type comboServicioRow struct {
		ComboNombre    string         `db:"combo_nombre"`
		ServicioNombre string         `db:"servicio_nombre"`
		Tiempo         string         `db:"tiempo"`
		Costo          sql.NullString `db:"costo"`
		Sesiones       int            `db:"sesiones"`
		Orden          int            `db:"orden"`
	}

	var comboServicios []comboServicioRow
	for _, item := range items {
		for orden, svc := range item.ServiciosIncluidos {
			comboServicios = append(comboServicios, comboServicioRow{
				ComboNombre:    strings.TrimSpace(item.Nombre),
				ServicioNombre: strings.TrimSpace(svc.Nombre),
				Tiempo:         strings.TrimSpace(svc.Tiempo),
				Costo:          extraerCosto(svc.Costo),
				Sesiones:       svc.Sesiones,
				Orden:          orden,
			})
		}
	}

	for i := 0; i < len(comboServicios); i += batchSize {
		fin := min(i+batchSize, len(comboServicios))
		if _, err := l.db.NamedExec(`
			INSERT INTO combo_servicios_temp
				(combo_nombre, servicio_nombre, tiempo, costo, sesiones, orden)
			VALUES
				(:combo_nombre, :servicio_nombre, :tiempo, :costo, :sesiones, :orden)
		`, comboServicios[i:fin]); err != nil {
			return err
		}
	}

	return nil
}

// Helpers de importación

// extraerCosto parsea strings como "70 Bs", "210  Bs", "150.5 Bs" → sql.NullString con el número.
// Devuelve NullString inválido si no hay número extraíble.
func extraerCosto(raw string) sql.NullString {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return sql.NullString{Valid: false}
	}
	match := reCosto.FindString(raw)
	if match == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: match, Valid: true}
}

// expandirLocales parsea "ARANJUEZ + CENTRO" → ["PASEO ARANJUEZ", "SAN MARTIN"]
func expandirLocales(local string) []string {
	local = strings.TrimSpace(local)
	if local == "" {
		return nil
	}

	var partes []string
	if strings.Contains(local, "+") {
		for _, p := range strings.Split(local, "+") {
			partes = append(partes, strings.TrimSpace(p))
		}
	} else {
		partes = []string{local}
	}

	var resultado []string
	for _, p := range partes {
		if mapped := mapearLocal(p); mapped != "" {
			resultado = append(resultado, mapped)
		}
	}
	return resultado
}

// mapearLocal normaliza los nombres del sheet al nombre exacto en la tabla locales.
func mapearLocal(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "CENTRO", "SAN MARTIN":
		return "SAN MARTIN"
	case "ARANJUEZ", "PASEO ARANJUEZ":
		return "PASEO ARANJUEZ"
	}
	return ""
}

func nombresAMaps(nombres []string) []map[string]interface{} {
	resultado := make([]map[string]interface{}, len(nombres))
	for i, n := range nombres {
		resultado[i] = map[string]interface{}{"nombre": n}
	}
	return resultado
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
