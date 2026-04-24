package pgsql

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
)

type ReservasRepo struct {
	db *sqlx.DB
}

func NewReservasRepo(db *sqlx.DB) *ReservasRepo {
	return &ReservasRepo{db: db}
}

// Lectura

type FiltroReservasPG struct {
	LocalID     *int
	Fecha       *time.Time
	FechaDesde  *time.Time
	FechaHasta  *time.Time
	Cliente     string
	TipoEspacio string // 'M' | 'B'
	PlanID      *int
	SoloActivas bool
}

func (r *ReservasRepo) GetReservas(f FiltroReservasPG) ([]models.ReservaPGCompleta, error) {
	conditions := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.LocalID != nil {
		conditions = append(conditions, fmt.Sprintf("r.local_id = $%d", idx))
		args = append(args, *f.LocalID)
		idx++
	}
	if f.Fecha != nil {
		conditions = append(conditions, fmt.Sprintf("r.fecha = $%d", idx))
		args = append(args, *f.Fecha)
		idx++
	}
	if f.FechaDesde != nil {
		conditions = append(conditions, fmt.Sprintf("r.fecha >= $%d", idx))
		args = append(args, *f.FechaDesde)
		idx++
	}
	if f.FechaHasta != nil {
		conditions = append(conditions, fmt.Sprintf("r.fecha <= $%d", idx))
		args = append(args, *f.FechaHasta)
		idx++
	}
	if f.Cliente != "" {
		conditions = append(conditions, fmt.Sprintf("r.cliente ILIKE $%d", idx))
		args = append(args, "%"+f.Cliente+"%")
		idx++
	}
	if f.TipoEspacio != "" {
		conditions = append(conditions, fmt.Sprintf("r.tipo_espacio = $%d", idx))
		args = append(args, strings.ToUpper(f.TipoEspacio))
		idx++
	}
	if f.PlanID != nil {
		conditions = append(conditions, fmt.Sprintf("r.plan_id = $%d", idx))
		args = append(args, *f.PlanID)
		idx++
	}
	if f.SoloActivas {
		conditions = append(conditions, "r.activo = TRUE")
	}

	query := fmt.Sprintf(`
		SELECT
			r.id,
			r.local_id,
			r.local_nombre,
			r.tipo_espacio,
			r.fecha,
			r.hora_desde::text,
			r.hora_hasta::text,
			r.cliente,
			r.plan_id,
			r.servicio_nombre,
			r.servicio_tiempo,
			r.precio,
			r.notas,
			r.activo,
			r.creado_en,
			r.actualizado_en
		FROM reservas r
		WHERE %s
		ORDER BY r.fecha, r.hora_desde
	`, strings.Join(conditions, " AND "))

	rows, err := r.db.Queryx(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error al consultar reservas: %w", err)
	}
	defer rows.Close()

	var reservas []models.ReservaPGCompleta
	for rows.Next() {
		var rv models.ReservaPGCompleta
		if err := rows.StructScan(&rv); err != nil {
			continue
		}

		// get detalle
		rv.Detalle, _ = r.getDetalleReserva(rv.ID)
		reservas = append(reservas, rv)
	}

	return reservas, nil
}

func (r *ReservasRepo) getDetalleReserva(reservaID int) ([]models.DetalleReservaPG, error) {
	var detalle []models.DetalleReservaPG
	err := r.db.Select(&detalle, `
		SELECT id, reserva_id, servicio_nombre, servicio_tiempo, precio, sesiones, notas
		FROM detalle_reservas
		WHERE reserva_id = $1
		ORDER BY id
	`, reservaID)
	return detalle, err
}

// Escritura

type CreateReservaInput struct {
	LocalID        int
	LocalNombre    string
	TipoEspacio    string
	Fecha          time.Time
	HoraDesde      string // "09:00"
	HoraHasta      string // "09:30"
	Cliente        string
	PlanID         *int
	ServicioNombre string
	ServicioTiempo string
	Precio         *float64
	Notas          string
	Detalle        []CrearDetalleInput
}

type CrearDetalleInput struct {
	ServicioNombre string
	ServicioTiempo string
	Precio         *float64
	Sesiones       int
	Notas          string
}

func (r *ReservasRepo) CreateReserva(input CreateReservaInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var reservaID int
	err = tx.QueryRowx(`
		INSERT INTO reservas (
			local_id, local_nombre, tipo_espacio,
			fecha, hora_desde, hora_hasta,
			cliente, plan_id,
			servicio_nombre, servicio_tiempo, precio, notas
		) VALUES (
			$1, $2, $3,
			$4, $5::time, $6::time,
			$7, $8,
			$9, $10, $11, $12
		)
		RETURNING id
	`,
		input.LocalID, input.LocalNombre, strings.ToUpper(input.TipoEspacio),
		input.Fecha, input.HoraDesde, input.HoraHasta,
		input.Cliente, input.PlanID,
		nullStr(input.ServicioNombre), nullStr(input.ServicioTiempo),
		input.Precio, nullStr(input.Notas),
	).Scan(&reservaID)
	if err != nil {
		return 0, fmt.Errorf("error al insertar reserva: %w", err)
	}

	for _, d := range input.Detalle {
		_, err = tx.Exec(`
			INSERT INTO detalle_reservas
				(reserva_id, servicio_nombre, servicio_tiempo, precio, sesiones, notas)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, reservaID,
			d.ServicioNombre, nullStr(d.ServicioTiempo),
			d.Precio, d.Sesiones, nullStr(d.Notas),
		)
		if err != nil {
			return 0, fmt.Errorf("error al insertar detalle: %w", err)
		}
	}

	// Manejo planes (por verse)
	if input.PlanID != nil {
		_, err = tx.Exec(`
			UPDATE planes SET sesiones_usadas = sesiones_usadas + 1
			WHERE id = $1
		`, *input.PlanID)
		if err != nil {
			return 0, fmt.Errorf("error al actualizar plan: %w", err)
		}
	}

	return reservaID, tx.Commit()
}

type updateReservaInput struct {
	ID             int
	TipoEspacio    *string
	Fecha          *time.Time
	HoraDesde      *string
	HoraHasta      *string
	ServicioNombre *string
	ServicioTiempo *string
	Precio         *float64
	Notas          *string
}

func (r *ReservasRepo) updateReserva(input updateReservaInput) error {
	sets := []string{"actualizado_en = NOW()"}
	args := []interface{}{}
	idx := 1

	if input.TipoEspacio != nil {
		sets = append(sets, fmt.Sprintf("tipo_espacio = $%d", idx))
		args = append(args, strings.ToUpper(*input.TipoEspacio))
		idx++
	}
	if input.Fecha != nil {
		sets = append(sets, fmt.Sprintf("fecha = $%d", idx))
		args = append(args, *input.Fecha)
		idx++
	}
	if input.HoraDesde != nil {
		sets = append(sets, fmt.Sprintf("hora_desde = $%d::time", idx))
		args = append(args, *input.HoraDesde)
		idx++
	}
	if input.HoraHasta != nil {
		sets = append(sets, fmt.Sprintf("hora_hasta = $%d::time", idx))
		args = append(args, *input.HoraHasta)
		idx++
	}
	if input.ServicioNombre != nil {
		sets = append(sets, fmt.Sprintf("servicio_nombre = $%d", idx))
		args = append(args, *input.ServicioNombre)
		idx++
	}
	if input.ServicioTiempo != nil {
		sets = append(sets, fmt.Sprintf("servicio_tiempo = $%d", idx))
		args = append(args, *input.ServicioTiempo)
		idx++
	}
	if input.Precio != nil {
		sets = append(sets, fmt.Sprintf("precio = $%d", idx))
		args = append(args, *input.Precio)
		idx++
	}
	if input.Notas != nil {
		sets = append(sets, fmt.Sprintf("notas = $%d", idx))
		args = append(args, *input.Notas)
		idx++
	}

	args = append(args, input.ID)
	query := fmt.Sprintf(
		"UPDATE reservas SET %s WHERE id = $%d",
		strings.Join(sets, ", "), idx,
	)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error al actualizar reserva: %w", err)
	}
	return nil
}

func (r *ReservasRepo) NullifyReserva(id int) error {
	_, err := r.db.Exec(
		"UPDATE reservas SET activo = FALSE, actualizado_en = NOW() WHERE id = $1", id,
	)
	return err
}
