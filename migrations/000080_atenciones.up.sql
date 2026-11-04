CREATE TABLE atenciones (
    id                 SERIAL PRIMARY KEY,
    reserva_id         INT NOT NULL UNIQUE REFERENCES reservas(id) ON DELETE RESTRICT,
    empleado_id        INT REFERENCES empleados(id) ON DELETE SET NULL,
    empleado_nombre    VARCHAR(200) NOT NULL,
    fecha_atencion     DATE NOT NULL,
    hora_inicio        TIME,
    hora_fin           TIME,
    local_id           INT REFERENCES locales(id) ON DELETE SET NULL,
    lugar              VARCHAR(200) NOT NULL,
    observaciones      TEXT,
    registrado_por_id  INT REFERENCES usuarios(id) ON DELETE SET NULL,
    registrado_por     VARCHAR(200),
    creado_en          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_atenciones_horas CHECK (hora_fin IS NULL OR hora_inicio IS NULL OR hora_fin > hora_inicio)
);

CREATE INDEX idx_atenciones_fecha ON atenciones (fecha_atencion DESC, id DESC);
CREATE INDEX idx_atenciones_local ON atenciones (local_id, fecha_atencion DESC);
CREATE INDEX idx_atenciones_empleado ON atenciones (empleado_id) WHERE empleado_id IS NOT NULL;

CREATE TABLE atencion_insumos (
    id                  SERIAL PRIMARY KEY,
    atencion_id         INT NOT NULL REFERENCES atenciones(id) ON DELETE CASCADE,
    item_id             INT REFERENCES inventario_items(id) ON DELETE SET NULL,
    item_nombre         VARCHAR(200) NOT NULL,
    unidad_medida       VARCHAR(30) NOT NULL,
    cantidad_usada      NUMERIC(12,3) NOT NULL,
    cantidad_descontada NUMERIC(12,3) NOT NULL DEFAULT 0,
    estado_descuento    VARCHAR(30) NOT NULL DEFAULT 'DESCONTADO',
    CONSTRAINT chk_atencion_insumos_cantidad CHECK (cantidad_usada > 0 AND cantidad_descontada >= 0),
    CONSTRAINT chk_atencion_insumos_estado CHECK (estado_descuento IN ('DESCONTADO', 'PENDIENTE_STOCK', 'NO_CATALOGADO'))
);

ALTER TABLE inventario_movimientos ADD COLUMN atencion_id INT REFERENCES atenciones(id) ON DELETE SET NULL;
CREATE INDEX idx_inv_mov_atencion ON inventario_movimientos (atencion_id) WHERE atencion_id IS NOT NULL;
