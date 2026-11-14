DROP INDEX IF EXISTS uq_mov_fin_gasto_fijo_periodo;
ALTER TABLE movimientos_financieros DROP CONSTRAINT IF EXISTS fk_mov_fin_gasto_fijo;
DROP TABLE IF EXISTS gastos_fijos;
