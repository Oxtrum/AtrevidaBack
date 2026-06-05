ALTER TABLE pagos
    DROP CONSTRAINT IF EXISTS chk_pagos_tipo_pago;

ALTER TABLE pagos
    ALTER COLUMN tipo_pago DROP NOT NULL;
