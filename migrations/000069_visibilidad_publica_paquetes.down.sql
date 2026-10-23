DELETE FROM configuracion_sistema WHERE clave = 'publico.mostrar_paquetes';
ALTER TABLE paquetes DROP COLUMN IF EXISTS visible_publico;
