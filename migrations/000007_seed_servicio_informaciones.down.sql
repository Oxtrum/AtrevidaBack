DELETE FROM servicio_local
WHERE servicio_id IN (
    SELECT s.id
    FROM servicios s
    JOIN categorias c ON c.id = s.categoria_id
    WHERE LOWER(s.nombre) = LOWER('cita consulta')
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
    WHERE LOWER(s.nombre) = LOWER('cita consulta')
      AND c.nombre = 'INFORMACIONES'
      AND COALESCE(s.tiempo, '') = '30 mins'
      AND COALESCE(s.costo, 0) = 0
      AND s.sesiones = 1
);

DELETE FROM categorias
WHERE nombre = 'INFORMACIONES'
  AND NOT EXISTS (
      SELECT 1
      FROM servicios s
      WHERE s.categoria_id = categorias.id
  );
