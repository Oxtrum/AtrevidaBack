DROP INDEX IF EXISTS idx_reservas_canceladas_local_fecha;
ALTER TABLE reservas DROP CONSTRAINT IF EXISTS chk_reservas_motivo_cancelacion_tipo;
ALTER TABLE reservas
    DROP COLUMN IF EXISTS cancelado_por,
    DROP COLUMN IF EXISTS cancelado_por_id,
    DROP COLUMN IF EXISTS cancelado_en,
    DROP COLUMN IF EXISTS motivo_cancelacion_detalle,
    DROP COLUMN IF EXISTS motivo_cancelacion_tipo;
