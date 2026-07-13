-- Revierte el rename *_texto -> *_snapshot de forma idempotente.
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT * FROM (VALUES
            ('plan_servicios', 'nombre_texto',          'nombre_snapshot'),
            ('plan_servicios', 'tiempo_texto',          'tiempo_snapshot'),
            ('plan_servicios', 'precio_unitario_texto', 'precio_unitario_snapshot'),
            ('planes',         'cliente_nombre_texto',  'cliente_nombre_snapshot'),
            ('planes',         'local_nombre_texto',    'local_nombre_snapshot'),
            ('planes',         'combo_nombre_texto',     'combo_nombre_snapshot')
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
