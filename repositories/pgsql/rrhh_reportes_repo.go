package pgsql

import (
	"fmt"
	"time"
)

// AtencionReporteFila es una atencion activa aplanada para el reporte de personal de RRHH.
type AtencionReporteFila struct {
	Fecha       time.Time `db:"fecha_atencion"`
	LocalID     *int      `db:"local_id"`
	LocalNombre string    `db:"local_nombre"`
	EmpleadoID  *int      `db:"empleado_id"`
	// Nombre actual del empleado si esta vinculado; si no, el nombre escrito al registrar.
	Nombre string  `db:"nombre"`
	Cargo  *string `db:"cargo"`
	// Minutos entre hora_inicio y hora_fin; nil si falta alguna.
	Minutos *float64 `db:"minutos"`
	// Identidad del cliente: su id, o su telefono, o la reserva si no hay otro dato.
	Cliente       string `db:"cliente"`
	Servicio      string `db:"servicio"`
	RegistradoPor string `db:"registrado_por"`
}

// AtencionesParaReporte devuelve las atenciones activas con fecha entre desde y hasta (incluidos).
func (r *RRHHRepo) AtencionesParaReporte(desde, hasta time.Time) ([]AtencionReporteFila, error) {
	out := []AtencionReporteFila{}
	err := r.db.Select(&out, `
		SELECT a.fecha_atencion, a.local_id,
		       COALESCE(l.nombre, NULLIF(BTRIM(a.lugar), ''), 'Sin local') AS local_nombre,
		       a.empleado_id,
		       COALESCE(NULLIF(BTRIM(CONCAT_WS(' ', e.nombre, e.apellido)), ''), BTRIM(a.empleado_nombre)) AS nombre,
		       e.cargo,
		       CASE WHEN a.hora_inicio IS NOT NULL AND a.hora_fin IS NOT NULL
		            THEN (EXTRACT(EPOCH FROM (a.hora_fin - a.hora_inicio)) / 60)::float8 END AS minutos,
		       COALESCE('c' || r.cliente_id::text, 't' || NULLIF(BTRIM(r.numero_telefono), ''), 'r' || r.id::text) AS cliente,
		       COALESCE(NULLIF(BTRIM(r.servicio_confirmado), ''), NULLIF(BTRIM(r.servicio_solicitado), ''), NULLIF(BTRIM(r.servicio_nombre), ''), '') AS servicio,
		       COALESCE(NULLIF(BTRIM(a.registrado_por), ''), 'Sin usuario') AS registrado_por
		FROM atenciones a
		JOIN reservas r ON r.id = a.reserva_id
		LEFT JOIN empleados e ON e.id = a.empleado_id
		LEFT JOIN locales l ON l.id = a.local_id
		WHERE a.activo = TRUE AND a.fecha_atencion BETWEEN $1 AND $2
		ORDER BY a.fecha_atencion, a.id`, desde, hasta)
	if err != nil {
		return nil, fmt.Errorf("error al leer atenciones para el reporte: %w", err)
	}
	return out, nil
}
