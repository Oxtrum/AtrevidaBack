DROP TABLE IF EXISTS plan_pago_aplicaciones;
DROP TABLE IF EXISTS plan_cuotas;
DROP TABLE IF EXISTS plan_movimientos;
DROP TABLE IF EXISTS plan_servicios;

ALTER TABLE reservas DROP COLUMN IF EXISTS plan_servicio_id;

ALTER TABLE planes
    DROP CONSTRAINT IF EXISTS chk_planes_estado,
    DROP CONSTRAINT IF EXISTS chk_planes_estado_cobranza,
    DROP CONSTRAINT IF EXISTS chk_planes_tipo_pago,
    DROP CONSTRAINT IF EXISTS chk_planes_moneda,
    DROP CONSTRAINT IF EXISTS chk_planes_importes_no_negativos,
    DROP CONSTRAINT IF EXISTS chk_planes_descuento_valido,
    DROP CONSTRAINT IF EXISTS chk_planes_fechas;

ALTER TABLE planes
    DROP COLUMN IF EXISTS codigo,
    DROP COLUMN IF EXISTS cliente_id,
    DROP COLUMN IF EXISTS cliente_nombre_snapshot,
    DROP COLUMN IF EXISTS local_nombre_snapshot,
    DROP COLUMN IF EXISTS combo_id_origen,
    DROP COLUMN IF EXISTS combo_nombre_snapshot,
    DROP COLUMN IF EXISTS fecha_inicio,
    DROP COLUMN IF EXISTS fecha_fin,
    DROP COLUMN IF EXISTS estado,
    DROP COLUMN IF EXISTS estado_cobranza,
    DROP COLUMN IF EXISTS tipo_pago,
    DROP COLUMN IF EXISTS subtotal,
    DROP COLUMN IF EXISTS descuento,
    DROP COLUMN IF EXISTS precio_total,
    DROP COLUMN IF EXISTS moneda,
    DROP COLUMN IF EXISTS creado_por,
    DROP COLUMN IF EXISTS actualizado_por,
    DROP COLUMN IF EXISTS actualizado_en;

DROP SEQUENCE IF EXISTS plan_codigo_seq;
