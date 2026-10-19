package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"

	"github.com/jmoiron/sqlx"
)

// Orden de resolucion de servicio por nombre: primero los que tienen insumos
// definidos (los nombres pueden repetirse con distinto precio), luego el id menor.
const resolverServicioSQL = `
	SELECT s.id FROM servicios s
	WHERE s.activo = TRUE AND LOWER(BTRIM(s.nombre)) = LOWER(BTRIM($1))
	ORDER BY EXISTS (SELECT 1 FROM servicio_insumos x WHERE x.servicio_id = s.id) DESC, s.id
	LIMIT 1`

// resolverServicioID busca el servicio por nombre probando los candidatos en orden.
// Las reservas guardan el nombre (servicio_confirmado / servicio_nombre), no el id.
func resolverServicioID(q sqlx.Queryer, nombres ...string) (int, bool, error) {
	for _, nombre := range nombres {
		if strings.TrimSpace(nombre) == "" {
			continue
		}
		var id int
		err := sqlx.Get(q, &id, resolverServicioSQL, nombre)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return 0, false, fmt.Errorf("error al resolver servicio: %w", err)
		}
		return id, true, nil
	}
	return 0, false, nil
}

func insumosDeServicio(q sqlx.Queryer, servicioID int) ([]models.ServicioInsumo, error) {
	insumos := []models.ServicioInsumo{}
	err := sqlx.Select(q, &insumos, `
		SELECT si.item_id, i.nombre AS item_nombre, i.unidad_medida, si.cantidad_por_sesion
		FROM servicio_insumos si JOIN inventario_items i ON i.id = si.item_id
		WHERE si.servicio_id = $1 AND i.activo = TRUE ORDER BY i.id`, servicioID)
	if err != nil {
		return nil, fmt.Errorf("error al leer insumos del servicio: %w", err)
	}
	return insumos, nil
}

// ReservaInventarioInfo son los datos de la reserva que el inventario necesita.
type ReservaInventarioInfo struct {
	LocalID  int            `db:"local_id"`
	Estado   string         `db:"estado"`
	Servicio sql.NullString `db:"servicio_confirmado"`
	Nombre   sql.NullString `db:"servicio_nombre"`
	Solicit  sql.NullString `db:"servicio_solicitado"`
}

func (i ReservaInventarioInfo) nombres() []string {
	return []string{i.Servicio.String, i.Nombre.String, i.Solicit.String}
}

func leerReservaInventario(q sqlx.Queryer, reservaID int) (*ReservaInventarioInfo, error) {
	var info ReservaInventarioInfo
	err := sqlx.Get(q, &info, `
		SELECT COALESCE(local_id, 0) AS local_id, COALESCE(estado, 'PENDIENTE') AS estado,
		       servicio_confirmado, servicio_nombre, servicio_solicitado
		FROM reservas WHERE id = $1`, reservaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("reserva no encontrada")
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer reserva: %w", err)
	}
	return &info, nil
}

// ─── Disponibilidad (RN-11) ───────────────────────────────────────────────────

// ComprometidoPorReservas suma lo que otras reservas AGENDADAS futuras del mismo
// local ya reservaron de cada item.
func comprometidoPorReservas(q sqlx.Queryer, localID, excluirReservaID int) (map[int]float64, error) {
	rows := []struct {
		ItemID int     `db:"item_id"`
		Total  float64 `db:"total"`
	}{}
	err := sqlx.Select(q, &rows, `
		SELECT si.item_id, COALESCE(SUM(si.cantidad_por_sesion), 0) AS total
		FROM reservas r
		JOIN LATERAL (
			SELECT s.id FROM servicios s
			WHERE s.activo = TRUE
			  AND LOWER(BTRIM(s.nombre)) = LOWER(BTRIM(COALESCE(NULLIF(r.servicio_confirmado, ''), r.servicio_nombre, r.servicio_solicitado)))
			ORDER BY EXISTS (SELECT 1 FROM servicio_insumos x WHERE x.servicio_id = s.id) DESC, s.id
			LIMIT 1
		) srv ON TRUE
		JOIN servicio_insumos si ON si.servicio_id = srv.id
		WHERE r.local_id = $1 AND r.activo = TRUE AND r.estado = 'AGENDADO'
		  AND r.fecha >= CURRENT_DATE AND r.id <> $2
		GROUP BY si.item_id`, localID, excluirReservaID)
	if err != nil {
		return nil, fmt.Errorf("error al calcular stock comprometido: %w", err)
	}
	out := make(map[int]float64, len(rows))
	for _, row := range rows {
		out[row.ItemID] = row.Total
	}
	return out, nil
}

func stockDeItems(q sqlx.Queryer, localID int, itemIDs []int) (map[int]float64, error) {
	out := map[int]float64{}
	if len(itemIDs) == 0 {
		return out, nil
	}
	holders := make([]string, len(itemIDs))
	args := []interface{}{localID}
	for i, id := range itemIDs {
		holders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	rows := []struct {
		ItemID   int     `db:"item_id"`
		Cantidad float64 `db:"cantidad"`
	}{}
	if err := sqlx.Select(q, &rows, fmt.Sprintf(
		`SELECT item_id, cantidad FROM inventario_stock WHERE local_id = $1 AND item_id IN (%s)`,
		strings.Join(holders, ",")), args...); err != nil {
		return nil, fmt.Errorf("error al leer stock: %w", err)
	}
	for _, row := range rows {
		out[row.ItemID] = row.Cantidad
	}
	return out, nil
}

// EvaluarDisponibilidad devuelve una advertencia por cada insumo del servicio
// que no alcanza en el local. No modifica nada (RN-11/RN-12).
func (r *InventarioRepo) EvaluarDisponibilidad(
	nombresServicio []string, localID, excluirReservaID int, considerarComprometido bool,
) ([]models.AdvertenciaInventario, error) {
	return evaluarDisponibilidad(r.db, nombresServicio, localID, excluirReservaID, considerarComprometido)
}

func evaluarDisponibilidad(
	q sqlx.Queryer, nombresServicio []string, localID, excluirReservaID int, considerarComprometido bool,
) ([]models.AdvertenciaInventario, error) {
	warnings := []models.AdvertenciaInventario{}
	if localID <= 0 {
		return warnings, nil
	}
	servicioID, ok, err := resolverServicioID(q, nombresServicio...)
	if err != nil || !ok {
		return warnings, err // reserva sin servicio del catalogo: sin insumos, sin advertencia
	}
	insumos, err := insumosDeServicio(q, servicioID)
	if err != nil || len(insumos) == 0 {
		return warnings, err
	}
	ids := make([]int, len(insumos))
	for i, in := range insumos {
		ids[i] = in.ItemID
	}
	stock, err := stockDeItems(q, localID, ids)
	if err != nil {
		return nil, err
	}
	var comprometido map[int]float64
	if considerarComprometido {
		if comprometido, err = comprometidoPorReservas(q, localID, excluirReservaID); err != nil {
			return nil, err
		}
	}
	for _, in := range insumos {
		disponible := redondear3(stock[in.ItemID] - comprometido[in.ItemID])
		if disponible < in.CantidadPorSesion {
			warnings = append(warnings, models.AdvertenciaInventario{
				ItemID: in.ItemID, ItemNombre: in.ItemNombre, Requerido: in.CantidadPorSesion,
				Disponible: disponible, Unidad: in.UnidadMedida,
			})
		}
	}
	return warnings, nil
}

// AdvertenciasDeReserva evalua la reserva ya guardada sin modificar nada.
func (r *InventarioRepo) AdvertenciasDeReserva(reservaID int, considerarComprometido bool) ([]models.AdvertenciaInventario, error) {
	info, err := leerReservaInventario(r.db, reservaID)
	if err != nil {
		return nil, err
	}
	return evaluarDisponibilidad(r.db, info.nombres(), info.LocalID, reservaID, considerarComprometido)
}

// RecalcularAlerta evalua la reserva ya guardada, persiste alerta_inventario y
// devuelve las advertencias. Las reservas completadas o rechazadas quedan sin alerta.
func (r *InventarioRepo) RecalcularAlerta(reservaID int, considerarComprometido bool) ([]models.AdvertenciaInventario, error) {
	info, err := leerReservaInventario(r.db, reservaID)
	if err != nil {
		return nil, err
	}
	warnings := []models.AdvertenciaInventario{}
	if info.Estado == "PENDIENTE" || info.Estado == "AGENDADO" {
		if warnings, err = evaluarDisponibilidad(r.db, info.nombres(), info.LocalID, reservaID, considerarComprometido); err != nil {
			return nil, err
		}
	}
	if _, err := r.db.Exec(
		`UPDATE reservas SET alerta_inventario = $1 WHERE id = $2 AND alerta_inventario IS DISTINCT FROM $1`,
		len(warnings) > 0, reservaID); err != nil {
		return nil, fmt.Errorf("error al guardar alerta de inventario: %w", err)
	}
	return warnings, nil
}

// RecalcularAlertasAbiertas reevalua alerta_inventario de las reservas futuras PENDIENTE/AGENDADO.
// El stock puede cambiar sin tocar las reservas (ajuste, vencimiento, compra, consumo de otra
// reserva), asi que el flag guardado al agendar queda desactualizado. localID nil = todos los locales.
func (r *InventarioRepo) RecalcularAlertasAbiertas(localID *int, considerarComprometido bool) error {
	ids := []int{}
	query := `
		SELECT id FROM reservas
		WHERE activo = TRUE AND fecha >= CURRENT_DATE
		  AND COALESCE(estado, 'PENDIENTE') IN ('PENDIENTE', 'AGENDADO')`
	args := []interface{}{}
	if localID != nil {
		query += ` AND local_id = $1`
		args = append(args, *localID)
	}
	if err := r.db.Select(&ids, query+` ORDER BY fecha, hora_desde`, args...); err != nil {
		return fmt.Errorf("error al listar reservas abiertas: %w", err)
	}
	for _, id := range ids {
		if _, err := r.RecalcularAlerta(id, considerarComprometido); err != nil {
			return err
		}
	}
	return nil
}

// EstadoDeReserva devuelve el estado actual de la reserva.
func (r *InventarioRepo) EstadoDeReserva(reservaID int) (string, error) {
	info, err := leerReservaInventario(r.db, reservaID)
	if err != nil {
		return "", err
	}
	return info.Estado, nil
}

// LocalDeReserva devuelve el local de la reserva (0 si no tiene).
func (r *InventarioRepo) LocalDeReserva(reservaID int) (int, error) {
	info, err := leerReservaInventario(r.db, reservaID)
	if err != nil {
		return 0, err
	}
	return info.LocalID, nil
}

// ─── Consumo / reversion (RN-14, RN-15, RN-16, RN-18) ─────────────────────────

func nombreItemEnError(insumo models.ServicioInsumo, err error) error {
	if errors.Is(err, repository.ErrStockInsuficiente) {
		return fmt.Errorf("%w: no alcanza %s", repository.ErrStockInsuficiente, insumo.ItemNombre)
	}
	return err
}

// ConsumirPorReservaTx descuenta los insumos del servicio de la reserva, dentro
// de la transaccion del llamador. Idempotente: si ya existe la salida de
// (reserva, item) la omite, asi reintentos o dobles clics no descuentan dos veces.
func ConsumirPorReservaTx(tx *sqlx.Tx, reservaID int, actor repository.ConsumoActor, permitirNegativo bool) error {
	info, err := leerReservaInventario(tx, reservaID)
	if err != nil {
		return err
	}
	servicioID, ok, err := resolverServicioID(tx, info.nombres()...)
	if err != nil || !ok {
		return err
	}
	insumos, err := insumosDeServicio(tx, servicioID)
	if err != nil {
		return err
	}
	for _, in := range insumos {
		var ya int
		if err := tx.Get(&ya, `
			SELECT COUNT(*) FROM inventario_movimientos
			WHERE reserva_id = $1 AND item_id = $2 AND tipo = 'SALIDA_SERVICIO'`, reservaID, in.ItemID); err != nil {
			return fmt.Errorf("error al verificar salida previa: %w", err)
		}
		if ya > 0 {
			continue
		}
		rid := reservaID
		if _, err := AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: in.ItemID, LocalID: info.LocalID, Tipo: repository.MovSalidaServicio,
			Cantidad: in.CantidadPorSesion, ReservaID: &rid, UsuarioID: actor.UsuarioID,
			UsuarioNombre: actor.UsuarioNombre, PermitirNegativo: permitirNegativo,
			Motivo: fmt.Sprintf("Reserva #%d", reservaID),
		}); err != nil {
			return nombreItemEnError(in, err)
		}
	}
	return nil
}

// RevertirPorReservaTx devuelve el stock de las salidas de la reserva que aun no
// fueron revertidas (RN-16). Idempotente.
func RevertirPorReservaTx(tx *sqlx.Tx, reservaID int, actor repository.ConsumoActor) error {
	salidas := []struct {
		ItemID   int     `db:"item_id"`
		LocalID  int     `db:"local_id"`
		Cantidad float64 `db:"cantidad"`
	}{}
	if err := tx.Select(&salidas, `
		SELECT m.item_id, m.local_id, m.cantidad FROM inventario_movimientos m
		WHERE m.reserva_id = $1 AND m.tipo = 'SALIDA_SERVICIO'
		  AND NOT EXISTS (SELECT 1 FROM inventario_movimientos x
		                  WHERE x.reserva_id = m.reserva_id AND x.item_id = m.item_id AND x.tipo = 'REVERSION_SERVICIO')`,
		reservaID); err != nil {
		return fmt.Errorf("error al buscar salidas de la reserva: %w", err)
	}
	for _, s := range salidas {
		rid := reservaID
		if _, err := AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: s.ItemID, LocalID: s.LocalID, Tipo: repository.MovReversionServicio, Cantidad: s.Cantidad,
			ReservaID: &rid, UsuarioID: actor.UsuarioID, UsuarioNombre: actor.UsuarioNombre,
			PermitirNegativo: true, Motivo: fmt.Sprintf("Reversion reserva #%d", reservaID),
		}); err != nil {
			return err
		}
	}
	return nil
}

// ConsumirPorPlanMovimientoTx descuenta insumos de una sesion de plan marcada sin
// reserva ligada (RN-18), con referencia al plan_movimiento. Idempotente.
func ConsumirPorPlanMovimientoTx(
	tx *sqlx.Tx, servicioID, localID, planMovimientoID int, actor repository.ConsumoActor, permitirNegativo bool,
) error {
	insumos, err := insumosDeServicio(tx, servicioID)
	if err != nil {
		return err
	}
	for _, in := range insumos {
		var ya int
		if err := tx.Get(&ya, `
			SELECT COUNT(*) FROM inventario_movimientos
			WHERE plan_movimiento_id = $1 AND item_id = $2 AND tipo = 'SALIDA_SERVICIO'`,
			planMovimientoID, in.ItemID); err != nil {
			return fmt.Errorf("error al verificar salida previa de plan: %w", err)
		}
		if ya > 0 {
			continue
		}
		pid := planMovimientoID
		if _, err := AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: in.ItemID, LocalID: localID, Tipo: repository.MovSalidaServicio,
			Cantidad: in.CantidadPorSesion, PlanMovimientoID: &pid, UsuarioID: actor.UsuarioID,
			UsuarioNombre: actor.UsuarioNombre, PermitirNegativo: permitirNegativo,
			Motivo: fmt.Sprintf("Sesion de plan (movimiento #%d)", planMovimientoID),
		}); err != nil {
			return nombreItemEnError(in, err)
		}
	}
	return nil
}
