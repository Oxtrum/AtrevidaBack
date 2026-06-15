UPDATE pagos
SET nombre_cajero = 'admin'
WHERE nombre_cajero IS NULL
   OR BTRIM(nombre_cajero) = '';

UPDATE pagos
SET username_cajero = 'admin'
WHERE username_cajero IS NULL
   OR BTRIM(username_cajero) = '';

ALTER TABLE pagos
    ALTER COLUMN nombre_cajero SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_pagos_nombre_cajero_no_vacio'
          AND conrelid = 'pagos'::regclass
    ) THEN
        ALTER TABLE pagos
            ADD CONSTRAINT chk_pagos_nombre_cajero_no_vacio
            CHECK (BTRIM(nombre_cajero) <> '');
    END IF;
END $$;
