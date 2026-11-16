-- Restaurar la regla histórica solo es válido mientras no existan varias
-- salidas de servicio para el mismo insumo y reserva.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM inventario_movimientos
        WHERE tipo = 'SALIDA_SERVICIO'
        GROUP BY reserva_id, item_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'no se puede revertir 000087: existen varias salidas de servicio por reserva e insumo';
    END IF;
END $$;

DROP INDEX IF EXISTS uq_inv_mov_salida_atencion;
DROP INDEX IF EXISTS uq_inv_mov_salida_reserva;

CREATE UNIQUE INDEX uq_inv_mov_salida_reserva
    ON inventario_movimientos (reserva_id, item_id)
    WHERE tipo = 'SALIDA_SERVICIO';
