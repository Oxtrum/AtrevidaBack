package pgsql

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	repository "atrevida-agenda-api/repositories"
)

// descontarInsumoDeCargo registra la SALIDA_VENTA del insumo de un cargo, dentro de la transaccion que crea el cargo.
func descontarInsumoDeCargo(tx *sqlx.Tx, cargoID int, in CargoInput) error {
	var item struct {
		Nombre string   `db:"nombre"`
		Costo  *float64 `db:"costo_referencia"`
	}
	err := tx.Get(&item, `SELECT nombre, costo_referencia FROM inventario_items WHERE id = $1 AND activo = TRUE`, *in.ItemID)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.ErrItemNoEncontrado
	}
	if err != nil {
		return fmt.Errorf("error al leer el insumo del cargo: %w", err)
	}
	cid := cargoID
	if _, err := AplicarMovimientoStock(tx, repository.MovimientoStockInput{
		ItemID: *in.ItemID, LocalID: *in.LocalID, Tipo: repository.MovSalidaVenta, Cantidad: *in.Cantidad,
		CostoUnitario: item.Costo, CargoID: &cid, UsuarioID: in.CreadoPorID, UsuarioNombre: in.CreadoPor,
		PermitirNegativo: in.PermitirStockNegativo, Motivo: fmt.Sprintf("Cargo #%d", cargoID),
	}); err != nil {
		if errors.Is(err, repository.ErrStockInsuficiente) {
			return fmt.Errorf("%w: no alcanza %s", repository.ErrStockInsuficiente, item.Nombre)
		}
		return err
	}
	return nil
}

// revertirInsumoDeCargo devuelve al stock lo que el cargo habia sacado y aun no se habia devuelto.
func revertirInsumoDeCargo(tx *sqlx.Tx, cargoID int, actor repository.ConsumoActor) error {
	salidas := []struct {
		ItemID   int     `db:"item_id"`
		LocalID  int     `db:"local_id"`
		Cantidad float64 `db:"cantidad"`
	}{}
	if err := tx.Select(&salidas, `
		SELECT m.item_id, m.local_id, m.cantidad FROM inventario_movimientos m
		WHERE m.cargo_id = $1 AND m.tipo = 'SALIDA_VENTA'
		  AND NOT EXISTS (SELECT 1 FROM inventario_movimientos x
		                  WHERE x.cargo_id = m.cargo_id AND x.item_id = m.item_id AND x.tipo = 'REVERSION_VENTA')
		ORDER BY m.id`, cargoID); err != nil {
		return fmt.Errorf("error al buscar el insumo del cargo: %w", err)
	}
	for _, s := range salidas {
		cid := cargoID
		if _, err := AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: s.ItemID, LocalID: s.LocalID, Tipo: repository.MovReversionVenta, Cantidad: s.Cantidad,
			CargoID: &cid, UsuarioID: actor.UsuarioID, UsuarioNombre: actor.UsuarioNombre,
			PermitirNegativo: true, Motivo: fmt.Sprintf("Anulacion del cargo #%d", cargoID),
		}); err != nil {
			return err
		}
	}
	return nil
}
