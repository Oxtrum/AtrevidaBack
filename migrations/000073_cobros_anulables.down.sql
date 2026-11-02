-- Revertir pierde el historial de cobros anulados: esas filas se eliminan antes de quitar las columnas.
DELETE FROM pago_aplicaciones WHERE anulado = TRUE;

DROP INDEX IF EXISTS idx_pago_aplicaciones_cargo_vigente;
DROP INDEX IF EXISTS idx_pago_aplicaciones_pago_vigente;
CREATE INDEX IF NOT EXISTS idx_pago_aplicaciones_cargo ON pago_aplicaciones (cargo_id);

-- La unicidad solo se puede restaurar si ningun pago cubre dos veces el mismo cargo; se suman antes.
WITH sumas AS (
    SELECT MIN(id) AS id, pago_id, cargo_id, SUM(monto) AS monto
    FROM pago_aplicaciones
    GROUP BY pago_id, cargo_id
    HAVING COUNT(*) > 1
)
UPDATE pago_aplicaciones a SET monto = s.monto FROM sumas s WHERE a.id = s.id;

DELETE FROM pago_aplicaciones a
USING pago_aplicaciones b
WHERE a.pago_id = b.pago_id AND a.cargo_id = b.cargo_id AND a.id > b.id;

ALTER TABLE pago_aplicaciones
    ADD CONSTRAINT uq_pago_aplicaciones UNIQUE (pago_id, cargo_id);

ALTER TABLE pago_aplicaciones DROP CONSTRAINT IF EXISTS chk_pago_aplicaciones_anulacion;
ALTER TABLE pago_aplicaciones
    DROP COLUMN IF EXISTS anulado_en,
    DROP COLUMN IF EXISTS motivo_anulacion,
    DROP COLUMN IF EXISTS anulado;
