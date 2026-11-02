-- Metodos de pago: efectivo, tarjeta, QR y transferencia. Lista cerrada; otro metodo exige una migracion.
-- Debe coincidir con services.TiposPago.
ALTER TABLE pagos DROP CONSTRAINT IF EXISTS chk_pagos_tipo_pago;
ALTER TABLE pagos
    ADD CONSTRAINT chk_pagos_tipo_pago
        CHECK (tipo_pago IN ('efectivo', 'tarjeta', 'qr', 'transferencia'));
