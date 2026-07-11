-- El combo es una promocion de catalogo. Sus datos financieros no representan
-- una compra: el snapshot contractual se creara en planes en una fase posterior.

ALTER TABLE combos
    ADD COLUMN IF NOT EXISTS tipo_precio VARCHAR(20) NOT NULL DEFAULT 'PRECIO_PAQUETE',
    ADD COLUMN IF NOT EXISTS precio_paquete NUMERIC(10,2),
    ADD COLUMN IF NOT EXISTS moneda VARCHAR(3) NOT NULL DEFAULT 'BOB';

UPDATE combos
SET precio_paquete = costo_total
WHERE precio_paquete IS NULL
  AND costo_total IS NOT NULL;

UPDATE combos
SET tipo_precio = 'POR_ITEMS'
WHERE precio_paquete IS NULL;

ALTER TABLE combos
    ADD CONSTRAINT chk_combos_tipo_precio
        CHECK (tipo_precio IN ('POR_ITEMS', 'PRECIO_PAQUETE')),
    ADD CONSTRAINT chk_combos_moneda
        CHECK (moneda ~ '^[A-Z]{3}$'),
    ADD CONSTRAINT chk_combos_precio_paquete
        CHECK (
            (tipo_precio = 'POR_ITEMS' AND precio_paquete IS NULL)
            OR (tipo_precio = 'PRECIO_PAQUETE' AND precio_paquete IS NOT NULL AND precio_paquete >= 0)
        );

-- servicio_texto deja de ser un fallback de lectura: pasa a ser el nombre
-- snapshot que se muestra siempre, incluso si cambia el servicio base.
UPDATE combo_servicios cs
SET servicio_texto = s.nombre
FROM servicios s
WHERE cs.servicio_id = s.id
  AND (cs.servicio_texto IS NULL OR BTRIM(cs.servicio_texto) = '');

ALTER TABLE combo_servicios
    ALTER COLUMN servicio_texto SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_combo_servicios_orden_activo
    ON combo_servicios(combo_id, orden)
    WHERE activo = TRUE;

CREATE INDEX IF NOT EXISTS idx_combos_activo_nombre
    ON combos(activo, nombre);

COMMENT ON COLUMN combos.tipo_precio IS
    'POR_ITEMS calcula el total desde las lineas; PRECIO_PAQUETE usa precio_paquete como promocion cerrada.';
COMMENT ON COLUMN combos.precio_paquete IS
    'Precio final de la promocion cuando tipo_precio es PRECIO_PAQUETE.';
COMMENT ON COLUMN combos.moneda IS
    'Codigo ISO de tres letras de la moneda del catalogo.';
COMMENT ON COLUMN combo_servicios.servicio_texto IS
    'Nombre snapshot mostrado por el combo; no se reconstruye desde servicios.';
