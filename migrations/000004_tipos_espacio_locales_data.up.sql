-- 000003_tipos_espacio_locales_data.up.sql
-- SAN MARTIN (id=1): 3 mesas, 1 bicicleta
-- PASEO ARANJUEZ (id=2): 4 mesas

INSERT INTO tipos_espacio_locales (tipo_espacio, cantidad_espacios, local_id)
VALUES
    ('M', 3, 1),
    ('B', 1, 1),
    ('M', 4, 2)
ON CONFLICT (tipo_espacio, local_id) DO UPDATE
    SET cantidad_espacios = EXCLUDED.cantidad_espacios;