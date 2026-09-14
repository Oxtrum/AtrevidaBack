-- NULL conserva la modalidad desconocida de reservas anteriores, sin backfill.
ALTER TABLE reservas ADD COLUMN costo_variable BOOLEAN;
