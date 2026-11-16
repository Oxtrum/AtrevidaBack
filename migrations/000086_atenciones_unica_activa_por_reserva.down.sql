-- No es posible restaurar la unicidad histórica si ya se registraron
-- atenciones desactivadas adicionales para una misma reserva.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM atenciones
        GROUP BY reserva_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'no se puede revertir 000086: existen varias atenciones para una reserva';
    END IF;
END $$;

DROP INDEX IF EXISTS ux_atenciones_reserva_activa;

ALTER TABLE atenciones
    ADD CONSTRAINT atenciones_reserva_id_key UNIQUE (reserva_id);
