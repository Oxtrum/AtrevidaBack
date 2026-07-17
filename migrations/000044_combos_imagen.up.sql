-- Portada opcional del combo. Guarda el path del objeto en Supabase Storage
-- (bucket configurable), no la URL completa. La URL publica se deriva en la API.
ALTER TABLE combos
    ADD COLUMN IF NOT EXISTS imagen_path VARCHAR(300);

COMMENT ON COLUMN combos.imagen_path IS
    'Path del objeto de portada en Supabase Storage, por ejemplo combos/12. NULL si no tiene imagen.';
