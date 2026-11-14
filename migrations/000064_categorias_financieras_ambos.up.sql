-- Una categoria financiera puede ser de INGRESO, de EGRESO o de AMBOS.
ALTER TABLE categorias_financieras DROP CONSTRAINT IF EXISTS chk_categorias_fin_naturaleza;
ALTER TABLE categorias_financieras
    ADD CONSTRAINT chk_categorias_fin_naturaleza CHECK (naturaleza IN ('INGRESO', 'EGRESO', 'AMBOS'));
