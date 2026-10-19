-- Las categorias AMBOS vuelven a EGRESO (naturaleza original por defecto).
UPDATE categorias_financieras SET naturaleza = 'EGRESO' WHERE naturaleza = 'AMBOS';
ALTER TABLE categorias_financieras DROP CONSTRAINT IF EXISTS chk_categorias_fin_naturaleza;
ALTER TABLE categorias_financieras
    ADD CONSTRAINT chk_categorias_fin_naturaleza CHECK (naturaleza IN ('INGRESO', 'EGRESO'));
