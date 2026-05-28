INSERT INTO public.usuarios (username, "password", activo, fecha_registro)
SELECT
    'admin',
    '$2a$10$5xSqoBjPpLTlvJXHjw1SPOFgAB5KeJbx3RwDbDmT5XYnlivpsSeCO',
    TRUE,
    '2026-05-28 15:48:37.658'::TIMESTAMPTZ
WHERE NOT EXISTS (
    SELECT 1
    FROM public.usuarios
    WHERE LOWER(username) = LOWER('admin')
);
