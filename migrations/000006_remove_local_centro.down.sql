INSERT INTO locales (nombre, activo)
VALUES ('CENTRO', TRUE)
ON CONFLICT (nombre) DO NOTHING;
