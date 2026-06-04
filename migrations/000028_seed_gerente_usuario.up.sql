INSERT INTO public.usuarios (username, "password", activo, fecha_registro, rol_id)
SELECT
    'gerente',
    '$2a$10$YRycp1pv8XML2thO37g1TehYR8zn8Wdx.uwfv8jEXp4T4GIkHuTIy',
    TRUE,
    NOW(),
    r.id
FROM public.roles r
WHERE LOWER(r.codigo) = LOWER('gerencia')
  AND NOT EXISTS (
      SELECT 1
      FROM public.usuarios u
      WHERE LOWER(u.username) = LOWER('gerente')
  );
