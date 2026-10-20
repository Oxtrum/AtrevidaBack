DELETE FROM configuracion_sistema WHERE clave IN
    ('rrhh.dia_pago_defecto', 'rrhh.aporte_laboral_pct', 'rrhh.aporte_patronal_pct', 'rrhh.dias_mes_faltas');
DROP INDEX IF EXISTS uq_mov_fin_planilla;
ALTER TABLE movimientos_financieros DROP COLUMN IF EXISTS planilla_id;
DROP TABLE IF EXISTS adelantos_empleado;
DROP TABLE IF EXISTS planillas;
DROP TABLE IF EXISTS empleados;
