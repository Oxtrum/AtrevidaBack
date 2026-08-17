package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var _ repository.PagosRepository = (*PagosRepo)(nil)

type PagosRepo struct {
	db *sqlx.DB
}

func NewPagosRepo(db *sqlx.DB) *PagosRepo {
	return &PagosRepo{db: db}
}

func (r *PagosRepo) GetPagos(filtro repository.FiltroPagos) ([]models.PagoPG, error) {
	conditions, args := pagoConditions(filtro)
	idx := len(args) + 1
	if filtro.CursorSet {
		conditions = append(conditions, fmt.Sprintf("(p.fecha_creacion, p.id) < ($%d, $%d)", idx, idx+1))
		args = append(args, filtro.CursorFecha, filtro.CursorID)
		idx += 2
	}
	limitClause := ""
	if filtro.PageLimit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, filtro.PageLimit)
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM pagos p
		WHERE %s
		ORDER BY p.fecha_creacion DESC, p.id DESC%s
	`, pagoSelectColumns(), strings.Join(conditions, " AND "), limitClause)

	var pagos []models.PagoPG
	if err := r.db.SelectContext(queryContext(filtro.Context), &pagos, query, args...); err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los pagos")
	}
	if pagos == nil {
		pagos = []models.PagoPG{}
	}

	return pagos, nil
}

func (r *PagosRepo) CountPagos(filtro repository.FiltroPagos) (int, error) {
	conditions, args := pagoConditions(filtro)
	query := fmt.Sprintf("SELECT COUNT(*) FROM pagos p WHERE %s", strings.Join(conditions, " AND "))
	var total int
	if err := r.db.GetContext(queryContext(filtro.Context), &total, query, args...); err != nil {
		return 0, fmt.Errorf("no se pudieron contar los pagos")
	}
	return total, nil
}

func pagoConditions(f repository.FiltroPagos) ([]string, []interface{}) {
	conditions := []string{"1=1"}
	args := []interface{}{}
	add := func(condition string, value interface{}) {
		conditions = append(conditions, fmt.Sprintf(condition, len(args)+1))
		args = append(args, value)
	}
	if f.Busqueda != "" {
		add(`TRANSLATE(LOWER(CONCAT_WS(' ',
			p.codigo_pago, p.local_nombre, p.cliente_nombre, p.cliente_nit,
			p.nombre_cajero, p.username_cajero, p.nombre_cajero_modificacion,
			p.username_cajero_modificacion, p.tipo_pago, p.estado
		)), 'áéíóúüñ', 'aeiouun') LIKE TRANSLATE(LOWER($%d), 'áéíóúüñ', 'aeiouun')`, "%"+f.Busqueda+"%")
	}
	if f.CodigoPago != "" {
		add("p.codigo_pago ILIKE $%d", "%"+f.CodigoPago+"%")
	}
	if f.LocalID != nil {
		add("p.local_id = $%d", *f.LocalID)
	}
	if f.LocalNombre != "" {
		add("p.local_nombre ILIKE $%d", "%"+f.LocalNombre+"%")
	}
	if f.ClienteID != nil {
		add("p.cliente_id = $%d", *f.ClienteID)
	}
	if f.ClienteNIT != "" {
		add("p.cliente_nit ILIKE $%d", "%"+f.ClienteNIT+"%")
	}
	if f.ClienteNombre != "" {
		add("p.cliente_nombre ILIKE $%d", "%"+f.ClienteNombre+"%")
	}
	if f.TipoPago != "" {
		add("p.tipo_pago = $%d", f.TipoPago)
	}
	if f.Estado != "" {
		add("p.estado = $%d", f.Estado)
	}
	if f.Activo != nil {
		add("p.activo = $%d", *f.Activo)
	}
	if f.IDCajero != nil {
		add("p.id_cajero = $%d", *f.IDCajero)
	}
	if f.NombreCajero != "" {
		add("p.nombre_cajero ILIKE $%d", "%"+f.NombreCajero+"%")
	}
	if f.UsernameCajero != "" {
		add("p.username_cajero ILIKE $%d", "%"+f.UsernameCajero+"%")
	}
	if f.IDCajeroModificacion != nil {
		add("p.id_cajero_modificacion = $%d", *f.IDCajeroModificacion)
	}
	if f.NombreCajeroModificacion != "" {
		add("p.nombre_cajero_modificacion ILIKE $%d", "%"+f.NombreCajeroModificacion+"%")
	}
	if f.UsernameCajeroModificacion != "" {
		add("p.username_cajero_modificacion ILIKE $%d", "%"+f.UsernameCajeroModificacion+"%")
	}
	return conditions, args
}

func (r *PagosRepo) GetPagoByCodigo(codigoPago string) (*models.PagoCompletoPG, error) {
	var pago models.PagoCompletoPG
	err := r.db.Get(&pago, fmt.Sprintf(`
		SELECT %s
		FROM pagos p
		WHERE p.codigo_pago = $1
		  AND p.activo = TRUE
	`, pagoSelectColumns()), codigoPago)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pago no encontrado")
		}
		return nil, fmt.Errorf("no se pudo obtener el pago")
	}

	detalle, err := r.getDetallePago(pago.ID)
	if err != nil {
		return nil, err
	}
	pago.Detalle = detalle

	return &pago, nil
}

func (r *PagosRepo) CreatePago(input repository.CrearPagoInput) (string, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var pagoID int
	var codigoPago string
	err = tx.QueryRowx(`
		INSERT INTO pagos (
			local_id, local_nombre, cliente_id, cliente_nit, cliente_nombre,
			subtotal, descuento, total_final, tipo_pago, estado, activo,
			id_cajero, nombre_cajero, username_cajero
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, codigo_pago
	`,
		input.LocalID,
		input.LocalNombre,
		input.ClienteID,
		input.ClienteNIT,
		input.ClienteNombre,
		*input.Subtotal,
		*input.Descuento,
		*input.TotalFinal,
		input.TipoPago,
		input.Estado,
		input.Activo,
		input.Cajero.ID,
		input.Cajero.Nombre,
		input.Cajero.Username,
	).Scan(&pagoID, &codigoPago)
	if err != nil {
		return "", pagoInsertError(err)
	}

	for _, d := range input.Detalle {
		_, err = tx.Exec(`
			INSERT INTO detalle_pagos (
				pago_id, servicio_id, servicio, precio_unitario, cantidad, subtotal
			)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, pagoID, d.ServicioID, d.Servicio, d.PrecioUnitario, d.Cantidad, d.Subtotal)
		if err != nil {
			return "", pagoInsertError(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("no se pudo crear el pago")
	}

	return codigoPago, nil
}

func (r *PagosRepo) UpdatePago(input repository.ActualizarPagoInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actual struct {
		ID        int     `db:"id"`
		Estado    string  `db:"estado"`
		Subtotal  float64 `db:"subtotal"`
		Descuento float64 `db:"descuento"`
	}
	err = tx.Get(&actual, `
		SELECT id, estado, subtotal, descuento
		FROM pagos
		WHERE codigo_pago = $1
		  AND activo = TRUE
	`, input.CodigoPago)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("pago no encontrado")
		}
		return fmt.Errorf("no se pudo obtener el pago")
	}
	if strings.EqualFold(actual.Estado, "PAGADO") {
		return fmt.Errorf("no se puede modificar un pago en estado PAGADO")
	}

	if input.Detalle != nil {
		if err := r.syncDetallePago(tx, actual.ID, *input.Detalle); err != nil {
			return err
		}
	}

	subtotalFinal := actual.Subtotal
	if input.Subtotal != nil {
		subtotalFinal = *input.Subtotal
	}
	if input.RecalcularSubtotal || input.RecalcularTotalFinal {
		recalculado, err := r.sumarDetallePago(tx, actual.ID)
		if err != nil {
			return err
		}
		if input.RecalcularSubtotal {
			input.Subtotal = &recalculado
			subtotalFinal = recalculado
		}
	}

	descuentoFinal := actual.Descuento
	if input.Descuento != nil {
		descuentoFinal = *input.Descuento
	}
	if input.RecalcularTotalFinal {
		totalFinal := subtotalFinal - descuentoFinal
		if totalFinal < 0 {
			return fmt.Errorf("descuento no puede ser mayor al subtotal")
		}
		input.TotalFinal = &totalFinal
	}

	sets := []string{"fecha_modificacion = NOW()"}
	args := []interface{}{}
	idx := 1

	if input.LocalID != nil {
		sets = append(sets, fmt.Sprintf("local_id = $%d", idx))
		args = append(args, *input.LocalID)
		idx++
	}
	if input.LocalNombre != nil {
		sets = append(sets, fmt.Sprintf("local_nombre = $%d", idx))
		args = append(args, *input.LocalNombre)
		idx++
	}
	if input.ClienteIDSet {
		sets = append(sets, fmt.Sprintf("cliente_id = $%d", idx))
		args = append(args, input.ClienteID)
		idx++
	}
	if input.ClienteNITSet {
		sets = append(sets, fmt.Sprintf("cliente_nit = $%d", idx))
		args = append(args, input.ClienteNIT)
		idx++
	}
	if input.ClienteNombre != nil {
		sets = append(sets, fmt.Sprintf("cliente_nombre = $%d", idx))
		args = append(args, *input.ClienteNombre)
		idx++
	}
	if input.Subtotal != nil {
		sets = append(sets, fmt.Sprintf("subtotal = $%d", idx))
		args = append(args, *input.Subtotal)
		idx++
	}
	if input.Descuento != nil {
		sets = append(sets, fmt.Sprintf("descuento = $%d", idx))
		args = append(args, *input.Descuento)
		idx++
	}
	if input.TotalFinal != nil {
		sets = append(sets, fmt.Sprintf("total_final = $%d", idx))
		args = append(args, *input.TotalFinal)
		idx++
	}
	if input.TipoPago != nil {
		sets = append(sets, fmt.Sprintf("tipo_pago = $%d", idx))
		args = append(args, *input.TipoPago)
		idx++
	}
	if input.Estado != nil {
		sets = append(sets, fmt.Sprintf("estado = $%d", idx))
		args = append(args, *input.Estado)
		idx++
	}
	if input.Activo != nil {
		sets = append(sets, fmt.Sprintf("activo = $%d", idx))
		args = append(args, *input.Activo)
		idx++
	}

	if len(sets) == 1 {
		return fmt.Errorf("debe enviar al menos un campo para actualizar")
	}
	sets = append(sets, fmt.Sprintf("id_cajero_modificacion = $%d", idx))
	args = append(args, input.Cajero.ID)
	idx++
	sets = append(sets, fmt.Sprintf("nombre_cajero_modificacion = $%d", idx))
	args = append(args, input.Cajero.Nombre)
	idx++
	sets = append(sets, fmt.Sprintf("username_cajero_modificacion = $%d", idx))
	args = append(args, input.Cajero.Username)
	idx++

	args = append(args, input.CodigoPago)
	query := fmt.Sprintf(`
		UPDATE pagos
		SET %s
		WHERE codigo_pago = $%d
		  AND activo = TRUE
	`, strings.Join(sets, ", "), idx)

	res, err := tx.Exec(query, args...)
	if err != nil {
		return pagoInsertError(err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("no se pudo actualizar el pago")
	}
	if rowsAffected == 0 {
		return fmt.Errorf("pago no encontrado")
	}

	return tx.Commit()
}

func (r *PagosRepo) DeletePago(codigoPago string) error {
	res, err := r.db.Exec(`
		UPDATE pagos
		SET activo = FALSE,
		    fecha_modificacion = NOW()
		WHERE codigo_pago = $1
		  AND activo = TRUE
	`, codigoPago)
	if err != nil {
		return fmt.Errorf("no se pudo eliminar el pago")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("no se pudo eliminar el pago")
	}
	if rowsAffected == 0 {
		return fmt.Errorf("pago no encontrado")
	}

	return nil
}

func (r *PagosRepo) GetResumenPagos(filtro repository.FiltroResumenPagos) (repository.PagoResumenAgregado, error) {
	var result repository.PagoResumenAgregado
	conditions := []string{
		"p.activo = TRUE",
		"p.estado = 'PAGADO'",
		"p.fecha_creacion >= $1",
		"p.fecha_creacion < $2",
	}
	args := []interface{}{filtro.FechaDesde, filtro.FechaHasta.AddDate(0, 0, 1)}
	idx := 3

	if filtro.Local != "" {
		conditions = append(conditions, fmt.Sprintf(`p.local_id = (
			SELECT l.id FROM locales l WHERE UPPER(l.nombre) = UPPER($%d) LIMIT 1
		)`, idx))
		args = append(args, filtro.Local)
		idx++
	}

	where := strings.Join(conditions, " AND ")
	ctx := queryContext(filtro.Context)
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return result, fmt.Errorf("no se pudo iniciar el resumen de pagos: %w", err)
	}
	defer tx.Rollback()

	totalesQuery := fmt.Sprintf(`WITH base AS (
		SELECT p.id, BTRIM(p.local_nombre) local_nombre, p.subtotal, p.descuento, p.total_final
		FROM pagos p WHERE %s
	), detalle AS (
		SELECT dp.pago_id, COALESCE(SUM(dp.cantidad),0)::int cantidad
		FROM detalle_pagos dp JOIN base b ON b.id=dp.pago_id GROUP BY dp.pago_id
	)
	SELECT COALESCE(base.local_nombre,'') local_nombre,
		GROUPING(base.local_nombre)::int::boolean es_general,
		COALESCE(SUM(base.subtotal),0) subtotal, COALESCE(SUM(base.descuento),0) descuento,
		COALESCE(SUM(base.total_final),0) total_final, COUNT(*)::int cantidad_pagos,
		COALESCE(SUM(detalle.cantidad),0)::int cantidad_servicios_vendidos
	FROM base LEFT JOIN detalle ON detalle.pago_id=base.id
	GROUP BY GROUPING SETS ((), (base.local_nombre))
	ORDER BY es_general, local_nombre`, where)
	if err := tx.SelectContext(ctx, &result.Totales, totalesQuery, args...); err != nil {
		return result, fmt.Errorf("no se pudo obtener totales del resumen de pagos: %w", err)
	}

	tiposQuery := fmt.Sprintf(`WITH base AS (
		SELECT BTRIM(p.local_nombre) local_nombre,
			COALESCE(NULLIF(BTRIM(p.tipo_pago),''),'sin_tipo') tipo_pago, p.total_final
		FROM pagos p WHERE %s
	)
	SELECT COALESCE(local_nombre,'') local_nombre,
		GROUPING(local_nombre)::int::boolean es_general, tipo_pago,
		COUNT(*)::int cantidad_pagos, COALESCE(SUM(total_final),0) total
	FROM base GROUP BY GROUPING SETS ((tipo_pago),(local_nombre,tipo_pago))
	ORDER BY es_general, local_nombre, tipo_pago`, where)
	if err := tx.SelectContext(ctx, &result.Tipos, tiposQuery, args...); err != nil {
		return result, fmt.Errorf("no se pudo obtener tipos del resumen de pagos: %w", err)
	}

	serviciosQuery := fmt.Sprintf(`WITH base AS (
		SELECT p.id, BTRIM(p.local_nombre) local_nombre FROM pagos p WHERE %s
	)
	SELECT COALESCE(base.local_nombre,'') local_nombre,
		GROUPING(base.local_nombre)::int::boolean es_general, BTRIM(dp.servicio) servicio,
		COALESCE(SUM(dp.cantidad),0)::int cantidad, COALESCE(SUM(dp.subtotal),0) monto_total
	FROM base JOIN detalle_pagos dp ON dp.pago_id=base.id
	WHERE NULLIF(BTRIM(dp.servicio),'') IS NOT NULL
	GROUP BY GROUPING SETS ((dp.servicio),(base.local_nombre,dp.servicio))
	ORDER BY es_general, local_nombre, servicio`, where)
	if err := tx.SelectContext(ctx, &result.Servicios, serviciosQuery, args...); err != nil {
		return result, fmt.Errorf("no se pudo obtener servicios del resumen de pagos: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("no se pudo confirmar el resumen de pagos: %w", err)
	}
	return result, nil
}

func (r *PagosRepo) getDetallePago(pagoID int) ([]models.DetallePagoPG, error) {
	var detalle []models.DetallePagoPG
	err := r.db.Select(&detalle, `
		SELECT id, pago_id, servicio_id, servicio, precio_unitario, cantidad, subtotal
		FROM detalle_pagos
		WHERE pago_id = $1
		ORDER BY id
	`, pagoID)
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener el detalle del pago")
	}

	return detalle, nil
}

func (r *PagosRepo) syncDetallePago(tx *sqlx.Tx, pagoID int, detalle []repository.ActualizarDetallePagoInput) error {
	ids := make([]int, 0, len(detalle))
	for _, d := range detalle {
		if d.ID != nil {
			ids = append(ids, *d.ID)
		}
	}

	if len(ids) > 0 {
		var existentes int
		err := tx.QueryRowx(`
			SELECT COUNT(DISTINCT id)
			FROM detalle_pagos
			WHERE pago_id = $1
			  AND id = ANY($2)
		`, pagoID, ids).Scan(&existentes)
		if err != nil {
			return fmt.Errorf("no se pudo validar el detalle del pago")
		}
		if existentes != len(ids) {
			return fmt.Errorf("detalle de pago no encontrado")
		}

		if _, err := tx.Exec(`
			DELETE FROM detalle_pagos
			WHERE pago_id = $1
			  AND NOT (id = ANY($2))
		`, pagoID, ids); err != nil {
			return fmt.Errorf("no se pudo sincronizar el detalle del pago")
		}
	} else {
		if _, err := tx.Exec(`DELETE FROM detalle_pagos WHERE pago_id = $1`, pagoID); err != nil {
			return fmt.Errorf("no se pudo sincronizar el detalle del pago")
		}
	}

	for _, d := range detalle {
		if d.ID != nil {
			continue
		}

		_, err := tx.Exec(`
			INSERT INTO detalle_pagos (
				pago_id, servicio_id, servicio, precio_unitario, cantidad, subtotal
			)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, pagoID, d.ServicioID, d.Servicio, d.PrecioUnitario, d.Cantidad, d.Subtotal)
		if err != nil {
			return pagoInsertError(err)
		}
	}

	return nil
}

func (r *PagosRepo) sumarDetallePago(tx *sqlx.Tx, pagoID int) (float64, error) {
	var total float64
	err := tx.QueryRowx(`
		SELECT COALESCE(SUM(subtotal), 0)
		FROM detalle_pagos
		WHERE pago_id = $1
	`, pagoID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("no se pudo recalcular el total del detalle")
	}

	return total, nil
}

func pagoSelectColumns() string {
	return `
		p.id,
		p.codigo_pago,
		p.local_id,
		p.local_nombre,
		p.cliente_id,
		COALESCE(p.cliente_nit, '') AS cliente_nit,
		p.cliente_nombre,
		p.subtotal,
		p.descuento,
		p.total_final,
		p.tipo_pago,
		p.estado,
		p.activo,
		p.id_cajero,
		COALESCE(p.nombre_cajero, '') AS nombre_cajero,
		p.username_cajero,
		p.id_cajero_modificacion,
		p.nombre_cajero_modificacion,
		p.username_cajero_modificacion,
		p.fecha_creacion,
		p.fecha_modificacion
	`
}

func pagoInsertError(err error) error {
	var pqErr *pgconn.PgError
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23503":
			return fmt.Errorf("referencia no encontrada: local, cliente o servicio invalido")
		case "23514":
			return fmt.Errorf("datos de pago invalidos")
		case "23505":
			return fmt.Errorf("ya existe un pago con ese codigo")
		}
	}
	return fmt.Errorf("no se pudo guardar el pago")
}
