-- Datos reproducibles para el recorrido manual de Atenciones y Consumos (Fase 6).
-- Ejecutar solo sobre una base de pruebas. El script es idempotente: elimina y
-- vuelve a crear sus propias filas identificadas con notas 'QA-F6:*'.
--
-- psql -h localhost -p 5433 -U postgres -d atrevida_fit_test \
--      -f scripts/seed_atenciones_consumos_qa.sql

BEGIN;

DELETE FROM atenciones
WHERE reserva_id IN (SELECT id FROM reservas WHERE notas LIKE 'QA-F6:%');

DELETE FROM reservas WHERE notas LIKE 'QA-F6:%';

CREATE TEMP TABLE qa_base ON COMMIT DROP AS
SELECT
    (SELECT id FROM locales WHERE activo ORDER BY id LIMIT 1) AS local_id,
    (SELECT nombre FROM locales WHERE activo ORDER BY id LIMIT 1) AS local_nombre,
    (SELECT id FROM empleados WHERE activo ORDER BY id LIMIT 1) AS empleado_id,
    (SELECT CONCAT_WS(' ', nombre, apellido) FROM empleados WHERE activo ORDER BY id LIMIT 1) AS empleado_nombre,
    (SELECT id FROM inventario_items WHERE activo ORDER BY id LIMIT 1) AS item_a,
    (SELECT nombre FROM inventario_items WHERE activo ORDER BY id LIMIT 1) AS item_a_nombre,
    (SELECT unidad_medida FROM inventario_items WHERE activo ORDER BY id LIMIT 1) AS item_a_unidad,
    COALESCE(
        (SELECT id FROM inventario_items WHERE activo ORDER BY id OFFSET 1 LIMIT 1),
        (SELECT id FROM inventario_items WHERE activo ORDER BY id LIMIT 1)
    ) AS item_b,
    COALESCE(
        (SELECT nombre FROM inventario_items WHERE activo ORDER BY id OFFSET 1 LIMIT 1),
        (SELECT nombre FROM inventario_items WHERE activo ORDER BY id LIMIT 1)
    ) AS item_b_nombre,
    COALESCE(
        (SELECT unidad_medida FROM inventario_items WHERE activo ORDER BY id OFFSET 1 LIMIT 1),
        (SELECT unidad_medida FROM inventario_items WHERE activo ORDER BY id LIMIT 1)
    ) AS item_b_unidad;

WITH escenarios(clave, fecha, hora_desde, hora_hasta, estado) AS (
    VALUES
        ('programada', CURRENT_DATE, '09:00'::time, '10:00'::time, 'AGENDADO'),
        ('pendiente_cierre', CURRENT_DATE - 1, '10:00'::time, '11:00'::time, 'AGENDADO'),
        ('atendida', CURRENT_DATE - 2, '11:00'::time, '12:00'::time, 'COMPLETADO'),
        ('sin_insumos', CURRENT_DATE - 3, '12:00'::time, '13:00'::time, 'COMPLETADO'),
        ('futura', CURRENT_DATE + 7, '14:00'::time, '15:00'::time, 'AGENDADO')
)
INSERT INTO reservas (
    local_id, local_nombre, tipo_espacio, fecha, hora_desde, hora_hasta,
    cliente, servicio_nombre, estado, notas
)
SELECT
    b.local_id,
    b.local_nombre,
    'M',
    e.fecha,
    e.hora_desde,
    e.hora_hasta,
    'QA F6 ' || e.clave,
    'QA F6 ' || e.clave,
    e.estado,
    'QA-F6:' || e.clave
FROM qa_base b
CROSS JOIN escenarios e;

INSERT INTO atenciones (
    reserva_id, empleado_id, empleado_nombre, fecha_atencion,
    hora_inicio, hora_fin, local_id, lugar, observaciones
)
SELECT
    r.id,
    b.empleado_id,
    COALESCE(b.empleado_nombre, 'QA F6'),
    r.fecha,
    r.hora_desde,
    r.hora_hasta,
    r.local_id,
    COALESCE(b.local_nombre, 'QA F6'),
    'Fixture QA-F6'
FROM reservas r
CROSS JOIN qa_base b
WHERE r.notas IN ('QA-F6:atendida', 'QA-F6:sin_insumos');

INSERT INTO atencion_insumos (
    atencion_id, item_id, item_nombre, unidad_medida,
    cantidad_usada, cantidad_descontada, estado_descuento
)
SELECT
    a.id,
    b.item_a,
    b.item_a_nombre,
    b.item_a_unidad,
    1.000,
    1.000,
    'DESCONTADO'
FROM atenciones a
JOIN reservas r ON r.id = a.reserva_id
CROSS JOIN qa_base b
WHERE r.notas = 'QA-F6:atendida'
  AND b.item_a IS NOT NULL;

INSERT INTO atencion_insumos (
    atencion_id, item_id, item_nombre, unidad_medida,
    cantidad_usada, cantidad_descontada, estado_descuento
)
SELECT
    a.id,
    b.item_b,
    b.item_b_nombre,
    b.item_b_unidad,
    2.500,
    0,
    'PENDIENTE_STOCK'
FROM atenciones a
JOIN reservas r ON r.id = a.reserva_id
CROSS JOIN qa_base b
WHERE r.notas = 'QA-F6:atendida'
  AND b.item_b IS NOT NULL;

COMMIT;

