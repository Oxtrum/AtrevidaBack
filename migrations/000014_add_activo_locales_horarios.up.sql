ALTER TABLE locales_horarios
    ADD COLUMN IF NOT EXISTS activo BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE locales_horarios
    DROP CONSTRAINT IF EXISTS uq_locales_horarios_tramo;

CREATE UNIQUE INDEX IF NOT EXISTS uq_locales_horarios_tramo_activo
    ON locales_horarios(local_id, dia_semana, hora_desde, hora_hasta)
    WHERE activo = TRUE;
