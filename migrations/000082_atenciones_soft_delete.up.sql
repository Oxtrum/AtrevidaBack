-- Conserva el historial de atenciones al desactivarlas desde el panel.
ALTER TABLE atenciones
    ADD COLUMN activo BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX idx_atenciones_activas_local_fecha
    ON atenciones (local_id, fecha_atencion DESC, id DESC)
    WHERE activo = TRUE;
