-- Foto del empleado: solo la clave del objeto en el storage (carpeta privada personal/), nunca el binario.
ALTER TABLE empleados ADD COLUMN IF NOT EXISTS foto_key VARCHAR(500);
COMMENT ON COLUMN empleados.foto_key IS 'Clave del objeto en el storage (personal/{id}/{uuid}.webp). Privada: se lee con URL firmada temporal.';
