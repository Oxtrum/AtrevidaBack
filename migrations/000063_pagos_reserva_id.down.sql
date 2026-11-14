DROP INDEX IF EXISTS uq_pagos_reserva_activo;
DROP INDEX IF EXISTS idx_pagos_reserva;
ALTER TABLE pagos DROP COLUMN IF EXISTS reserva_id;
