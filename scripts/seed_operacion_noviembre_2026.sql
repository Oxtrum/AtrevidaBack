-- Carga operativa sintética para atrevida_fit_test.
-- Cubre la semana actual y la siguiente con reservas, cierres, cancelaciones,
-- inasistencias, insumos y stock visibles en las vistas administrativas.
--
-- Es idempotente: solo reemplaza reservas y atenciones marcadas con
-- "CARGA-NOV-2026:". No ejecutar sobre una base de producción.
--
-- psql -w -h localhost -p 5433 -U postgres -d atrevida_fit_test \
--   -f scripts/seed_operacion_noviembre_2026.sql

BEGIN;

DELETE FROM atenciones
WHERE reserva_id IN (
    SELECT id FROM reservas WHERE notas LIKE 'CARGA-NOV-2026:%'
);
DELETE FROM reservas WHERE notas LIKE 'CARGA-NOV-2026:%';

CREATE TEMP TABLE carga_locales ON COMMIT DROP AS
SELECT id, nombre
FROM locales
WHERE activo = TRUE AND nombre IN ('SAN MARTIN', 'PASEO ARANJUEZ');

DO $$
BEGIN
    IF (SELECT COUNT(*) FROM carga_locales) <> 2 THEN
        RAISE EXCEPTION 'La carga requiere los locales SAN MARTIN y PASEO ARANJUEZ';
    END IF;
END $$;

-- Personal verosímil para probar filtros y el calendario por personal.
INSERT INTO empleados (nombre, apellido, ci, cargo, telefono, local_id, sueldo_base, fecha_ingreso, activo, notas)
SELECT v.nombre, v.apellido, v.ci, v.cargo, v.telefono, l.id, v.sueldo, DATE '2025-02-01', TRUE, 'Carga operativa noviembre 2026'
FROM (
    VALUES
        ('Camila', 'Rojas', '7439218', 'Terapeuta corporal', '70145822', 'SAN MARTIN', 4200.00::numeric),
        ('Lucía', 'Fernández', '6893451', 'Cosmetóloga', '70961347', 'PASEO ARANJUEZ', 4500.00::numeric),
        ('Mariana', 'Salvatierra', '8127604', 'Fisioterapeuta', '72189436', 'SAN MARTIN', 4800.00::numeric),
        ('Paola', 'Vargas', '7650389', 'Auxiliar de cabina', '77412659', 'PASEO ARANJUEZ', 3600.00::numeric)
) AS v(nombre, apellido, ci, cargo, telefono, local_nombre, sueldo)
JOIN carga_locales l ON l.nombre = v.local_nombre
WHERE NOT EXISTS (SELECT 1 FROM empleados e WHERE e.ci = v.ci);

CREATE TEMP TABLE carga_personal ON COMMIT DROP AS
SELECT e.id, CONCAT_WS(' ', e.nombre, e.apellido) AS nombre, l.nombre AS local_nombre
FROM empleados e
JOIN locales l ON l.id = e.local_id
WHERE e.ci IN ('7439218', '6893451', '8127604', '7650389') AND e.activo = TRUE;

-- Catálogo suficientemente variado para reservas, consumos y configuración de servicios.
INSERT INTO inventario_items (nombre, descripcion, unidad_medida, costo_referencia, stock_minimo, activo)
SELECT v.nombre, 'Carga operativa noviembre 2026', v.unidad, v.costo, v.minimo, TRUE
FROM (
    VALUES
        ('Aceite de masaje de almendras 1 L', 'ML', 0.09::numeric, 30.000::numeric),
        ('Gel conductor profesional 5 L', 'ML', 0.07::numeric, 100.000::numeric),
        ('Mascarilla hidratante con ácido hialurónico', 'UNIDAD', 18.00::numeric, 12.000::numeric),
        ('Guantes de nitrilo talla M', 'UNIDAD', 0.85::numeric, 40.000::numeric),
        ('Crema reductora con cafeína 500 ml', 'ML', 0.11::numeric, 50.000::numeric),
        ('Toalla desechable premium', 'UNIDAD', 1.20::numeric, 25.000::numeric)
) AS v(nombre, unidad, costo, minimo)
WHERE NOT EXISTS (SELECT 1 FROM inventario_items i WHERE i.nombre = v.nombre);

CREATE TEMP TABLE carga_items ON COMMIT DROP AS
SELECT i.id, i.nombre, i.unidad_medida,
       CASE i.nombre
           WHEN 'Aceite de masaje de almendras 1 L' THEN 180.000::numeric
           WHEN 'Gel conductor profesional 5 L' THEN 260.000::numeric
           WHEN 'Mascarilla hidratante con ácido hialurónico' THEN 28.000::numeric
           WHEN 'Guantes de nitrilo talla M' THEN 180.000::numeric
           WHEN 'Crema reductora con cafeína 500 ml' THEN 40.000::numeric
           WHEN 'Toalla desechable premium' THEN 90.000::numeric
       END AS stock_objetivo
FROM inventario_items i
WHERE i.nombre IN (
    'Aceite de masaje de almendras 1 L', 'Gel conductor profesional 5 L',
    'Mascarilla hidratante con ácido hialurónico', 'Guantes de nitrilo talla M',
    'Crema reductora con cafeína 500 ml', 'Toalla desechable premium'
);

INSERT INTO inventario_stock (item_id, local_id, cantidad)
SELECT i.id, l.id, i.stock_objetivo
FROM carga_items i CROSS JOIN carga_locales l
ON CONFLICT (item_id, local_id) DO UPDATE
SET cantidad = GREATEST(inventario_stock.cantidad, EXCLUDED.cantidad), fecha_modificacion = NOW();

INSERT INTO servicios (nombre, tiempo, costo, sesiones, activo, tipo_espacio_requerido, requiere_evaluacion, costo_variable)
SELECT v.nombre, v.tiempo, v.costo, 1, TRUE, v.tipo, FALSE, FALSE
FROM (
    VALUES
        ('Masaje descontracturante integral', '60 min', 280.00::numeric, 'M'),
        ('Drenaje linfático manual', '60 min', 320.00::numeric, 'M'),
        ('Limpieza facial revitalizante', '75 min', 360.00::numeric, 'M'),
        ('Radiofrecuencia corporal reafirmante', '50 min', 390.00::numeric, 'M'),
        ('Sesión de presoterapia', '45 min', 230.00::numeric, 'B'),
        ('Ritual relajante con piedras calientes', '90 min', 430.00::numeric, 'M')
) AS v(nombre, tiempo, costo, tipo)
WHERE NOT EXISTS (SELECT 1 FROM servicios s WHERE s.nombre = v.nombre);

CREATE TEMP TABLE carga_servicios ON COMMIT DROP AS
SELECT id, nombre, tipo_espacio_requerido
FROM servicios
WHERE nombre IN (
    'Masaje descontracturante integral', 'Drenaje linfático manual', 'Limpieza facial revitalizante',
    'Radiofrecuencia corporal reafirmante', 'Sesión de presoterapia', 'Ritual relajante con piedras calientes'
);

-- Insumos sugeridos para que el formulario de Atención también tenga datos útiles.
INSERT INTO servicio_insumos (servicio_id, item_id, cantidad_por_sesion)
SELECT s.id, i.id,
       CASE
           WHEN s.nombre = 'Limpieza facial revitalizante' AND i.nombre = 'Mascarilla hidratante con ácido hialurónico' THEN 1.000
           WHEN s.nombre = 'Sesión de presoterapia' AND i.nombre = 'Toalla desechable premium' THEN 1.000
           WHEN i.nombre = 'Guantes de nitrilo talla M' THEN 2.000
           WHEN s.nombre = 'Radiofrecuencia corporal reafirmante' AND i.nombre = 'Gel conductor profesional 5 L' THEN 35.000
           WHEN s.nombre = 'Drenaje linfático manual' AND i.nombre = 'Crema reductora con cafeína 500 ml' THEN 18.000
           WHEN i.nombre = 'Aceite de masaje de almendras 1 L' THEN 25.000
           ELSE 0
       END
FROM carga_servicios s
JOIN carga_items i ON (
    i.nombre = 'Guantes de nitrilo talla M'
    OR (s.nombre IN ('Masaje descontracturante integral', 'Ritual relajante con piedras calientes') AND i.nombre = 'Aceite de masaje de almendras 1 L')
    OR (s.nombre = 'Drenaje linfático manual' AND i.nombre = 'Crema reductora con cafeína 500 ml')
    OR (s.nombre = 'Limpieza facial revitalizante' AND i.nombre = 'Mascarilla hidratante con ácido hialurónico')
    OR (s.nombre = 'Radiofrecuencia corporal reafirmante' AND i.nombre = 'Gel conductor profesional 5 L')
    OR (s.nombre = 'Sesión de presoterapia' AND i.nombre = 'Toalla desechable premium')
)
WHERE NOT EXISTS (
    SELECT 1 FROM servicio_insumos si WHERE si.servicio_id = s.id AND si.item_id = i.id
);

-- 42 reservas: tres por día desde ayer hasta dentro de doce días. Las fechas
-- abarcan toda la semana actual y la próxima sin llenar artificialmente cada hora.
WITH agenda AS (
    SELECT
        ROW_NUMBER() OVER (ORDER BY d, slot) AS n,
        CURRENT_DATE + d AS fecha,
        CASE slot WHEN 1 THEN '09:00'::time WHEN 2 THEN '11:00'::time ELSE '15:30'::time END AS hora_desde,
        CASE slot WHEN 1 THEN '10:00'::time WHEN 2 THEN '12:00'::time ELSE '16:30'::time END AS hora_hasta,
        d,
        slot
    FROM generate_series(-1, 12) AS d
    CROSS JOIN generate_series(1, 3) AS slot
), preparados AS (
    SELECT a.*,
        (ARRAY['Valeria Quiroga','Daniela Méndez','Sofía Arce','María José Paredes','Carolina Lema','Andrea Villarroel','Natalia Suárez','Fernanda Céspedes','Paola Rivero','Gabriela Molina','Mónica Flores','Elena Cárdenas','Verónica Alba','Laura Sánchez','Renata Blanco'])[1 + ((a.n - 1) % 15)] AS cliente,
        CASE
            WHEN a.d < 0 OR (a.d = 0 AND a.slot = 1) THEN 'COMPLETADO'
            WHEN a.d = 0 AND a.slot = 2 THEN 'CANCELADO'
            WHEN a.d > 0 AND a.n % 11 = 0 THEN 'RECHAZADO'
            WHEN a.d > 0 AND a.n % 9 = 0 THEN 'CANCELADO'
            WHEN a.d > 0 AND a.n % 7 = 0 THEN 'PENDIENTE'
            ELSE 'AGENDADO'
        END AS estado
    FROM agenda a
), con_servicio AS (
    SELECT p.*, s.nombre AS servicio, COALESCE(s.tipo_espacio_requerido, 'M') AS tipo_espacio
    FROM preparados p
    JOIN LATERAL (
        SELECT * FROM carga_servicios ORDER BY id OFFSET ((p.n - 1) % 6) LIMIT 1
    ) s ON TRUE
), con_local AS (
    SELECT cs.*, l.id AS local_id, l.nombre AS local_nombre
    FROM con_servicio cs
    JOIN LATERAL (
        SELECT * FROM carga_locales ORDER BY id OFFSET ((cs.n - 1) % 2) LIMIT 1
    ) l ON TRUE
)
INSERT INTO reservas (
    local_id, local_nombre, tipo_espacio, fecha, hora_desde, hora_hasta, cliente,
    numero_telefono, telefono_e164, servicio_nombre, servicio_solicitado, servicio_confirmado,
    estado, notas, notificado, motivo_cancelacion_tipo, motivo_cancelacion_detalle,
    cancelado_en, cancelado_por
)
SELECT local_id, local_nombre, tipo_espacio, fecha, hora_desde, hora_hasta, cliente,
       '7' || LPAD((6100000 + n)::text, 7, '0'), '+5917' || LPAD((6100000 + n)::text, 7, '0'),
       servicio, servicio, servicio, estado,
       'CARGA-NOV-2026:reserva-' || n, (n % 3 = 0),
       CASE WHEN estado = 'CANCELADO' AND n % 2 = 0 THEN 'INASISTENCIA'
            WHEN estado = 'CANCELADO' THEN 'REPROGRAMACION' END,
       CASE WHEN estado = 'CANCELADO' AND n % 2 <> 0 THEN 'La clienta solicitó reprogramar por un compromiso familiar.' END,
       CASE WHEN estado = 'CANCELADO' THEN NOW() - INTERVAL '2 hours' END,
       CASE WHEN estado = 'CANCELADO' THEN 'Mariana Salvatierra' END
FROM con_local;

-- Refuerzo de la Jornada de hoy: permite revisar simultáneamente todos los
-- resultados operativos sin depender de la hora exacta en que se pruebe.
WITH jornada(cliente, local_nombre, servicio, hora_desde, hora_hasta, estado, motivo, detalle) AS (
    VALUES
        ('Isabel Montaño', 'SAN MARTIN', 'Masaje descontracturante integral', '08:00'::time, '09:00'::time, 'COMPLETADO', NULL::text, NULL::text),
        ('Teresa Ávila', 'PASEO ARANJUEZ', 'Limpieza facial revitalizante', '09:00'::time, '10:15'::time, 'COMPLETADO', NULL::text, NULL::text),
        ('Marcela Rosales', 'SAN MARTIN', 'Drenaje linfático manual', '10:00'::time, '11:00'::time, 'AGENDADO', NULL::text, NULL::text),
        ('Natalia Cabrera', 'PASEO ARANJUEZ', 'Sesión de presoterapia', '11:00'::time, '11:45'::time, 'AGENDADO', NULL::text, NULL::text),
        ('Silvia Taboada', 'SAN MARTIN', 'Radiofrecuencia corporal reafirmante', '12:00'::time, '12:50'::time, 'CANCELADO', 'INASISTENCIA', NULL::text),
        ('Carla Jiménez', 'PASEO ARANJUEZ', 'Ritual relajante con piedras calientes', '13:00'::time, '14:30'::time, 'CANCELADO', 'REPROGRAMACION', 'La clienta solicitó cambiar su atención para la próxima semana.'),
        ('Paula Valdivia', 'SAN MARTIN', 'Masaje descontracturante integral', '14:00'::time, '15:00'::time, 'AGENDADO', NULL::text, NULL::text),
        ('Beatriz Lanza', 'PASEO ARANJUEZ', 'Limpieza facial revitalizante', '15:00'::time, '16:15'::time, 'AGENDADO', NULL::text, NULL::text),
        ('Ana Belén Rocha', 'SAN MARTIN', 'Sesión de presoterapia', '16:00'::time, '16:45'::time, 'AGENDADO', NULL::text, NULL::text),
        ('Mónica Delgadillo', 'PASEO ARANJUEZ', 'Drenaje linfático manual', '17:00'::time, '18:00'::time, 'AGENDADO', NULL::text, NULL::text),
        ('Lorena Mercado', 'SAN MARTIN', 'Radiofrecuencia corporal reafirmante', '18:00'::time, '18:50'::time, 'AGENDADO', NULL::text, NULL::text)
)
INSERT INTO reservas (
    local_id, local_nombre, tipo_espacio, fecha, hora_desde, hora_hasta, cliente,
    numero_telefono, telefono_e164, servicio_nombre, servicio_solicitado, servicio_confirmado,
    estado, notas, notificado, motivo_cancelacion_tipo, motivo_cancelacion_detalle,
    cancelado_en, cancelado_por
)
SELECT l.id, l.nombre, COALESCE(s.tipo_espacio_requerido, 'M'), CURRENT_DATE, j.hora_desde, j.hora_hasta,
       j.cliente, '76' || LPAD(ROW_NUMBER() OVER ()::text, 6, '0'), '+59176' || LPAD(ROW_NUMBER() OVER ()::text, 6, '0'),
       s.nombre, s.nombre, s.nombre, j.estado,
       'CARGA-NOV-2026:jornada-hoy-' || ROW_NUMBER() OVER (), TRUE,
       j.motivo, j.detalle,
       CASE WHEN j.estado = 'CANCELADO' THEN NOW() - INTERVAL '30 minutes' END,
       CASE WHEN j.estado = 'CANCELADO' THEN 'Mariana Salvatierra' END
FROM jornada j
JOIN carga_locales l ON l.nombre = j.local_nombre
JOIN carga_servicios s ON s.nombre = j.servicio;

-- Cierres reales de ayer y de la mañana actual, con dos personas distintas.
INSERT INTO atenciones (
    reserva_id, empleado_id, empleado_nombre, fecha_atencion, hora_inicio, hora_fin,
    local_id, lugar, observaciones, registrado_por
)
SELECT r.id, p.id, p.nombre, r.fecha, r.hora_desde, r.hora_hasta, r.local_id,
       CASE WHEN r.tipo_espacio = 'B' THEN 'Área de presoterapia' ELSE 'Cabina principal' END,
       CASE WHEN r.id % 4 = 0 THEN 'La clienta reportó buena tolerancia y evolución favorable.'
            ELSE 'Atención realizada sin novedades.' END,
       'Mariana Salvatierra'
FROM reservas r
JOIN LATERAL (
    SELECT * FROM carga_personal cp
    WHERE cp.local_nombre = r.local_nombre
    ORDER BY cp.id OFFSET (r.id % 2) LIMIT 1
) p ON TRUE
WHERE r.notas LIKE 'CARGA-NOV-2026:%' AND r.estado = 'COMPLETADO';

-- Consumos históricos: la mayoría descontados, dos con deducción pendiente
-- para probar alertas, edición y los filtros del Historial.
INSERT INTO atencion_insumos (
    atencion_id, item_id, item_nombre, unidad_medida, cantidad_usada, cantidad_descontada, estado_descuento
)
SELECT a.id, i.id, i.nombre, i.unidad_medida,
       CASE i.unidad_medida WHEN 'ML' THEN 25.000 ELSE 2.000 END,
       CASE i.unidad_medida WHEN 'ML' THEN 25.000 ELSE 2.000 END,
       'DESCONTADO'
FROM atenciones a
JOIN reservas r ON r.id = a.reserva_id
JOIN LATERAL (
    SELECT * FROM carga_items ORDER BY id OFFSET (a.id % 6) LIMIT 1
) i ON TRUE
WHERE r.notas LIKE 'CARGA-NOV-2026:%';

INSERT INTO atencion_insumos (
    atencion_id, item_id, item_nombre, unidad_medida, cantidad_usada, cantidad_descontada, estado_descuento
)
SELECT a.id, i.id, i.nombre, i.unidad_medida, 2.000, 2.000, 'DESCONTADO'
FROM atenciones a
JOIN reservas r ON r.id = a.reserva_id
JOIN carga_items i ON i.nombre = 'Guantes de nitrilo talla M'
WHERE r.notas LIKE 'CARGA-NOV-2026:%' AND a.id % 2 = 0;

-- Una atención conserva el uso declarado, pero queda pendiente de descontar
-- para probar los avisos y la edición de deducciones pendientes.
UPDATE atencion_insumos
SET cantidad_descontada = 0, estado_descuento = 'PENDIENTE_STOCK'
WHERE id = (
    SELECT ai.id
    FROM atencion_insumos ai
    JOIN atenciones a ON a.id = ai.atencion_id
    JOIN reservas r ON r.id = a.reserva_id
    WHERE r.notas LIKE 'CARGA-NOV-2026:%'
    ORDER BY ai.id
    LIMIT 1
);

COMMIT;

-- Resumen esperado aproximado: 42 reservas, 4 atenciones, cancelaciones,
-- rechazos, pendientes, agendadas y al menos una deducción pendiente.
SELECT estado, COUNT(*) AS reservas
FROM reservas
WHERE notas LIKE 'CARGA-NOV-2026:%'
GROUP BY estado
ORDER BY estado;

SELECT COUNT(DISTINCT a.id) AS atenciones, COUNT(*) FILTER (WHERE ai.estado_descuento = 'PENDIENTE_STOCK') AS deducciones_pendientes
FROM atenciones a
JOIN reservas r ON r.id = a.reserva_id
LEFT JOIN atencion_insumos ai ON ai.atencion_id = a.id
WHERE r.notas LIKE 'CARGA-NOV-2026:%';
