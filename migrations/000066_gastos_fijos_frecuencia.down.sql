ALTER TABLE gastos_fijos DROP CONSTRAINT IF EXISTS chk_gastos_fijos_frecuencia;
ALTER TABLE gastos_fijos DROP COLUMN IF EXISTS frecuencia;
