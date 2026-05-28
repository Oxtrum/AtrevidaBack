DELETE FROM servicio_local
WHERE servicio_id IN (
    SELECT id
    FROM servicios
    WHERE LOWER(nombre) IN (LOWER('Evaluacion gratuita'), LOWER('Evaluación gratuita'))
);

DELETE FROM servicios
WHERE LOWER(nombre) IN (LOWER('Evaluacion gratuita'), LOWER('Evaluación gratuita'))
  AND costo = 0.00
  AND sesiones = 1;
