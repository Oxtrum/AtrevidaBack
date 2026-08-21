-- Transición compatible: numero_telefono conserva el formato legacy para
-- consumidores anteriores; telefono_e164 almacena la forma internacional
-- canónica cuando puede resolverse sin ambigüedad.
ALTER TABLE clientes
    ADD COLUMN IF NOT EXISTS telefono_e164 VARCHAR(16);

ALTER TABLE reservas
    ALTER COLUMN numero_telefono TYPE VARCHAR(20),
    ADD COLUMN IF NOT EXISTS telefono_e164 VARCHAR(16);

ALTER TABLE clientes
    ADD CONSTRAINT chk_clientes_telefono_e164
    CHECK (telefono_e164 IS NULL OR telefono_e164 ~ '^\+[1-9][0-9]{1,14}$')
    NOT VALID;

ALTER TABLE reservas
    ADD CONSTRAINT chk_reservas_telefono_e164
    CHECK (telefono_e164 IS NULL OR telefono_e164 ~ '^\+[1-9][0-9]{1,14}$')
    NOT VALID;
