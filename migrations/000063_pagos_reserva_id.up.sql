-- Enlaza un cobro con la reserva que cobra. Permite saber que reservas completadas ya se
-- cobraron y cuales siguen por cobrar. Es opcional: los cobros de mostrador no llevan reserva.
ALTER TABLE pagos
    ADD COLUMN IF NOT EXISTS reserva_id INT REFERENCES reservas(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_pagos_reserva ON pagos(reserva_id) WHERE reserva_id IS NOT NULL;

-- Una reserva tiene como maximo un cobro activo; anular el pago (activo = FALSE) libera la reserva.
CREATE UNIQUE INDEX IF NOT EXISTS uq_pagos_reserva_activo
    ON pagos(reserva_id) WHERE reserva_id IS NOT NULL AND activo = TRUE;

COMMENT ON COLUMN pagos.reserva_id IS
    'Reserva que origino el cobro; nulo para ventas de mostrador.';
