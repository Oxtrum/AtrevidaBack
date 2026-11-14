package pgsql

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	repository "atrevida-agenda-api/repositories"
)

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
