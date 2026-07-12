-- Duracion por sesion del paquete (minutos). El admin la interpreta; no la usa el agendado.
-- Las sesiones del combo pasan a definirse a nivel paquete (sesiones_totales directo),
-- no derivadas de la suma de lineas.
ALTER TABLE combos
    ADD COLUMN IF NOT EXISTS duracion_min INT;

COMMENT ON COLUMN combos.duracion_min IS
    'Duracion sugerida por sesion del paquete, en minutos. Referencia para el administrador.';
