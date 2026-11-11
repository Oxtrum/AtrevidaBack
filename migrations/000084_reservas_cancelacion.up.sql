-- Cancelaciones posteriores a una reserva ya agendada.
ALTER TABLE reservas
    ADD COLUMN motivo_cancelacion_tipo VARCHAR(30),
    ADD COLUMN motivo_cancelacion_detalle TEXT,
    ADD COLUMN cancelado_en TIMESTAMPTZ,
    ADD COLUMN cancelado_por_id INT REFERENCES usuarios(id) ON DELETE SET NULL,
    ADD COLUMN cancelado_por VARCHAR(200);

ALTER TABLE reservas
    ADD CONSTRAINT chk_reservas_motivo_cancelacion_tipo
    CHECK (motivo_cancelacion_tipo IS NULL OR motivo_cancelacion_tipo IN (
        'INASISTENCIA', 'REPROGRAMACION', 'SOLICITUD_CLIENTE', 'OPERATIVO_LOCAL', 'OTRO'
    ));

CREATE INDEX idx_reservas_canceladas_local_fecha
    ON reservas (local_id, fecha DESC, id DESC)
    WHERE activo = TRUE AND estado = 'CANCELADO';
