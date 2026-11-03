-- Cargo de insumo a cuenta del cliente: guarda el insumo y la cantidad, y su salida de stock queda en el kardex
-- ligada al cargo (SALIDA_VENTA); al anular el cargo se devuelve (REVERSION_VENTA).

ALTER TABLE cargos_cliente
    ADD COLUMN IF NOT EXISTS item_id INT REFERENCES inventario_items(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cantidad NUMERIC(12,3);

ALTER TABLE cargos_cliente
    ADD CONSTRAINT chk_cargos_cliente_insumo
        CHECK (item_id IS NULL OR (tipo = 'INSUMO' AND cantidad > 0));

COMMENT ON COLUMN cargos_cliente.item_id IS 'Insumo del inventario que se cobra; solo en cargos de tipo INSUMO.';
COMMENT ON COLUMN cargos_cliente.cantidad IS 'Cantidad del insumo cobrada; la salida de stock usa esta cantidad.';

ALTER TABLE inventario_movimientos
    ADD COLUMN IF NOT EXISTS cargo_id INT REFERENCES cargos_cliente(id) ON DELETE SET NULL;

-- A lo sumo una salida y una reversion por (cargo, item): un reintento no descuenta ni devuelve dos veces.
CREATE UNIQUE INDEX IF NOT EXISTS uq_inv_mov_salida_cargo
    ON inventario_movimientos (cargo_id, item_id) WHERE tipo = 'SALIDA_VENTA' AND cargo_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_inv_mov_reversion_cargo
    ON inventario_movimientos (cargo_id, item_id) WHERE tipo = 'REVERSION_VENTA' AND cargo_id IS NOT NULL;

COMMENT ON COLUMN inventario_movimientos.cargo_id IS
    'Cargo de la cuenta de un cliente que origino la salida (SALIDA_VENTA) o su reversion (REVERSION_VENTA).';
