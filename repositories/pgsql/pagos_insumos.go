package pgsql

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	repository "atrevida-agenda-api/repositories"
)

// descontarInsumosVendidos registra una SALIDA_VENTA por cada insumo vendido en el pago, dentro de la transaccion
// del cobro. Si el mismo insumo aparece en dos lineas se suman en una sola salida (una por pago e item).
func descontarInsumosVendidos(tx *sqlx.Tx, pagoID int, codigoPago string, input repository.CrearPagoInput) error {
	orden := []int{}
	cantidades := map[int]int{}
	for _, d := range input.Detalle {
		if d.ItemID == nil {
			continue
		}
		if _, visto := cantidades[*d.ItemID]; !visto {
			orden = append(orden, *d.ItemID)
		}
		cantidades[*d.ItemID] += d.Cantidad
	}
	for _, itemID := range orden {
		var item struct {
			Nombre string   `db:"nombre"`
			Costo  *float64 `db:"costo_referencia"`
		}
		err := tx.Get(&item, `SELECT nombre, costo_referencia FROM inventario_items WHERE id = $1 AND activo = TRUE`, itemID)
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ErrItemNoEncontrado
		}
		if err != nil {
			return fmt.Errorf("error al leer el insumo vendido: %w", err)
		}
		pid := pagoID
		if _, err := AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: itemID, LocalID: input.LocalID, Tipo: repository.MovSalidaVenta,
			Cantidad: float64(cantidades[itemID]), CostoUnitario: item.Costo, PagoID: &pid,
			UsuarioID: input.Cajero.ID, UsuarioNombre: input.Cajero.Nombre,
			PermitirNegativo: input.PermitirStockNegativo, Motivo: "Venta " + codigoPago,
		}); err != nil {
			if errors.Is(err, repository.ErrStockInsuficiente) {
				return fmt.Errorf("%w: no alcanza %s", repository.ErrStockInsuficiente, item.Nombre)
			}
			return err
		}
	}
	return nil
}

// revertirInsumosVendidos devuelve al stock lo que el pago habia sacado y aun no se habia devuelto. Idempotente por
// (pago, item): el indice uq_inv_mov_reversion_venta y el NOT EXISTS evitan devolver dos veces.
func revertirInsumosVendidos(tx *sqlx.Tx, pagoID int, codigoPago string, actor repository.ConsumoActor) error {
	salidas := []struct {
		ItemID   int     `db:"item_id"`
		LocalID  int     `db:"local_id"`
		Cantidad float64 `db:"cantidad"`
	}{}
	if err := tx.Select(&salidas, `
		SELECT m.item_id, m.local_id, m.cantidad FROM inventario_movimientos m
		WHERE m.pago_id = $1 AND m.tipo = 'SALIDA_VENTA'
		  AND NOT EXISTS (SELECT 1 FROM inventario_movimientos x
		                  WHERE x.pago_id = m.pago_id AND x.item_id = m.item_id AND x.tipo = 'REVERSION_VENTA')
		ORDER BY m.id`, pagoID); err != nil {
		return fmt.Errorf("error al buscar los insumos vendidos del pago: %w", err)
	}
	for _, s := range salidas {
		pid := pagoID
		if _, err := AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: s.ItemID, LocalID: s.LocalID, Tipo: repository.MovReversionVenta, Cantidad: s.Cantidad,
			PagoID: &pid, UsuarioID: actor.UsuarioID, UsuarioNombre: actor.UsuarioNombre,
			PermitirNegativo: true, Motivo: "Anulacion " + codigoPago,
		}); err != nil {
			return err
		}
	}
	return nil
}
