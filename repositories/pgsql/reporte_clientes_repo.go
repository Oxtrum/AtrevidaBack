package pgsql

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ReporteClientesRepo lee pagos, reservas y deudas aplanados para el reporte de clientes.
type ReporteClientesRepo struct {
	db *sqlx.DB
}

func NewReporteClientesRepo(db *sqlx.DB) *ReporteClientesRepo {
	return &ReporteClientesRepo{db: db}
}

// PagoClienteFila es un pago PAGADO activo. ClienteID nil = venta sin cliente asignado.
type PagoClienteFila struct {
	Fecha       time.Time `db:"fecha"`
	LocalID     *int      `db:"local_id"`
	LocalNombre string    `db:"local_nombre"`
	ClienteID   *int      `db:"cliente_id"`
	Monto       float64   `db:"monto"`
}

// PagosParaReporte devuelve los pagos PAGADO activos del rango (incluidos ambos dias), con la misma regla de
// fechas que /bd/pagos/resumen y el reporte financiero para que los totales coincidan.
func (r *ReporteClientesRepo) PagosParaReporte(desde, hasta time.Time) ([]PagoClienteFila, error) {
	out := []PagoClienteFila{}
	err := r.db.Select(&out, `
		SELECT (p.fecha_creacion AT TIME ZONE 'UTC')::date AS fecha, p.local_id,
		       COALESCE(l.nombre, p.local_nombre) AS local_nombre, p.cliente_id, p.total_final AS monto
		FROM pagos p
		LEFT JOIN locales l ON l.id = p.local_id
		WHERE p.activo = TRUE AND p.estado = 'PAGADO'
		  AND p.fecha_creacion >= $1 AND p.fecha_creacion < $2
		ORDER BY p.fecha_creacion, p.id`, desde, hasta.AddDate(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("error al leer pagos para el reporte de clientes: %w", err)
	}
	return out, nil
}

// ReservaClienteFila es una reserva activa. ClienteID nil = reserva por vincular.
type ReservaClienteFila struct {
	Fecha       time.Time `db:"fecha"`
	LocalID     *int      `db:"local_id"`
	LocalNombre string    `db:"local_nombre"`
	ClienteID   *int      `db:"cliente_id"`
	Estado      string    `db:"estado"`
	// MotivoCancelacion es el tipo de cancelacion (INASISTENCIA, REPROGRAMACION...); vacio si no aplica.
	MotivoCancelacion string `db:"motivo_cancelacion"`
	Servicio          string `db:"servicio"`
}

// ReservasParaReporte devuelve las reservas activas con fecha entre desde y hasta (incluidos).
func (r *ReporteClientesRepo) ReservasParaReporte(desde, hasta time.Time) ([]ReservaClienteFila, error) {
	out := []ReservaClienteFila{}
	err := r.db.Select(&out, `
		SELECT r.fecha, r.local_id, COALESCE(l.nombre, r.local_nombre) AS local_nombre, r.cliente_id,
		       UPPER(COALESCE(r.estado, '')) AS estado,
		       COALESCE(r.motivo_cancelacion_tipo, '') AS motivo_cancelacion,
		       COALESCE(NULLIF(BTRIM(r.servicio_confirmado), ''), NULLIF(BTRIM(r.servicio_solicitado), ''), NULLIF(BTRIM(r.servicio_nombre), ''), '') AS servicio
		FROM reservas r
		LEFT JOIN locales l ON l.id = r.local_id
		WHERE r.activo = TRUE AND r.fecha BETWEEN $1::date AND $2::date
		ORDER BY r.fecha, r.id`, desde, hasta)
	if err != nil {
		return nil, fmt.Errorf("error al leer reservas para el reporte de clientes: %w", err)
	}
	return out, nil
}

// ClienteBasico identifica a un cliente y la fecha de su primera actividad (primer pago o primera cita completada).
type ClienteBasico struct {
	ID       int    `db:"id"`
	Nombre   string `db:"nombre"`
	Telefono string `db:"telefono"`
	// PrimeraActividad es nil si el cliente nunca pago ni completo una cita.
	PrimeraActividad *time.Time `db:"primera_actividad"`
}

// ClientesConPrimeraActividad devuelve todos los clientes con la fecha de su primer pago o primera cita completada.
func (r *ReporteClientesRepo) ClientesConPrimeraActividad() ([]ClienteBasico, error) {
	out := []ClienteBasico{}
	err := r.db.Select(&out, `
		SELECT c.id, BTRIM(CONCAT_WS(' ', c.nombre, c.apellido)) AS nombre, COALESCE(c.numero_telefono, '') AS telefono,
		       LEAST(
		           (SELECT MIN((p.fecha_creacion AT TIME ZONE 'UTC')::date) FROM pagos p
		            WHERE p.cliente_id = c.id AND p.activo = TRUE AND p.estado = 'PAGADO'),
		           (SELECT MIN(r.fecha) FROM reservas r
		            WHERE r.cliente_id = c.id AND r.activo = TRUE AND UPPER(r.estado) = 'COMPLETADO')
		       ) AS primera_actividad
		FROM clientes c`)
	if err != nil {
		return nil, fmt.Errorf("error al leer clientes para el reporte: %w", err)
	}
	return out, nil
}

// DeudaClienteFila es lo que un cliente debe hoy en un local (LocalID nil = cargos sin local).
type DeudaClienteFila struct {
	ClienteID   int       `db:"cliente_id"`
	LocalID     *int      `db:"local_id"`
	LocalNombre string    `db:"local_nombre"`
	Pendiente   float64   `db:"pendiente"`
	Cargos      int       `db:"cargos"`
	Desde       time.Time `db:"desde"`
}

// SaldoAFavorFila es el dinero de abonos que un cliente todavia no aplico a ningun cargo.
type SaldoAFavorFila struct {
	ClienteID  int     `db:"cliente_id"`
	Disponible float64 `db:"disponible"`
}

// SaldosAFavor devuelve, por cliente, lo que queda sin aplicar de sus abonos activos y pagados.
func (r *ReporteClientesRepo) SaldosAFavor() ([]SaldoAFavorFila, error) {
	out := []SaldoAFavorFila{}
	err := r.db.Select(&out, `
		SELECT x.cliente_id, SUM(x.disponible)::float8 AS disponible
		FROM (
			SELECT p.cliente_id,
			       p.total_final - COALESCE((SELECT SUM(a.monto) FROM pago_aplicaciones a
			                                 WHERE a.pago_id = p.id AND a.anulado = FALSE), 0) AS disponible
			FROM pagos p
			WHERE p.es_abono = TRUE AND p.activo = TRUE AND p.estado = 'PAGADO' AND p.cliente_id IS NOT NULL
		) x
		GROUP BY x.cliente_id
		HAVING SUM(x.disponible) >= 0.01`)
	if err != nil {
		return nil, fmt.Errorf("error al leer saldos a favor para el reporte de clientes: %w", err)
	}
	return out, nil
}

// DeudasVigentes suma, por cliente y local, lo pendiente de los cargos no anulados (monto del cargo menos sus
// cobros vigentes). No descuenta el saldo a favor sin aplicar: eso lo hace el servicio con SaldosAFavor.
func (r *ReporteClientesRepo) DeudasVigentes() ([]DeudaClienteFila, error) {
	out := []DeudaClienteFila{}
	err := r.db.Select(&out, `
		SELECT x.cliente_id, x.local_id, COALESCE(l.nombre, 'Sin local') AS local_nombre,
		       SUM(x.pendiente)::float8 AS pendiente, COUNT(*) AS cargos,
		       MIN((x.fecha_hecho AT TIME ZONE 'UTC')::date) AS desde
		FROM (
			SELECT c.cliente_id, c.local_id, c.fecha_hecho,
			       c.monto - COALESCE((SELECT SUM(a.monto) FROM pago_aplicaciones a
			                           WHERE a.cargo_id = c.id AND a.anulado = FALSE), 0) AS pendiente
			FROM cargos_cliente c
			WHERE c.anulado = FALSE
		) x
		LEFT JOIN locales l ON l.id = x.local_id
		WHERE x.pendiente >= 0.01
		GROUP BY x.cliente_id, x.local_id, l.nombre
		ORDER BY x.cliente_id`)
	if err != nil {
		return nil, fmt.Errorf("error al leer deudas para el reporte de clientes: %w", err)
	}
	return out, nil
}
