DROP INDEX IF EXISTS idx_combos_activo_nombre;
DROP INDEX IF EXISTS idx_combo_servicios_orden_activo;

ALTER TABLE combo_servicios
    ALTER COLUMN servicio_texto DROP NOT NULL;

ALTER TABLE combos
    DROP CONSTRAINT IF EXISTS chk_combos_precio_paquete,
    DROP CONSTRAINT IF EXISTS chk_combos_moneda,
    DROP CONSTRAINT IF EXISTS chk_combos_tipo_precio,
    DROP COLUMN IF EXISTS moneda,
    DROP COLUMN IF EXISTS precio_paquete,
    DROP COLUMN IF EXISTS tipo_precio;
