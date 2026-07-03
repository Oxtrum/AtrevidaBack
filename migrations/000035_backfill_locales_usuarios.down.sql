UPDATE usuarios u
SET local_id = NULL,
    nombre_local = NULL
FROM roles r
WHERE u.rol_id = r.id
  AND LOWER(r.codigo) <> LOWER('admin_sys')
  AND u.local_id = 1
  AND u.nombre_local = 'SAN MARTIN';
