ALTER TABLE reservas
DROP COLUMN IF EXISTS servicio_confirmado,
DROP COLUMN IF EXISTS servicio_solicitado;

ALTER TABLE servicios
DROP COLUMN IF EXISTS requiere_evaluacion;
