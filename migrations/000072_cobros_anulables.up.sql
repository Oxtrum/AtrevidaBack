-- Las filas de pago_aplicaciones (que parte de un pago cubre que cargo) dejan de borrarse: al anular un pago o un
-- cargo se marcan como anuladas, con motivo y fecha, para conservar el historial de dinero.

ALTER TABLE pago_aplicaciones
    ADD COLUMN IF NOT EXISTS anulado          BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS motivo_anulacion TEXT,
    ADD COLUMN IF NOT EXISTS anulado_en       TIMESTAMPTZ;

ALTER TABLE pago_aplicaciones
    ADD CONSTRAINT chk_pago_aplicaciones_anulacion
        CHECK (NOT anulado OR motivo_anulacion IS NOT NULL);

-- Cada cobro es un registro nuevo: ya no se suma sobre una fila existente, asi que se quita la unicidad.
ALTER TABLE pago_aplicaciones DROP CONSTRAINT IF EXISTS uq_pago_aplicaciones;

CREATE INDEX IF NOT EXISTS idx_pago_aplicaciones_pago_vigente
    ON pago_aplicaciones (pago_id) WHERE anulado = FALSE;
DROP INDEX IF EXISTS idx_pago_aplicaciones_cargo;
CREATE INDEX IF NOT EXISTS idx_pago_aplicaciones_cargo_vigente
    ON pago_aplicaciones (cargo_id) WHERE anulado = FALSE;

COMMENT ON COLUMN pago_aplicaciones.anulado IS 'Cobro anulado: ya no cuenta como pagado, pero la fila se conserva.';
COMMENT ON COLUMN pago_aplicaciones.motivo_anulacion IS 'Por que se anulo: pago o cargo anulado, con su motivo.';
