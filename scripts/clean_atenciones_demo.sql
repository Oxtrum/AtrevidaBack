-- Borra los datos de prueba cargados por scripts/seed_atenciones_demo.sql (reservas con notas = 'DEMO-ATENCION'
-- y sus atenciones). Incluye las atenciones que se registren a mano sobre esas reservas al probar.
-- No toca nada que no lleve esa marca. Uso: go run ./cmd/_sql -f scripts/clean_atenciones_demo.sql
BEGIN;
UPDATE inventario_movimientos SET atencion_id = NULL
WHERE atencion_id IN (SELECT a.id FROM atenciones a JOIN reservas r ON r.id = a.reserva_id WHERE r.notas = 'DEMO-ATENCION');
DELETE FROM atenciones WHERE reserva_id IN (SELECT id FROM reservas WHERE notas = 'DEMO-ATENCION');
DELETE FROM reservas WHERE notas = 'DEMO-ATENCION';
COMMIT;
