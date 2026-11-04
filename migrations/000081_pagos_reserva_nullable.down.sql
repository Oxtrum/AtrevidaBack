-- La columna y los indices ya existian desde migraciones anteriores a la 81.
COMMENT ON COLUMN pagos.reserva_id IS
    'Reserva que origino el cobro; nulo para ventas de mostrador.';
