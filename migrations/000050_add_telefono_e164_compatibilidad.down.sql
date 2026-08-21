ALTER TABLE clientes
    DROP CONSTRAINT IF EXISTS chk_clientes_telefono_e164,
    DROP COLUMN IF EXISTS telefono_e164;

ALTER TABLE reservas
    DROP CONSTRAINT IF EXISTS chk_reservas_telefono_e164,
    DROP COLUMN IF EXISTS telefono_e164;
