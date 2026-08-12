CREATE INDEX IF NOT EXISTS idx_pagos_paginacion_fecha_id
    ON pagos (fecha_creacion DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_planes_paginacion_fecha_id
    ON planes (creado_en DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_planes_local_paginacion_fecha_id
    ON planes (local_id, creado_en DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_clientes_paginacion_nombre_id
    ON clientes (apellido, nombre, id);

CREATE INDEX IF NOT EXISTS idx_reservas_paginacion_activas
    ON reservas (local_nombre, fecha, hora_desde, id)
    WHERE activo = TRUE;

CREATE INDEX IF NOT EXISTS idx_paquetes_paginacion_nombre_id
    ON paquetes (nombre, id);

CREATE INDEX IF NOT EXISTS idx_combos_paginacion_activos
    ON combos (nombre, id)
    WHERE activo = TRUE;
