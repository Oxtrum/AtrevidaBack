DROP INDEX IF EXISTS idx_reservas_alerta_inventario;
ALTER TABLE reservas DROP COLUMN IF EXISTS alerta_inventario;
