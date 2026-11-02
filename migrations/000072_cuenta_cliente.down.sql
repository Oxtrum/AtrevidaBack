DROP TABLE IF EXISTS pago_aplicaciones;
DROP TABLE IF EXISTS cargos_cliente;
DROP INDEX IF EXISTS idx_pagos_abono_cliente;
ALTER TABLE pagos DROP CONSTRAINT IF EXISTS chk_pagos_abono_con_cliente;
ALTER TABLE pagos DROP COLUMN IF EXISTS es_abono;
