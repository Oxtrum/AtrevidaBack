CREATE SEQUENCE compras_inventario_codigo_seq;

CREATE TABLE compras_inventario (
    id               SERIAL PRIMARY KEY,
    codigo           VARCHAR(20) NOT NULL UNIQUE
                     DEFAULT ('COMP-' || LPAD(nextval('compras_inventario_codigo_seq')::TEXT, 6, '0')),
    local_id         INT NOT NULL REFERENCES locales(id) ON DELETE RESTRICT,
    proveedor        VARCHAR(200),
    fecha            DATE NOT NULL,
    total            NUMERIC(12,2) NOT NULL,
    estado           VARCHAR(20) NOT NULL DEFAULT 'REGISTRADA',
    notas            TEXT,
    motivo_anulacion TEXT,
    id_usuario       INT REFERENCES usuarios(id) ON DELETE SET NULL,
    nombre_usuario   VARCHAR(200),
    fecha_creacion   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_compras_total CHECK (total >= 0),
    CONSTRAINT chk_compras_estado CHECK (estado IN ('REGISTRADA', 'ANULADA'))
);
CREATE INDEX idx_compras_local_fecha ON compras_inventario (local_id, fecha DESC);

CREATE TABLE detalle_compras (
    id             SERIAL PRIMARY KEY,
    compra_id      INT NOT NULL REFERENCES compras_inventario(id) ON DELETE CASCADE,
    item_id        INT NOT NULL REFERENCES inventario_items(id) ON DELETE RESTRICT,
    cantidad       NUMERIC(12,3) NOT NULL,
    costo_unitario NUMERIC(12,2) NOT NULL,
    subtotal       NUMERIC(12,2) NOT NULL,
    CONSTRAINT chk_detalle_compras_cantidad CHECK (cantidad > 0),
    CONSTRAINT chk_detalle_compras_costo CHECK (costo_unitario >= 0),
    CONSTRAINT chk_detalle_compras_subtotal CHECK (subtotal >= 0)
);
CREATE INDEX idx_detalle_compras_compra ON detalle_compras (compra_id);

ALTER TABLE inventario_movimientos
    ADD CONSTRAINT fk_inv_mov_compra FOREIGN KEY (compra_id) REFERENCES compras_inventario(id) ON DELETE SET NULL;
ALTER TABLE movimientos_financieros
    ADD CONSTRAINT fk_mov_fin_compra FOREIGN KEY (compra_id) REFERENCES compras_inventario(id) ON DELETE SET NULL;
CREATE INDEX idx_mov_fin_compra ON movimientos_financieros (compra_id) WHERE compra_id IS NOT NULL;
