-- Agrupa los servicios del combo/plan en sesiones y registra si cada sesión se realizó.
ALTER TABLE combo_servicios
    ADD COLUMN IF NOT EXISTS sesion_numero INT NOT NULL DEFAULT 1;

ALTER TABLE plan_servicios
    ADD COLUMN IF NOT EXISTS sesion_numero   INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS realizado       BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS fecha_realizado TIMESTAMPTZ;

COMMENT ON COLUMN combo_servicios.sesion_numero IS
    'Numero de sesion (1-based) a la que pertenece el servicio dentro del combo.';
COMMENT ON COLUMN plan_servicios.sesion_numero IS
    'Numero de sesion a la que pertenece la linea del plan.';
COMMENT ON COLUMN plan_servicios.realizado IS
    'TRUE cuando la sesion de esta linea fue marcada como realizada.';
COMMENT ON COLUMN plan_servicios.fecha_realizado IS
    'Momento en que se marco realizada; NULL si esta pendiente.';
