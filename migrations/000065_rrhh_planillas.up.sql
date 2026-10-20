-- Recursos Humanos: empleados, adelantos y planillas de sueldo.
CREATE TABLE empleados (
    id                 SERIAL PRIMARY KEY,
    nombre             VARCHAR(100) NOT NULL,
    apellido           VARCHAR(100) NOT NULL,
    ci                 VARCHAR(30),
    cargo              VARCHAR(100),
    telefono           VARCHAR(30),
    local_id           INT REFERENCES locales(id) ON DELETE RESTRICT,
    usuario_id         INT REFERENCES usuarios(id) ON DELETE SET NULL,
    sueldo_base        NUMERIC(12,2) NOT NULL,
    dia_pago           SMALLINT,
    aplica_aportes     BOOLEAN NOT NULL DEFAULT TRUE,
    metodo_pago        VARCHAR(30),
    fecha_ingreso      DATE NOT NULL,
    fecha_baja         DATE,
    activo             BOOLEAN NOT NULL DEFAULT TRUE,
    notas              TEXT,
    fecha_creacion     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_empleados_sueldo CHECK (sueldo_base > 0),
    CONSTRAINT chk_empleados_dia_pago CHECK (dia_pago IS NULL OR dia_pago BETWEEN 1 AND 31),
    CONSTRAINT chk_empleados_baja CHECK (fecha_baja IS NULL OR fecha_baja >= fecha_ingreso),
    CONSTRAINT chk_empleados_metodo CHECK (
        metodo_pago IS NULL OR metodo_pago IN ('efectivo', 'qr', 'transferencia', 'debito_automatico'))
);
COMMENT ON COLUMN empleados.dia_pago IS 'NULL = usa rrhh.dia_pago_defecto de configuracion_sistema.';

CREATE TABLE planillas (
    id                 SERIAL PRIMARY KEY,
    empleado_id        INT NOT NULL REFERENCES empleados(id) ON DELETE RESTRICT,
    tipo               VARCHAR(10) NOT NULL DEFAULT 'SUELDO',
    periodo            DATE NOT NULL,
    fecha_pago         DATE NOT NULL,
    sueldo_base        NUMERIC(12,2) NOT NULL,
    bonos              NUMERIC(12,2) NOT NULL DEFAULT 0,
    dias_falta         NUMERIC(5,2) NOT NULL DEFAULT 0,
    descuento_faltas   NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_ganado       NUMERIC(12,2) NOT NULL,
    aporte_laboral     NUMERIC(12,2) NOT NULL DEFAULT 0,
    adelantos          NUMERIC(12,2) NOT NULL DEFAULT 0,
    otros_descuentos   NUMERIC(12,2) NOT NULL DEFAULT 0,
    neto_pagar         NUMERIC(12,2) NOT NULL,
    aporte_patronal    NUMERIC(12,2) NOT NULL DEFAULT 0,
    costo_empresa      NUMERIC(12,2) NOT NULL,
    estado             VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE',
    motivo_anulacion   TEXT,
    notas              TEXT,
    fecha_creacion     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_planillas_tipo CHECK (tipo IN ('SUELDO', 'AGUINALDO')),
    CONSTRAINT chk_planillas_estado CHECK (estado IN ('PENDIENTE', 'PAGADO', 'ANULADO')),
    CONSTRAINT chk_planillas_montos CHECK (
        bonos >= 0 AND dias_falta >= 0 AND descuento_faltas >= 0 AND aporte_laboral >= 0 AND adelantos >= 0
        AND otros_descuentos >= 0 AND aporte_patronal >= 0 AND neto_pagar >= 0 AND costo_empresa > 0)
);
-- Una planilla vigente por empleado, periodo y tipo: la generacion es idempotente (una anulada se puede regenerar a mano).
CREATE UNIQUE INDEX uq_planillas_empleado_periodo ON planillas (empleado_id, periodo, tipo) WHERE estado <> 'ANULADO';
CREATE INDEX idx_planillas_periodo ON planillas (periodo);

CREATE TABLE adelantos_empleado (
    id                 SERIAL PRIMARY KEY,
    empleado_id        INT NOT NULL REFERENCES empleados(id) ON DELETE RESTRICT,
    fecha              DATE NOT NULL,
    monto              NUMERIC(12,2) NOT NULL,
    motivo             TEXT,
    planilla_id        INT REFERENCES planillas(id) ON DELETE SET NULL,
    anulado            BOOLEAN NOT NULL DEFAULT FALSE,
    fecha_creacion     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_adelantos_monto CHECK (monto > 0)
);
CREATE INDEX idx_adelantos_empleado ON adelantos_empleado (empleado_id) WHERE planilla_id IS NULL AND NOT anulado;

-- El egreso de cada planilla vive en movimientos_financieros (flujo y reportes ya lo suman).
ALTER TABLE movimientos_financieros ADD COLUMN planilla_id INT REFERENCES planillas(id) ON DELETE RESTRICT;
CREATE UNIQUE INDEX uq_mov_fin_planilla ON movimientos_financieros (planilla_id) WHERE planilla_id IS NOT NULL;

INSERT INTO configuracion_sistema (clave, valor, tipo, descripcion) VALUES
    ('rrhh.dia_pago_defecto', '1', 'int', 'Dia del mes en que se paga el sueldo si el empleado no tiene dia propio.'),
    ('rrhh.aporte_laboral_pct', '12.71', 'string', 'Porcentaje de aportes de ley retenido al empleado (AFP) sobre el total ganado.'),
    ('rrhh.aporte_patronal_pct', '16.71', 'string', 'Porcentaje de aportes patronales sobre el total ganado (costo para la empresa).'),
    ('rrhh.dias_mes_faltas', '30', 'int', 'Dias base para valorar una falta: sueldo_base / dias.')
ON CONFLICT (clave) DO NOTHING;
