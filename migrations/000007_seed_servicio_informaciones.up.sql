INSERT INTO categorias (nombre)
VALUES ('INFORMACIONES')
ON CONFLICT (nombre) DO NOTHING;

WITH servicio_insertado AS (
    INSERT INTO servicios (
        nombre,
        categoria_id,
        tiempo,
        costo,
        sesiones,
        activo,
        tipo_espacio_requerido
    )
    SELECT
        'cita consulta',
        c.id,
        '30 mins',
        0,
        1,
        TRUE,
        NULL
    FROM categorias c
    WHERE c.nombre = 'INFORMACIONES'
      AND NOT EXISTS (
          SELECT 1
          FROM servicios s
          JOIN categorias cx ON cx.id = s.categoria_id
          WHERE LOWER(s.nombre) = LOWER('cita consulta')
            AND cx.nombre = 'INFORMACIONES'
            AND COALESCE(s.tiempo, '') = '30 mins'
            AND COALESCE(s.costo, 0) = 0
            AND s.sesiones = 1
      )
    RETURNING id
),
servicio_objetivo AS (
    SELECT id FROM servicio_insertado
    UNION ALL
    SELECT s.id
    FROM servicios s
    JOIN categorias c ON c.id = s.categoria_id
    WHERE LOWER(s.nombre) = LOWER('cita consulta')
      AND c.nombre = 'INFORMACIONES'
      AND COALESCE(s.tiempo, '') = '30 mins'
      AND COALESCE(s.costo, 0) = 0
      AND s.sesiones = 1
    LIMIT 1
)
INSERT INTO servicio_local (servicio_id, local_id)
SELECT so.id, l.id
FROM servicio_objetivo so
CROSS JOIN locales l
WHERE NOT EXISTS (
    SELECT 1
    FROM servicio_local sl
    WHERE sl.servicio_id = so.id
      AND sl.local_id = l.id
);
