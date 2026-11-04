DROP INDEX IF EXISTS idx_inv_mov_atencion;
ALTER TABLE inventario_movimientos DROP COLUMN IF EXISTS atencion_id;
DROP TABLE IF EXISTS atencion_insumos;
DROP TABLE IF EXISTS atenciones;
