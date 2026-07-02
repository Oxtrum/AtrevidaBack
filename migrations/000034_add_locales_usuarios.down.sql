ALTER TABLE usuarios
    DROP COLUMN IF EXISTS nombre_local,
    DROP COLUMN IF EXISTS local_id;
