INSERT INTO locales (nombre, activo) VALUES
    ('SAN MARTIN',      TRUE),
    ('PASEO ARANJUEZ',  TRUE),
    ('CENTRO',          TRUE)
ON CONFLICT (nombre) DO NOTHING;