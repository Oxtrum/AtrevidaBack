-- Falla si todavia hay usuarios con este rol: primero hay que cambiarles el rol o eliminarlos.
DELETE FROM roles WHERE LOWER(codigo) = LOWER('coordinacion');
