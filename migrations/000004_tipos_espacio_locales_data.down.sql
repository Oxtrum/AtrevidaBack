-- 000003_tipos_espacio_locales_data.down.sql
DELETE FROM tipos_espacio_locales
WHERE (tipo_espacio = 'M' AND local_id = 1)
   OR (tipo_espacio = 'B' AND local_id = 1)
   OR (tipo_espacio = 'M' AND local_id = 2);