-- Quita las filas de venta del kardex (no devuelve stock: este down es solo para volver a una base sin la funcion).
DELETE FROM inventario_movimientos WHERE tipo IN ('SALIDA_VENTA', 'REVERSION_VENTA');

DROP INDEX IF EXISTS uq_inv_mov_reversion_venta;
DROP INDEX IF EXISTS uq_inv_mov_salida_venta;

ALTER TABLE inventario_movimientos DROP CONSTRAINT IF EXISTS chk_inv_mov_tipo;
ALTER TABLE inventario_movimientos ADD CONSTRAINT chk_inv_mov_tipo CHECK (tipo IN
    ('ENTRADA_COMPRA', 'SALIDA_SERVICIO', 'REVERSION_SERVICIO', 'AJUSTE_POSITIVO', 'AJUSTE_NEGATIVO'));
ALTER TABLE inventario_movimientos DROP COLUMN IF EXISTS pago_id;

DROP INDEX IF EXISTS idx_detalle_pagos_item;
ALTER TABLE detalle_pagos DROP CONSTRAINT IF EXISTS chk_detalle_pagos_servicio_o_item;
ALTER TABLE detalle_pagos DROP COLUMN IF EXISTS item_id;
