ALTER TABLE plan_servicios
    DROP COLUMN IF EXISTS fecha_realizado,
    DROP COLUMN IF EXISTS realizado,
    DROP COLUMN IF EXISTS sesion_numero;

ALTER TABLE combo_servicios
    DROP COLUMN IF EXISTS sesion_numero;
