DELETE FROM categorias_financieras c
WHERE c.nombre IN ('Alquiler', 'Banco / Créditos', 'Sueldos y personal', 'Servicios básicos',
                   'Compras de inventario', 'Mantenimiento', 'Marketing', 'Otros egresos', 'Otros ingresos')
  AND NOT EXISTS (SELECT 1 FROM movimientos_financieros m WHERE m.categoria_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM gastos_fijos g WHERE g.categoria_id = c.id);
