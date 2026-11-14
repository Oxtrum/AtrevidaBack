CREATE TABLE inventario_movimientos (
    id                 SERIAL PRIMARY KEY,
    item_id            INT NOT NULL REFERENCES inventario_items(id) ON DELETE RESTRICT,
    local_id           INT NOT NULL REFERENCES locales(id) ON DELETE RESTRICT,
    tipo               VARCHAR(30) NOT NULL,
    cantidad           NUMERIC(12,3) NOT NULL,
    stock_resultante   NUMERIC(12,3) NOT NULL,
    costo_unitario     NUMERIC(12,2),
    reserva_id         INT REFERENCES reservas(id) ON DELETE SET NULL,
    plan_movimiento_id INT REFERENCES plan_movimientos(id) ON DELETE SET NULL,
    compra_id          INT,
    motivo             TEXT,
    usuario_id         INT REFERENCES usuarios(id) ON DELETE SET NULL,
    usuario_nombre     VARCHAR(200),
    creado_en          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_inv_mov_tipo CHECK (tipo IN
        ('ENTRADA_COMPRA', 'SALIDA_SERVICIO', 'REVERSION_SERVICIO', 'AJUSTE_POSITIVO', 'AJUSTE_NEGATIVO')),
    CONSTRAINT chk_inv_mov_cantidad CHECK (cantidad > 0)
);

CREATE INDEX idx_inv_mov_item_local_fecha ON inventario_movimientos (item_id, local_id, creado_en DESC);
CREATE INDEX idx_inv_mov_fecha ON inventario_movimientos (creado_en DESC);

-- RN-15: a lo sumo una salida por (reserva, item); reintentos no descuentan dos veces.
-- La reversion se registra como REVERSION_SERVICIO y libera la salida (ver repositorio).
CREATE UNIQUE INDEX uq_inv_mov_salida_reserva
    ON inventario_movimientos (reserva_id, item_id) WHERE tipo = 'SALIDA_SERVICIO';

-- RN-18: idem para sesiones de plan sin reserva ligada.
CREATE UNIQUE INDEX uq_inv_mov_salida_plan_mov
    ON inventario_movimientos (plan_movimiento_id, item_id)
    WHERE tipo = 'SALIDA_SERVICIO' AND plan_movimiento_id IS NOT NULL;

-- Una sola reversion por (reserva, item): evita devolver el stock dos veces.
CREATE UNIQUE INDEX uq_inv_mov_reversion_reserva
    ON inventario_movimientos (reserva_id, item_id) WHERE tipo = 'REVERSION_SERVICIO';

COMMENT ON TABLE inventario_movimientos IS 'Kardex. cantidad siempre > 0; el signo lo da el tipo.';
