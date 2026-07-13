-- Renombra el estado pre-pago BORRADOR -> RESERVADO (reservado, esperando cobro).
-- Solo afecta planes.estado; pagos.estado tiene su propio 'BORRADOR' sin relacion.
ALTER TABLE planes ALTER COLUMN estado DROP DEFAULT;
ALTER TABLE planes DROP CONSTRAINT IF EXISTS chk_planes_estado;
UPDATE planes SET estado = 'RESERVADO' WHERE estado = 'BORRADOR';
ALTER TABLE planes ADD CONSTRAINT chk_planes_estado
    CHECK (estado IN ('RESERVADO', 'ACTIVO', 'COMPLETADO', 'VENCIDO', 'CANCELADO'));
ALTER TABLE planes ALTER COLUMN estado SET DEFAULT 'RESERVADO';
COMMENT ON COLUMN planes.estado IS
    'Estado contractual: RESERVADO, ACTIVO, COMPLETADO, VENCIDO, CANCELADO.';
