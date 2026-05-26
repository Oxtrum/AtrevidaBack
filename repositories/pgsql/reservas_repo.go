package pgsql

import (
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
	conditions := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.LocalNombre != "" {
		conditions = append(conditions, fmt.Sprintf("UPPER(r.local_nombre) = UPPER($%d)", idx))
		args = append(args, f.LocalNombre)
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
	if f.NumeroTelefono != "" {
		digitos := soloDigitosTelefono(f.NumeroTelefono)
		last8 := digitos
		if len(last8) > 8 {
			last8 = last8[len(last8)-8:]
		}
		conditions = append(conditions, fmt.Sprintf(`(
			BTRIM(COALESCE(r.numero_telefono, '')) = BTRIM($%d)
			OR regexp_replace(COALESCE(r.numero_telefono, ''), '\D', '', 'g') = $%d
			OR RIGHT(regexp_replace(COALESCE(r.numero_telefono, ''), '\D', '', 'g'), 8) = $%d
		)`, idx, idx+1, idx+2))
		args = append(args, f.NumeroTelefono, digitos, last8)
		idx += 3
	}
	if f.ServicioSolicitado != "" {
		conditions = append(conditions, fmt.Sprintf("COALESCE(r.servicio_solicitado, '') ILIKE $%d", idx))
		args = append(args, "%"+f.ServicioSolicitado+"%")
		idx++
	}
	if f.ServicioConfirmado != "" {
		conditions = append(conditions, fmt.Sprintf("COALESCE(r.servicio_confirmado, '') ILIKE $%d", idx))
		args = append(args, "%"+f.ServicioConfirmado+"%")
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
			r.id, r.local_id, r.local_nombre, r.tipo_espacio,
			r.fecha, r.hora_desde::text, r.hora_hasta::text,
			r.cliente, r.estado, r.numero_telefono, r.plan_id, r.servicio_nombre,
			r.servicio_solicitado, r.servicio_confirmado, r.servicio_tiempo,
			r.precio, r.notas, r.activo, COALESCE(r.notificado, FALSE) AS notificado,
			r.creado_en, r.actualizado_en
		FROM reservas r
		WHERE %s
		ORDER BY r.local_nombre, r.fecha, r.hora_desde
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
		rv.Detalle, _ = r.getDetalleReserva(rv.ID)
		reservas = append(reservas, rv)
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

	rv.Detalle, _ = r.getDetalleReserva(rv.ID)
	return &rv, nil
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

	if input.PlanID != nil {
		_, err = tx.Exec(
			`UPDATE planes SET sesiones_usadas = sesiones_usadas + 1 WHERE id = $1`, *input.PlanID,
		)
		if err != nil {
			return 0, fmt.Errorf("error al actualizar plan: %w", err)
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

	args = append(args, reservaID)
	_, err = tx.Exec(
		fmt.Sprintf("UPDATE reservas SET %s WHERE id = $%d", strings.Join(sets, ", "), idx),
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

	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("reserva con id %d no encontrada o inactiva", id)
	}

	return nil
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
		return fmt.Errorf("el tipo '%s' no está disponible en este local", tipo)
	}

	var ocupados int
	err = tx.QueryRowx(`
		SELECT COUNT(*) FROM reservas
		WHERE local_id = $1 AND tipo_espacio = $2 AND fecha = $3
		  AND activo = TRUE
		  AND hora_desde < $5::time AND hora_hasta > $4::time
		  AND id != $6
	`, localID, tipo, fecha, horaDesde, horaHasta, excludeID).Scan(&ocupados)
	if err != nil {
		return fmt.Errorf("error al verificar disponibilidad: %w", err)
	}

	if ocupados >= capacidad {
		return fmt.Errorf("No hay ambientes disponibles para este servicio en esa fecha y horario")
	}
	return nil

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
			Tipo:       tipoLetraANombre(rv.TipoEspacio),
			Cliente:    rv.Cliente,
			Notificado: rv.Notificado,
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
			continue
		}
		resultado = append(resultado, c)
	}
	return resultado, nil
}
