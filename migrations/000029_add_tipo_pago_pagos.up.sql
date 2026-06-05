ALTER TABLE pagos
    ADD COLUMN IF NOT EXISTS tipo_pago VARCHAR(20);

COMMENT ON COLUMN pagos.tipo_pago IS
    'Tipo de pago utilizado: efectivo o qr.';
