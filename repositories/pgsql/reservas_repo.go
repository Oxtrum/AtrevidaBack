package pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var _ repository.ReservasPGRepository = (*ReservasRepo)(nil)

type ReservasRepo struct {
	db *sqlx.DB
}

func NewReservasRepo(db *sqlx.DB) *ReservasRepo {
	return &ReservasRepo{db: db}
}

// GET
func (r *ReservasRepo) GetReservas(f repository.FiltroReservasPG) ([]models.ReservaPGCompleta, error) {
	conditions, args := reservaConditions(f)
	idx := len(args) + 1
	if f.CursorSet {
		if f.OrdenCronologico {
			conditions = append(conditions, fmt.Sprintf("(r.fecha, r.hora_desde, r.local_nombre, r.id) > ($%d, $%d, $%d, $%d)", idx, idx+1, idx+2, idx+3))
			args = append(args, f.CursorFecha, f.CursorHora, f.CursorLocal, f.CursorID)
		} else {
			conditions = append(conditions, fmt.Sprintf("(r.local_nombre, r.fecha, r.hora_desde, r.id) > ($%d, $%d, $%d, $%d)", idx, idx+1, idx+2, idx+3))
			args = append(args, f.CursorLocal, f.CursorFecha, f.CursorHora, f.CursorID)
		}
		idx += 4
	}
	limitClause := ""
	if f.PageLimit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, f.PageLimit)
	}

	query := fmt.Sprintf(`
		SELECT
			r.id, r.local_id, r.local_nombre, r.tipo_espacio,
			r.fecha, r.hora_desde::text, r.hora_hasta::text,
			r.cliente, r.estado, r.numero_telefono, r.plan_id, r.servicio_nombre,
			r.servicio_solicitado, r.servicio_confirmado, r.servicio_tiempo,
			r.precio, r.notas, r.activo, COALESCE(r.notificado, FALSE) AS notificado,
			r.creado_en, r.actualizado_en
		FROM reservas r
		WHERE %s
		ORDER BY %s%s
	`, strings.Join(conditions, " AND "), reservaOrderClause(f), limitClause)

	rows, err := r.db.QueryxContext(queryContext(f.Context), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error al consultar reservas: %w", err)
	}
	var reservas []models.ReservaPGCompleta
	for rows.Next() {
		var rv models.ReservaPGCompleta
		if err := rows.StructScan(&rv); err != nil {
			rows.Close()
			return nil, fmt.Errorf("error al leer reserva: %w", err)
		}
		reservas = append(reservas, rv)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("error al recorrer reservas: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("error al cerrar consulta de reservas: %w", err)
	}

	return reservas, nil
}

func reservaOrderClause(f repository.FiltroReservasPG) string {
	if f.OrdenCronologico {
		return "r.fecha, r.hora_desde, r.local_nombre, r.id"
	}
	return "r.local_nombre, r.fecha, r.hora_desde, r.id"
}

func (r *ReservasRepo) CountReservas(f repository.FiltroReservasPG) (int, error) {
	conditions, args := reservaConditions(f)
	query := fmt.Sprintf("SELECT COUNT(*) FROM reservas r WHERE %s", strings.Join(conditions, " AND "))
	var total int
	if err := r.db.GetContext(queryContext(f.Context), &total, query, args...); err != nil {
		return 0, fmt.Errorf("error al contar reservas: %w", err)
	}
	return total, nil
}

func reservaConditions(f repository.FiltroReservasPG) ([]string, []interface{}) {
	conditions := []string{"1=1"}
	args := []interface{}{}
	add := func(condition string, value interface{}) {
		conditions = append(conditions, fmt.Sprintf(condition, len(args)+1))
		args = append(args, value)
	}
	if f.LocalID != nil {
		add("r.local_id = $%d", *f.LocalID)
	}
	if f.LocalNombre != "" {
		add("r.local_id = (SELECT l.id FROM locales l WHERE UPPER(l.nombre) = UPPER($%d) LIMIT 1)", f.LocalNombre)
	}
	if f.Fecha != nil {
		add("r.fecha = $%d", *f.Fecha)
	}
	if f.FechaDesde != nil {
		add("r.fecha >= $%d", *f.FechaDesde)
	}
	if f.FechaHasta != nil {
		add("r.fecha <= $%d", *f.FechaHasta)
	}
	if f.Cliente != "" {
		add("r.cliente ILIKE $%d", "%"+f.Cliente+"%")
	}
	if f.NumeroTelefono != "" {
		digitos := soloDigitosTelefono(f.NumeroTelefono)
		last8 := digitos
		if len(last8) > 8 {
			last8 = last8[len(last8)-8:]
		}
		idx := len(args) + 1
		conditions = append(conditions, fmt.Sprintf("(BTRIM(COALESCE(r.numero_telefono, '')) = BTRIM($%d) OR regexp_replace(COALESCE(r.numero_telefono, ''), '\\D', '', 'g') = $%d OR RIGHT(regexp_replace(COALESCE(r.numero_telefono, ''), '\\D', '', 'g'), 8) = $%d)", idx, idx+1, idx+2))
		args = append(args, f.NumeroTelefono, digitos, last8)
	}
	if f.ServicioSolicitado != "" {
		add("COALESCE(r.servicio_solicitado, '') ILIKE $%d", "%"+f.ServicioSolicitado+"%")
	}
	if f.ServicioConfirmado != "" {
		add("COALESCE(r.servicio_confirmado, '') ILIKE $%d", "%"+f.ServicioConfirmado+"%")
	}
	if f.Busqueda != "" {
		add(`TRANSLATE(LOWER(CONCAT_WS(' ', r.id::text, r.cliente, r.numero_telefono, r.servicio_nombre,
			COALESCE(r.servicio_solicitado, ''), COALESCE(r.servicio_confirmado, ''),
			r.local_nombre, TO_CHAR(r.fecha, 'YYYY-MM-DD'))), 'áéíóúüñ', 'aeiouun')
			LIKE TRANSLATE(LOWER($%d), 'áéíóúüñ', 'aeiouun')`, "%"+f.Busqueda+"%")
	}
	if f.Estado != "" {
		add("r.estado = $%d", f.Estado)
	}
	if f.ExcluirEstado != "" {
		add("r.estado <> $%d", f.ExcluirEstado)
	}
	if f.TipoEspacio != "" {
		add("r.tipo_espacio = $%d", strings.ToUpper(f.TipoEspacio))
	}
	if f.PlanID != nil {
		add("r.plan_id = $%d", *f.PlanID)
	}
	if f.SoloActivas {
		conditions = append(conditions, "r.activo = TRUE")
	}
	if f.VigenteFecha != nil && f.VigenteHora != "" {
		idx := len(args) + 1
		condition := fmt.Sprintf("(r.fecha > $%d OR (r.fecha = $%d AND r.hora_hasta >= $%d))", idx, idx, idx+1)
		if f.VigenciaPendientes {
			condition = "(r.estado <> 'PENDIENTE' OR " + condition + ")"
		}
		conditions = append(conditions, condition)
		args = append(args, *f.VigenteFecha, f.VigenteHora)
	}
	return conditions, args
}

func (r *ReservasRepo) GetReservasAgendadasNoNotificadas(ctx context.Context, localNombre string, limit int) ([]models.ReservaPGCompleta, error) {
	conditions := []string{"r.activo = TRUE", "r.estado = 'AGENDADO'", "COALESCE(r.notificado, FALSE) = FALSE"}
	args := []interface{}{}
	if strings.TrimSpace(localNombre) != "" {
		conditions = append(conditions, `r.local_id = (
			SELECT l.id FROM locales l WHERE UPPER(l.nombre) = UPPER($1) LIMIT 1
		)`)
		args = append(args, strings.TrimSpace(localNombre))
	}
	args = append(args, limit)
	query := fmt.Sprintf(`
		SELECT r.id, r.local_id, r.local_nombre, r.tipo_espacio,
			r.fecha, r.hora_desde::text, r.hora_hasta::text,
			r.cliente, r.estado, r.numero_telefono, r.plan_id, r.servicio_nombre,
			r.servicio_solicitado, r.servicio_confirmado, r.servicio_tiempo,
			r.precio, r.notas, r.activo, COALESCE(r.notificado, FALSE) AS notificado,
			r.creado_en, r.actualizado_en
		FROM reservas r WHERE %s
		ORDER BY r.creado_en DESC, r.id DESC LIMIT $%d
	`, strings.Join(conditions, " AND "), len(args))
	var reservas []models.ReservaPGCompleta
	if err := r.db.SelectContext(queryContext(ctx), &reservas, query, args...); err != nil {
		return nil, fmt.Errorf("error al consultar notificaciones de reservas: %w", err)
	}
	if reservas == nil {
		reservas = []models.ReservaPGCompleta{}
	}
	return reservas, nil
}

func (r *ReservasRepo) GetReservaByID(id int) (*models.ReservaPGCompleta, error) {
	query := `
		SELECT
			r.id, r.local_id, r.local_nombre, r.tipo_espacio,
			r.fecha, r.hora_desde::text, r.hora_hasta::text,
			r.cliente, r.estado, r.numero_telefono, r.plan_id, r.servicio_nombre,
			r.servicio_solicitado, r.servicio_confirmado, r.servicio_tiempo,
			r.precio, r.notas, r.activo, COALESCE(r.notificado, FALSE) AS notificado,
			r.creado_en, r.actualizado_en
		FROM reservas r
		WHERE r.id = $1
	`
	var rv models.ReservaPGCompleta
	err := r.db.Get(&rv, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reserva no encontrada")
		}
		return nil, fmt.Errorf("error al obtener reserva por id: %w", err)
	}

	rv.Detalle, err = r.getDetalleReserva(rv.ID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener detalle de reserva: %w", err)
	}
	return &rv, nil
}

func (r *ReservasRepo) GetLocalIDByNombre(nombre string) (int, error) {
	var localID int
	err := r.db.Get(&localID, `
		SELECT id
		FROM locales
		WHERE UPPER(nombre) = UPPER($1)
		  AND activo = TRUE
	`, strings.TrimSpace(nombre))
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("local no encontrado")
		}
		return 0, fmt.Errorf("error al obtener local: %w", err)
	}

	return localID, nil
}

func (r *ReservasRepo) GetResumenPagosReservas(f repository.FiltroResumenPagosReservas) (repository.ResumenPagosReservas, error) {
	conditions := []string{
		"p.activo = TRUE",
		"p.estado = 'PAGADO'",
		"p.fecha_creacion >= $1::date",
		"p.fecha_creacion < ($2::date + INTERVAL '1 day')",
	}
	args := []interface{}{f.FechaDesde, f.FechaHasta, f.Fecha}
	idx := 4

	if f.LocalID != nil {
		conditions = append(conditions, fmt.Sprintf("p.local_id = $%d", idx))
		args = append(args, *f.LocalID)
		idx++
	}
	if f.LocalNombre != "" {
		conditions = append(conditions, fmt.Sprintf(`p.local_id = (
			SELECT l.id FROM locales l WHERE UPPER(l.nombre) = UPPER($%d) LIMIT 1
		)`, idx))
		args = append(args, f.LocalNombre)
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(p.total_final) FILTER (WHERE p.fecha_creacion::date = $3::date), 0) AS ingresos_dia,
			COALESCE(SUM(p.total_final), 0) AS ingresos_semana,
			COALESCE(SUM(p.total_final) FILTER (WHERE EXTRACT(ISODOW FROM p.fecha_creacion::date) = 1), 0) AS ingresos_lunes,
			COALESCE(SUM(p.total_final) FILTER (WHERE EXTRACT(ISODOW FROM p.fecha_creacion::date) = 2), 0) AS ingresos_martes,
			COALESCE(SUM(p.total_final) FILTER (WHERE EXTRACT(ISODOW FROM p.fecha_creacion::date) = 3), 0) AS ingresos_miercoles,
			COALESCE(SUM(p.total_final) FILTER (WHERE EXTRACT(ISODOW FROM p.fecha_creacion::date) = 4), 0) AS ingresos_jueves,
			COALESCE(SUM(p.total_final) FILTER (WHERE EXTRACT(ISODOW FROM p.fecha_creacion::date) = 5), 0) AS ingresos_viernes,
			COALESCE(SUM(p.total_final) FILTER (WHERE EXTRACT(ISODOW FROM p.fecha_creacion::date) = 6), 0) AS ingresos_sabado,
			COUNT(*) FILTER (WHERE p.fecha_creacion::date = $3::date) AS cancelaciones_dia
		FROM pagos p
		WHERE %s
	`, strings.Join(conditions, " AND "))

	var resumen repository.ResumenPagosReservas
	if err := r.db.GetContext(queryContext(f.Context), &resumen, query, args...); err != nil {
		return resumen, fmt.Errorf("no se pudo obtener el resumen de pagos para reservas")
	}

	return resumen, nil
}

func (r *ReservasRepo) getDetalleReserva(reservaID int) ([]models.DetalleReservaPG, error) {
	var detalle []models.DetalleReservaPG
	err := r.db.Select(&detalle, `
		SELECT id, reserva_id, servicio_nombre, servicio_tiempo, precio, sesiones, notas
		FROM detalle_reservas WHERE reserva_id = $1 ORDER BY id
	`, reservaID)
	return detalle, err
}

// POST
func (r *ReservasRepo) CreateReserva(input repository.CreateReservaInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var localID int
	err = tx.QueryRowx(
		`SELECT id FROM locales WHERE UPPER(nombre) = UPPER($1)`, input.LocalNombre,
	).Scan(&localID)
	if err != nil {
		return 0, fmt.Errorf("local '%s' no encontrado", input.LocalNombre)
	}

	if err := r.validarCapacidad(tx, localID, input.TipoEspacio, input.Fecha, input.HoraDesde, input.HoraHasta, 0); err != nil {
		return 0, err
	}

	var reservaID int
	err = tx.QueryRowx(`
		INSERT INTO reservas (
			local_id, local_nombre, tipo_espacio,
			fecha, hora_desde, hora_hasta,
			cliente, estado, numero_telefono, plan_id, servicio_nombre,
			servicio_solicitado, servicio_confirmado, precio, notas
		) VALUES ($1,$2,$3,$4,$5::time,$6::time,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id
	`,
		localID, input.LocalNombre, strings.ToUpper(input.TipoEspacio),
		input.Fecha, input.HoraDesde, input.HoraHasta,
		input.Cliente, input.Estado, nullStr(input.NumeroTelefono), input.PlanID,
		nullStr(input.ServicioNombre), nullStr(input.ServicioSolicitado),
		input.ServicioConfirmado, input.Precio, nullStr(input.Notas),
	).Scan(&reservaID)
	if err != nil {
		return 0, fmt.Errorf("error al insertar reserva: %w", err)
	}

	for _, d := range input.Detalle {
		_, err = tx.Exec(`
			INSERT INTO detalle_reservas (reserva_id, servicio_nombre, servicio_tiempo, precio, sesiones, notas)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, reservaID, d.ServicioNombre, nullStr(d.ServicioTiempo), d.Precio, d.Sesiones, nullStr(d.Notas))
		if err != nil {
			return 0, fmt.Errorf("error al insertar detalle: %w", err)
		}
	}

	return reservaID, tx.Commit()
}

// PATCH
func (r *ReservasRepo) UpdateReserva(input repository.UpdateReservaInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var reservaID int
	var localID int

	var fechaActual time.Time
	var horaDesdeActual string
	var horaHastaActual string
	var tipoActual string

	err = tx.QueryRowx(`
		SELECT
			r.id,
			r.local_id,
			r.fecha,
			r.hora_desde::text,
			r.hora_hasta::text,
			r.tipo_espacio
		FROM reservas r
		WHERE r.id = $1
		AND r.activo = TRUE
		LIMIT 1
	`, input.Id).Scan(
		&reservaID,
		&localID,
		&fechaActual,
		&horaDesdeActual,
		&horaHastaActual,
		&tipoActual,
	)

	if err != nil {
		return fmt.Errorf("Reserva no encontrada")
	}

	// Un cambio de local mueve la reserva a otro conjunto de espacios: la
	// capacidad debe validarse contra el local destino, no contra el actual.
	localNombreFinal := ""
	if input.NuevoLocal != nil {
		localNombreFinal = strings.TrimSpace(*input.NuevoLocal)
		err = tx.QueryRowx(
			`SELECT id, nombre FROM locales WHERE UPPER(nombre) = UPPER($1)`, localNombreFinal,
		).Scan(&localID, &localNombreFinal)
		if err != nil {
			return fmt.Errorf("local '%s' no encontrado", strings.TrimSpace(*input.NuevoLocal))
		}
	}

	if input.NuevaFecha != nil {

		today := time.Now().Truncate(24 * time.Hour)

		if input.NuevaFecha.Before(today) {
			return fmt.Errorf(
				"No se puede modificar una reserva para una fecha pasada | fecha recibida: %s | fecha actual: %s",
				input.NuevaFecha.Format(time.RFC3339),
				time.Now().Format(time.RFC3339),
			)
		}
	}

	fechaFinal := fechaActual
	if input.NuevaFecha != nil {
		fechaFinal = *input.NuevaFecha
	}

	horaDesdeFinal := horaDesdeActual
	if input.NuevaHoraDesde != nil {
		horaDesdeFinal = *input.NuevaHoraDesde
	}

	horaHastaFinal := horaHastaActual
	if input.NuevaHoraHasta != nil {
		horaHastaFinal = *input.NuevaHoraHasta
	}

	tipoFinal := tipoActual
	if input.NuevoTipo != nil {
		tipoFinal = strings.ToUpper(*input.NuevoTipo)
	}

	err = r.validarCapacidad(
		tx,
		localID,
		tipoFinal,
		fechaFinal,
		horaDesdeFinal,
		horaHastaFinal,
		reservaID,
	)

	if err != nil {
		return err
	}

	sets := []string{"actualizado_en = NOW()"}
	args := []interface{}{}
	idx := 1

	if input.NuevoTipo != nil {
		sets = append(sets, fmt.Sprintf("tipo_espacio = $%d", idx))
		args = append(args, strings.ToUpper(*input.NuevoTipo))
		idx++
	}
	if input.NuevaFecha != nil {
		sets = append(sets, fmt.Sprintf("fecha = $%d", idx))
		args = append(args, *input.NuevaFecha)
		idx++
	}
	if input.NuevaHoraDesde != nil {
		sets = append(sets, fmt.Sprintf("hora_desde = $%d::time", idx))
		args = append(args, *input.NuevaHoraDesde)
		idx++
	}
	if input.NuevaHoraHasta != nil {
		sets = append(sets, fmt.Sprintf("hora_hasta = $%d::time", idx))
		args = append(args, *input.NuevaHoraHasta)
		idx++
	}
	if input.NuevoServicio != nil {
		sets = append(sets, fmt.Sprintf("servicio_nombre = $%d", idx))
		args = append(args, *input.NuevoServicio)
		idx++
	}
	if input.NuevoServicioSolicitado != nil {
		sets = append(sets, fmt.Sprintf("servicio_solicitado = $%d", idx))
		args = append(args, *input.NuevoServicioSolicitado)
		idx++
	}
	if input.NuevoServicioConfirmado != nil {
		sets = append(sets, fmt.Sprintf("servicio_confirmado = $%d", idx))
		args = append(args, *input.NuevoServicioConfirmado)
		idx++
	}
	if input.NuevoNumeroTelefono != nil {
		sets = append(sets, fmt.Sprintf("numero_telefono = $%d", idx))
		args = append(args, *input.NuevoNumeroTelefono)
		idx++
	}
	if input.NuevoPrecio != nil {
		sets = append(sets, fmt.Sprintf("precio = $%d", idx))
		args = append(args, *input.NuevoPrecio)
		idx++
	}
	if input.NuevasNotas != nil {
		sets = append(sets, fmt.Sprintf("notas = $%d", idx))
		args = append(args, *input.NuevasNotas)
		idx++
	}
	if input.NuevoCliente != nil {
		sets = append(sets, fmt.Sprintf("cliente = $%d", idx))
		args = append(args, *input.NuevoCliente)
		idx++
	}
	if input.NuevoLocal != nil {
		sets = append(sets, fmt.Sprintf("local_id = $%d", idx))
		args = append(args, localID)
		idx++
		sets = append(sets, fmt.Sprintf("local_nombre = $%d", idx))
		args = append(args, localNombreFinal)
		idx++
	}
	if input.LimpiarPlanID {
		sets = append(sets, "plan_id = NULL")
	} else if input.NuevoPlanID != nil {
		sets = append(sets, fmt.Sprintf("plan_id = $%d", idx))
		args = append(args, *input.NuevoPlanID)
		idx++
	}
	if input.ResetNotificado {
		sets = append(sets, "notificado = FALSE")
	}

	args = append(args, reservaID)
	_, err = tx.Exec(
		fmt.Sprintf("UPDATE reservas SET %s WHERE id = $%d AND activo = TRUE", strings.Join(sets, ", "), idx),
		args...,
	)
	if err != nil {
		return fmt.Errorf("error al actualizar reserva: %w", err)
	}

	return tx.Commit()
}

func (r *ReservasRepo) AnularReserva(id int) error {
	res, err := r.db.Exec(
		`UPDATE reservas SET activo = FALSE, actualizado_en = NOW() WHERE id = $1 AND activo = TRUE`, id,
	)
	if err != nil {
		return fmt.Errorf("error al eliminar reserva: %w", err)
	}
	n, err := affectedRows(res, "eliminar reserva")
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("reserva con id %d no encontrada o inactiva", id)
	}
	return nil
}

func (r *ReservasRepo) UpdateReservaNotificado(id int, notificado bool) error {
	res, err := r.db.Exec(
		`UPDATE reservas SET notificado = $1, actualizado_en = NOW() WHERE id = $2 AND activo = TRUE`,
		notificado,
		id,
	)
	if err != nil {
		return fmt.Errorf("error al actualizar notificacion de reserva: %w", err)
	}

	n, err := affectedRows(res, "actualizar notificacion de reserva")
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("reserva con id %d no encontrada o inactiva", id)
	}

	return nil
}

func (r *ReservasRepo) UpdateReservasNotificado(ids []int, notificado bool) (int, error) {
	query, args, err := sqlx.In(
		`UPDATE reservas SET notificado = ?, actualizado_en = NOW() WHERE activo = TRUE AND id IN (?)`,
		notificado,
		ids,
	)
	if err != nil {
		return 0, fmt.Errorf("error al preparar actualizacion masiva de notificaciones: %w", err)
	}

	res, err := r.db.Exec(r.db.Rebind(query), args...)
	if err != nil {
		return 0, fmt.Errorf("error al actualizar notificaciones de reservas: %w", err)
	}

	n, err := affectedRows(res, "actualizar notificaciones de reservas")
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func (r *ReservasRepo) UpdateReservaEstado(input repository.UpdateReservaEstadoInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if input.TipoEspacio != nil {
		var localID int
		var fecha time.Time
		var horaDesde string
		var horaHasta string

		err = tx.QueryRowx(`
			SELECT local_id, fecha, hora_desde::text, hora_hasta::text
			FROM reservas
			WHERE id = $1 AND activo = TRUE
		`, input.ID).Scan(&localID, &fecha, &horaDesde, &horaHasta)
		if err != nil {
			return fmt.Errorf("reserva no encontrada")
		}

		if err := r.validarCapacidad(tx, localID, *input.TipoEspacio, fecha, horaDesde, horaHasta, input.ID); err != nil {
			return err
		}
	}

	sets := []string{"estado = $1", "actualizado_en = NOW()"}
	args := []interface{}{input.Estado}
	idx := 2

	if input.ServicioConfirmado != nil {
		sets = append(sets, fmt.Sprintf("servicio_confirmado = $%d", idx))
		args = append(args, *input.ServicioConfirmado)
		idx++
	}
	if input.Precio != nil {
		sets = append(sets, fmt.Sprintf("precio = $%d", idx))
		args = append(args, *input.Precio)
		idx++
	}
	if input.TipoEspacio != nil {
		sets = append(sets, fmt.Sprintf("tipo_espacio = $%d", idx))
		args = append(args, strings.ToUpper(*input.TipoEspacio))
		idx++
	}

	args = append(args, input.ID)
	result, err := tx.Exec(
		fmt.Sprintf(`UPDATE reservas SET %s WHERE id = $%d AND activo = TRUE`, strings.Join(sets, ", "), idx),
		args...,
	)
	if err != nil {
		return fmt.Errorf("error al actualizar estado de reserva: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error al verificar actualizacion de estado: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("reserva no encontrada")
	}

	return tx.Commit()
}

// Validación de espacios y cantidades (no ocupado para esos ambientes en ese momento)

func (r *ReservasRepo) validarCapacidad(
	tx *sqlx.Tx, localID int, tipoEspacio string,
	fecha time.Time, horaDesde, horaHasta string, excludeID int,
) error {
	tipo := strings.ToUpper(tipoEspacio)

	var capacidad int
	err := tx.QueryRowx(`
		SELECT cantidad_espacios FROM tipos_espacio_locales
		WHERE local_id = $1 AND tipo_espacio = $2
	`, localID, tipo).Scan(&capacidad)
	if err != nil {
		// El texto "no está disponible" es el que el handler mapea a 409.
		return fmt.Errorf(
			"este servicio necesita %s, y esta sucursal no está disponible para ese tipo de ambiente. Elige otra sucursal u otro servicio",
			nombreTipoEspacio(tipo, 2),
		)
	}

	var ocupados int
	err = tx.QueryRowx(consultaMaxConcurrencia(),
		localID, tipo, fecha, horaDesde, horaHasta, excludeID).Scan(&ocupados)
	if err != nil {
		return fmt.Errorf("error al verificar disponibilidad: %w", err)
	}

	if ocupados >= capacidad {
		// El prefijo "No hay ambientes" es el que el handler mapea a 409: no
		// cambiarlo sin ajustar también handlers/reservas_pg_handler.go.
		return fmt.Errorf(
			"No hay ambientes disponibles: %s de esta sucursal (%d en total) ya están ocupados en algún momento entre las %s y las %s del %s. Elige otro horario, acorta la duración o revisa la agenda del día",
			nombreTipoEspacio(tipo, capacidad), capacidad,
			horaCorta(horaDesde), horaCorta(horaHasta), fecha.Format("02/01/2006"),
		)
	}
	return nil

}

// nombreTipoEspacio traduce el código guardado ('M'/'B') a algo legible para el
// staff, concordando en número con la capacidad del local.
func nombreTipoEspacio(tipo string, capacidad int) string {
	singular, plural := "el ambiente", "los ambientes"
	switch tipo {
	case "M":
		singular, plural = "la mesa", "las mesas"
	case "B":
		singular, plural = "la bicicleta", "las bicicletas"
	}
	if capacidad == 1 {
		return singular
	}
	return plural
}

// horaCorta recorta los segundos que PostgreSQL agrega a las columnas TIME
// ("15:00:00" → "15:00") para que el mensaje se lea natural.
func horaCorta(hora string) string {
	if len(hora) >= 5 {
		return hora[:5]
	}
	return hora
}

// consultaMaxConcurrencia mide la concurrencia máxima de reservas activas en
// cualquier instante dentro del rango [hora_desde, hora_hasta). Contar todas las
// reservas que solapan el rango sobrestima la ocupación cuando sub-bloques
// distintos están ocupados por reservas distintas (ej: 12:00-12:30 y 12:30-13:00
// con capacidad 2): cada instante tiene 1 ocupada de 2 y una nueva de 60min
// debería entrar. Se evalúa en cada instante relevante (inicio del rango pedido y
// horarios de inicio de las reservas existentes que solapan) y se toma el máximo.
func consultaMaxConcurrencia() string {
	return `
		SELECT COALESCE(MAX(depth), 0)
		FROM (
			SELECT (
				SELECT COUNT(*)
				FROM reservas r
				WHERE r.local_id = $1 AND r.tipo_espacio = $2 AND r.fecha = $3
				  AND r.activo = TRUE AND r.id != $6
				  AND r.hora_desde <= t.instant AND r.hora_hasta > t.instant
			) AS depth
			FROM (
				SELECT $4::time AS instant
				UNION
				SELECT hora_desde FROM reservas r2
				WHERE r2.local_id = $1 AND r2.tipo_espacio = $2 AND r2.fecha = $3
				  AND r2.activo = TRUE AND r2.id != $6
				  AND r2.hora_desde < $5::time AND r2.hora_hasta > $4::time
			) t
		) s`
}

// BuildJerarquia (Derivado de la que se tenia )

func BuildJerarquia(reservas []models.ReservaPGCompleta) []models.LocalReservas {
	type slotKey struct{ horaDesde, horaHasta string }
	type semanaKey struct{ inicio time.Time }

	localesMap := map[string]map[semanaKey]map[string]map[slotKey][]models.ReservaItem{}

	for _, rv := range reservas {
		local := rv.LocalNombre
		weekday := int(rv.Fecha.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		lunes := rv.Fecha.AddDate(0, 0, -(weekday - 1))
		sk := semanaKey{inicio: lunes}
		dia := diaNombre(rv.Fecha.Weekday())
		slot := slotKey{horaDesde: rv.HoraDesde, horaHasta: rv.HoraHasta}

		item := models.ReservaItem{
			ID:            rv.ID,
			Tipo:          tipoLetraANombre(rv.TipoEspacio),
			Cliente:       rv.Cliente,
			Local:         rv.LocalNombre,
			Fecha:         rv.Fecha.Format("2006-01-02"),
			HoraDesde:     rv.HoraDesde,
			HoraHasta:     rv.HoraHasta,
			HoraHastaReal: rv.HoraHastaOriginal,
			PlanID:        rv.PlanID,
			Notificado:    rv.Notificado,
		}
		if !rv.CreadoEn.IsZero() {
			item.CreadoEn = rv.CreadoEn.Format(time.RFC3339)
		}
		if !rv.ActualizadoEn.IsZero() {
			item.ActualizadoEn = rv.ActualizadoEn.Format(time.RFC3339)
		}
		if rv.ServicioNombre != nil {
			item.Servicio = *rv.ServicioNombre
		}
		if rv.ServicioSolicitado != nil {
			item.ServicioSolicitado = *rv.ServicioSolicitado
		}
		if rv.ServicioConfirmado != nil {
			item.ServicioConfirmado = *rv.ServicioConfirmado
		}
		if rv.Estado != nil {
			item.Estado = *rv.Estado
		}
		if rv.NumeroTelefono != nil {
			item.NumeroTelefono = *rv.NumeroTelefono
		}

		if localesMap[local] == nil {
			localesMap[local] = map[semanaKey]map[string]map[slotKey][]models.ReservaItem{}
		}
		if localesMap[local][sk] == nil {
			localesMap[local][sk] = map[string]map[slotKey][]models.ReservaItem{}
		}
		if localesMap[local][sk][dia] == nil {
			localesMap[local][sk][dia] = map[slotKey][]models.ReservaItem{}
		}
		localesMap[local][sk][dia][slot] = append(localesMap[local][sk][dia][slot], item)
	}

	var resultado []models.LocalReservas
	for localNombre, semanasMap := range localesMap {
		semanas := make([]semanaKey, 0, len(semanasMap))
		for sk := range semanasMap {
			semanas = append(semanas, sk)
		}
		sort.Slice(semanas, func(i, j int) bool {
			return semanas[i].inicio.Before(semanas[j].inicio)
		})

		var semanasOut []models.Semana
		for _, sk := range semanas {
			diasMap := semanasMap[sk]

			slotsMap := map[slotKey]map[string][]models.ReservaItem{}
			for dia, slots := range diasMap {
				for slot, items := range slots {
					if slotsMap[slot] == nil {
						slotsMap[slot] = map[string][]models.ReservaItem{}
					}
					slotsMap[slot][dia] = items
				}
			}

			slotKeys := make([]slotKey, 0, len(slotsMap))
			for sk := range slotsMap {
				slotKeys = append(slotKeys, sk)
			}
			sort.Slice(slotKeys, func(i, j int) bool {
				return slotKeys[i].horaDesde < slotKeys[j].horaDesde
			})

			var reservasOut []models.ReservaSlot
			for _, slot := range slotKeys {
				reservasOut = append(reservasOut, models.ReservaSlot{
					Hora: formatHora(slot.horaDesde) + " a " + formatHora(slot.horaHasta),
					Dias: slotsMap[slot],
				})
			}

			viernes := sk.inicio.AddDate(0, 0, 4)
			titulo := fmt.Sprintf("SEMANA %s AL %s DE %s",
				sk.inicio.Format("02"),
				viernes.Format("02"),
				mesNombre(sk.inicio.Month()),
			)

			semanasOut = append(semanasOut, models.Semana{
				Titulo:   titulo,
				Reservas: reservasOut,
			})
		}

		resultado = append(resultado, models.LocalReservas{
			Local:   localNombre,
			Semanas: semanasOut,
		})
	}

	return resultado
}

// Helpers
func diaNombre(d time.Weekday) string {
	return map[time.Weekday]string{
		time.Monday: "LUNES", time.Tuesday: "MARTES", time.Wednesday: "MIÉRCOLES",
		time.Thursday: "JUEVES", time.Friday: "VIERNES", time.Saturday: "SÁBADO", time.Sunday: "DOMINGO",
	}[d]
}

func mesNombre(m time.Month) string {
	return map[time.Month]string{
		time.January: "ENERO", time.February: "FEBRERO", time.March: "MARZO",
		time.April: "ABRIL", time.May: "MAYO", time.June: "JUNIO",
		time.July: "JULIO", time.August: "AGOSTO", time.September: "SEPTIEMBRE",
		time.October: "OCTUBRE", time.November: "NOVIEMBRE", time.December: "DICIEMBRE",
	}[m]
}

func tipoLetraANombre(letra string) string {
	switch strings.ToUpper(letra) {
	case "M":
		return "mesa"
	case "B":
		return "bicicleta"
	}
	return strings.ToLower(letra)
}

func formatHora(h string) string {
	h = strings.TrimSuffix(h, ":00")
	// h = strings.TrimPrefix(h, "0") // NO TRIM ZERO
	return h
}

func soloDigitosTelefono(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// GetCapacidades retorna tipos de espacio y su cantidad per local
func (r *ReservasRepo) GetCapacidades(localNombre string) ([]repository.CapacidadLocal, error) {
	query := `
		SELECT l.nombre AS local_nombre, t.tipo_espacio, t.cantidad_espacios AS capacidad
		FROM tipos_espacio_locales t
		JOIN locales l ON l.id = t.local_id
		WHERE l.activo = TRUE
	`
	args := []interface{}{}
	if localNombre != "" {
		query += " AND UPPER(l.nombre) = UPPER($1)"
		args = append(args, localNombre)
	}
	query += " ORDER BY l.nombre, t.tipo_espacio"

	rows, err := r.db.Queryx(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error al consultar capacidades: %w", err)
	}
	defer rows.Close()

	var resultado []repository.CapacidadLocal
	for rows.Next() {
		var c repository.CapacidadLocal
		if err := rows.Scan(&c.LocalNombre, &c.TipoEspacio, &c.Capacidad); err != nil {
			return nil, fmt.Errorf("error al leer capacidad: %w", err)
		}
		resultado = append(resultado, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error al recorrer capacidades: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("error al cerrar consulta de capacidades: %w", err)
	}
	return resultado, nil
}
