-- Backfill idempotente: agrupa combos sin paquete por nombre-familia.
DO $$
DECLARE
    fam RECORD;
    nuevo_paquete_id INT;
BEGIN
    FOR fam IN
        SELECT
            regexp_replace(cb.nombre, '\s*-\s*\d+\s*SESI[OÓ]N(ES)?\s*$', '', 'i') AS familia,
            (array_agg(cb.id ORDER BY cb.sesiones_totales, cb.id))[1] AS primer_combo_id
        FROM combos cb
        WHERE cb.paquete_id IS NULL AND cb.activo = TRUE
        GROUP BY 1
    LOOP
        INSERT INTO paquetes (nombre, descripcion, categoria_id, imagen_path, moneda, activo)
        SELECT
            fam.familia,
            cb.descripcion, cb.categoria_id, cb.imagen_path,
            COALESCE(cb.moneda, 'BOB'), TRUE
        FROM combos cb WHERE cb.id = fam.primer_combo_id
        RETURNING id INTO nuevo_paquete_id;

        -- Servicios base = servicios del primer tier, deduplicados por (servicio_id, texto).
        INSERT INTO paquete_servicios (paquete_id, servicio_id, servicio_texto, costo, orden, activo)
        SELECT DISTINCT ON (cs.servicio_id, cs.servicio_texto)
            nuevo_paquete_id, cs.servicio_id, cs.servicio_texto, COALESCE(cs.costo, 0), cs.orden, TRUE
        FROM combo_servicios cs
        WHERE cs.combo_id = fam.primer_combo_id AND cs.activo = TRUE
        ORDER BY cs.servicio_id, cs.servicio_texto, cs.orden;

        -- Locales = union de todos los tiers de la familia.
        INSERT INTO paquete_local (paquete_id, local_id)
        SELECT DISTINCT nuevo_paquete_id, cl.local_id
        FROM combo_local cl
        JOIN combos cb ON cb.id = cl.combo_id
        WHERE regexp_replace(cb.nombre, '\s*-\s*\d+\s*SESI[OÓ]N(ES)?\s*$', '', 'i') = fam.familia
          AND cb.paquete_id IS NULL
        ON CONFLICT DO NOTHING;

        -- Enlazar todos los combos de la familia como tiers.
        UPDATE combos cb
        SET paquete_id = nuevo_paquete_id, actualizado_en = NOW()
        WHERE regexp_replace(cb.nombre, '\s*-\s*\d+\s*SESI[OÓ]N(ES)?\s*$', '', 'i') = fam.familia
          AND cb.paquete_id IS NULL;
    END LOOP;
END $$;
