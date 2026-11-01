-- Cuenta del cliente (PLAN_CUENTA_CLIENTE_Y_AJUSTES §5.3): cargos (lo que debe), abonos (lo que entrega) y
-- aplicaciones (que parte de un abono cubre que cargo). El saldo no se guarda: se calcula.

-- Un abono es un pago de un cliente registrado, sin detalle de servicios, que se reparte entre sus cargos.
-- Las ventas de mostrador (con detalle) siguen siendo es_abono = FALSE y no mueven el saldo.
ALTER TABLE pagos ADD COLUMN IF NOT EXISTS es_abono BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE pagos ADD CONSTRAINT chk_pagos_abono_con_cliente
    CHECK (NOT es_abono OR cliente_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pagos_abono_cliente
    ON pagos (cliente_id) WHERE es_abono = TRUE AND activo = TRUE;

COMMENT ON COLUMN pagos.es_abono IS
    'Pago a cuenta de un cliente registrado: no tiene detalle y se reparte entre sus cargos con pago_aplicaciones.';

CREATE TABLE IF NOT EXISTS cargos_cliente (
    id               SERIAL PRIMARY KEY,
    cliente_id       INT NOT NULL REFERENCES clientes(id) ON DELETE RESTRICT,
    local_id         INT REFERENCES locales(id) ON DELETE SET NULL,
    tipo             VARCHAR(12) NOT NULL,
    concepto         VARCHAR(300) NOT NULL,
    monto            NUMERIC(10,2) NOT NULL,
    -- Cuando ocurrio el hecho que se cobra; editable al crear. registrado_en lo pone el servidor.
    fecha_hecho      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    registrado_en    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reserva_id       INT REFERENCES reservas(id) ON DELETE SET NULL,
    plan_id          INT REFERENCES planes(id) ON DELETE SET NULL,
    anulado          BOOLEAN NOT NULL DEFAULT FALSE,
    motivo_anulacion TEXT,
    anulado_en       TIMESTAMPTZ,
    creado_por_id    INT,
    creado_por       VARCHAR(200),

    CONSTRAINT chk_cargos_cliente_tipo
        CHECK (tipo IN ('PAQUETE', 'SERVICIO', 'INSUMO', 'OTRO')),
    -- Un cargo en 0 es valido: consumo sin cobro que igual queda registrado.
    CONSTRAINT chk_cargos_cliente_monto
        CHECK (monto >= 0),
    CONSTRAINT chk_cargos_cliente_anulacion
        CHECK (NOT anulado OR motivo_anulacion IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_cargos_cliente_cliente
    ON cargos_cliente (cliente_id, fecha_hecho, id) WHERE anulado = FALSE;
CREATE INDEX IF NOT EXISTS idx_cargos_cliente_reserva
    ON cargos_cliente (reserva_id) WHERE reserva_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cargos_cliente_plan
    ON cargos_cliente (plan_id) WHERE plan_id IS NOT NULL;

COMMENT ON TABLE cargos_cliente IS 'Lo que un cliente debe: paquete, servicio, insumo u otro. Se anula con motivo, nunca se borra.';
COMMENT ON COLUMN cargos_cliente.fecha_hecho IS 'Cuando ocurrio el hecho que se cobra; ordena tablas y reportes.';
COMMENT ON COLUMN cargos_cliente.registrado_en IS 'Momento real del registro; lo pone el servidor y no se edita.';

CREATE TABLE IF NOT EXISTS pago_aplicaciones (
    id            SERIAL PRIMARY KEY,
    pago_id       INT NOT NULL REFERENCES pagos(id) ON DELETE CASCADE,
    cargo_id      INT NOT NULL REFERENCES cargos_cliente(id) ON DELETE CASCADE,
    monto         NUMERIC(10,2) NOT NULL,
    registrado_en TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_pago_aplicaciones_monto CHECK (monto > 0),
    -- Un abono aplica a un cargo en una sola fila; aplicar mas suma al monto existente.
    CONSTRAINT uq_pago_aplicaciones UNIQUE (pago_id, cargo_id)
);

CREATE INDEX IF NOT EXISTS idx_pago_aplicaciones_cargo ON pago_aplicaciones (cargo_id);

COMMENT ON TABLE pago_aplicaciones IS 'Que parte de un abono cubre que cargo. Un abono no aplica mas que su monto ni un cargo recibe mas que el suyo (se valida en la aplicacion).';
