ALTER TABLE combo_servicios
    ADD COLUMN IF NOT EXISTS servicio_texto VARCHAR(500),
    ALTER COLUMN servicio_id DROP NOT NULL;