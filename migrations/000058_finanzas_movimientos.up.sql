CREATE TABLE categorias_financieras (
    id         SERIAL PRIMARY KEY,
    nombre     VARCHAR(100) NOT NULL,
    naturaleza VARCHAR(10) NOT NULL,
    activo     BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT chk_categorias_fin_naturaleza CHECK (naturaleza IN ('INGRESO', 'EGRESO'))
);
CREATE UNIQUE INDEX uq_categorias_financieras_nombre ON categorias_financieras (LOWER(nombre), naturaleza);

CREATE SEQUENCE movimientos_financieros_codigo_seq;

CREATE TABLE movimientos_financieros (
    id                 SERIAL PRIMARY KEY,
    codigo             VARCHAR(20) NOT NULL UNIQUE
                       DEFAULT ('MOV-' || LPAD(nextval('movimientos_financieros_codigo_seq')::TEXT, 6, '0')),
    naturaleza         VARCHAR(10) NOT NULL,
    clase              VARCHAR(10),
    categoria_id       INT NOT NULL REFERENCES categorias_financieras(id) ON DELETE RESTRICT,
    gasto_fijo_id      INT,
    periodo            DATE,
    compra_id          INT,
    local_id           INT REFERENCES locales(id) ON DELETE RESTRICT,
    local_nombre       VARCHAR(100),
    fecha              DATE NOT NULL,
    monto              NUMERIC(12,2) NOT NULL,
    descripcion        TEXT,
    proveedor          VARCHAR(200),
    metodo_pago        VARCHAR(30),
    estado             VARCHAR(20) NOT NULL DEFAULT 'PAGADO',
    comprobante_key    VARCHAR(500),
    motivo_anulacion   TEXT,
    id_usuario         INT REFERENCES usuarios(id) ON DELETE SET NULL,
    nombre_usuario     VARCHAR(200),
    fecha_creacion     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_mov_fin_naturaleza CHECK (naturaleza IN ('INGRESO', 'EGRESO')),
    CONSTRAINT chk_mov_fin_clase CHECK (
        (naturaleza = 'EGRESO' AND clase IN ('FIJO', 'VARIABLE')) OR (naturaleza = 'INGRESO' AND clase IS NULL)),
    CONSTRAINT chk_mov_fin_monto CHECK (monto > 0),
    CONSTRAINT chk_mov_fin_estado CHECK (estado IN ('PENDIENTE', 'PAGADO', 'ANULADO')),
    CONSTRAINT chk_mov_fin_metodo CHECK (
        metodo_pago IS NULL OR metodo_pago IN ('efectivo', 'qr', 'transferencia', 'debito_automatico'))
);
CREATE INDEX idx_mov_fin_fecha_local ON movimientos_financieros (fecha, local_id) WHERE estado <> 'ANULADO';
CREATE INDEX idx_mov_fin_categoria ON movimientos_financieros (categoria_id);

COMMENT ON TABLE movimientos_financieros IS 'Egresos (FIJO/VARIABLE) y otros ingresos. Las ventas siguen en pagos; no se duplican aqui.';
