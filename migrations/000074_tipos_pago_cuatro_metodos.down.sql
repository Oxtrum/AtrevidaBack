-- Los pagos con tarjeta o transferencia no caben en la restriccion anterior: se pasan a efectivo para poder revertir.
UPDATE pagos SET tipo_pago = 'efectivo' WHERE tipo_pago IN ('tarjeta', 'transferencia');

ALTER TABLE pagos DROP CONSTRAINT IF EXISTS chk_pagos_tipo_pago;
ALTER TABLE pagos
    ADD CONSTRAINT chk_pagos_tipo_pago
        CHECK (tipo_pago IN ('efectivo', 'qr'));
