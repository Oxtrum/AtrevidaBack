CREATE INDEX IF NOT EXISTS idx_detalle_reservas_reserva_id_id
    ON detalle_reservas (reserva_id, id);

CREATE INDEX IF NOT EXISTS idx_reservas_notificaciones_local_creado
    ON reservas (local_id, creado_en DESC, id DESC)
    WHERE activo = TRUE
      AND estado = 'AGENDADO'
      AND COALESCE(notificado, FALSE) = FALSE;

CREATE INDEX IF NOT EXISTS idx_reservas_local_fecha_hora_activas
    ON reservas (local_id, fecha, hora_desde)
    WHERE activo = TRUE;

CREATE INDEX IF NOT EXISTS idx_pagos_reporte_local_fecha
    ON pagos (local_id, fecha_creacion, id)
    WHERE activo = TRUE
      AND estado = 'PAGADO';

CREATE INDEX IF NOT EXISTS idx_pagos_reporte_fecha_local
    ON pagos (fecha_creacion, local_id, id)
    WHERE activo = TRUE
      AND estado = 'PAGADO';
