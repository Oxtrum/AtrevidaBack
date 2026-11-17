-- Volver a exigir cantidad: los insumos de uso variable no tienen una cantidad que restaurar, se quitan del servicio.
DELETE FROM servicio_insumos WHERE cantidad_por_sesion IS NULL;

ALTER TABLE servicio_insumos DROP CONSTRAINT IF EXISTS chk_servicio_insumos_cantidad;
ALTER TABLE servicio_insumos
    ADD CONSTRAINT chk_servicio_insumos_cantidad CHECK (cantidad_por_sesion > 0);

ALTER TABLE servicio_insumos ALTER COLUMN cantidad_por_sesion SET NOT NULL;

COMMENT ON COLUMN servicio_insumos.cantidad_por_sesion IS NULL;
