ALTER TABLE combo_servicios
    DROP COLUMN IF EXISTS servicio_texto,
    ALTER COLUMN servicio_id SET NOT NULL;