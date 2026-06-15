ALTER TABLE pagos
    DROP CONSTRAINT IF EXISTS fk_pagos_id_cajero_modificacion,
    DROP CONSTRAINT IF EXISTS fk_pagos_id_cajero;

ALTER TABLE pagos
    DROP COLUMN IF EXISTS username_cajero_modificacion,
    DROP COLUMN IF EXISTS nombre_cajero_modificacion,
    DROP COLUMN IF EXISTS id_cajero_modificacion,
    DROP COLUMN IF EXISTS username_cajero,
    DROP COLUMN IF EXISTS nombre_cajero,
    DROP COLUMN IF EXISTS id_cajero;
