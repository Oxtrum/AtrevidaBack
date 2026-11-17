-- La cantidad por sesion pasa a ser opcional: NULL = uso variable. El insumo se sugiere al registrar la
-- atencion con la cantidad vacia (se escribe lo que se uso) y no participa del consumo automatico,
-- las alertas ni el stock comprometido de las reservas, porque no hay una cantidad fija que reservar.
ALTER TABLE servicio_insumos ALTER COLUMN cantidad_por_sesion DROP NOT NULL;

ALTER TABLE servicio_insumos DROP CONSTRAINT IF EXISTS chk_servicio_insumos_cantidad;
ALTER TABLE servicio_insumos
    ADD CONSTRAINT chk_servicio_insumos_cantidad CHECK (cantidad_por_sesion IS NULL OR cantidad_por_sesion > 0);

COMMENT ON COLUMN servicio_insumos.cantidad_por_sesion IS
    'Cantidad fija que consume cada sesion. NULL = uso variable: se indica en cada atencion.';
