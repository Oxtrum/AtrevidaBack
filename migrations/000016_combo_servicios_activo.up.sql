ALTER TABLE combo_servicios
    ADD COLUMN IF NOT EXISTS activo BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_combo_servicios_combo_activo
    ON combo_servicios(combo_id)
    WHERE activo = TRUE;
