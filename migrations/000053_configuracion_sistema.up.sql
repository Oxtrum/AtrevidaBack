-- Parametros editables por el administrador (clave/valor tipado).
CREATE TABLE configuracion_sistema (
    clave              VARCHAR(100) PRIMARY KEY,
    valor              TEXT NOT NULL,
    tipo               VARCHAR(10) NOT NULL,
    descripcion        TEXT,
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_configuracion_tipo CHECK (tipo IN ('string', 'bool', 'int'))
);

COMMENT ON TABLE configuracion_sistema IS 'Parametros del sistema (inventario y finanzas) editables solo por admin_sys.';

INSERT INTO configuracion_sistema (clave, valor, tipo, descripcion) VALUES
    ('inventario.politica_sin_stock', 'ADVERTIR', 'string', 'ADVERTIR registra la reserva con aviso; BLOQUEAR la rechaza en el panel interno.'),
    ('inventario.descontar_en_estado', 'COMPLETADO', 'string', 'Estado de la reserva en el que se descuentan los insumos: COMPLETADO o AGENDADO.'),
    ('inventario.permitir_stock_negativo', 'true', 'bool', 'Si al descontar no alcanza el stock, permite dejarlo negativo.'),
    ('inventario.considerar_comprometido', 'true', 'bool', 'Resta lo comprometido por reservas agendadas futuras al estimar disponibilidad.'),
    ('finanzas.estado_gasto_fijo_generado', 'PENDIENTE', 'string', 'Estado con el que nace el egreso generado desde un gasto fijo: PENDIENTE o PAGADO.'),
    ('finanzas.dias_aviso_gasto_fijo', '3', 'int', 'Dias antes del cobro para avisar en el dashboard.'),
    ('finanzas.zona_horaria', 'America/La_Paz', 'string', 'Zona horaria para calcular fechas de cobro.'),
    ('finanzas.moneda', 'Bs', 'string', 'Simbolo de moneda en pantallas y reportes.')
ON CONFLICT (clave) DO NOTHING;
