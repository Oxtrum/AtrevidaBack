-- La referencia es opcional por compatibilidad; Caja y el frontend no dependen de ella.
ALTER TABLE pagos
    ADD COLUMN IF NOT EXISTS reserva_id INT REFERENCES reservas(id) ON DELETE SET NULL;

ALTER TABLE pagos ALTER COLUMN reserva_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pagos_reserva
    ON pagos(reserva_id) WHERE reserva_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_pagos_reserva_activo
    ON pagos(reserva_id) WHERE reserva_id IS NOT NULL AND activo = TRUE;

COMMENT ON COLUMN pagos.reserva_id IS
    'Referencia opcional a una reserva; no modifica su estado ni crea un pago automaticamente.';

-- Limpia la tabla transitoria de la version anterior de la migracion 81. El rollback previo restaura sus vinculos.
DROP TABLE IF EXISTS legacy_pagos_reservas;
