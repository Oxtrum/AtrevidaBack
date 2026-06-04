CREATE SEQUENCE IF NOT EXISTS pagos_codigo_pago_seq;

CREATE TABLE IF NOT EXISTS pagos (
    id                 SERIAL PRIMARY KEY,
    codigo_pago        VARCHAR(20) NOT NULL UNIQUE DEFAULT ('PAGO-' || LPAD(nextval('pagos_codigo_pago_seq')::TEXT, 6, '0')),
    local_id           INT NOT NULL REFERENCES locales(id) ON DELETE RESTRICT,
    local_nombre       VARCHAR(100) NOT NULL,
    cliente_id         INT REFERENCES clientes(id) ON DELETE SET NULL,
    cliente_nit        VARCHAR(50) NOT NULL,
    cliente_nombre     VARCHAR(200) NOT NULL,
    subtotal           NUMERIC(10,2) NOT NULL DEFAULT 0,
    descuento          NUMERIC(10,2) NOT NULL DEFAULT 0,
    total_final        NUMERIC(10,2) NOT NULL DEFAULT 0,
    estado             VARCHAR(20) NOT NULL DEFAULT 'BORRADOR',
    activo             BOOLEAN NOT NULL DEFAULT TRUE,
    fecha_creacion     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_pagos_estado
        CHECK (estado IN ('PAGADO', 'BORRADOR', 'PENDIENTE')),
    CONSTRAINT chk_pagos_importes_no_negativos
        CHECK (subtotal >= 0 AND descuento >= 0 AND total_final >= 0),
    CONSTRAINT chk_pagos_descuento_valido
        CHECK (descuento <= subtotal)
);

ALTER SEQUENCE pagos_codigo_pago_seq OWNED BY pagos.codigo_pago;

CREATE TABLE IF NOT EXISTS detalle_pagos (
    id              SERIAL PRIMARY KEY,
    pago_id         INT NOT NULL REFERENCES pagos(id) ON DELETE CASCADE,
    servicio_id     INT REFERENCES servicios(id) ON DELETE SET NULL,
    servicio        VARCHAR(500) NOT NULL,
    precio_unitario NUMERIC(10,2) NOT NULL,
    cantidad        INT NOT NULL DEFAULT 1,
    subtotal        NUMERIC(10,2) NOT NULL,

    CONSTRAINT chk_detalle_pagos_precio_no_negativo
        CHECK (precio_unitario >= 0),
    CONSTRAINT chk_detalle_pagos_cantidad_positiva
        CHECK (cantidad > 0),
    CONSTRAINT chk_detalle_pagos_subtotal_no_negativo
        CHECK (subtotal >= 0)
);

CREATE INDEX IF NOT EXISTS idx_pagos_local
    ON pagos(local_id);

CREATE INDEX IF NOT EXISTS idx_pagos_cliente
    ON pagos(cliente_id);

CREATE INDEX IF NOT EXISTS idx_pagos_estado
    ON pagos(estado);

CREATE INDEX IF NOT EXISTS idx_detalle_pagos_pago
    ON detalle_pagos(pago_id);

COMMENT ON TABLE pagos IS
    'Cabecera de pagos independiente de reservas y planes.';
COMMENT ON COLUMN pagos.codigo_pago IS
    'Codigo publico incremental del pago usado por la API.';
COMMENT ON COLUMN pagos.local_nombre IS
    'Snapshot del nombre del local al momento de registrar el pago.';
COMMENT ON COLUMN pagos.cliente_id IS
    'Referencia opcional al cliente registrado.';
COMMENT ON COLUMN pagos.estado IS
    'Estado informativo del pago: PAGADO, BORRADOR o PENDIENTE.';
COMMENT ON TABLE detalle_pagos IS
    'Detalle de servicios cobrados dentro de un pago.';
COMMENT ON COLUMN detalle_pagos.servicio_id IS
    'Referencia opcional al servicio base cobrado.';
COMMENT ON COLUMN detalle_pagos.servicio IS
    'Texto del servicio cobrado al momento de registrar el pago.';
