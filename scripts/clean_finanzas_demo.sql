-- Borra los datos de prueba cargados por scripts/seed_finanzas_demo.sql (todo lo marcado 'DEMO').
-- No toca nada que no lleve esa marca. Uso: go run ./cmd/_sql -f scripts/clean_finanzas_demo.sql
BEGIN;
DELETE FROM inventario_movimientos WHERE item_id IN (SELECT id FROM inventario_items WHERE codigo LIKE 'DEMO-I-%');
DELETE FROM movimientos_financieros WHERE codigo LIKE 'DEMO-M-%';
DELETE FROM planilla_bonos WHERE planilla_id IN (SELECT id FROM planillas WHERE notas = 'DEMO');
DELETE FROM adelantos_empleado WHERE motivo LIKE 'DEMO%';
DELETE FROM planillas WHERE notas = 'DEMO';
DELETE FROM empleados WHERE notas = 'DEMO';
DELETE FROM gastos_fijos WHERE notas = 'DEMO';
DELETE FROM detalle_compras WHERE compra_id IN (SELECT id FROM compras_inventario WHERE codigo LIKE 'DEMO-C-%');
DELETE FROM compras_inventario WHERE codigo LIKE 'DEMO-C-%';
DELETE FROM pagos WHERE codigo_pago LIKE 'DEMO-P-%';
DELETE FROM inventario_stock WHERE item_id IN (SELECT id FROM inventario_items WHERE codigo LIKE 'DEMO-I-%');
DELETE FROM inventario_items WHERE codigo LIKE 'DEMO-I-%';
DELETE FROM clientes WHERE ci LIKE 'DEMO-%';
DELETE FROM tipos_bono WHERE descripcion = 'DEMO';
DELETE FROM categorias_inventario c WHERE c.nombre IN ('Insumos de cabina', 'Cosmética y venta', 'Descartables', 'Limpieza')
  AND NOT EXISTS (SELECT 1 FROM inventario_items i WHERE i.categoria_id = c.id);
COMMIT;
