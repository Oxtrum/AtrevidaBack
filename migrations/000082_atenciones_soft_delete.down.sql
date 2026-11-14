DROP INDEX IF EXISTS idx_atenciones_activas_local_fecha;
ALTER TABLE atenciones DROP COLUMN IF EXISTS activo;
