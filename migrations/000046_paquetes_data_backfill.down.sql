-- Best-effort: desvincula tiers y borra los paquetes creados por el backfill.
UPDATE combos SET paquete_id = NULL WHERE paquete_id IS NOT NULL;
DELETE FROM paquetes;
