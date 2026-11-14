DROP INDEX IF EXISTS idx_mov_fin_compra;
ALTER TABLE movimientos_financieros DROP CONSTRAINT IF EXISTS fk_mov_fin_compra;
ALTER TABLE inventario_movimientos DROP CONSTRAINT IF EXISTS fk_inv_mov_compra;
DROP TABLE IF EXISTS detalle_compras;
DROP TABLE IF EXISTS compras_inventario;
DROP SEQUENCE IF EXISTS compras_inventario_codigo_seq;
