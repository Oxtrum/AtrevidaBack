-- Visibilidad en la pagina principal: un paquete puede ocultarse sin darlo de baja
-- y la seccion completa de paquetes puede apagarse desde Configuracion.
ALTER TABLE paquetes ADD COLUMN IF NOT EXISTS visible_publico BOOLEAN NOT NULL DEFAULT TRUE;

INSERT INTO configuracion_sistema (clave, valor, tipo, descripcion) VALUES
    ('publico.mostrar_paquetes', 'true', 'bool', 'Muestra la seccion de paquetes en la pagina principal. En false se oculta sin afectar la administracion.')
ON CONFLICT (clave) DO NOTHING;
