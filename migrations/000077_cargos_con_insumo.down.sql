-- Quita las filas de kardex ligadas a cargos (no devuelve stock: este down es solo para volver a una base sin la funcion).
DELETE FROM inventario_movimientos WHERE cargo_id IS NOT NULL;

DROP INDEX IF EXISTS uq_inv_mov_reversion_cargo;
DROP INDEX IF EXISTS uq_inv_mov_salida_cargo;
ALTER TABLE inventario_movimientos DROP COLUMN IF EXISTS cargo_id;

ALTER TABLE cargos_cliente DROP CONSTRAINT IF EXISTS chk_cargos_cliente_insumo;
ALTER TABLE cargos_cliente DROP COLUMN IF EXISTS cantidad;
ALTER TABLE cargos_cliente DROP COLUMN IF EXISTS item_id;
