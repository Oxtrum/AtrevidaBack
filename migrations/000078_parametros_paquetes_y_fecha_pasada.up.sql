-- Cuando avanza la sesion de un paquete: al completar la reserva (por defecto) o al agendarla.
-- Tambien se siembra cobros.dias_maximo_fecha_pasada, que el servicio ya valida pero no tenia fila y no se podia editar.
INSERT INTO configuracion_sistema (clave, valor, tipo, descripcion) VALUES
    ('paquetes.sesion_en_estado', 'COMPLETADO', 'string', 'Estado de la reserva en el que avanza la sesion de su paquete: COMPLETADO o AGENDADO.'),
    ('cobros.dias_maximo_fecha_pasada', '0', 'int', 'Dias hacia atras permitidos para la fecha de un cobro o cargo. 0 = sin limite.')
ON CONFLICT (clave) DO NOTHING;
