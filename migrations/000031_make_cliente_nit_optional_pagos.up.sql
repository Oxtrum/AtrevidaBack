ALTER TABLE pagos
    ALTER COLUMN cliente_nit DROP NOT NULL;

COMMENT ON COLUMN pagos.cliente_nit IS
    'NIT opcional del cliente al momento de registrar el pago.';
