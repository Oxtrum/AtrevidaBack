INSERT INTO categorias_financieras (nombre, naturaleza) VALUES
    ('Alquiler', 'EGRESO'),
    ('Banco / Créditos', 'EGRESO'),
    ('Sueldos y personal', 'EGRESO'),
    ('Servicios básicos', 'EGRESO'),
    ('Compras de inventario', 'EGRESO'),
    ('Mantenimiento', 'EGRESO'),
    ('Marketing', 'EGRESO'),
    ('Otros egresos', 'EGRESO'),
    ('Otros ingresos', 'INGRESO')
ON CONFLICT DO NOTHING;
