UPDATE pagos
SET cliente_nit = ''
WHERE cliente_nit IS NULL;

ALTER TABLE pagos
    ALTER COLUMN cliente_nit SET NOT NULL;
