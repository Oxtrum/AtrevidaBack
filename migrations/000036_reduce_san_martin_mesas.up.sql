UPDATE tipos_espacio_locales tel
SET cantidad_espacios = 2
FROM locales l
WHERE l.id = tel.local_id
  AND l.nombre = 'SAN MARTIN'
  AND tel.tipo_espacio = 'M';
