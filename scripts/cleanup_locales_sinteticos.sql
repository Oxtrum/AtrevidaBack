-- Limpia locales sintéticos y todos sus registros dependientes de atrevida_fit_test.
-- Conserva exclusivamente SAN MARTIN y PASEO ARANJUEZ.
-- No ejecutar sobre una base de producción.

BEGIN;

CREATE TEMP TABLE locales_objetivo ON COMMIT DROP AS
SELECT id
FROM locales
WHERE nombre NOT IN ('SAN MARTIN', 'PASEO ARANJUEZ');

CREATE TEMP TABLE reservas_objetivo ON COMMIT DROP AS
SELECT id FROM reservas WHERE local_id IN (SELECT id FROM locales_objetivo);

CREATE TEMP TABLE atenciones_objetivo ON COMMIT DROP AS
SELECT id FROM atenciones
WHERE local_id IN (SELECT id FROM locales_objetivo)
   OR reserva_id IN (SELECT id FROM reservas_objetivo);

CREATE TEMP TABLE planes_objetivo ON COMMIT DROP AS
SELECT id FROM planes WHERE local_id IN (SELECT id FROM locales_objetivo);

CREATE TEMP TABLE pagos_objetivo ON COMMIT DROP AS
SELECT id FROM pagos
WHERE local_id IN (SELECT id FROM locales_objetivo)
   OR reserva_id IN (SELECT id FROM reservas_objetivo);

CREATE TEMP TABLE cargos_objetivo ON COMMIT DROP AS
SELECT id FROM cargos_cliente
WHERE local_id IN (SELECT id FROM locales_objetivo)
   OR reserva_id IN (SELECT id FROM reservas_objetivo)
   OR plan_id IN (SELECT id FROM planes_objetivo);

-- Dependencias de pagos, cargos y planes.
DELETE FROM plan_pago_aplicaciones
WHERE pago_id IN (SELECT id FROM pagos_objetivo)
   OR plan_cuota_id IN (
       SELECT pc.id FROM plan_cuotas pc WHERE pc.plan_id IN (SELECT id FROM planes_objetivo)
   );

DELETE FROM pago_aplicaciones
WHERE pago_id IN (SELECT id FROM pagos_objetivo)
   OR cargo_id IN (SELECT id FROM cargos_objetivo);

DELETE FROM detalle_pagos WHERE pago_id IN (SELECT id FROM pagos_objetivo);

-- Movimientos de inventario y finanzas son hijos de las entidades operativas.
DELETE FROM inventario_movimientos
WHERE local_id IN (SELECT id FROM locales_objetivo)
   OR reserva_id IN (SELECT id FROM reservas_objetivo)
   OR atencion_id IN (SELECT id FROM atenciones_objetivo)
   OR pago_id IN (SELECT id FROM pagos_objetivo)
   OR cargo_id IN (SELECT id FROM cargos_objetivo);

DELETE FROM movimientos_financieros
WHERE local_id IN (SELECT id FROM locales_objetivo);

DELETE FROM atenciones
WHERE id IN (SELECT id FROM atenciones_objetivo);

DELETE FROM pagos WHERE id IN (SELECT id FROM pagos_objetivo);
DELETE FROM cargos_cliente WHERE id IN (SELECT id FROM cargos_objetivo);

DELETE FROM detalle_compras
WHERE compra_id IN (
    SELECT id FROM compras_inventario WHERE local_id IN (SELECT id FROM locales_objetivo)
);
DELETE FROM compras_inventario WHERE local_id IN (SELECT id FROM locales_objetivo);
DELETE FROM gastos_fijos WHERE local_id IN (SELECT id FROM locales_objetivo);

-- Planillas y adelantos deben salir antes de su personal.
DELETE FROM adelantos_empleado
WHERE empleado_id IN (SELECT id FROM empleados WHERE local_id IN (SELECT id FROM locales_objetivo));
DELETE FROM planillas
WHERE empleado_id IN (SELECT id FROM empleados WHERE local_id IN (SELECT id FROM locales_objetivo));
DELETE FROM empleados WHERE local_id IN (SELECT id FROM locales_objetivo);

-- Las reservas pueden apuntar a planes; se eliminan antes de estos. Las tablas
-- detalle_reservas y plan_movimientos se resuelven por sus reglas ON DELETE.
DELETE FROM reservas WHERE id IN (SELECT id FROM reservas_objetivo);
DELETE FROM planes WHERE id IN (SELECT id FROM planes_objetivo);

DELETE FROM inventario_stock WHERE local_id IN (SELECT id FROM locales_objetivo);

-- El resto de relaciones por local se eliminan por CASCADE; estas sentencias
-- explícitas dejan la limpieza legible y no dependen de esa implementación.
DELETE FROM categorias_locales WHERE local_id IN (SELECT id FROM locales_objetivo);
DELETE FROM combo_local WHERE local_id IN (SELECT id FROM locales_objetivo);
DELETE FROM locales_horarios WHERE local_id IN (SELECT id FROM locales_objetivo);
DELETE FROM paquete_local WHERE local_id IN (SELECT id FROM locales_objetivo);
DELETE FROM servicio_local WHERE local_id IN (SELECT id FROM locales_objetivo);
DELETE FROM tipos_espacio_locales WHERE local_id IN (SELECT id FROM locales_objetivo);

DELETE FROM locales WHERE id IN (SELECT id FROM locales_objetivo);

COMMIT;

SELECT id, nombre FROM locales ORDER BY id;
