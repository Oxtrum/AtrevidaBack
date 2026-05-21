ALTER TABLE servicios
ADD COLUMN IF NOT EXISTS requiere_evaluacion BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE servicios s
SET requiere_evaluacion = FALSE
WHERE LOWER(s.nombre) IN (
    LOWER('Evaluacion gratuita'),
    LOWER('Evaluación gratuita')
)
OR s.tipo_espacio_requerido = 'B'
OR LOWER(s.nombre) LIKE LOWER('%E-Pulse Bike%')
OR LOWER(s.nombre) IN (
    LOWER('Limpieza Facial'),
    LOWER('Limpieza Facial Premium')
);

UPDATE servicios
SET activo = FALSE,
    requiere_evaluacion = TRUE
WHERE LOWER(nombre) LIKE LOWER('%E-Pulse Bike%')
  AND sesiones > 1;

UPDATE servicios s
SET activo = FALSE,
    requiere_evaluacion = TRUE
FROM categorias c
WHERE s.categoria_id = c.id
  AND UPPER(c.nombre) LIKE '%COMBO%';

ALTER TABLE reservas
ADD COLUMN IF NOT EXISTS servicio_solicitado VARCHAR(200),
ADD COLUMN IF NOT EXISTS servicio_confirmado VARCHAR(200);

UPDATE reservas
SET servicio_solicitado = servicio_nombre
WHERE servicio_solicitado IS NULL
  AND servicio_nombre IS NOT NULL;
