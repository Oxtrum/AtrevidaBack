package pgsql

import (
	"fmt"
	"strings"
	"time"

	"atrevida-agenda-api/models"
)

// FiltroFlujo acota el flujo de caja unificado. Fechas inclusivas.
type FiltroFlujo struct {
	Desde   time.Time
	Hasta   time.Time
	LocalID *int
	Origen  string // VENTA, COMPRA, GASTO_FIJO, GASTO_VARIABLE, OTRO_INGRESO
	Estado  string // PAGADO, PENDIENTE, ANULADO
	Q       string
	Limit   int
}

// flujoBaseSQL une ventas (pagos) y movimientos financieros en una sola lista. Las compras de
// inventario ya generan su egreso en movimientos_financieros, por eso no se leen aparte.
// Los pagos en BORRADOR o inactivos no son movimientos de caja y se excluyen.
const flujoBaseSQL = `
	SELECT 'pago-' || p.id AS clave, 'VENTA' AS origen, p.codigo_pago AS codigo, 'INGRESO' AS naturaleza,
	       'Venta' AS categoria, ('Venta a ' || p.cliente_nombre) AS detalle,
	       p.local_id, p.local_nombre, p.total_final AS monto, p.estado,
	       p.fecha_creacion AS ts, p.fecha_creacion::date AS fecha
	FROM pagos p
	WHERE p.activo = TRUE AND p.estado IN ('PAGADO', 'PENDIENTE')
	UNION ALL
	SELECT 'mov-' || m.id, CASE WHEN m.compra_id IS NOT NULL THEN 'COMPRA'
	                           WHEN m.naturaleza = 'INGRESO' THEN 'OTRO_INGRESO'
	                           WHEN m.clase = 'FIJO' THEN 'GASTO_FIJO' ELSE 'GASTO_VARIABLE' END,
	       m.codigo, m.naturaleza, c.nombre, COALESCE(NULLIF(m.descripcion, ''), m.proveedor, ''),
	       m.local_id, m.local_nombre, m.monto, m.estado, m.fecha::timestamptz, m.fecha
	FROM movimientos_financieros m
	JOIN categorias_financieras c ON c.id = m.categoria_id`

// flujoWhere arma los filtros sobre la lista unida. conLocal=false omite el local (comparativa por sucursal).
func flujoWhere(f FiltroFlujo, conLocal bool) (string, []interface{}) {
	conds := []string{"b.fecha >= $1", "b.fecha <= $2"}
	args := []interface{}{f.Desde, f.Hasta}
	add := func(cond string, v interface{}) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if conLocal && f.LocalID != nil {
		add("b.local_id = $%d", *f.LocalID)
	}
	if f.Origen != "" {
		add("b.origen = $%d", f.Origen)
	}
	if f.Estado != "" {
		add("b.estado = $%d", f.Estado)
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			"(LOWER(b.codigo) LIKE $%[1]d OR LOWER(b.categoria) LIKE $%[1]d OR LOWER(b.detalle) LIKE $%[1]d OR LOWER(COALESCE(b.local_nombre, '')) LIKE $%[1]d)", n))
	}
	return strings.Join(conds, " AND "), args
}

// ListFlujo devuelve los movimientos del periodo, mas recientes primero, con el saldo acumulado
// (solo cuentan los PAGADO) calculado sobre todo el rango filtrado antes de aplicar el limite.
func (r *FinanzasRepo) ListFlujo(f FiltroFlujo) ([]models.FlujoMovimiento, bool, error) {
	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	where, args := flujoWhere(f, true)
	args = append(args, limit+1)
	out := []models.FlujoMovimiento{}
	err := r.db.Select(&out, fmt.Sprintf(`
		WITH b AS (%s)
		SELECT * FROM (
			SELECT b.clave, b.fecha::text AS fecha, b.origen, b.codigo, b.naturaleza, b.categoria, b.detalle,
			       b.local_id, b.local_nombre, b.monto, b.estado, b.ts,
			       SUM(CASE WHEN b.estado = 'PAGADO' THEN (CASE WHEN b.naturaleza = 'INGRESO' THEN b.monto ELSE -b.monto END) ELSE 0 END)
			         OVER (ORDER BY b.ts, b.clave) AS saldo
			FROM b WHERE %s
		) t ORDER BY t.ts DESC, t.clave DESC LIMIT $%d`, flujoBaseSQL, where, len(args)), args...)
	if err != nil {
		return nil, false, fmt.Errorf("error al listar el flujo de caja: %w", err)
	}
	truncado := len(out) > limit
	if truncado {
		out = out[:limit]
	}
	return out, truncado, nil
}

// TotalesFlujo suma el rango filtrado: lo cobrado/pagado y lo pendiente.
func (r *FinanzasRepo) TotalesFlujo(f FiltroFlujo) (models.FlujoTotales, error) {
	where, args := flujoWhere(f, true)
	var t models.FlujoTotales
	err := r.db.Get(&t, fmt.Sprintf(`
		WITH b AS (%s)
		SELECT
			COALESCE(SUM(b.monto) FILTER (WHERE b.naturaleza = 'INGRESO' AND b.estado = 'PAGADO'), 0) AS ingresos,
			COALESCE(SUM(b.monto) FILTER (WHERE b.naturaleza = 'EGRESO'  AND b.estado = 'PAGADO'), 0) AS egresos,
			COALESCE(SUM(b.monto) FILTER (WHERE b.naturaleza = 'INGRESO' AND b.estado = 'PENDIENTE'), 0) AS por_cobrar,
			COALESCE(SUM(b.monto) FILTER (WHERE b.naturaleza = 'EGRESO'  AND b.estado = 'PENDIENTE'), 0) AS por_pagar,
			COUNT(*) AS movimientos
		FROM b WHERE %s`, flujoBaseSQL, where), args...)
	if err != nil {
		return t, fmt.Errorf("error al totalizar el flujo de caja: %w", err)
	}
	t.Balance = t.Ingresos - t.Egresos
	return t, nil
}

// FlujoPorLocal compara las sucursales con los mismos filtros, sin acotar por local.
func (r *FinanzasRepo) FlujoPorLocal(f FiltroFlujo) ([]models.FlujoLocal, error) {
	where, args := flujoWhere(f, false)
	out := []models.FlujoLocal{}
	err := r.db.Select(&out, fmt.Sprintf(`
		WITH b AS (%s)
		SELECT COALESCE(b.local_id, 0) AS local_id, COALESCE(b.local_nombre, 'General') AS local,
		       COALESCE(SUM(b.monto) FILTER (WHERE b.naturaleza = 'INGRESO' AND b.estado = 'PAGADO'), 0) AS ingresos,
		       COALESCE(SUM(b.monto) FILTER (WHERE b.naturaleza = 'EGRESO'  AND b.estado = 'PAGADO'), 0) AS egresos
		FROM b WHERE %s
		GROUP BY 1, 2 ORDER BY 2`, flujoBaseSQL, where), args...)
	if err != nil {
		return nil, fmt.Errorf("error al comparar locales: %w", err)
	}
	for i := range out {
		out[i].Balance = out[i].Ingresos - out[i].Egresos
	}
	return out, nil
}
