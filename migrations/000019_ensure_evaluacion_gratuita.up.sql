WITH categoria_servicios AS (
    SELECT id
    FROM categorias
    WHERE UPPER(nombre) = 'SERVICIOS'
    LIMIT 1
),
servicio_insertado AS (
    INSERT INTO servicios (
        nombre,
        categoria_id,
        tiempo,
        costo,
        sesiones,
        activo,
        tipo_espacio_requerido,
        requiere_evaluacion
    )
    SELECT
        'Evaluacion gratuita',
        categoria_servicios.id,
        '30 min',
        0.00,
        1,
        TRUE,
        'M',
        FALSE
    FROM categoria_servicios
    WHERE NOT EXISTS (
        SELECT 1
        FROM servicios
        WHERE LOWER(nombre) IN (LOWER('Evaluacion gratuita'), LOWER('Evaluación gratuita'))
    )
    RETURNING id
),
servicio_evaluacion AS (
    SELECT id
    FROM servicio_insertado
    UNION
    SELECT id
    FROM servicios
    WHERE LOWER(nombre) IN (LOWER('Evaluacion gratuita'), LOWER('Evaluación gratuita'))
)
UPDATE servicios
SET requiere_evaluacion = FALSE,
    activo = TRUE,
    tipo_espacio_requerido = COALESCE(tipo_espacio_requerido, 'M')
WHERE id IN (SELECT id FROM servicio_evaluacion);

INSERT INTO servicio_local (servicio_id, local_id)
SELECT s.id, l.id
FROM servicios s
CROSS JOIN locales l
WHERE LOWER(s.nombre) IN (LOWER('Evaluacion gratuita'), LOWER('Evaluación gratuita'))
  AND l.activo = TRUE
ON CONFLICT DO NOTHING;
