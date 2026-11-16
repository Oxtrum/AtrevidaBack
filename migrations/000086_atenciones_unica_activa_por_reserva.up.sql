-- Conserva el historial de atenciones desactivadas y evita que una reserva
-- tenga más de una atención vigente al mismo tiempo.
ALTER TABLE atenciones
    DROP CONSTRAINT IF EXISTS atenciones_reserva_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS ux_atenciones_reserva_activa
    ON atenciones (reserva_id)
    WHERE activo = TRUE;
