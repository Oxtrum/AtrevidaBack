-- Frecuencia de cobro de los gastos fijos: antes todos eran mensuales.
-- dia_cobro aplica a las frecuencias de mes en adelante; DIARIA/SEMANAL/QUINCENAL cuentan desde fecha_inicio.
ALTER TABLE gastos_fijos ADD COLUMN frecuencia VARCHAR(12) NOT NULL DEFAULT 'MENSUAL';
ALTER TABLE gastos_fijos ADD CONSTRAINT chk_gastos_fijos_frecuencia
    CHECK (frecuencia IN ('DIARIA','SEMANAL','QUINCENAL','MENSUAL','BIMESTRAL','TRIMESTRAL','SEMESTRAL','ANUAL'));
