COMMENT ON COLUMN combos.costo_total IS NULL;
COMMENT ON COLUMN combo_servicios.costo IS NULL;
COMMENT ON COLUMN combo_servicios.servicio_texto IS NULL;
COMMENT ON COLUMN combo_servicios.servicio_id IS NULL;
COMMENT ON TABLE combo_servicios IS NULL;
COMMENT ON TABLE combos IS NULL;

ALTER TABLE combo_servicios
    DROP CONSTRAINT IF EXISTS chk_combo_servicios_snapshot_servicio,
    DROP CONSTRAINT IF EXISTS chk_combo_servicios_sesiones_positivas,
    DROP CONSTRAINT IF EXISTS chk_combo_servicios_costo_no_negativo;

ALTER TABLE combos
    DROP CONSTRAINT IF EXISTS chk_combos_sesiones_totales_positivas,
    DROP CONSTRAINT IF EXISTS chk_combos_costo_total_no_negativo;

ALTER TABLE combos
    DROP COLUMN IF EXISTS actualizado_en,
    DROP COLUMN IF EXISTS creado_en,
    DROP COLUMN IF EXISTS descripcion;
