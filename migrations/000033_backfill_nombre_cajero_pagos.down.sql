ALTER TABLE pagos
    DROP CONSTRAINT IF EXISTS chk_pagos_nombre_cajero_no_vacio;

ALTER TABLE pagos
    ALTER COLUMN nombre_cajero DROP NOT NULL;
