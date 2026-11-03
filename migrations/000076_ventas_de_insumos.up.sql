-- Venta de insumos desde la caja: una linea del pago puede ser un insumo del inventario y su salida queda
-- en el kardex, ligada al pago que la origino.

-- Una linea es un servicio (servicio_id) o un insumo (item_id), nunca las dos; sin ninguna es una linea libre.
ALTER TABLE detalle_pagos
    ADD COLUMN IF NOT EXISTS item_id INT REFERENCES inventario_items(id) ON DELETE SET NULL;

ALTER TABLE detalle_pagos
    ADD CONSTRAINT chk_detalle_pagos_servicio_o_item CHECK (servicio_id IS NULL OR item_id IS NULL);

CREATE INDEX IF NOT EXISTS idx_detalle_pagos_item ON detalle_pagos (item_id) WHERE item_id IS NOT NULL;

COMMENT ON COLUMN detalle_pagos.item_id IS
    'Insumo vendido en esta linea. Al crear el pago se registra SALIDA_VENTA en el kardex; al anularlo, REVERSION_VENTA.';

ALTER TABLE inventario_movimientos
    ADD COLUMN IF NOT EXISTS pago_id INT REFERENCES pagos(id) ON DELETE SET NULL;

ALTER TABLE inventario_movimientos DROP CONSTRAINT chk_inv_mov_tipo;
ALTER TABLE inventario_movimientos ADD CONSTRAINT chk_inv_mov_tipo CHECK (tipo IN
    ('ENTRADA_COMPRA', 'SALIDA_SERVICIO', 'REVERSION_SERVICIO', 'AJUSTE_POSITIVO', 'AJUSTE_NEGATIVO',
     'SALIDA_VENTA', 'REVERSION_VENTA'));

-- A lo sumo una salida y una reversion por (pago, item): un reintento no descuenta ni devuelve dos veces.
CREATE UNIQUE INDEX IF NOT EXISTS uq_inv_mov_salida_venta
    ON inventario_movimientos (pago_id, item_id) WHERE tipo = 'SALIDA_VENTA';
CREATE UNIQUE INDEX IF NOT EXISTS uq_inv_mov_reversion_venta
    ON inventario_movimientos (pago_id, item_id) WHERE tipo = 'REVERSION_VENTA';

COMMENT ON COLUMN inventario_movimientos.pago_id IS
    'Pago de caja que origino la salida (SALIDA_VENTA) o su reversion (REVERSION_VENTA).';
