CREATE TABLE IF NOT EXISTS locales_horarios (
    id         SERIAL PRIMARY KEY,
    local_id    INT NOT NULL REFERENCES locales(id) ON DELETE CASCADE,
    dia_semana  INT NOT NULL,
    hora_desde  TIME NOT NULL,
    hora_hasta  TIME NOT NULL,

    CONSTRAINT chk_locales_horarios_dia_semana
        CHECK (dia_semana BETWEEN 1 AND 7),
    CONSTRAINT chk_locales_horarios_rango_horas
        CHECK (hora_desde < hora_hasta),
    CONSTRAINT uq_locales_horarios_tramo
        UNIQUE (local_id, dia_semana, hora_desde, hora_hasta)
);

CREATE INDEX idx_locales_horarios_local_dia
    ON locales_horarios(local_id, dia_semana);
