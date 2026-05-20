DELETE FROM servicio_local
WHERE servicio_id IN (
    SELECT s.id
    FROM servicios s
    JOIN categorias c ON c.id = s.categoria_id
    WHERE LOWER(s.nombre) = LOWER('Realizar tratamiento')
      AND c.nombre = 'INFORMACIONES'
      AND COALESCE(s.tiempo, '') = '30 mins'
      AND COALESCE(s.costo, 0) = 0
      AND s.sesiones = 1
);

DELETE FROM servicios
WHERE id IN (
    SELECT s.id
    FROM servicios s
    JOIN categorias c ON c.id = s.categoria_id
    WHERE LOWER(s.nombre) = LOWER('Realizar tratamiento')
      AND c.nombre = 'INFORMACIONES'
      AND COALESCE(s.tiempo, '') = '30 mins'
      AND COALESCE(s.costo, 0) = 0
      AND s.sesiones = 1
);

UPDATE servicios s
SET nombre = 'cita consulta'
FROM categorias c
WHERE s.categoria_id = c.id
  AND c.nombre = 'INFORMACIONES'
  AND LOWER(s.nombre) = LOWER('Evaluacion gratuita')
  AND COALESCE(s.tiempo, '') = '30 mins'
  AND COALESCE(s.costo, 0) = 0
  AND s.sesiones = 1;
