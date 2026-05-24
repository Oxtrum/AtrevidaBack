DROP INDEX IF EXISTS idx_combo_servicios_combo_activo;

ALTER TABLE combo_servicios
    DROP COLUMN IF EXISTS activo;
