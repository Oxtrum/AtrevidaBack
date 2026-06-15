package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

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
	conditions := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if filtro.CodigoPago != "" {
		conditions = append(conditions, fmt.Sprintf("p.codigo_pago ILIKE $%d", idx))
		args = append(args, "%"+filtro.CodigoPago+"%")
		idx++
	}
	if filtro.LocalID != nil {
		conditions = append(conditions, fmt.Sprintf("p.local_id = $%d", idx))
		args = append(args, *filtro.LocalID)
		idx++
	}
	if filtro.LocalNombre != "" {
		conditions = append(conditions, fmt.Sprintf("p.local_nombre ILIKE $%d", idx))
		args = append(args, "%"+filtro.LocalNombre+"%")
		idx++
	}
	if filtro.ClienteID != nil {
		conditions = append(conditions, fmt.Sprintf("p.cliente_id = $%d", idx))
		args = append(args, *filtro.ClienteID)
		idx++
	}
	if filtro.ClienteNIT != "" {
		conditions = append(conditions, fmt.Sprintf("p.cliente_nit ILIKE $%d", idx))
		args = append(args, "%"+filtro.ClienteNIT+"%")
		idx++
	}
	if filtro.ClienteNombre != "" {
		conditions = append(conditions, fmt.Sprintf("p.cliente_nombre ILIKE $%d", idx))
		args = append(args, "%"+filtro.ClienteNombre+"%")
		idx++
	}
	if filtro.TipoPago != "" {
		conditions = append(conditions, fmt.Sprintf("p.tipo_pago = $%d", idx))
		args = append(args, filtro.TipoPago)
		idx++
	}
	if filtro.Estado != "" {
		conditions = append(conditions, fmt.Sprintf("p.estado = $%d", idx))
		args = append(args, filtro.Estado)
		idx++
	}
	if filtro.Activo != nil {
		conditions = append(conditions, fmt.Sprintf("p.activo = $%d", idx))
		args = append(args, *filtro.Activo)
		idx++
	}
	if filtro.IDCajero != nil {
		conditions = append(conditions, fmt.Sprintf("p.id_cajero = $%d", idx))
		args = append(args, *filtro.IDCajero)
		idx++
	}
	if filtro.NombreCajero != "" {
		conditions = append(conditions, fmt.Sprintf("p.nombre_cajero ILIKE $%d", idx))
		args = append(args, "%"+filtro.NombreCajero+"%")
		idx++
	}
	if filtro.UsernameCajero != "" {
		conditions = append(conditions, fmt.Sprintf("p.username_cajero ILIKE $%d", idx))
		args = append(args, "%"+filtro.UsernameCajero+"%")
		idx++
	}
	if filtro.IDCajeroModificacion != nil {
		conditions = append(conditions, fmt.Sprintf("p.id_cajero_modificacion = $%d", idx))
		args = append(args, *filtro.IDCajeroModificacion)
		idx++
	}
	if filtro.NombreCajeroModificacion != "" {
		conditions = append(conditions, fmt.Sprintf("p.nombre_cajero_modificacion ILIKE $%d", idx))
		args = append(args, "%"+filtro.NombreCajeroModificacion+"%")
		idx++
	}
	if filtro.UsernameCajeroModificacion != "" {
		conditions = append(conditions, fmt.Sprintf("p.username_cajero_modificacion ILIKE $%d", idx))
		args = append(args, "%"+filtro.UsernameCajeroModificacion+"%")
		idx++
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM pagos p
		WHERE %s
		ORDER BY p.fecha_creacion DESC, p.id DESC
	`, pagoSelectColumns(), strings.Join(conditions, " AND "))

	var pagos []models.PagoPG
	if err := r.db.Select(&pagos, query, args...); err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los pagos")
	}

	return pagos, nil
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

func (r *PagosRepo) GetResumenPagos(filtro repository.FiltroResumenPagos) ([]repository.PagoResumenRow, error) {
	conditions := []string{
		"p.activo = TRUE",
		"p.estado = 'PAGADO'",
		"p.fecha_creacion::date >= $1::date",
		"p.fecha_creacion::date <= $2::date",
	}
	args := []interface{}{filtro.FechaDesde, filtro.FechaHasta}
	idx := 3

	if filtro.Local != "" {
		conditions = append(conditions, fmt.Sprintf("UPPER(p.local_nombre) = UPPER($%d)", idx))
		args = append(args, filtro.Local)
		idx++
	}

	query := fmt.Sprintf(`
		SELECT
			p.id AS pago_id,
			p.local_nombre,
			p.tipo_pago,
			p.subtotal,
			p.descuento,
			p.total_final,
			dp.servicio,
			dp.cantidad,
			dp.subtotal AS detalle_subtotal
		FROM pagos p
		LEFT JOIN detalle_pagos dp ON dp.pago_id = p.id
		WHERE %s
		ORDER BY p.local_nombre, p.id, dp.id
	`, strings.Join(conditions, " AND "))

	var rows []repository.PagoResumenRow
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, fmt.Errorf("no se pudo obtener el resumen de pagos")
	}

	return rows, nil
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
		`, pagoID, pq.Array(ids)).Scan(&existentes)
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
		`, pagoID, pq.Array(ids)); err != nil {
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
	var pqErr *pq.Error
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
