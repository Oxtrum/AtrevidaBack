-- El plan es una compra viva de un cliente. Conserva snapshots contractuales de
-- lo adquirido: cliente, local, servicios, sesiones, precio y moneda. Un plan
-- puede originarse desde un combo (combo_id_origen como trazabilidad) o ser
-- completamente manual. Los cambios posteriores del catalogo no modifican
-- planes existentes.
--
-- Las columnas heredadas (cliente, combo_id, combo_nombre, sesiones_totales,
-- sesiones_usadas, costo_total, activo) se mantienen temporalmente para
-- compatibilidad con reservas historicas.

CREATE SEQUENCE IF NOT EXISTS plan_codigo_seq;

ALTER TABLE planes
    ADD COLUMN IF NOT EXISTS codigo VARCHAR(20) UNIQUE DEFAULT ('PLAN-' || LPAD(nextval('plan_codigo_seq')::TEXT, 6, '0')),
    ADD COLUMN IF NOT EXISTS cliente_id INT REFERENCES clientes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cliente_nombre_snapshot VARCHAR(200),
    ADD COLUMN IF NOT EXISTS local_nombre_snapshot VARCHAR(100),
    ADD COLUMN IF NOT EXISTS combo_id_origen INT REFERENCES combos(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS combo_nombre_snapshot VARCHAR(200),
    ADD COLUMN IF NOT EXISTS fecha_inicio DATE,
    ADD COLUMN IF NOT EXISTS fecha_fin DATE,
    ADD COLUMN IF NOT EXISTS estado VARCHAR(20) NOT NULL DEFAULT 'BORRADOR',
    ADD COLUMN IF NOT EXISTS estado_cobranza VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE',
    ADD COLUMN IF NOT EXISTS tipo_pago VARCHAR(20) NOT NULL DEFAULT 'UNICO',
    ADD COLUMN IF NOT EXISTS subtotal NUMERIC(10,2),
    ADD COLUMN IF NOT EXISTS descuento NUMERIC(10,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS precio_total NUMERIC(10,2),
    ADD COLUMN IF NOT EXISTS moneda VARCHAR(3) NOT NULL DEFAULT 'BOB',
    ADD COLUMN IF NOT EXISTS creado_por INT REFERENCES usuarios(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS actualizado_por INT REFERENCES usuarios(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMPTZ;

-- Backfill seguro: copiar datos existentes a las nuevas columnas
UPDATE planes
SET cliente_nombre_snapshot = COALESCE(NULLIF(BTRIM(cliente), ''), 'SIN NOMBRE'),
    local_nombre_snapshot = COALESCE((SELECT nombre FROM locales WHERE id = planes.local_id), ''),
    combo_id_origen = combo_id,
    combo_nombre_snapshot = combo_nombre,
    estado = CASE WHEN activo = TRUE THEN 'ACTIVO' ELSE 'CANCELADO' END,
    estado_cobranza = 'PAGADO',
    tipo_pago = 'UNICO',
    subtotal = costo_total,
    precio_total = costo_total
WHERE cliente_nombre_snapshot IS NULL;

ALTER TABLE planes
    ALTER COLUMN cliente_nombre_snapshot SET NOT NULL,
    ALTER COLUMN subtotal SET NOT NULL,
    ALTER COLUMN precio_total SET NOT NULL,
    ADD CONSTRAINT chk_planes_estado
        CHECK (estado IN ('BORRADOR', 'ACTIVO', 'COMPLETADO', 'VENCIDO', 'CANCELADO')),
    ADD CONSTRAINT chk_planes_estado_cobranza
        CHECK (estado_cobranza IN ('PENDIENTE', 'PARCIAL', 'PAGADO', 'VENCIDO')),
    ADD CONSTRAINT chk_planes_tipo_pago
        CHECK (tipo_pago IN ('UNICO', 'CUOTAS')),
    ADD CONSTRAINT chk_planes_moneda
        CHECK (moneda ~ '^[A-Z]{3}$'),
    ADD CONSTRAINT chk_planes_importes_no_negativos
        CHECK (subtotal >= 0 AND descuento >= 0 AND precio_total >= 0),
    ADD CONSTRAINT chk_planes_descuento_valido
        CHECK (descuento <= subtotal),
    ADD CONSTRAINT chk_planes_fechas
        CHECK (fecha_fin IS NULL OR fecha_inicio IS NULL OR fecha_fin >= fecha_inicio);

ALTER SEQUENCE plan_codigo_seq OWNED BY planes.codigo;

-- Indices para busqueda administrativa
CREATE INDEX IF NOT EXISTS idx_planes_codigo ON planes(codigo);
CREATE INDEX IF NOT EXISTS idx_planes_estado ON planes(estado);
CREATE INDEX IF NOT EXISTS idx_planes_estado_cobranza ON planes(estado_cobranza);
CREATE INDEX IF NOT EXISTS idx_planes_cliente_id ON planes(cliente_id);
CREATE INDEX IF NOT EXISTS idx_planes_fecha_fin ON planes(fecha_fin);
CREATE INDEX IF NOT EXISTS idx_planes_local_estado ON planes(local_id, estado);

COMMENT ON COLUMN planes.codigo IS
    'Codigo publico incremental del plan usado por la API.';
COMMENT ON COLUMN planes.cliente_id IS
    'Referencia opcional al cliente registrado.';
COMMENT ON COLUMN planes.cliente_nombre_snapshot IS
    'Snapshot del nombre del cliente al momento de contratar.';
COMMENT ON COLUMN planes.local_nombre_snapshot IS
    'Snapshot del nombre del local al momento de contratar.';
COMMENT ON COLUMN planes.combo_id_origen IS
    'Combo de catalogo que origino el plan; solo trazabilidad.';
COMMENT ON COLUMN planes.combo_nombre_snapshot IS
    'Snapshot del nombre del combo al momento de contratar.';
COMMENT ON COLUMN planes.fecha_inicio IS
    'Fecha opcional de inicio de vigencia.';
COMMENT ON COLUMN planes.fecha_fin IS
    'Fecha opcional de vencimiento. Nulo indica vigencia indefinida.';
COMMENT ON COLUMN planes.estado IS
    'Estado contractual: BORRADOR, ACTIVO, COMPLETADO, VENCIDO, CANCELADO.';
COMMENT ON COLUMN planes.estado_cobranza IS
    'Estado de cobranza: PENDIENTE, PARCIAL, PAGADO, VENCIDO.';
COMMENT ON COLUMN planes.tipo_pago IS
    'Modalidad de pago: UNICO o CUOTAS.';
COMMENT ON COLUMN planes.subtotal IS
    'Suma de precios unitarios por sesiones contratadas antes de descuento.';
COMMENT ON COLUMN planes.descuento IS
    'Descuento total aplicado al plan.';
COMMENT ON COLUMN planes.precio_total IS
    'Precio final contratado (subtotal - descuento).';

-- Plan servicios: snapshot de cada servicio contratado dentro del plan
CREATE TABLE IF NOT EXISTS plan_servicios (
    id                    SERIAL PRIMARY KEY,
    plan_id               INT NOT NULL REFERENCES planes(id) ON DELETE CASCADE,
    servicio_id_origen    INT REFERENCES servicios(id) ON DELETE SET NULL,
    nombre_snapshot       VARCHAR(500) NOT NULL,
    tiempo_snapshot       VARCHAR(50),
    precio_unitario_snapshot NUMERIC(10,2),
    sesiones_contratadas  INT NOT NULL,
    orden                 INT NOT NULL DEFAULT 0,
    creado_en             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_plan_servicios_sesiones_positivas
        CHECK (sesiones_contratadas > 0),
    CONSTRAINT chk_plan_servicios_precio_no_negativo
        CHECK (precio_unitario_snapshot IS NULL OR precio_unitario_snapshot >= 0)
);

CREATE INDEX IF NOT EXISTS idx_plan_servicios_plan ON plan_servicios(plan_id);
CREATE INDEX IF NOT EXISTS idx_plan_servicios_orden ON plan_servicios(plan_id, orden);

COMMENT ON TABLE plan_servicios IS
    'Lineas contratadas dentro de un plan. Datos congelados al momento de la compra.';
COMMENT ON COLUMN plan_servicios.servicio_id_origen IS
    'Referencia opcional al servicio de catalogo; solo trazabilidad.';
COMMENT ON COLUMN plan_servicios.nombre_snapshot IS
    'Nombre del servicio al momento de contratar.';
COMMENT ON COLUMN plan_servicios.tiempo_snapshot IS
    'Duracion del servicio al momento de contratar.';
COMMENT ON COLUMN plan_servicios.precio_unitario_snapshot IS
    'Precio unitario por sesion al momento de contratar.';
COMMENT ON COLUMN plan_servicios.sesiones_contratadas IS
    'Cantidad de sesiones contratadas para esta linea.';

-- Plan movimientos: libro de auditoria de sesiones
CREATE TABLE IF NOT EXISTS plan_movimientos (
    id              SERIAL PRIMARY KEY,
    plan_servicio_id INT NOT NULL REFERENCES plan_servicios(id) ON DELETE CASCADE,
    reserva_id      INT REFERENCES reservas(id) ON DELETE SET NULL,
    tipo            VARCHAR(20) NOT NULL,
    cantidad        INT NOT NULL,
    motivo          TEXT,
    usuario_id      INT REFERENCES usuarios(id) ON DELETE SET NULL,
    creado_en       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_plan_movimientos_tipo
        CHECK (tipo IN ('RESERVA', 'CONSUMO', 'LIBERACION', 'AJUSTE')),
    CONSTRAINT chk_plan_movimientos_cantidad
        CHECK (cantidad > 0),
    CONSTRAINT uq_plan_movimiento_reserva
        UNIQUE (plan_servicio_id, reserva_id)
);

CREATE INDEX IF NOT EXISTS idx_plan_movimientos_servicio ON plan_movimientos(plan_servicio_id);
CREATE INDEX IF NOT EXISTS idx_plan_movimientos_reserva ON plan_movimientos(reserva_id);
CREATE INDEX IF NOT EXISTS idx_plan_movimientos_tipo ON plan_movimientos(tipo);
CREATE INDEX IF NOT EXISTS idx_plan_movimientos_creado ON plan_movimientos(creado_en);

COMMENT ON TABLE plan_movimientos IS
    'Libro de movimientos de sesiones: reserva, consumo, liberacion o ajuste.';
COMMENT ON COLUMN plan_movimientos.plan_servicio_id IS
    'Linea del plan afectada por el movimiento.';
COMMENT ON COLUMN plan_movimientos.reserva_id IS
    'Reserva que origino el movimiento; nulo para ajustes manuales.';
COMMENT ON COLUMN plan_movimientos.tipo IS
    'Tipo de movimiento: RESERVA, CONSUMO, LIBERACION, AJUSTE.';
COMMENT ON COLUMN plan_movimientos.cantidad IS
    'Cantidad de sesiones afectadas (siempre positiva).';
COMMENT ON COLUMN plan_movimientos.motivo IS
    'Motivo obligatorio para AJUSTE; opcional para otros tipos.';
COMMENT ON COLUMN plan_movimientos.usuario_id IS
    'Usuario que realizo el movimiento o ajuste.';

-- Plan cuotas: deuda programada
CREATE TABLE IF NOT EXISTS plan_cuotas (
    id          SERIAL PRIMARY KEY,
    plan_id     INT NOT NULL REFERENCES planes(id) ON DELETE CASCADE,
    numero      INT NOT NULL,
    vencimiento DATE,
    monto       NUMERIC(10,2) NOT NULL,
    estado      VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE',
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_plan_cuotas_numero_positivo
        CHECK (numero > 0),
    CONSTRAINT chk_plan_cuotas_monto_no_negativo
        CHECK (monto >= 0),
    CONSTRAINT chk_plan_cuotas_estado
        CHECK (estado IN ('PENDIENTE', 'PAGADO', 'VENCIDO')),
    CONSTRAINT uq_plan_cuota_numero
        UNIQUE (plan_id, numero)
);

CREATE INDEX IF NOT EXISTS idx_plan_cuotas_plan ON plan_cuotas(plan_id);
CREATE INDEX IF NOT EXISTS idx_plan_cuotas_estado ON plan_cuotas(estado);

COMMENT ON TABLE plan_cuotas IS
    'Cuotas programadas del plan. Una cuota UNICO equivale a una sola cuota.';
COMMENT ON COLUMN plan_cuotas.numero IS
    'Numero de cuota dentro del plan (1-based).';
COMMENT ON COLUMN plan_cuotas.vencimiento IS
    'Fecha opcional de vencimiento de la cuota.';
COMMENT ON COLUMN plan_cuotas.estado IS
    'Estado de la cuota: PENDIENTE, PAGADO, VENCIDO.';

-- Plan pago aplicaciones: vincula pagos de caja con cuotas del plan
CREATE TABLE IF NOT EXISTS plan_pago_aplicaciones (
    id              SERIAL PRIMARY KEY,
    plan_cuota_id   INT NOT NULL REFERENCES plan_cuotas(id) ON DELETE CASCADE,
    pago_id         INT NOT NULL REFERENCES pagos(id) ON DELETE RESTRICT,
    monto_aplicado  NUMERIC(10,2) NOT NULL,
    creado_en       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_plan_pago_aplicaciones_monto_positivo
        CHECK (monto_aplicado > 0),
    CONSTRAINT uq_plan_pago_aplicacion
        UNIQUE (plan_cuota_id, pago_id)
);

CREATE INDEX IF NOT EXISTS idx_plan_pago_aplicaciones_cuota ON plan_pago_aplicaciones(plan_cuota_id);
CREATE INDEX IF NOT EXISTS idx_plan_pago_aplicaciones_pago ON plan_pago_aplicaciones(pago_id);

COMMENT ON TABLE plan_pago_aplicaciones IS
    'Distribucion de un pago de caja entre cuotas del plan.';

-- Agregar plan_servicio_id a reservas para integracion futura
ALTER TABLE reservas
    ADD COLUMN IF NOT EXISTS plan_servicio_id INT REFERENCES plan_servicios(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_reservas_plan_servicio ON reservas(plan_servicio_id);

COMMENT ON COLUMN reservas.plan_servicio_id IS
    'Linea de plan consumida por esta reserva. Se agregara en fase de integracion.';
