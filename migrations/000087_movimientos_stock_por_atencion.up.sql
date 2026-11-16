-- Las salidas históricas de una reserva ya no deben impedir una nueva atención
-- tras desactivar la anterior. Se conserva la idempotencia de los flujos
-- antiguos sin atención y se vuelve única la salida de cada atención.
DROP INDEX IF EXISTS uq_inv_mov_salida_reserva;

CREATE UNIQUE INDEX uq_inv_mov_salida_reserva
    ON inventario_movimientos (reserva_id, item_id)
    WHERE tipo = 'SALIDA_SERVICIO' AND atencion_id IS NULL;

CREATE UNIQUE INDEX uq_inv_mov_salida_atencion
    ON inventario_movimientos (atencion_id, item_id)
    WHERE tipo = 'SALIDA_SERVICIO' AND atencion_id IS NOT NULL;
