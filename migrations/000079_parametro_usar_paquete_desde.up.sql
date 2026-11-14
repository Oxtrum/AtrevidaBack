-- Desde cuando se puede usar un paquete: CUENTA (apenas se vende, aunque quede debiendo) o PAGADO (solo pagado).
INSERT INTO configuracion_sistema (clave, valor, tipo, descripcion) VALUES
    ('paquetes.usar_desde', 'CUENTA', 'string', 'Desde cuando se puede usar un paquete: CUENTA (apenas se vende, aunque quede a cuenta) o PAGADO (solo cuando ya esta pagado).')
ON CONFLICT (clave) DO NOTHING;
