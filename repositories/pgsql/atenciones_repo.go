package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/pagination"
	repository "atrevida-agenda-api/repositories"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type AtencionesRepo struct{ db *sqlx.DB }

func NewAtencionesRepo(db *sqlx.DB) *AtencionesRepo { return &AtencionesRepo{db: db} }

type RegistrarAtencionInput struct {
	ReservaID                  int
	LocalID                    *int
	EmpleadoID                 *int
	EmpleadoNombre             string
	FechaAtencion              time.Time
	HoraInicio, HoraFin        *string
	Lugar                      string
	Observaciones              *string
	ConfirmarStockInsuficiente bool
	Actor                      repository.ConsumoActor
	Insumos                    []AtencionInsumoInput
	PoliticaSinStock           string
}

type ActualizarAtencionInput struct {
	AtencionID                 int
	LocalID                    *int
	EmpleadoID                 *int
	EmpleadoNombre             string
	FechaAtencion              time.Time
	HoraInicio, HoraFin        *string
	Lugar                      string
	Observaciones              *string
	ConfirmarStockInsuficiente bool
	Actor                      repository.ConsumoActor
	Insumos                    []AtencionInsumoInput
	PoliticaSinStock           string
}

type AtencionInsumoInput struct {
	ItemID   int
	Cantidad float64
}

func normalizarNombresServicio(nombres []string) []string {
	normalizados := make([]string, 0, len(nombres))
	for _, nombre := range nombres {
		normalizados = append(normalizados, strings.ToLower(strings.TrimSpace(nombre)))
	}
	return normalizados
}

func (r *AtencionesRepo) SugerenciasPersonal(q string, localID *int) ([]models.PersonalSugerido, error) {
	args := []any{"%" + strings.TrimSpace(q) + "%"}
	where := "e.activo = TRUE AND CONCAT_WS(' ', e.nombre, e.apellido) ILIKE $1"
	if localID != nil {
		args = append(args, *localID)
		where += " AND (e.local_id IS NULL OR e.local_id = $2)"
	}
	out := []models.PersonalSugerido{}
	err := r.db.Select(&out, `SELECT e.id, CONCAT_WS(' ', e.nombre, e.apellido) AS nombre, e.cargo, e.local_id, l.nombre AS local_nombre
		FROM empleados e LEFT JOIN locales l ON l.id=e.local_id WHERE `+where+` ORDER BY e.apellido, e.nombre LIMIT 20`, args...)
	return out, err
}

func (r *AtencionesRepo) InsumosSugeridos(reservaID int, localID *int) ([]models.ServicioInsumo, error) {
	var nombres []string
	if err := r.db.Select(&nombres, `SELECT nombre FROM (SELECT servicio_nombre AS nombre FROM reservas WHERE id=$1 AND ($2::int IS NULL OR local_id=$2) UNION SELECT servicio_confirmado FROM reservas WHERE id=$1 AND ($2::int IS NULL OR local_id=$2) UNION SELECT servicio_solicitado FROM reservas WHERE id=$1 AND ($2::int IS NULL OR local_id=$2)) s WHERE NULLIF(BTRIM(nombre),'') IS NOT NULL`, reservaID, localID); err != nil {
		return nil, err
	}
	if len(nombres) == 0 {
		return []models.ServicioInsumo{}, nil
	}
	var servicioID int
	if err := r.db.Get(&servicioID, `SELECT id FROM servicios WHERE activo=TRUE AND LOWER(BTRIM(nombre))=ANY($1) ORDER BY id LIMIT 1`, pq.Array(normalizarNombresServicio(nombres))); err != nil {
		return []models.ServicioInsumo{}, nil
	}
	out := []models.ServicioInsumo{}
	err := r.db.Select(&out, `SELECT si.item_id,i.nombre AS item_nombre,i.unidad_medida,si.cantidad_por_sesion,COALESCE(st.cantidad,0) AS disponible FROM servicio_insumos si JOIN inventario_items i ON i.id=si.item_id JOIN reservas r ON r.id=$2 AND ($3::int IS NULL OR r.local_id=$3) LEFT JOIN inventario_stock st ON st.item_id=i.id AND st.local_id=r.local_id WHERE si.servicio_id=$1 ORDER BY i.nombre`, servicioID, reservaID, localID)
	return out, err
}

func (r *AtencionesRepo) Registrar(in RegistrarAtencionInput) (*models.Atencion, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var reserva struct {
		LocalID int    `db:"local_id"`
		Estado  string `db:"estado"`
	}
	err = tx.Get(&reserva, `SELECT local_id, COALESCE(estado,'PENDIENTE') estado FROM reservas WHERE id=$1 AND activo=TRUE FOR UPDATE`, in.ReservaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("reserva no encontrada")
	}
	if err != nil {
		return nil, err
	}
	if in.LocalID != nil && reserva.LocalID != *in.LocalID {
		return nil, errors.New("no tienes acceso a ese local")
	}
	if strings.ToUpper(strings.TrimSpace(reserva.Estado)) != "AGENDADO" {
		return nil, errors.New("solo se puede atender una reserva AGENDADA")
	}
	if in.EmpleadoID != nil {
		var n int
		err = tx.Get(&n, `SELECT COUNT(*) FROM empleados WHERE id=$1 AND activo=TRUE AND (local_id IS NULL OR local_id=$2)`, *in.EmpleadoID, reserva.LocalID)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, errors.New("empleado no disponible para este local")
		}
	}
	var id int
	err = tx.QueryRowx(`INSERT INTO atenciones (reserva_id,empleado_id,empleado_nombre,fecha_atencion,hora_inicio,hora_fin,local_id,lugar,observaciones,registrado_por_id,registrado_por)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, in.ReservaID, in.EmpleadoID, in.EmpleadoNombre, in.FechaAtencion, in.HoraInicio, in.HoraFin, reserva.LocalID, in.Lugar, in.Observaciones, in.Actor.UsuarioID, nullStr(in.Actor.UsuarioNombre)).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("no se pudo registrar la atencion: %w", err)
	}
	type stockPendiente struct {
		itemID     int
		requerido  float64
		pendiente  float64
		disponible float64
	}
	pendientes := make([]stockPendiente, 0)
	stockPrevio := make([]int, 0)
	tipoReversion := repository.MovReversionServicio
	reservaID := in.ReservaID
	atencionID := id
	solicitados := make(map[int]bool, len(in.Insumos))
	for _, line := range in.Insumos {
		solicitados[line.ItemID] = true
	}
	var consumosPrevios []struct {
		ItemID   int     `db:"item_id"`
		Cantidad float64 `db:"cantidad"`
	}
	if err = tx.Select(&consumosPrevios, `SELECT item_id, SUM(CASE WHEN tipo='SALIDA_SERVICIO' THEN cantidad ELSE -cantidad END) AS cantidad FROM inventario_movimientos WHERE reserva_id=$1 AND tipo IN ('SALIDA_SERVICIO','REVERSION_SERVICIO') GROUP BY item_id HAVING SUM(CASE WHEN tipo='SALIDA_SERVICIO' THEN cantidad ELSE -cantidad END) > 0`, in.ReservaID); err != nil {
		return nil, fmt.Errorf("no se pudo revisar los descuentos previos: %w", err)
	}
	tipoRevertido := make([]struct {
		itemID   int
		cantidad float64
	}, 0)
	for _, consumo := range consumosPrevios {
		if solicitados[consumo.ItemID] {
			continue
		}
		if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{ItemID: consumo.ItemID, LocalID: reserva.LocalID, Tipo: tipoReversion, Cantidad: consumo.Cantidad, ReservaID: &reservaID, AtencionID: &atencionID, UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: true, Motivo: fmt.Sprintf("Insumo no utilizado en atencion #%d, reserva #%d", id, in.ReservaID)}); err != nil {
			return nil, fmt.Errorf("fallo_ajuste_inventario: %w", err)
		}
		tipoRevertido = append(tipoRevertido, struct {
			itemID   int
			cantidad float64
		}{itemID: consumo.ItemID, cantidad: consumo.Cantidad})
	}
	for _, line := range in.Insumos {
		var item struct {
			Nombre string `db:"nombre"`
			Unidad string `db:"unidad_medida"`
		}
		err = tx.Get(&item, `SELECT nombre, unidad_medida FROM inventario_items WHERE id=$1 AND activo=TRUE`, line.ItemID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("item de inventario no encontrado")
		}
		if err != nil {
			return nil, err
		}
		if _, err = tx.Exec(`INSERT INTO inventario_stock(item_id,local_id,cantidad) VALUES($1,$2,0) ON CONFLICT(item_id,local_id) DO NOTHING`, line.ItemID, reserva.LocalID); err != nil {
			return nil, fmt.Errorf("no se pudo preparar el stock para la atencion: %w", err)
		}
		var disponible float64
		if err = tx.Get(&disponible, `SELECT cantidad FROM inventario_stock WHERE item_id=$1 AND local_id=$2 FOR UPDATE`, line.ItemID, reserva.LocalID); err != nil {
			return nil, fmt.Errorf("no se pudo consultar el stock para la atencion: %w", err)
		}
		var descontadoPrevio float64
		if err = tx.Get(&descontadoPrevio, `SELECT COALESCE(SUM(CASE WHEN tipo='SALIDA_SERVICIO' THEN cantidad ELSE -cantidad END),0) FROM inventario_movimientos WHERE reserva_id=$1 AND item_id=$2 AND tipo IN ('SALIDA_SERVICIO','REVERSION_SERVICIO')`, in.ReservaID, line.ItemID); err != nil {
			return nil, fmt.Errorf("no se pudo verificar el consumo previo del insumo: %w", err)
		}
		if descontadoPrevio < 0 {
			descontadoPrevio = 0
		}
		if descontadoPrevio > line.Cantidad {
			if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{ItemID: line.ItemID, LocalID: reserva.LocalID, Tipo: tipoReversion, Cantidad: descontadoPrevio - line.Cantidad, ReservaID: &reservaID, AtencionID: &atencionID, UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: true, Motivo: fmt.Sprintf("Ajuste de atencion #%d, reserva #%d", id, in.ReservaID)}); err != nil {
				return nil, fmt.Errorf("fallo_ajuste_inventario: %w", err)
			}
			descontadoPrevio = line.Cantidad
		}
		faltante := line.Cantidad - descontadoPrevio
		if disponible < faltante {
			if strings.EqualFold(strings.TrimSpace(in.PoliticaSinStock), "BLOQUEAR") && !in.ConfirmarStockInsuficiente {
				return nil, fmt.Errorf("%w: stock disponible %v, se requieren %v adicionales del insumo %s", repository.ErrStockInsuficiente, disponible, faltante, item.Nombre)
			}
			if _, err = tx.Exec(`INSERT INTO atencion_insumos(atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento) VALUES($1,$2,$3,$4,$5,$6,'PENDIENTE_STOCK')`, id, line.ItemID, item.Nombre, item.Unidad, line.Cantidad, descontadoPrevio); err != nil {
				return nil, fmt.Errorf("no se pudo registrar el insumo pendiente: %w", err)
			}
			pendientes = append(pendientes, stockPendiente{itemID: line.ItemID, requerido: line.Cantidad, pendiente: faltante, disponible: disponible})
			continue
		}
		if faltante > 0 {
			if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{ItemID: line.ItemID, LocalID: reserva.LocalID, Tipo: repository.MovSalidaServicio, Cantidad: faltante, ReservaID: &reservaID, AtencionID: &atencionID, UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: false, Motivo: fmt.Sprintf("Atencion #%d, reserva #%d", id, in.ReservaID)}); err != nil {
				return nil, fmt.Errorf("fallo_descuento_inventario: %w", err)
			}
		} else {
			stockPrevio = append(stockPrevio, line.ItemID)
		}
		if _, err = tx.Exec(`INSERT INTO atencion_insumos(atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada) VALUES($1,$2,$3,$4,$5,$5)`, id, line.ItemID, item.Nombre, item.Unidad, line.Cantidad); err != nil {
			return nil, err
		}
	}
	if err = MarcarSesionPorReservaTx(tx, in.ReservaID); err != nil {
		return nil, fmt.Errorf("fallo_consumo_sesion: %w", err)
	}
	if _, err = tx.Exec(`UPDATE reservas SET estado='COMPLETADO', actualizado_en=NOW() WHERE id=$1`, in.ReservaID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	for _, pendiente := range pendientes {
		log.Printf("[atenciones] event=stock_pendiente reserva_id=%d atencion_id=%d local_id=%d item_id=%d requerido=%v pendiente=%v disponible=%v", in.ReservaID, id, reserva.LocalID, pendiente.itemID, pendiente.requerido, pendiente.pendiente, pendiente.disponible)
	}
	for _, itemID := range stockPrevio {
		log.Printf("[atenciones] event=stock_previamente_descontado reserva_id=%d atencion_id=%d local_id=%d item_id=%d", in.ReservaID, id, reserva.LocalID, itemID)
	}
	for _, item := range tipoRevertido {
		log.Printf("[atenciones] event=stock_revertido_no_usado reserva_id=%d atencion_id=%d local_id=%d item_id=%d cantidad=%v", in.ReservaID, id, reserva.LocalID, item.itemID, item.cantidad)
	}
	return r.Obtener(id)
}

func (r *AtencionesRepo) Obtener(id int) (*models.Atencion, error) {
	var a models.Atencion
	err := r.db.Get(&a, `SELECT id,reserva_id,empleado_id,empleado_nombre,fecha_atencion,hora_inicio::text hora_inicio,hora_fin::text hora_fin,local_id,lugar,observaciones,registrado_por_id,registrado_por,creado_en,activo,'ATENDIDA' AS resultado FROM atenciones WHERE id=$1 AND activo=TRUE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("atencion no encontrada")
	}
	if err != nil {
		return nil, err
	}
	err = r.db.Select(&a.Insumos, `SELECT id,atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento FROM atencion_insumos WHERE atencion_id=$1 ORDER BY id`, id)
	return &a, err
}

const servicioReservaExpr = "COALESCE(NULLIF(r.servicio_confirmado,''), NULLIF(r.servicio_solicitado,''), NULLIF(r.servicio_nombre,''), '')"

// acentosSQL permite comparar texto sin distinguir mayusculas, minusculas ni
// acentos sin depender de la extension unaccent.
const acentosSQL = `U&'\00E1\00E9\00ED\00F3\00FA\00FC\00F1'`

// AtencionesOrdenables lista las columnas por las que se puede ordenar el
// historial (ordenar_por). La clave es el nombre del campo en la respuesta.
var AtencionesOrdenables = pagination.Columns{
	"fecha_atencion":     "h.fecha_atencion",
	"hora_inicio":        "h.hora_inicio",
	"cliente_nombre":     textoOrden("h.cliente_nombre"),
	"servicio_reserva":   textoOrden("h.servicio_reserva"),
	"empleado_nombre":    textoOrden("h.empleado_nombre"),
	"local_nombre":       textoOrden("h.local_nombre"),
	"reserva_id":         "h.reserva_id",
	"insumos_pendientes": "h.insumos_pendientes",
	"resultado":          "h.resultado",
}

var normalizarBusquedaAtencion = strings.NewReplacer(
	"\u00e1", "a", "\u00e9", "e", "\u00ed", "i", "\u00f3", "o", "\u00fa", "u", "\u00fc", "u", "\u00f1", "n",
	"\u00c1", "a", "\u00c9", "e", "\u00cd", "i", "\u00d3", "o", "\u00da", "u", "\u00dc", "u", "\u00d1", "n",
)

func textoAtencionNormalizado(v string) string {
	return normalizarBusquedaAtencion.Replace(strings.ToLower(strings.TrimSpace(v)))
}

func textoBusquedaAtencion(columna string) string {
	return fmt.Sprintf("TRANSLATE(LOWER(COALESCE(%s, '')), %s, 'aeiouun')", columna, acentosSQL)
}

func atencionesOrderClause(f repository.FiltroAtenciones) string {
	return pagination.Sort{Key: f.OrdenarPor, Desc: f.OrdenDesc}.OrderBy(AtencionesOrdenables, "h.fecha_atencion DESC, h.id DESC", "h.id")
}

// historialAtencionesCTE unifica atenciones activas y reservas canceladas. Una
// cancelación usa un id negativo para no colisionar con el id de una atención y
// para poder conservar el cursor homogéneo sin inventar una atención vacía.
const historialAtencionesCTE = `
WITH historial AS (
    SELECT a.id, a.id AS atencion_id, a.reserva_id,
           COALESCE(NULLIF(r.servicio_confirmado,''), NULLIF(r.servicio_solicitado,''), NULLIF(r.servicio_nombre,''), '') AS servicio_reserva,
           COALESCE(r.cliente, '') AS cliente_nombre,
           a.empleado_id, a.empleado_nombre, a.fecha_atencion,
           TO_CHAR(a.hora_inicio, 'HH24:MI') AS hora_inicio, TO_CHAR(a.hora_fin, 'HH24:MI') AS hora_fin,
           a.local_id, COALESCE(l.nombre, r.local_nombre, '') AS local_nombre,
           a.lugar, a.observaciones, a.registrado_por_id, a.registrado_por, a.creado_en, a.activo,
           'ATENDIDA'::text AS resultado, NULL::text AS motivo_cancelacion_tipo, NULL::text AS motivo_cancelacion_detalle,
           (SELECT COUNT(*) FROM atencion_insumos ai WHERE ai.atencion_id = a.id) AS insumos_registrados,
           (SELECT string_agg(ai.item_nombre, ', ' ORDER BY ai.item_nombre) FROM atencion_insumos ai WHERE ai.atencion_id = a.id) AS insumos_resumen,
           (SELECT COUNT(*) FROM atencion_insumos ai WHERE ai.atencion_id = a.id AND ai.estado_descuento = 'PENDIENTE_STOCK') AS insumos_pendientes,
           (SELECT string_agg(ai.item_nombre, ', ' ORDER BY ai.item_nombre) FROM atencion_insumos ai WHERE ai.atencion_id = a.id AND ai.estado_descuento = 'PENDIENTE_STOCK') AS insumos_pendientes_detalle
    FROM atenciones a
    JOIN reservas r ON r.id = a.reserva_id
    LEFT JOIN locales l ON l.id = a.local_id
    WHERE a.activo = TRUE
    UNION ALL
    SELECT -r.id AS id, NULL::int AS atencion_id, r.id AS reserva_id,
           COALESCE(NULLIF(r.servicio_confirmado,''), NULLIF(r.servicio_solicitado,''), NULLIF(r.servicio_nombre,''), '') AS servicio_reserva,
           COALESCE(r.cliente, '') AS cliente_nombre,
           NULL::int AS empleado_id, ''::text AS empleado_nombre, r.fecha AS fecha_atencion,
           TO_CHAR(r.hora_desde, 'HH24:MI') AS hora_inicio, TO_CHAR(r.hora_hasta, 'HH24:MI') AS hora_fin,
           r.local_id, COALESCE(l.nombre, r.local_nombre, '') AS local_nombre,
           ''::text AS lugar, NULL::text AS observaciones, r.cancelado_por_id AS registrado_por_id, r.cancelado_por AS registrado_por,
           r.cancelado_en AS creado_en, TRUE AS activo,
           'CANCELADA'::text AS resultado, r.motivo_cancelacion_tipo, r.motivo_cancelacion_detalle,
           0 AS insumos_registrados, NULL::text AS insumos_resumen, 0 AS insumos_pendientes, NULL::text AS insumos_pendientes_detalle
    FROM reservas r
    LEFT JOIN locales l ON l.id = r.local_id
    WHERE r.activo = TRUE AND UPPER(BTRIM(COALESCE(r.estado, ''))) = 'CANCELADO'
)`

// atencionesWhere arma el WHERE de la proyección histórica; con cursor excluye
// lo ya visto. Se aplica después de unificar resultados para que conteo y lista
// utilicen exactamente los mismos filtros.
func atencionesWhere(f repository.FiltroAtenciones, withCursor bool) (string, []interface{}) {
	conds := []string{"TRUE"}
	args := []interface{}{}
	add := func(cond string, v interface{}) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.LocalID != nil {
		add("h.local_id = $%d", *f.LocalID)
	}
	if f.FechaDesde != nil {
		add("h.fecha_atencion >= $%d", *f.FechaDesde)
	}
	if f.FechaHasta != nil {
		add("h.fecha_atencion <= $%d", *f.FechaHasta)
	}
	if f.EmpleadoID != nil {
		add("h.empleado_id = $%d", *f.EmpleadoID)
	}
	if f.ItemID != nil {
		add("EXISTS (SELECT 1 FROM atencion_insumos ai WHERE ai.atencion_id = h.atencion_id AND ai.item_id = $%d)", *f.ItemID)
	}
	if f.PendientesStock != nil {
		if *f.PendientesStock {
			conds = append(conds, "EXISTS (SELECT 1 FROM atencion_insumos ai WHERE ai.atencion_id = h.atencion_id AND ai.estado_descuento = 'PENDIENTE_STOCK')")
		} else {
			conds = append(conds, "NOT EXISTS (SELECT 1 FROM atencion_insumos ai WHERE ai.atencion_id = h.atencion_id AND ai.estado_descuento = 'PENDIENTE_STOCK')")
		}
	}
	switch f.Resultado {
	case "ATENDIDA", "CANCELADA":
		add("h.resultado = $%d", f.Resultado)
	case "INASISTENCIA":
		conds = append(conds, "h.resultado = 'CANCELADA' AND h.motivo_cancelacion_tipo = 'INASISTENCIA'")
	}
	if v := textoAtencionNormalizado(f.Cliente); v != "" {
		add(textoBusquedaAtencion("h.cliente_nombre")+" LIKE $%d", "%"+v+"%")
	}
	if v := textoAtencionNormalizado(f.Servicio); v != "" {
		add(textoBusquedaAtencion("h.servicio_reserva")+" LIKE $%d", "%"+v+"%")
	}
	if v := textoAtencionNormalizado(f.Empleado); v != "" {
		add(textoBusquedaAtencion("h.empleado_nombre")+" LIKE $%d", "%"+v+"%")
	}
	if q := textoAtencionNormalizado(f.Q); q != "" {
		args = append(args, "%"+q+"%")
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			"(%s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR CAST(h.reserva_id AS TEXT) LIKE $%d)",
			textoBusquedaAtencion("h.cliente_nombre"), n,
			textoBusquedaAtencion("h.servicio_reserva"), n,
			textoBusquedaAtencion("h.empleado_nombre"), n,
			textoBusquedaAtencion("h.observaciones"), n,
			textoBusquedaAtencion("h.lugar"), n,
			n,
		))
	}
	if withCursor && f.CursorSet {
		args = append(args, f.CursorFecha, f.CursorID)
		conds = append(conds, fmt.Sprintf("(h.fecha_atencion, h.id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	return strings.Join(conds, " AND "), args
}

// buildAtencionesQuery arma el listado del historial. Sin orden pedido pagina
// por valor (fecha_atencion, id); con orden pedido pagina por desplazamiento.
func buildAtencionesQuery(f repository.FiltroAtenciones) (string, []interface{}) {
	ordenado := f.OrdenarPor != ""
	where, args := atencionesWhere(f, !ordenado)
	limit := f.PageLimit
	if limit <= 0 {
		limit = f.Limit
		if limit <= 0 || limit > 200 {
			limit = 50
		}
	}
	args = append(args, limit)
	pageClause := fmt.Sprintf("LIMIT $%d", len(args))
	if ordenado && f.PageLimit > 0 && f.Offset > 0 {
		args = append(args, f.Offset)
		pageClause += fmt.Sprintf(" OFFSET $%d", len(args))
	}
	query := fmt.Sprintf(`%s
		SELECT id, reserva_id, servicio_reserva, cliente_nombre, empleado_id, empleado_nombre, fecha_atencion,
		       hora_inicio, hora_fin, local_id, local_nombre, lugar, observaciones, registrado_por_id, registrado_por,
		       creado_en, activo, resultado, motivo_cancelacion_tipo, motivo_cancelacion_detalle, insumos_registrados,
		       insumos_resumen, insumos_pendientes, insumos_pendientes_detalle
		FROM historial h WHERE %s ORDER BY %s %s`, historialAtencionesCTE, where, atencionesOrderClause(f), pageClause)
	return query, args
}

func (r *AtencionesRepo) Listar(f repository.FiltroAtenciones) ([]models.Atencion, error) {
	query, args := buildAtencionesQuery(f)
	out := []models.Atencion{}
	if err := r.db.Select(&out, query, args...); err != nil {
		return nil, fmt.Errorf("error al listar atenciones: %w", err)
	}
	return out, nil
}

// Contar cuenta el historial con los mismos filtros (sin cursor).
func (r *AtencionesRepo) Contar(f repository.FiltroAtenciones) (int, error) {
	where, args := atencionesWhere(f, false)
	var total int
	err := r.db.Get(&total, fmt.Sprintf(`%s SELECT COUNT(*) FROM historial h WHERE %s`, historialAtencionesCTE, where), args...)
	if err != nil {
		return 0, fmt.Errorf("error al contar atenciones: %w", err)
	}
	return total, err
}

// jornadaWhere concentra los filtros del listado de Jornada y de su contador.
// El estado se filtra en SQL para que la paginación no dependa de descargar el
// día completo. La derivación usa la misma fecha/hora que el service.
func jornadaWhere(f repository.FiltroJornada) (string, []interface{}) {
	conds := []string{
		"r.activo = TRUE",
		"r.fecha = $1",
		"(UPPER(BTRIM(COALESCE(r.estado, ''))) IN ('AGENDADO', 'CANCELADO') OR a.id IS NOT NULL)",
	}
	args := []interface{}{f.Fecha}
	if f.LocalID != nil {
		args = append(args, *f.LocalID)
		conds = append(conds, fmt.Sprintf("r.local_id = $%d", len(args)))
	}
	if q := textoAtencionNormalizado(f.Q); q != "" {
		args = append(args, "%"+q+"%")
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			"(%s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR CAST(r.id AS TEXT) LIKE $%d)",
			textoBusquedaAtencion("r.cliente"), n,
			textoBusquedaAtencion(servicioReservaExpr), n,
			textoBusquedaAtencion("a.empleado_nombre"), n,
			textoBusquedaAtencion("r.local_nombre"), n,
			n,
		))
	}
	switch f.Estado {
	case "ATENDIDA":
		conds = append(conds, "a.id IS NOT NULL")
	case "CANCELADA":
		conds = append(conds, "a.id IS NULL", "UPPER(BTRIM(COALESCE(r.estado, ''))) = 'CANCELADO'")
	case "PROGRAMADA", "PENDIENTE_CIERRE":
		conds = append(conds, "a.id IS NULL", "UPPER(BTRIM(COALESCE(r.estado, ''))) = 'AGENDADO'")
		// Las fechas de filtros se parsean en UTC, pero representan una fecha de
		// calendario local. Se reconstruyen en la zona de Jornada antes de
		// compararlas para no convertir el día actual en "pasado" por el huso.
		fechaAhora := time.Date(f.Ahora.Year(), f.Ahora.Month(), f.Ahora.Day(), 0, 0, 0, 0, f.Ahora.Location())
		fechaConsulta := time.Date(f.Fecha.Year(), f.Fecha.Month(), f.Fecha.Day(), 0, 0, 0, 0, f.Ahora.Location())
		if f.Estado == "PROGRAMADA" {
			switch {
			case fechaConsulta.After(fechaAhora):
				// Toda la jornada consultada sigue por delante.
			case fechaConsulta.Before(fechaAhora):
				conds = append(conds, "FALSE")
			default:
				args = append(args, f.Ahora.Format("15:04"))
				conds = append(conds, fmt.Sprintf("r.hora_hasta > $%d::time", len(args)))
			}
		} else {
			switch {
			case fechaConsulta.Before(fechaAhora):
				// Toda la jornada consultada ya venció.
			case fechaConsulta.After(fechaAhora):
				conds = append(conds, "FALSE")
			default:
				args = append(args, f.Ahora.Format("15:04"))
				conds = append(conds, fmt.Sprintf("r.hora_hasta <= $%d::time", len(args)))
			}
		}
	}
	return strings.Join(conds, " AND "), args
}

// Jornada devuelve las reservas AGENDADAS o CANCELADAS del dia y las que ya
// tienen una atención activa, con paginación estable por hora y reserva.
func (r *AtencionesRepo) Jornada(f repository.FiltroJornada) ([]repository.JornadaRow, error) {
	where, args := jornadaWhere(f)
	pageClause := ""
	if f.PageLimit != -1 {
		limit := f.PageLimit
		if limit <= 0 {
			limit = 500
		}
		args = append(args, limit)
		pageClause = fmt.Sprintf("LIMIT $%d", len(args))
		if f.PageLimit > 0 && f.Offset > 0 {
			args = append(args, f.Offset)
			pageClause += fmt.Sprintf(" OFFSET $%d", len(args))
		}
	}
	query := fmt.Sprintf(`
		SELECT r.id AS reserva_id, r.fecha,
		       TO_CHAR(r.hora_desde, 'HH24:MI') AS hora_desde,
		       TO_CHAR(r.hora_hasta, 'HH24:MI') AS hora_hasta,
		       COALESCE(r.cliente, '') AS cliente,
		       %s AS servicio,
		       r.local_id, COALESCE(r.local_nombre, '') AS local_nombre,
		       COALESCE(NULLIF(BTRIM(r.estado), ''), 'PENDIENTE') AS estado_reserva,
		       r.motivo_cancelacion_tipo,
		       a.id AS atencion_id, a.empleado_nombre,
		       (SELECT COUNT(*) FROM atencion_insumos ai WHERE ai.atencion_id = a.id) AS insumos_registrados,
		       (SELECT string_agg(ai.item_nombre, ', ' ORDER BY ai.item_nombre) FROM atencion_insumos ai WHERE ai.atencion_id = a.id) AS insumos_resumen,
		       (SELECT COUNT(*) FROM atencion_insumos ai WHERE ai.atencion_id = a.id AND ai.estado_descuento = 'PENDIENTE_STOCK') AS insumos_pendientes
		FROM reservas r
		LEFT JOIN atenciones a ON a.reserva_id = r.id AND a.activo = TRUE
		WHERE %s
		ORDER BY r.hora_desde ASC, r.id ASC
		%s`, servicioReservaExpr, where, pageClause)
	out := []repository.JornadaRow{}
	if err := r.db.Select(&out, query, args...); err != nil {
		return nil, fmt.Errorf("error al listar la jornada: %w", err)
	}
	return out, nil
}

// ContarJornada cuenta las mismas filas que Jornada sin el cursor de página.
func (r *AtencionesRepo) ContarJornada(f repository.FiltroJornada) (int, error) {
	where, args := jornadaWhere(f)
	var total int
	err := r.db.Get(&total, fmt.Sprintf(`
		SELECT COUNT(*)
		FROM reservas r
		LEFT JOIN atenciones a ON a.reserva_id = r.id AND a.activo = TRUE
		WHERE %s`, where), args...)
	if err != nil {
		return 0, fmt.Errorf("error al contar la jornada: %w", err)
	}
	return total, nil
}

func calendarioPersonalWhere(f repository.FiltroCalendarioPersonal) (string, []interface{}) {
	conds := []string{"a.activo = TRUE"}
	args := []interface{}{}
	add := func(cond string, value interface{}) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	add("a.fecha_atencion >= $%d", f.FechaDesde)
	add("a.fecha_atencion <= $%d", f.FechaHasta)
	if f.LocalID != nil {
		add("a.local_id = $%d", *f.LocalID)
	}
	if q := textoAtencionNormalizado(f.Cliente); q != "" {
		add(textoBusquedaAtencion("r.cliente")+" LIKE $%d", "%"+q+"%")
	}
	if q := textoAtencionNormalizado(f.Servicio); q != "" {
		add(textoBusquedaAtencion(servicioReservaExpr)+" LIKE $%d", "%"+q+"%")
	}
	if q := textoAtencionNormalizado(f.Empleado); q != "" {
		add(textoBusquedaAtencion("a.empleado_nombre")+" LIKE $%d", "%"+q+"%")
	}
	if q := textoAtencionNormalizado(f.Q); q != "" {
		args = append(args, "%"+q+"%")
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			"(%s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR CAST(a.reserva_id AS TEXT) LIKE $%d)",
			textoBusquedaAtencion("r.cliente"), n,
			textoBusquedaAtencion(servicioReservaExpr), n,
			textoBusquedaAtencion("a.empleado_nombre"), n,
			textoBusquedaAtencion("l.nombre"), n,
			n,
		))
	}
	return strings.Join(conds, " AND "), args
}

// CalendarioPersonal lista atenciones activas para el calendario semanal que
// prioriza quién atendió. No incluye líneas de insumo como contenido principal.
func (r *AtencionesRepo) CalendarioPersonal(f repository.FiltroCalendarioPersonal) ([]models.CalendarioPersonalEvento, error) {
	where, args := calendarioPersonalWhere(f)
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
		       TO_CHAR(a.hora_inicio, 'HH24:MI') AS hora_inicio,
		       TO_CHAR(a.hora_fin, 'HH24:MI') AS hora_fin,
		       COALESCE(r.cliente, '') AS cliente,
		       %s AS servicio,
		       a.local_id, COALESCE(l.nombre, '') AS local_nombre,
		       a.empleado_id, COALESCE(a.empleado_nombre, '') AS empleado_nombre,
		       COALESCE(a.lugar, '') AS lugar,
		       (SELECT COUNT(*) FROM atencion_insumos ai WHERE ai.atencion_id = a.id AND ai.estado_descuento <> 'DESCONTADO') AS insumos_pendientes
		FROM atenciones a
		JOIN reservas r ON r.id = a.reserva_id
		LEFT JOIN locales l ON l.id = a.local_id
		WHERE %s
		ORDER BY a.fecha_atencion ASC, a.hora_inicio ASC NULLS LAST, a.id ASC
		%s`, servicioReservaExpr, where, pageClause)
	out := []models.CalendarioPersonalEvento{}
	if err := r.db.Select(&out, query, args...); err != nil {
		return nil, fmt.Errorf("error al listar calendario de personal: %w", err)
	}
	return out, nil
}

func (r *AtencionesRepo) ContarCalendarioPersonal(f repository.FiltroCalendarioPersonal) (int, error) {
	where, args := calendarioPersonalWhere(f)
	var total int
	err := r.db.Get(&total, fmt.Sprintf(`
		SELECT COUNT(*) FROM atenciones a
		JOIN reservas r ON r.id = a.reserva_id
		LEFT JOIN locales l ON l.id = a.local_id
		WHERE %s`, where), args...)
	if err != nil {
		return 0, fmt.Errorf("error al contar calendario de personal: %w", err)
	}
	return total, nil
}

// seguimientoQuery concentra los hechos pendientes de la operación. El CTE
// evita mezclar la semántica de Historial (todos los hechos) con la bandeja de
// trabajo (solo acciones por resolver).
func seguimientoQuery(f repository.FiltroSeguimiento, count bool) (string, []any) {
	args := []any{f.Ahora.Format("2006-01-02"), f.Ahora.Format("15:04")}
	add := func(value any) string { args = append(args, value); return fmt.Sprintf("$%d", len(args)) }
	conds := []string{"fecha >= " + add(f.FechaDesde), "fecha <= " + add(f.FechaHasta)}
	if f.LocalID != nil {
		conds = append(conds, "local_id = "+add(*f.LocalID))
	}
	if tipo := strings.ToUpper(strings.TrimSpace(f.Tipo)); tipo != "" {
		conds = append(conds, "tipo = "+add(tipo))
	}
	if q := textoAtencionNormalizado(f.Q); q != "" {
		p := add("%" + q + "%")
		conds = append(conds, fmt.Sprintf("(%s LIKE %s OR %s LIKE %s OR %s LIKE %s OR %s LIKE %s OR CAST(reserva_id AS TEXT) LIKE %s)", textoBusquedaAtencion("cliente"), p, textoBusquedaAtencion("servicio"), p, textoBusquedaAtencion("empleado_nombre"), p, textoBusquedaAtencion("local_nombre"), p, p))
	}
	selectColumns := "tipo, atencion_id, reserva_id, TO_CHAR(fecha, 'YYYY-MM-DD') AS fecha, hora_inicio, hora_fin, cliente, servicio, local_id, local_nombre, empleado_nombre, motivo, insumos_pendientes, detalle_insumos_pendientes, puede_atender, puede_editar_atencion"
	if count {
		selectColumns = "COUNT(*)"
	}
	query := fmt.Sprintf(`WITH casos AS (
		SELECT 'DEDUCCION_PENDIENTE'::text AS tipo, a.id AS atencion_id, a.reserva_id, a.fecha_atencion AS fecha,
		       TO_CHAR(a.hora_inicio,'HH24:MI') AS hora_inicio, TO_CHAR(a.hora_fin,'HH24:MI') AS hora_fin, COALESCE(r.cliente,'') AS cliente, %s AS servicio,
		       a.local_id, COALESCE(l.nombre,r.local_nombre,'') AS local_nombre, a.empleado_nombre,
		       NULL::text AS motivo, COUNT(ai.id)::int AS insumos_pendientes,
		       string_agg(CONCAT(ai.item_nombre, ': ', ai.cantidad_usada - ai.cantidad_descontada, ' ', ai.unidad_medida), ', ' ORDER BY ai.item_nombre) AS detalle_insumos_pendientes,
		       FALSE AS puede_atender, TRUE AS puede_editar_atencion
		FROM atenciones a JOIN reservas r ON r.id=a.reserva_id LEFT JOIN locales l ON l.id=a.local_id
		JOIN atencion_insumos ai ON ai.atencion_id=a.id AND ai.estado_descuento='PENDIENTE_STOCK'
		WHERE a.activo=TRUE GROUP BY a.id,r.id,l.id
		UNION ALL
		SELECT CASE WHEN r.motivo_cancelacion_tipo='INASISTENCIA' THEN 'INASISTENCIA' ELSE 'CANCELACION' END,
		       NULL::int, r.id, r.fecha, TO_CHAR(r.hora_desde,'HH24:MI'), TO_CHAR(r.hora_hasta,'HH24:MI'), COALESCE(r.cliente,''), %s,
		       r.local_id, COALESCE(l.nombre,r.local_nombre,''), NULL::text,
		       NULLIF(CONCAT_WS(' · ',r.motivo_cancelacion_tipo,r.motivo_cancelacion_detalle),'') AS motivo,
		       0, NULL::text, FALSE, FALSE
		FROM reservas r LEFT JOIN locales l ON l.id=r.local_id
		WHERE r.activo=TRUE AND r.estado='CANCELADO'
		UNION ALL
		SELECT 'PENDIENTE_CIERRE', NULL::int, r.id, r.fecha, TO_CHAR(r.hora_desde,'HH24:MI'), TO_CHAR(r.hora_hasta,'HH24:MI'), COALESCE(r.cliente,''), %s,
		       r.local_id, COALESCE(l.nombre,r.local_nombre,''), NULL::text, NULL::text,
		       0, NULL::text, TRUE, FALSE
		FROM reservas r LEFT JOIN locales l ON l.id=r.local_id
		LEFT JOIN atenciones a ON a.reserva_id=r.id AND a.activo=TRUE
		WHERE r.activo=TRUE AND r.estado='AGENDADO' AND a.id IS NULL
		  AND (r.fecha < $1 OR (r.fecha=$1 AND r.hora_hasta <= $2::time))
	)
	SELECT %s FROM casos WHERE %s`, servicioReservaExpr, servicioReservaExpr, servicioReservaExpr, selectColumns, strings.Join(conds, " AND "))
	if !count {
		query += " ORDER BY fecha DESC, hora_inicio DESC NULLS LAST, reserva_id DESC"
		// -1 se usa internamente para el resumen completo; no debe heredar el
		// límite de seguridad de la lista paginada.
		if f.PageLimit != -1 {
			limit := f.PageLimit
			if limit <= 0 {
				limit = 500
			}
			args = append(args, limit)
			query += fmt.Sprintf(" LIMIT $%d", len(args))
			if f.PageLimit > 0 && f.Offset > 0 {
				args = append(args, f.Offset)
				query += fmt.Sprintf(" OFFSET $%d", len(args))
			}
		}
	}
	return query, args
}

func (r *AtencionesRepo) Seguimiento(f repository.FiltroSeguimiento) ([]models.SeguimientoItem, error) {
	query, args := seguimientoQuery(f, false)
	out := []models.SeguimientoItem{}
	if err := r.db.Select(&out, query, args...); err != nil {
		return nil, fmt.Errorf("error al listar seguimiento: %w", err)
	}
	return out, nil
}

func (r *AtencionesRepo) ContarSeguimiento(f repository.FiltroSeguimiento) (int, error) {
	query, args := seguimientoQuery(f, true)
	var total int
	if err := r.db.Get(&total, query, args...); err != nil {
		return 0, fmt.Errorf("error al contar seguimiento: %w", err)
	}
	return total, nil
}

func (r *AtencionesRepo) Desactivar(id int, localID *int, restituirStock bool, actor repository.ConsumoActor) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var atencion struct {
		ReservaID int `db:"reserva_id"`
		LocalID   int `db:"local_id"`
	}
	if err = tx.Get(&atencion, `SELECT reserva_id,local_id FROM atenciones WHERE id=$1 AND activo=TRUE FOR UPDATE`, id); errors.Is(err, sql.ErrNoRows) {
		return errors.New("atencion no encontrada")
	} else if err != nil {
		return err
	}
	if localID != nil && atencion.LocalID != *localID {
		return errors.New("no tienes acceso a ese local")
	}
	if restituirStock {
		var lineas []struct {
			ItemID   int     `db:"item_id"`
			Cantidad float64 `db:"cantidad_descontada"`
		}
		if err = tx.Select(&lineas, `SELECT item_id,cantidad_descontada FROM atencion_insumos WHERE atencion_id=$1 AND item_id IS NOT NULL AND cantidad_descontada>0`, id); err != nil {
			return fmt.Errorf("no se pudieron leer los descuentos de la atencion: %w", err)
		}
		reservaID, atencionID := atencion.ReservaID, id
		for _, linea := range lineas {
			if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{
				ItemID: linea.ItemID, LocalID: atencion.LocalID, Tipo: repository.MovReversionServicio,
				Cantidad: linea.Cantidad, ReservaID: &reservaID, AtencionID: &atencionID,
				UsuarioID: actor.UsuarioID, UsuarioNombre: actor.UsuarioNombre, PermitirNegativo: true,
				Motivo: fmt.Sprintf("Restitucion al desactivar atencion #%d", id),
			}); err != nil {
				return fmt.Errorf("no se pudo restituir el stock: %w", err)
			}
		}
	}
	if _, err = tx.Exec(`UPDATE atenciones SET activo=FALSE WHERE id=$1`, id); err != nil {
		return fmt.Errorf("no se pudo desactivar la atencion: %w", err)
	}
	return tx.Commit()
}

func (r *AtencionesRepo) Actualizar(in ActualizarAtencionInput) (*models.Atencion, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var atencion struct {
		ReservaID int `db:"reserva_id"`
		LocalID   int `db:"local_id"`
	}
	if err = tx.Get(&atencion, `SELECT reserva_id,local_id FROM atenciones WHERE id=$1 AND activo=TRUE FOR UPDATE`, in.AtencionID); errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("atencion no encontrada")
	} else if err != nil {
		return nil, err
	}
	if in.LocalID != nil && atencion.LocalID != *in.LocalID {
		return nil, errors.New("no tienes acceso a ese local")
	}
	var pendientes int
	if err = tx.Get(&pendientes, `SELECT COUNT(*) FROM atencion_insumos WHERE atencion_id=$1 AND estado_descuento='PENDIENTE_STOCK'`, in.AtencionID); err != nil {
		return nil, err
	}
	if pendientes == 0 {
		return nil, errors.New("solo se pueden editar atenciones con deducciones pendientes")
	}
	if in.EmpleadoID != nil {
		var existe int
		if err = tx.Get(&existe, `SELECT COUNT(*) FROM empleados WHERE id=$1 AND activo=TRUE AND (local_id IS NULL OR local_id=$2)`, *in.EmpleadoID, atencion.LocalID); err != nil {
			return nil, err
		}
		if existe == 0 {
			return nil, errors.New("empleado no disponible para este local")
		}
	}
	type lineaActual struct {
		ItemID             sql.NullInt64 `db:"item_id"`
		ItemNombre         string        `db:"item_nombre"`
		UnidadMedida       string        `db:"unidad_medida"`
		CantidadUsada      float64       `db:"cantidad_usada"`
		CantidadDescontada float64       `db:"cantidad_descontada"`
		EstadoDescuento    string        `db:"estado_descuento"`
	}
	var anteriores []lineaActual
	if err = tx.Select(&anteriores, `SELECT item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento FROM atencion_insumos WHERE atencion_id=$1`, in.AtencionID); err != nil {
		return nil, err
	}
	porItem := make(map[int]float64, len(anteriores))
	for _, anterior := range anteriores {
		if anterior.ItemID.Valid {
			porItem[int(anterior.ItemID.Int64)] = anterior.CantidadDescontada
		}
	}
	solicitados := make(map[int]bool, len(in.Insumos))
	for _, linea := range in.Insumos {
		solicitados[linea.ItemID] = true
	}
	reservaID, atencionID := atencion.ReservaID, in.AtencionID
	for itemID, cantidadDescontada := range porItem {
		if solicitados[itemID] || cantidadDescontada <= 0 {
			continue
		}
		if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: itemID, LocalID: atencion.LocalID, Tipo: repository.MovReversionServicio,
			Cantidad: cantidadDescontada, ReservaID: &reservaID, AtencionID: &atencionID,
			UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: true,
			Motivo: fmt.Sprintf("Insumo quitado al editar atencion #%d", in.AtencionID),
		}); err != nil {
			return nil, fmt.Errorf("no se pudo restituir el insumo quitado: %w", err)
		}
	}
	if _, err = tx.Exec(`UPDATE atenciones SET empleado_id=$1,empleado_nombre=$2,fecha_atencion=$3,hora_inicio=$4,hora_fin=$5,lugar=$6,observaciones=$7 WHERE id=$8`, in.EmpleadoID, in.EmpleadoNombre, in.FechaAtencion, in.HoraInicio, in.HoraFin, in.Lugar, in.Observaciones, in.AtencionID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`DELETE FROM atencion_insumos WHERE atencion_id=$1`, in.AtencionID); err != nil {
		return nil, err
	}
	for _, anterior := range anteriores {
		if anterior.ItemID.Valid {
			continue
		}
		if _, err = tx.Exec(`INSERT INTO atencion_insumos(atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento) VALUES($1,NULL,$2,$3,$4,$5,$6)`, in.AtencionID, anterior.ItemNombre, anterior.UnidadMedida, anterior.CantidadUsada, anterior.CantidadDescontada, anterior.EstadoDescuento); err != nil {
			return nil, err
		}
	}
	for _, linea := range in.Insumos {
		var item struct {
			Nombre string `db:"nombre"`
			Unidad string `db:"unidad_medida"`
		}
		if err = tx.Get(&item, `SELECT nombre,unidad_medida FROM inventario_items WHERE id=$1 AND activo=TRUE`, linea.ItemID); errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("item de inventario no encontrado")
		} else if err != nil {
			return nil, err
		}
		anterior := porItem[linea.ItemID]
		descontado := anterior
		if anterior > linea.Cantidad {
			restituir := anterior - linea.Cantidad
			if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{
				ItemID: linea.ItemID, LocalID: atencion.LocalID, Tipo: repository.MovReversionServicio,
				Cantidad: restituir, ReservaID: &reservaID, AtencionID: &atencionID,
				UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: true,
				Motivo: fmt.Sprintf("Ajuste de insumo al editar atencion #%d", in.AtencionID),
			}); err != nil {
				return nil, fmt.Errorf("no se pudo ajustar el stock del insumo: %w", err)
			}
			descontado = linea.Cantidad
		} else if anterior < linea.Cantidad {
			if _, err = tx.Exec(`INSERT INTO inventario_stock(item_id,local_id,cantidad) VALUES($1,$2,0) ON CONFLICT(item_id,local_id) DO NOTHING`, linea.ItemID, atencion.LocalID); err != nil {
				return nil, err
			}
			var disponible float64
			if err = tx.Get(&disponible, `SELECT cantidad FROM inventario_stock WHERE item_id=$1 AND local_id=$2 FOR UPDATE`, linea.ItemID, atencion.LocalID); err != nil {
				return nil, err
			}
			adicional := linea.Cantidad - anterior
			if disponible < adicional {
				if strings.EqualFold(strings.TrimSpace(in.PoliticaSinStock), "BLOQUEAR") && !in.ConfirmarStockInsuficiente {
					return nil, fmt.Errorf("%w: stock disponible %v, se requieren %v adicionales del insumo %s", repository.ErrStockInsuficiente, disponible, adicional, item.Nombre)
				}
			} else {
				if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{
					ItemID: linea.ItemID, LocalID: atencion.LocalID, Tipo: repository.MovSalidaServicio,
					Cantidad: adicional, ReservaID: &reservaID, AtencionID: &atencionID,
					UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre,
					Motivo: fmt.Sprintf("Ajuste de consumo al editar atencion #%d", in.AtencionID),
				}); err != nil {
					return nil, err
				}
				descontado = linea.Cantidad
			}
		}
		estado := "DESCONTADO"
		if descontado < linea.Cantidad {
			estado = "PENDIENTE_STOCK"
		}
		if _, err = tx.Exec(`INSERT INTO atencion_insumos(atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento) VALUES($1,$2,$3,$4,$5,$6,$7)`, in.AtencionID, linea.ItemID, item.Nombre, item.Unidad, linea.Cantidad, descontado, estado); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Obtener(in.AtencionID)
}
