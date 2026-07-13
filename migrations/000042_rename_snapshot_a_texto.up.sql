-- Corrige el split-brain de nombres de columna en planes/plan_servicios.
-- El 000039 fue reescrito in-place para renombrar *_snapshot -> *_texto, pero las
-- DBs que ya habian aplicado el 000039 viejo conservaron las columnas *_snapshot
-- (migrate las considera "aplicadas" y no re-corre el rename). Este 000042 renombra
-- de forma idempotente: solo actua cuando existe la columna vieja y no la nueva, por
-- lo que es no-op en DBs frescas que ya nacieron con *_texto.
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT * FROM (VALUES
            ('plan_servicios', 'nombre_snapshot',          'nombre_texto'),
            ('plan_servicios', 'tiempo_snapshot',          'tiempo_texto'),
            ('plan_servicios', 'precio_unitario_snapshot', 'precio_unitario_texto'),
            ('planes',         'cliente_nombre_snapshot',  'cliente_nombre_texto'),
            ('planes',         'local_nombre_snapshot',    'local_nombre_texto'),
            ('planes',         'combo_nombre_snapshot',     'combo_nombre_texto')
        ) AS t(tbl, oldc, newc)
    LOOP
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = r.tbl AND column_name = r.oldc
        ) AND NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = r.tbl AND column_name = r.newc
        ) THEN
            EXECUTE format('ALTER TABLE %I RENAME COLUMN %I TO %I', r.tbl, r.oldc, r.newc);
        END IF;
    END LOOP;
END $$;
