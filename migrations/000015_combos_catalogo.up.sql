ALTER TABLE combos
    ADD COLUMN IF NOT EXISTS descripcion TEXT,
    ADD COLUMN IF NOT EXISTS creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE combo_servicios cs
SET servicio_texto = s.nombre
FROM servicios s
WHERE cs.servicio_id = s.id
  AND (cs.servicio_texto IS NULL OR BTRIM(cs.servicio_texto) = '');

ALTER TABLE combos
    ADD CONSTRAINT chk_combos_costo_total_no_negativo
        CHECK (costo_total IS NULL OR costo_total >= 0),
    ADD CONSTRAINT chk_combos_sesiones_totales_positivas
        CHECK (sesiones_totales > 0);

ALTER TABLE combo_servicios
    ADD CONSTRAINT chk_combo_servicios_costo_no_negativo
        CHECK (costo IS NULL OR costo >= 0),
    ADD CONSTRAINT chk_combo_servicios_sesiones_positivas
        CHECK (sesiones > 0),
    ADD CONSTRAINT chk_combo_servicios_snapshot_servicio
        CHECK (
            servicio_id IS NOT NULL
            OR NULLIF(BTRIM(servicio_texto), '') IS NOT NULL
        );

COMMENT ON TABLE combos IS
    'Catalogo de combos sugeridos. No representa planes contratados por clientes.';
COMMENT ON TABLE combo_servicios IS
    'Servicios sugeridos dentro de un combo. Pueden referenciar servicios o guardar texto plano como snapshot.';
COMMENT ON COLUMN combo_servicios.servicio_id IS
    'Referencia opcional al servicio base usado como origen del item.';
COMMENT ON COLUMN combo_servicios.servicio_texto IS
    'Nombre del servicio mostrado en el combo.';
COMMENT ON COLUMN combo_servicios.costo IS
    'Precio unitario del servicio dentro del combo.';
COMMENT ON COLUMN combos.costo_total IS
    'Precio total sugerido del combo de catalogo.';
