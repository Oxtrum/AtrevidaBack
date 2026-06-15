ALTER TABLE pagos
    ADD COLUMN IF NOT EXISTS id_cajero INT,
    ADD COLUMN IF NOT EXISTS nombre_cajero VARCHAR(200),
    ADD COLUMN IF NOT EXISTS username_cajero VARCHAR(100),
    ADD COLUMN IF NOT EXISTS id_cajero_modificacion INT,
    ADD COLUMN IF NOT EXISTS nombre_cajero_modificacion VARCHAR(200),
    ADD COLUMN IF NOT EXISTS username_cajero_modificacion VARCHAR(100);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_pagos_id_cajero'
          AND conrelid = 'pagos'::regclass
    ) THEN
        ALTER TABLE pagos
            ADD CONSTRAINT fk_pagos_id_cajero
            FOREIGN KEY (id_cajero)
            REFERENCES usuarios(id)
            ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_pagos_id_cajero_modificacion'
          AND conrelid = 'pagos'::regclass
    ) THEN
        ALTER TABLE pagos
            ADD CONSTRAINT fk_pagos_id_cajero_modificacion
            FOREIGN KEY (id_cajero_modificacion)
            REFERENCES usuarios(id)
            ON DELETE SET NULL;
    END IF;
END $$;

COMMENT ON COLUMN pagos.id_cajero IS
    'ID opcional del usuario cajero que registro el pago.';
COMMENT ON COLUMN pagos.nombre_cajero IS
    'Nombre completo del cajero que registro el pago.';
COMMENT ON COLUMN pagos.username_cajero IS
    'Username opcional del cajero que registro el pago.';
COMMENT ON COLUMN pagos.id_cajero_modificacion IS
    'ID opcional del usuario cajero que modifico el pago por ultima vez.';
COMMENT ON COLUMN pagos.nombre_cajero_modificacion IS
    'Nombre completo opcional del cajero que modifico el pago por ultima vez.';
COMMENT ON COLUMN pagos.username_cajero_modificacion IS
    'Username opcional del cajero que modifico el pago por ultima vez.';
