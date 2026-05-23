DROP INDEX IF EXISTS uq_locales_horarios_tramo_activo;

ALTER TABLE locales_horarios
    DROP COLUMN IF EXISTS activo;

ALTER TABLE locales_horarios
    ADD CONSTRAINT uq_locales_horarios_tramo
        UNIQUE (local_id, dia_semana, hora_desde, hora_hasta);
