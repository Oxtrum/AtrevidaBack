UPDATE pagos
SET tipo_pago = 'efectivo'
WHERE tipo_pago IS NULL;

ALTER TABLE pagos
    ALTER COLUMN tipo_pago SET NOT NULL;

ALTER TABLE pagos
    ADD CONSTRAINT chk_pagos_tipo_pago
        CHECK (tipo_pago IN ('efectivo', 'qr'));
