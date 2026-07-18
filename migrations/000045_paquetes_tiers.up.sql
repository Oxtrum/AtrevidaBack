-- Paquetes: fuente de verdad del catalogo. Los combos pasan a ser tiers derivados.
CREATE TABLE IF NOT EXISTS paquetes (
    id             SERIAL PRIMARY KEY,
    nombre         VARCHAR(200) NOT NULL,
    descripcion    TEXT,
    categoria_id   INT REFERENCES categorias(id) ON DELETE SET NULL,
    imagen_path    VARCHAR(300),
    moneda         VARCHAR(3) NOT NULL DEFAULT 'BOB',
    activo         BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS paquete_servicios (
    id             SERIAL PRIMARY KEY,
    paquete_id     INT NOT NULL REFERENCES paquetes(id) ON DELETE CASCADE,
    servicio_id    INT REFERENCES servicios(id) ON DELETE SET NULL,
    servicio_texto VARCHAR(200),
    costo          NUMERIC(12,2) NOT NULL DEFAULT 0,
    orden          INT NOT NULL DEFAULT 0,
    activo         BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_paquete_servicios_paquete ON paquete_servicios(paquete_id);

CREATE TABLE IF NOT EXISTS paquete_local (
    paquete_id INT NOT NULL REFERENCES paquetes(id) ON DELETE CASCADE,
    local_id   INT NOT NULL REFERENCES locales(id) ON DELETE CASCADE,
    PRIMARY KEY (paquete_id, local_id)
);

ALTER TABLE combos
    ADD COLUMN IF NOT EXISTS paquete_id     INT REFERENCES paquetes(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS precio_regular NUMERIC(12,2),
    ADD COLUMN IF NOT EXISTS nota           VARCHAR(200);
CREATE INDEX IF NOT EXISTS idx_combos_paquete ON combos(paquete_id);

COMMENT ON TABLE  paquetes IS 'Familia del catalogo; fuente de verdad. Cada combo es un tier derivado.';
COMMENT ON COLUMN combos.paquete_id IS 'Tier: paquete al que pertenece este combo. NULL solo en combos legacy sin migrar.';
COMMENT ON COLUMN combos.precio_regular IS 'Precio regular tachado (promo). NULL si no hay descuento.';
COMMENT ON COLUMN combos.nota IS 'Detalle marketing corto del tier (ej. tratamiento completo de 5 semanas).';
