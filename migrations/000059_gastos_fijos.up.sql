CREATE TABLE gastos_fijos (
    id                 SERIAL PRIMARY KEY,
    nombre             VARCHAR(200) NOT NULL,
    categoria_id       INT NOT NULL REFERENCES categorias_financieras(id) ON DELETE RESTRICT,
    monto              NUMERIC(12,2) NOT NULL,
    dia_cobro          SMALLINT NOT NULL,
    local_id           INT REFERENCES locales(id) ON DELETE RESTRICT,
    metodo_pago        VARCHAR(30),
    proveedor          VARCHAR(200),
    fecha_inicio       DATE NOT NULL,
    fecha_fin          DATE,
    activo             BOOLEAN NOT NULL DEFAULT TRUE,
    notas              TEXT,
    fecha_creacion     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_gastos_fijos_monto CHECK (monto > 0),
    CONSTRAINT chk_gastos_fijos_dia CHECK (dia_cobro BETWEEN 1 AND 31),
    CONSTRAINT chk_gastos_fijos_vigencia CHECK (fecha_fin IS NULL OR fecha_fin >= fecha_inicio),
    CONSTRAINT chk_gastos_fijos_metodo CHECK (
        metodo_pago IS NULL OR metodo_pago IN ('efectivo', 'qr', 'transferencia', 'debito_automatico'))
);

ALTER TABLE movimientos_financieros
    ADD CONSTRAINT fk_mov_fin_gasto_fijo FOREIGN KEY (gasto_fijo_id) REFERENCES gastos_fijos(id) ON DELETE RESTRICT;

-- RN-32: un solo egreso por (gasto fijo, periodo); la generacion es idempotente.
CREATE UNIQUE INDEX uq_mov_fin_gasto_fijo_periodo
    ON movimientos_financieros (gasto_fijo_id, periodo) WHERE gasto_fijo_id IS NOT NULL;
