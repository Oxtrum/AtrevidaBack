-- Promociones extraídas de las piezas gráficas entregadas en Recursos/ el 2026-07-11.
-- Son datos de catálogo: no crean planes, pagos ni reservas.

INSERT INTO categorias (nombre)
SELECT 'COMBOS'
WHERE NOT EXISTS (
    SELECT 1 FROM categorias WHERE UPPER(nombre) = 'COMBOS'
);

WITH seed (
    nombre,
    descripcion,
    precio_paquete,
    sesiones_totales
) AS (
    VALUES
        (
            'PAQUETE FIT PREMIUM',
            'Promoción FIT Premium de 10 sesiones. Incluye ultracavitación, radiofrecuencia, vacumterapia + RF, lipoláser, ondas rusas, maderoterapia y masaje reductor o tonificador. Precio especial publicado: Bs 1000.',
            1000.00::NUMERIC,
            10
        ),
        (
            'PAQUETE FIT',
            'Promoción FIT de 10 sesiones. Incluye ultracavitación, radiofrecuencia, vacumterapia + RF, lipoláser, ondas rusas, maderoterapia y masaje reductor o tonificador. Precio regular publicado: Bs 890; promoción especial: Bs 679.',
            679.00::NUMERIC,
            10
        ),
        ('QUEMADOR DE GRASA - 1 SESIÓN', 'Mesoterapia quemador de grasa potente. Incluye 2 ampollas.', 300.00::NUMERIC, 1),
        ('QUEMADOR DE GRASA - 3 SESIONES', 'Mesoterapia quemador de grasa potente. Reducción de grasa localizada y activación del metabolismo. Pago contado.', 750.00::NUMERIC, 3),
        ('QUEMADOR DE GRASA - 5 SESIONES', 'Mesoterapia quemador de grasa potente. Tratamiento completo de 5 semanas. Pago contado.', 1250.00::NUMERIC, 5),
        ('QUEMADOR DE GRASA - 10 SESIONES', 'Mesoterapia quemador de grasa potente. Tratamiento completo de 10 semanas. Pago contado.', 2350.00::NUMERIC, 10),
        ('MESOTERAPIA DMAE - 1 SESIÓN', 'Mesoterapia DMAE tensor y tonificador. Incluye 2 ampollas y efecto tensor inmediato.', 350.00::NUMERIC, 1),
        ('MESOTERAPIA DMAE - 3 SESIONES', 'Mesoterapia DMAE tensor y tonificador. Mejor firmeza. Pago contado.', 900.00::NUMERIC, 3),
        ('MESOTERAPIA DMAE - 5 SESIONES', 'Mesoterapia DMAE tensor y tonificador. Tonificación más visible. Pago contado.', 1500.00::NUMERIC, 5),
        ('MESOTERAPIA DMAE - 10 SESIONES', 'Mesoterapia DMAE tensor y tonificador. Tratamiento completo de 10 semanas. Pago contado.', 2900.00::NUMERIC, 10),
        ('E - PULSE BIKE - 1 SESIÓN', 'Programa E - Pulse Bike: adelgazar, tonificar, rehabilitador y resistencia.', 70.00::NUMERIC, 1),
        ('E - PULSE BIKE - 5 SESIONES', 'Programa E - Pulse Bike: adelgazar, tonificar, rehabilitador y resistencia.', 250.00::NUMERIC, 5),
        ('E - PULSE BIKE - 10 SESIONES', 'Programa E - Pulse Bike: adelgazar, tonificar, rehabilitador y resistencia.', 350.00::NUMERIC, 10),
        ('E - PULSE BIKE - 20 SESIONES', 'Programa E - Pulse Bike: adelgazar, tonificar, rehabilitador y resistencia.', 500.00::NUMERIC, 20)
)
INSERT INTO combos (
    nombre,
    descripcion,
    categoria_id,
    tipo_precio,
    precio_paquete,
    moneda,
    costo_total,
    sesiones_totales,
    activo,
    creado_en,
    actualizado_en
)
SELECT
    seed.nombre,
    seed.descripcion,
    categoria.id,
    'PRECIO_PAQUETE',
    seed.precio_paquete,
    'BOB',
    seed.precio_paquete,
    seed.sesiones_totales,
    TRUE,
    NOW(),
    NOW()
FROM seed
CROSS JOIN LATERAL (
    SELECT id
    FROM categorias
    WHERE UPPER(nombre) = 'COMBOS'
    ORDER BY id
    LIMIT 1
) categoria;

-- Todas las promociones, excepto E - Pulse Bike, se publican en ambos locales.
INSERT INTO combo_local (combo_id, local_id)
SELECT combo.id, local.id
FROM combos combo
JOIN locales local ON (
    (combo.nombre LIKE 'E - PULSE BIKE%' AND UPPER(local.nombre) = 'SAN MARTIN')
    OR
    (combo.nombre NOT LIKE 'E - PULSE BIKE%' AND UPPER(local.nombre) IN ('SAN MARTIN', 'PASEO ARANJUEZ'))
)
WHERE combo.nombre IN (
    'PAQUETE FIT PREMIUM',
    'PAQUETE FIT',
    'QUEMADOR DE GRASA - 1 SESIÓN',
    'QUEMADOR DE GRASA - 3 SESIONES',
    'QUEMADOR DE GRASA - 5 SESIONES',
    'QUEMADOR DE GRASA - 10 SESIONES',
    'MESOTERAPIA DMAE - 1 SESIÓN',
    'MESOTERAPIA DMAE - 3 SESIONES',
    'MESOTERAPIA DMAE - 5 SESIONES',
    'MESOTERAPIA DMAE - 10 SESIONES',
    'E - PULSE BIKE - 1 SESIÓN',
    'E - PULSE BIKE - 5 SESIONES',
    'E - PULSE BIKE - 10 SESIONES',
    'E - PULSE BIKE - 20 SESIONES'
)
ON CONFLICT DO NOTHING;

-- La línea conserva el precio unitario equivalente para referencia. El precio
-- final del catálogo continúa siendo precio_paquete, no la suma dinámica.
WITH lineas (combo_nombre, servicio_texto, tiempo, costo_unitario, sesiones) AS (
    VALUES
        ('PAQUETE FIT PREMIUM', 'Sesión FIT Premium personalizada: ultracavitación, radiofrecuencia, vacumterapia + RF, lipoláser, ondas rusas, maderoterapia y masaje reductor o tonificador', '50 min', 100.00::NUMERIC, 10),
        ('PAQUETE FIT', 'Sesión FIT personalizada: ultracavitación, radiofrecuencia, vacumterapia + RF, lipoláser, ondas rusas, maderoterapia y masaje reductor o tonificador', '50 min', 67.90::NUMERIC, 10),
        ('QUEMADOR DE GRASA - 1 SESIÓN', 'Mesoterapia quemador de grasa potente', '50 min', 300.00::NUMERIC, 1),
        ('QUEMADOR DE GRASA - 3 SESIONES', 'Mesoterapia quemador de grasa potente', '50 min', 250.00::NUMERIC, 3),
        ('QUEMADOR DE GRASA - 5 SESIONES', 'Mesoterapia quemador de grasa potente', '50 min', 250.00::NUMERIC, 5),
        ('QUEMADOR DE GRASA - 10 SESIONES', 'Mesoterapia quemador de grasa potente', '50 min', 235.00::NUMERIC, 10),
        ('MESOTERAPIA DMAE - 1 SESIÓN', 'Mesoterapia DMAE tensor y tonificador', '50 min', 350.00::NUMERIC, 1),
        ('MESOTERAPIA DMAE - 3 SESIONES', 'Mesoterapia DMAE tensor y tonificador', '50 min', 300.00::NUMERIC, 3),
        ('MESOTERAPIA DMAE - 5 SESIONES', 'Mesoterapia DMAE tensor y tonificador', '50 min', 300.00::NUMERIC, 5),
        ('MESOTERAPIA DMAE - 10 SESIONES', 'Mesoterapia DMAE tensor y tonificador', '50 min', 290.00::NUMERIC, 10),
        ('E - PULSE BIKE - 1 SESIÓN', 'E - Pulse Bike', '30 min', 70.00::NUMERIC, 1),
        ('E - PULSE BIKE - 5 SESIONES', 'E - Pulse Bike', '30 min', 50.00::NUMERIC, 5),
        ('E - PULSE BIKE - 10 SESIONES', 'E - Pulse Bike', '30 min', 35.00::NUMERIC, 10),
        ('E - PULSE BIKE - 20 SESIONES', 'E - Pulse Bike', '30 min', 25.00::NUMERIC, 20)
)
INSERT INTO combo_servicios (
    combo_id,
    servicio_id,
    servicio_texto,
    tiempo,
    costo,
    sesiones,
    orden,
    activo
)
SELECT
    combo.id,
    NULL,
    lineas.servicio_texto,
    lineas.tiempo,
    lineas.costo_unitario,
    lineas.sesiones,
    0,
    TRUE
FROM lineas
JOIN combos combo ON combo.nombre = lineas.combo_nombre;
