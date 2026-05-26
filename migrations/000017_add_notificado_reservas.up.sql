ALTER TABLE reservas
    ADD COLUMN IF NOT EXISTS notificado BOOLEAN DEFAULT FALSE;

UPDATE reservas
SET notificado = FALSE
WHERE notificado IS NULL;
