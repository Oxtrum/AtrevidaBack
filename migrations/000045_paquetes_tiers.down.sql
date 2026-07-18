ALTER TABLE combos
    DROP COLUMN IF EXISTS nota,
    DROP COLUMN IF EXISTS precio_regular,
    DROP COLUMN IF EXISTS paquete_id;
DROP TABLE IF EXISTS paquete_local;
DROP TABLE IF EXISTS paquete_servicios;
DROP TABLE IF EXISTS paquetes;
