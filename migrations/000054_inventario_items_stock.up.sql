CREATE TABLE categorias_inventario (
    id     SERIAL PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL,
    activo BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE UNIQUE INDEX uq_categorias_inventario_nombre ON categorias_inventario (LOWER(nombre));

CREATE SEQUENCE inventario_items_codigo_seq;

CREATE TABLE inventario_items (
    id                 SERIAL PRIMARY KEY,
    codigo             VARCHAR(30) NOT NULL UNIQUE
                       DEFAULT ('INV-' || LPAD(nextval('inventario_items_codigo_seq')::TEXT, 6, '0')),
    nombre             VARCHAR(200) NOT NULL,
    descripcion        TEXT,
    categoria_id       INT REFERENCES categorias_inventario(id) ON DELETE SET NULL,
    unidad_medida      VARCHAR(20) NOT NULL DEFAULT 'UNIDAD',
    costo_referencia   NUMERIC(12,2),
    stock_minimo       NUMERIC(12,3) NOT NULL DEFAULT 0,
    imagen_key         VARCHAR(500),
    activo             BOOLEAN NOT NULL DEFAULT TRUE,
    fecha_creacion     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_inventario_items_unidad CHECK (unidad_medida IN ('UNIDAD', 'ML', 'GR', 'CAJA')),
    CONSTRAINT chk_inventario_items_stock_minimo CHECK (stock_minimo >= 0),
    CONSTRAINT chk_inventario_items_costo CHECK (costo_referencia IS NULL OR costo_referencia >= 0)
);
CREATE INDEX idx_inventario_items_categoria ON inventario_items (categoria_id) WHERE activo = TRUE;

CREATE TABLE inventario_stock (
    item_id            INT NOT NULL REFERENCES inventario_items(id) ON DELETE RESTRICT,
    local_id           INT NOT NULL REFERENCES locales(id) ON DELETE RESTRICT,
    cantidad           NUMERIC(12,3) NOT NULL DEFAULT 0,
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (item_id, local_id)
);

COMMENT ON COLUMN inventario_items.imagen_key IS 'Clave del objeto en el storage; nunca el binario. La URL se arma en backend.';
COMMENT ON TABLE inventario_stock IS 'Stock por item y local. Solo se modifica junto con un movimiento de kardex (RN-02).';
