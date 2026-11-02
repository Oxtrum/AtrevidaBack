-- Momento real en que se digito el pago. fecha_creacion pasa a ser la fecha del hecho: cuando ocurrio el cobro,
-- editable al crear (p. ej. al registrar de noche lo anotado en papel). Reportes, orden e indices siguen usando
-- fecha_creacion; registrado_en solo sirve de auditoria y no se edita.
ALTER TABLE pagos ADD COLUMN IF NOT EXISTS registrado_en TIMESTAMPTZ;

-- Los pagos existentes se registraron cuando ocurrieron.
UPDATE pagos SET registrado_en = fecha_creacion WHERE registrado_en IS NULL;

ALTER TABLE pagos ALTER COLUMN registrado_en SET DEFAULT NOW();
ALTER TABLE pagos ALTER COLUMN registrado_en SET NOT NULL;

COMMENT ON COLUMN pagos.fecha_creacion IS
    'Fecha del hecho: cuando ocurrio el cobro. Puede ser anterior a registrado_en si se registro despues.';
COMMENT ON COLUMN pagos.registrado_en IS
    'Momento real en que se registro el pago en el sistema; lo pone el servidor y no se edita.';
