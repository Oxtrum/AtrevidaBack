-- CI: documento de identidad de la persona; identifica al cliente de forma
-- fiable, cosa que nombre+apellido no hacen y el telefono deja de hacer al
-- cambiar de numero.
-- NIT: dato de facturacion por defecto del cliente. Lo que se factura en un
-- pago concreto sigue viviendo en pagos.cliente_nit y puede diferir.
-- Ambas nullables: no romper las filas existentes ni bloquear el registro de
-- alguien que no trae el carnet.
ALTER TABLE clientes
    ADD COLUMN IF NOT EXISTS ci  VARCHAR(20),
    ADD COLUMN IF NOT EXISTS nit VARCHAR(20);
