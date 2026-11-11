package pgsql

import (
	"fmt"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"

	"github.com/lib/pq"
)

// consumosEventosWhere arma el WHERE de los eventos del calendario. El filtro
// por estado se aplica a nivel de atencion: PENDIENTE exige al menos una linea
// pendiente y DESCONTADO exige que ninguna lo este.
func consumosEventosWhere(f repository.FiltroConsumos) (string, []interface{}) {
	conds := []string{"a.activo = TRUE"}
	args := []interface{}{}
	add := func(cond string, v interface{}) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	add("a.fecha_atencion >= $%d", f.FechaDesde)
	add("a.fecha_atencion <= $%d", f.FechaHasta)
	if f.LocalID != nil {
		add("a.local_id = $%d", *f.LocalID)
	}
	if f.EmpleadoID != nil {
		add("a.empleado_id = $%d", *f.EmpleadoID)
	}
	if f.ItemID != nil {
		add("EXISTS (SELECT 1 FROM atencion_insumos ai WHERE ai.atencion_id = a.id AND ai.item_id = $%d)", *f.ItemID)
	}
	switch f.EstadoDescuento {
	case "PENDIENTE":
		conds = append(conds, "EXISTS (SELECT 1 FROM atencion_insumos ai WHERE ai.atencion_id = a.id AND ai.estado_descuento <> 'DESCONTADO')")
	case "DESCONTADO":
		conds = append(conds, "NOT EXISTS (SELECT 1 FROM atencion_insumos ai WHERE ai.atencion_id = a.id AND ai.estado_descuento <> 'DESCONTADO')")
	}
	if v := textoAtencionNormalizado(f.Cliente); v != "" {
		add(textoBusquedaAtencion("r.cliente")+" LIKE $%d", "%"+v+"%")
	}
	if v := textoAtencionNormalizado(f.Servicio); v != "" {
		add(textoBusquedaAtencion(servicioReservaExpr)+" LIKE $%d", "%"+v+"%")
	}
	if v := textoAtencionNormalizado(f.Empleado); v != "" {
		add(textoBusquedaAtencion("a.empleado_nombre")+" LIKE $%d", "%"+v+"%")
	}
	if q := textoAtencionNormalizado(f.Q); q != "" {
		args = append(args, "%"+q+"%")
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			"(%s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR CAST(a.reserva_id AS TEXT) LIKE $%d OR EXISTS (SELECT 1 FROM atencion_insumos ai WHERE ai.atencion_id = a.id AND %s LIKE $%d))",
			textoBusquedaAtencion("r.cliente"), n,
			textoBusquedaAtencion(servicioReservaExpr), n,
			textoBusquedaAtencion("a.empleado_nombre"), n,
			textoBusquedaAtencion("l.nombre"), n,
			n,
			textoBusquedaAtencion("ai.item_nombre"), n,
		))
	}
	return strings.Join(conds, " AND "), args
}

// consumosResumenWhere arma el WHERE del reporte agregado. El filtro por item
// y por estado se aplica sobre la linea de consumo, no sobre la atencion.
func consumosResumenWhere(f repository.FiltroConsumos) (string, []interface{}) {
	conds := []string{"a.activo = TRUE"}
	args := []interface{}{}
	add := func(cond string, v interface{}) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	add("a.fecha_atencion >= $%d", f.FechaDesde)
	add("a.fecha_atencion <= $%d", f.FechaHasta)
	if f.LocalID != nil {
		add("a.local_id = $%d", *f.LocalID)
	}
	if f.EmpleadoID != nil {
		add("a.empleado_id = $%d", *f.EmpleadoID)
	}
	if f.ItemID != nil {
		add("ai.item_id = $%d", *f.ItemID)
	}
	switch f.EstadoDescuento {
	case "PENDIENTE":
		conds = append(conds, "ai.estado_descuento <> 'DESCONTADO'")
	case "DESCONTADO":
		conds = append(conds, "ai.estado_descuento = 'DESCONTADO'")
	}
	if v := textoAtencionNormalizado(f.Cliente); v != "" {
		add(textoBusquedaAtencion("r.cliente")+" LIKE $%d", "%"+v+"%")
	}
	if v := textoAtencionNormalizado(f.Servicio); v != "" {
		add(textoBusquedaAtencion(servicioReservaExpr)+" LIKE $%d", "%"+v+"%")
	}
	if v := textoAtencionNormalizado(f.Empleado); v != "" {
		add(textoBusquedaAtencion("a.empleado_nombre")+" LIKE $%d", "%"+v+"%")
	}
	if q := textoAtencionNormalizado(f.Q); q != "" {
		args = append(args, "%"+q+"%")
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			"(%s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR CAST(a.reserva_id AS TEXT) LIKE $%d OR %s LIKE $%d)",
			textoBusquedaAtencion("r.cliente"), n,
			textoBusquedaAtencion(servicioReservaExpr), n,
			textoBusquedaAtencion("a.empleado_nombre"), n,
			textoBusquedaAtencion("l.nombre"), n,
			n,
			textoBusquedaAtencion("ai.item_nombre"), n,
		))
	}
	return strings.Join(conds, " AND "), args
}

// buildConsumosEventosQuery arma el listado del calendario. Ordena por fecha,
// hora y atencion para que la grilla semanal sea estable.
func buildConsumosEventosQuery(f repository.FiltroConsumos) (string, []interface{}) {
	where, args := consumosEventosWhere(f)
	limit := f.PageLimit
	if limit <= 0 {
		limit = 500
	}
	args = append(args, limit)
	pageClause := fmt.Sprintf("LIMIT $%d", len(args))
	if f.PageLimit > 0 && f.Offset > 0 {
		args = append(args, f.Offset)
		pageClause += fmt.Sprintf(" OFFSET $%d", len(args))
	}
	query := fmt.Sprintf(`
		SELECT a.id AS atencion_id, a.reserva_id,
		       TO_CHAR(a.fecha_atencion, 'YYYY-MM-DD') AS fecha,
		       TO_CHAR(COALESCE(a.hora_inicio, r.hora_desde), 'HH24:MI') AS hora_inicio,
		       TO_CHAR(COALESCE(a.hora_fin, r.hora_hasta), 'HH24:MI') AS hora_fin,
		       (a.hora_inicio IS NULL) AS horario_programado,
		       COALESCE(r.cliente, '') AS cliente,
		       %s AS servicio,
		       a.local_id, COALESCE(l.nombre, '') AS local_nombre,
		       a.empleado_id, COALESCE(a.empleado_nombre, '') AS empleado_nombre,
		       COALESCE(a.lugar, '') AS lugar
		FROM atenciones a
		JOIN reservas r ON r.id = a.reserva_id
		LEFT JOIN locales l ON l.id = a.local_id
		WHERE %s
		ORDER BY a.fecha_atencion ASC, COALESCE(a.hora_inicio, r.hora_desde) ASC NULLS LAST, a.id ASC
		%s`, servicioReservaExpr, where, pageClause)
	return query, args
}

// buildConsumosResumenQuery arma el reporte agregado por insumo y unidad.
func buildConsumosResumenQuery(f repository.FiltroConsumos) (string, []interface{}) {
	where, args := consumosResumenWhere(f)
	query := fmt.Sprintf(`
		SELECT ai.item_id, ai.item_nombre, ai.unidad_medida,
		       COALESCE(SUM(ai.cantidad_usada), 0) AS cantidad_usada,
		       COALESCE(SUM(ai.cantidad_descontada), 0) AS cantidad_descontada,
		       COALESCE(SUM(GREATEST(ai.cantidad_usada - ai.cantidad_descontada, 0)), 0) AS cantidad_pendiente,
		       COUNT(DISTINCT ai.atencion_id) AS atenciones
		FROM atencion_insumos ai
		JOIN atenciones a ON a.id = ai.atencion_id
		JOIN reservas r ON r.id = a.reserva_id
		LEFT JOIN locales l ON l.id = a.local_id
		WHERE %s
		GROUP BY ai.item_id, ai.item_nombre, ai.unidad_medida
		ORDER BY ai.item_nombre ASC, ai.unidad_medida ASC`, where)
	return query, args
}

// buildConsumosEmpleadosQuery arma el reporte agregado por empleado. Usa
// LEFT JOIN para conservar atenciones sin insumos y agrupa por empleado_id;
// cuando el empleado fue eliminado (id NULL), separa por el nombre snapshot.
func buildConsumosEmpleadosQuery(f repository.FiltroConsumos) (string, []interface{}) {
	where, args := consumosResumenWhere(f)
	query := fmt.Sprintf(`
		SELECT a.empleado_id,
		       (array_agg(a.empleado_nombre ORDER BY a.fecha_atencion DESC, a.id DESC))[1] AS empleado_nombre,
		       COUNT(DISTINCT a.id) AS atenciones,
		       COUNT(ai.id) AS insumos_registrados,
		       COUNT(DISTINCT (ai.item_id, ai.item_nombre, ai.unidad_medida)) FILTER (WHERE ai.id IS NOT NULL) AS insumos_distintos,
		       COUNT(ai.id) FILTER (WHERE ai.estado_descuento <> 'DESCONTADO') AS insumos_pendientes,
		       COUNT(DISTINCT a.id) FILTER (WHERE ai.estado_descuento <> 'DESCONTADO') AS atenciones_pendientes,
		       COALESCE(string_agg(DISTINCT l.nombre, ', ' ORDER BY l.nombre), '') AS locales
		FROM atenciones a
		JOIN reservas r ON r.id = a.reserva_id
		LEFT JOIN locales l ON l.id = a.local_id
		LEFT JOIN atencion_insumos ai ON ai.atencion_id = a.id
		WHERE %s
		GROUP BY a.empleado_id, CASE WHEN a.empleado_id IS NULL THEN a.empleado_nombre ELSE '' END
		ORDER BY empleado_nombre ASC`, where)
	return query, args
}

func (r *InventarioRepo) ListarConsumos(f repository.FiltroConsumos) ([]models.ConsumoEvento, error) {
	query, args := buildConsumosEventosQuery(f)
	eventos := []models.ConsumoEvento{}
	if err := r.db.Select(&eventos, query, args...); err != nil {
		return nil, fmt.Errorf("error al listar consumos: %w", err)
	}
	if len(eventos) == 0 {
		return eventos, nil
	}
	ids := make([]int, len(eventos))
	for i := range eventos {
		ids[i] = eventos[i].AtencionID
	}
	type linea struct {
		AtencionID int `db:"atencion_id"`
		models.ConsumoInsumo
	}
	lineas := []linea{}
	if err := r.db.Select(&lineas, `
		SELECT atencion_id, item_id, item_nombre, unidad_medida,
		       cantidad_usada, cantidad_descontada, estado_descuento
		FROM atencion_insumos
		WHERE atencion_id = ANY($1)
		ORDER BY atencion_id, id`, pq.Array(ids)); err != nil {
		return nil, fmt.Errorf("error al listar insumos de los consumos: %w", err)
	}
	porAtencion := make(map[int][]models.ConsumoInsumo, len(eventos))
	for _, fila := range lineas {
		insumo := fila.ConsumoInsumo
		if pendiente := insumo.CantidadUsada - insumo.CantidadDescontada; pendiente > 0 {
			insumo.CantidadPendiente = redondear3(pendiente)
		}
		porAtencion[fila.AtencionID] = append(porAtencion[fila.AtencionID], insumo)
	}
	for i := range eventos {
		eventos[i].Insumos = porAtencion[eventos[i].AtencionID]
		if eventos[i].Insumos == nil {
			eventos[i].Insumos = []models.ConsumoInsumo{}
		}
		eventos[i].InsumosRegistrados = len(eventos[i].Insumos)
		for _, insumo := range eventos[i].Insumos {
			if insumo.EstadoDescuento != "DESCONTADO" {
				eventos[i].InsumosPendientes++
			}
		}
	}
	return eventos, nil
}

func (r *InventarioRepo) ContarConsumos(f repository.FiltroConsumos) (int, error) {
	where, args := consumosEventosWhere(f)
	var total int
	err := r.db.Get(&total, fmt.Sprintf(`
		SELECT COUNT(*)
		FROM atenciones a
		JOIN reservas r ON r.id = a.reserva_id
		LEFT JOIN locales l ON l.id = a.local_id
		WHERE %s`, where), args...)
	if err != nil {
		return 0, fmt.Errorf("error al contar consumos: %w", err)
	}
	return total, nil
}

func (r *InventarioRepo) ResumenConsumos(f repository.FiltroConsumos) ([]models.ConsumoResumen, error) {
	query, args := buildConsumosResumenQuery(f)
	out := []models.ConsumoResumen{}
	if err := r.db.Select(&out, query, args...); err != nil {
		return nil, fmt.Errorf("error al resumir consumos: %w", err)
	}
	for i := range out {
		out[i].CantidadUsada = redondear3(out[i].CantidadUsada)
		out[i].CantidadDescontada = redondear3(out[i].CantidadDescontada)
		out[i].CantidadPendiente = redondear3(out[i].CantidadPendiente)
	}
	return out, nil
}

func (r *InventarioRepo) ResumenConsumosPorEmpleado(f repository.FiltroConsumos) ([]models.ConsumoEmpleadoResumen, error) {
	query, args := buildConsumosEmpleadosQuery(f)
	out := []models.ConsumoEmpleadoResumen{}
	if err := r.db.Select(&out, query, args...); err != nil {
		return nil, fmt.Errorf("error al resumir consumos por empleado: %w", err)
	}
	return out, nil
}
