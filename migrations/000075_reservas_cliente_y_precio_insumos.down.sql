ALTER TABLE inventario_items DROP CONSTRAINT IF EXISTS chk_inventario_items_precio_venta;
ALTER TABLE inventario_items DROP COLUMN IF EXISTS precio_venta;

DROP INDEX IF EXISTS idx_reservas_por_vincular;
DROP INDEX IF EXISTS idx_reservas_cliente_id;
ALTER TABLE reservas DROP COLUMN IF EXISTS cliente_id;
