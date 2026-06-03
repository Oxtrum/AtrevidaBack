CREATE TABLE IF NOT EXISTS categorias_locales (
    categoria_id INT NOT NULL REFERENCES categorias(id) ON DELETE CASCADE,
    local_id     INT NOT NULL REFERENCES locales(id) ON DELETE CASCADE,
    PRIMARY KEY (categoria_id, local_id)
);

INSERT INTO categorias_locales (categoria_id, local_id)
SELECT c.id, l.id
FROM categorias c
CROSS JOIN locales l
WHERE l.activo = TRUE
ON CONFLICT DO NOTHING;
