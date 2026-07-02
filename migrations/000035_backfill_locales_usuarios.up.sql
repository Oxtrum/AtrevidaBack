UPDATE usuarios u
SET local_id = 1,
    nombre_local = 'SAN MARTIN'
FROM roles r
WHERE u.rol_id = r.id
  AND LOWER(r.codigo) <> LOWER('admin_sys');

UPDATE usuarios u
SET local_id = NULL,
    nombre_local = NULL
FROM roles r
WHERE u.rol_id = r.id
  AND LOWER(r.codigo) = LOWER('admin_sys');
