ALTER TABLE usuarios
    ADD COLUMN IF NOT EXISTS local_id INT,
    ADD COLUMN IF NOT EXISTS nombre_local VARCHAR(100);

COMMENT ON COLUMN usuarios.local_id IS
    'ID del local asignado al usuario. Nullable y sin foreign key para conservar el dato historico si el local se elimina.';

COMMENT ON COLUMN usuarios.nombre_local IS
    'Nombre del local asignado al usuario. Se guarda desnormalizado desde locales.nombre al registrar el usuario.';
