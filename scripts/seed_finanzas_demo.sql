-- Datos de prueba para Finanzas (ventas, egresos, gastos fijos, compras, inventario y RRHH), por local.
-- Cubre desde el inicio del mes de hace 3 meses hasta hoy, y es reproducible (setseed).
-- Todo queda marcado para poder borrarlo con scripts/clean_finanzas_demo.sql:
--   pagos.codigo_pago, movimientos_financieros.codigo, compras_inventario.codigo, inventario_items.codigo = 'DEMO-*'
--   empleados/planillas/gastos_fijos.notas = 'DEMO', adelantos_empleado.motivo = 'DEMO ...', clientes.ci = 'DEMO-*'
-- Uso: go run ./cmd/_sql -f scripts/seed_finanzas_demo.sql   (o ejecutarlo con cualquier cliente de Postgres)

CREATE OR REPLACE FUNCTION pg_temp.demo_stock(p_item int, p_local int) RETURNS numeric AS $f$
  SELECT COALESCE((SELECT cantidad FROM inventario_stock WHERE item_id = p_item AND local_id = p_local), 0)
$f$ LANGUAGE sql;

-- Misma logica que AplicarMovimientoStock del backend: actualiza stock y deja el renglon del kardex.
CREATE OR REPLACE FUNCTION pg_temp.demo_mov_stock(
  p_item int, p_local int, p_tipo text, p_cant numeric, p_costo numeric, p_compra int, p_pago int,
  p_ts timestamptz, p_motivo text, p_uid int, p_unombre text
) RETURNS void AS $f$
DECLARE
  v_nuevo numeric;
  v_signo int := CASE WHEN p_tipo IN ('ENTRADA_COMPRA', 'AJUSTE_POSITIVO', 'REVERSION_SERVICIO', 'REVERSION_VENTA') THEN 1 ELSE -1 END;
BEGIN
  INSERT INTO inventario_stock (item_id, local_id, cantidad) VALUES (p_item, p_local, 0)
  ON CONFLICT (item_id, local_id) DO NOTHING;
  UPDATE inventario_stock SET cantidad = cantidad + v_signo * p_cant, fecha_modificacion = p_ts
  WHERE item_id = p_item AND local_id = p_local
  RETURNING cantidad INTO v_nuevo;
  INSERT INTO inventario_movimientos
    (item_id, local_id, tipo, cantidad, stock_resultante, costo_unitario, compra_id, pago_id, motivo, usuario_id, usuario_nombre, creado_en)
  VALUES (p_item, p_local, p_tipo, p_cant, v_nuevo, p_costo, p_compra, p_pago, p_motivo, p_uid, p_unombre, p_ts);
END
$f$ LANGUAGE plpgsql;

DO $seed$
DECLARE
  tz         constant text := 'America/La_Paz';
  v_hoy      date := CURRENT_DATE;
  v_ini      date := (date_trunc('month', CURRENT_DATE) - interval '3 months')::date;
  v_locs     int[];
  v_lid      int;
  v_lname    text;
  v_admin    int;
  v_ger      int;
  v_ger_local int;
  v_items    int[];
  v_clientes int[];
  v_cnames   text[];
  v_pc int := 0;  -- contador de pagos
  v_mc int := 0;  -- contador de movimientos
  v_cc int := 0;  -- contador de compras
  c_sueldos  int;
  c_compras  int;
  v_d        date;
  v_m        date;
  r          record;
  g          record;
  e          record;
  -- ventas
  v_n int; v_i int; v_h int; v_hmax int; v_mf numeric; v_base numeric;
  v_pago int; v_sub numeric; v_desc numeric; v_tipo text; v_estado text; v_activo boolean;
  v_cli int; v_cli_nombre text; v_ts timestamptz; v_cajero int; v_cajero_nom text;
  v_nlin int; v_k int; v_grupo text; v_qty int; v_item_idx int; v_item int;
  -- compras
  v_compra int; v_total numeric; v_costo numeric; v_prov text; v_metodo text; v_cant numeric; v_codigo text;
  -- planillas
  v_dia int; v_fpago date; v_bonos numeric; v_faltas numeric; v_descf numeric; v_totg numeric; v_lab numeric;
  v_pat numeric; v_adel numeric; v_neto numeric; v_costoemp numeric; v_planilla int;
  -- movimientos
  v_est text; v_monto numeric; v_fecha date; v_arr text;
BEGIN
  PERFORM setseed(0.37);

  IF EXISTS (SELECT 1 FROM pagos WHERE codigo_pago LIKE 'DEMO-%') THEN
    RAISE EXCEPTION 'Ya hay datos demo cargados. Corre primero scripts/clean_finanzas_demo.sql';
  END IF;

  SELECT array_agg(id ORDER BY id) INTO v_locs FROM locales WHERE activo;
  SELECT id INTO v_admin FROM usuarios WHERE username = 'admin';
  SELECT id, local_id INTO v_ger, v_ger_local FROM usuarios WHERE username = 'gerente';
  SELECT id INTO c_sueldos FROM categorias_financieras WHERE nombre = 'Sueldos y personal' AND naturaleza = 'EGRESO';
  SELECT id INTO c_compras FROM categorias_financieras WHERE LOWER(nombre) = 'compras de inventario' AND naturaleza = 'EGRESO';

  -- ── Categorias e insumos ────────────────────────────────────────────────────
  INSERT INTO categorias_inventario (nombre)
  SELECT n FROM (VALUES ('Insumos de cabina'), ('Cosmética y venta'), ('Descartables'), ('Limpieza')) v(n)
  WHERE NOT EXISTS (SELECT 1 FROM categorias_inventario c WHERE c.nombre = v.n);

  INSERT INTO inventario_items (codigo, nombre, descripcion, categoria_id, unidad_medida, costo_referencia, stock_minimo, precio_venta)
  SELECT 'DEMO-I-' || lpad(t.n::text, 2, '0'), t.nombre, 'DEMO',
         (SELECT id FROM categorias_inventario WHERE nombre = t.cat LIMIT 1), t.unidad, t.costo, t.minimo, t.venta
  FROM (VALUES
    (1, 'Gel conductor para ultrasonido', 'Insumos de cabina', 'LT', 35, 4, NULL::numeric),
    (2, 'Aceite de masaje', 'Insumos de cabina', 'LT', 60, 3, NULL),
    (3, 'Crema reductora', 'Cosmética y venta', 'FRASCO', 85, 4, 140),
    (4, 'Crema reafirmante facial', 'Cosmética y venta', 'FRASCO', 70, 4, 120),
    (5, 'Sábanas descartables', 'Descartables', 'PAQUETE', 45, 5, NULL),
    (6, 'Guantes de látex', 'Descartables', 'CAJA', 38, 4, NULL),
    (7, 'Toallas desechables', 'Descartables', 'PAQUETE', 30, 5, NULL),
    (8, 'Faja térmica', 'Cosmética y venta', 'UNIDAD', 55, 3, 99),
    (9, 'Protector solar', 'Cosmética y venta', 'FRASCO', 48, 3, 85),
    (10, 'Alcohol en gel', 'Limpieza', 'LT', 22, 3, NULL)
  ) t(n, nombre, cat, unidad, costo, minimo, venta);
  SELECT array_agg(id ORDER BY codigo) INTO v_items FROM inventario_items WHERE codigo LIKE 'DEMO-I-%';

  -- ── Clientes ────────────────────────────────────────────────────────────────
  INSERT INTO clientes (nombre, apellido, numero_telefono, ci)
  SELECT t.nombre, t.apellido, '7100' || lpad(t.n::text, 4, '0'), 'DEMO-' || lpad(t.n::text, 2, '0')
  FROM (VALUES
    (1, 'María', 'Gutiérrez'), (2, 'Carla', 'Mendoza'), (3, 'Verónica', 'Salazar'), (4, 'Paola', 'Camacho'),
    (5, 'Sandra', 'Rivero'), (6, 'Gabriela', 'Ortuño'), (7, 'Fernanda', 'Claros'), (8, 'Jimena', 'Arce'),
    (9, 'Daniela', 'Zurita'), (10, 'Patricia', 'Montaño'), (11, 'Lorena', 'Vásquez'), (12, 'Natalia', 'Antezana'),
    (13, 'Silvia', 'Heredia'), (14, 'Alejandra', 'Pinto')
  ) t(n, nombre, apellido);
  SELECT array_agg(id ORDER BY ci), array_agg(nombre || ' ' || apellido ORDER BY ci)
    INTO v_clientes, v_cnames FROM clientes WHERE ci LIKE 'DEMO-%';

  -- ── Catalogo de servicios por local (para armar las ventas) ────────────────
  CREATE TEMP TABLE demo_svc ON COMMIT DROP AS
  SELECT sl.local_id, s.id AS servicio_id, s.nombre, s.costo,
         CASE WHEN s.costo <= 150 THEN 'barato' WHEN s.costo <= 1500 THEN 'medio' ELSE 'caro' END AS grupo
  FROM servicio_local sl JOIN servicios s ON s.id = sl.servicio_id
  WHERE s.activo AND s.costo > 0;

  -- ── RRHH: empleados, adelantos, planillas y su egreso ──────────────────────
  INSERT INTO tipos_bono (nombre, descripcion)
  SELECT n, 'DEMO' FROM (VALUES ('Bono de producción'), ('Bono por meta de ventas')) v(n)
  WHERE NOT EXISTS (SELECT 1 FROM tipos_bono t WHERE t.nombre = v.n);

  INSERT INTO empleados (nombre, apellido, ci, cargo, local_id, sueldo_base, dia_pago, aplica_aportes, metodo_pago, fecha_ingreso, notas)
  SELECT t.nombre, t.apellido, 'DEMO-E' || lpad(t.n::text, 2, '0'), t.cargo, v_locs[t.loc], t.sueldo, t.dia, t.aportes, t.metodo, t.ingreso::date, 'DEMO'
  FROM (VALUES
    (1, 'Camila', 'Rojas', 'Esteticista', 1, 3200, NULL::int, true, 'transferencia', '2025-02-03'),
    (2, 'Daniela', 'Pérez', 'Masajista', 1, 2800, 5, true, 'efectivo', '2025-04-14'),
    (3, 'Lucía', 'Vargas', 'Recepcionista', 1, 2500, NULL, true, 'transferencia', '2025-06-02'),
    (4, 'Andrea', 'Flores', 'Esteticista', 2, 3200, 5, true, 'transferencia', '2025-03-10'),
    (5, 'Marcela', 'Soliz', 'Masajista', 2, 2800, NULL, true, 'efectivo', '2025-07-21'),
    (6, 'Rosa', 'Quispe', 'Limpieza', 2, 2200, 5, false, 'efectivo', '2025-09-01')
  ) t(n, nombre, apellido, cargo, loc, sueldo, dia, aportes, metodo, ingreso);

  INSERT INTO adelantos_empleado (empleado_id, fecha, monto, motivo, anulado)
  SELECT (SELECT id FROM empleados WHERE ci = t.ci), t.fecha::date, t.monto, 'DEMO ' || t.motivo, t.anulado
  FROM (VALUES
    ('DEMO-E03', '2026-08-20', 500, 'Adelanto por gastos médicos', false),
    ('DEMO-E02', '2026-09-18', 400, 'Adelanto de sueldo', false),
    ('DEMO-E05', '2026-09-25', 300, 'Registrado por error', true),
    ('DEMO-E06', date_trunc('month', CURRENT_DATE)::date - 12, 300, 'Adelanto para útiles escolares', false)
  ) t(ci, fecha, monto, motivo, anulado);

  FOR v_m IN SELECT generate_series(v_ini, date_trunc('month', v_hoy)::date, interval '1 month')::date LOOP
    FOR e IN SELECT em.*, l.nombre AS lnombre FROM empleados em LEFT JOIN locales l ON l.id = em.local_id
             WHERE em.notas = 'DEMO' ORDER BY em.id LOOP
      v_dia := COALESCE(e.dia_pago, 1);
      v_fpago := v_m + (v_dia - 1);
      CONTINUE WHEN v_fpago > v_hoy;

      v_bonos := CASE WHEN e.cargo = 'Esteticista' AND v_m >= v_ini + 31 THEN 150 + 50 * (e.id % 3) ELSE 0 END;
      v_faltas := CASE WHEN e.ci = 'DEMO-E05' AND to_char(v_m, 'YYYY-MM') = to_char(v_ini + 31, 'YYYY-MM') THEN 1 ELSE 0 END;
      v_descf := round(e.sueldo_base / 30 * v_faltas, 2);
      v_totg := round(e.sueldo_base + v_bonos - v_descf, 2);
      v_lab := CASE WHEN e.aplica_aportes THEN round(v_totg * 12.71 / 100, 2) ELSE 0 END;
      v_pat := CASE WHEN e.aplica_aportes THEN round(v_totg * 16.71 / 100, 2) ELSE 0 END;
      SELECT COALESCE(SUM(monto), 0) INTO v_adel FROM adelantos_empleado
        WHERE empleado_id = e.id AND planilla_id IS NULL AND NOT anulado AND fecha <= v_fpago;
      v_neto := round(v_totg - v_lab - v_adel, 2);
      v_costoemp := round(v_totg + v_pat, 2);

      INSERT INTO planillas (empleado_id, tipo, periodo, fecha_pago, sueldo_base, bonos, dias_falta, descuento_faltas, total_ganado,
                             aporte_laboral, adelantos, otros_descuentos, neto_pagar, aporte_patronal, costo_empresa, estado, notas,
                             fecha_creacion, fecha_modificacion)
      VALUES (e.id, 'SUELDO', v_m, v_fpago, e.sueldo_base, v_bonos, v_faltas, v_descf, v_totg, v_lab, v_adel, 0, v_neto, v_pat,
              v_costoemp, 'PAGADO', 'DEMO', (v_fpago::timestamp + interval '8 hours') AT TIME ZONE tz,
              (v_fpago::timestamp + interval '9 hours') AT TIME ZONE tz)
      RETURNING id INTO v_planilla;

      UPDATE adelantos_empleado SET planilla_id = v_planilla
        WHERE empleado_id = e.id AND planilla_id IS NULL AND NOT anulado AND fecha <= v_fpago;
      IF v_bonos > 0 THEN
        INSERT INTO planilla_bonos (planilla_id, tipo_bono_id, monto, nota)
        VALUES (v_planilla, (SELECT id FROM tipos_bono WHERE nombre = 'Bono de producción' LIMIT 1), v_bonos, 'DEMO');
      END IF;

      v_mc := v_mc + 1;
      INSERT INTO movimientos_financieros (codigo, naturaleza, clase, categoria_id, planilla_id, periodo, local_id, local_nombre, fecha,
                                           monto, descripcion, proveedor, metodo_pago, estado, nombre_usuario, fecha_creacion, fecha_modificacion)
      VALUES ('DEMO-M-' || lpad(v_mc::text, 5, '0'), 'EGRESO', 'FIJO', c_sueldos, v_planilla, v_m, e.local_id, e.lnombre, v_fpago,
              v_costoemp, 'Sueldo ' || to_char(v_m, 'YYYY-MM') || ' - ' || e.nombre || ' ' || e.apellido,
              e.nombre || ' ' || e.apellido, e.metodo_pago, 'PAGADO', 'sistema',
              (v_fpago::timestamp + interval '8 hours') AT TIME ZONE tz, (v_fpago::timestamp + interval '9 hours') AT TIME ZONE tz);
    END LOOP;
  END LOOP;

  -- ── Gastos fijos (plantillas) y su cobro mensual ───────────────────────────
  INSERT INTO gastos_fijos (nombre, categoria_id, monto, dia_cobro, local_id, metodo_pago, proveedor, fecha_inicio, frecuencia, notas)
  SELECT t.nombre, (SELECT id FROM categorias_financieras WHERE nombre = t.cat AND naturaleza = 'EGRESO' LIMIT 1), t.monto, t.dia,
         CASE WHEN t.loc IS NULL THEN NULL ELSE v_locs[t.loc] END, t.metodo, t.proveedor, (v_ini - 30), 'MENSUAL', 'DEMO'
  FROM (VALUES
    ('Alquiler San Martín', 'Alquiler', 3500, 5, 1, 'transferencia', 'Inmobiliaria Los Pinos'),
    ('Alquiler Paseo Aranjuez', 'Alquiler', 4200, 3, 2, 'transferencia', 'Inversiones Aranjuez SRL'),
    ('Electricidad San Martín', 'Servicios básicos', 650, 15, 1, 'qr', 'ELFEC S.A.'),
    ('Electricidad Paseo Aranjuez', 'Servicios básicos', 580, 15, 2, 'qr', 'ELFEC S.A.'),
    ('Internet San Martín', 'Servicios básicos', 250, 2, 1, 'debito_automatico', 'Tigo'),
    ('Internet Paseo Aranjuez', 'Servicios básicos', 250, 2, 2, 'debito_automatico', 'Tigo'),
    ('Crédito de equipos', 'Banco / Créditos', 1800, 20, 1, 'debito_automatico', 'Banco Mercantil Santa Cruz'),
    ('Publicidad en redes', 'Marketing', 900, 12, 2, 'qr', 'Meta Ads'),
    ('Contabilidad externa', 'Otros egresos', 600, 25, NULL, 'transferencia', 'Estudio Contable Mamani')
  ) t(nombre, cat, monto, dia, loc, metodo, proveedor);

  FOR v_m IN SELECT generate_series(v_ini, date_trunc('month', v_hoy)::date, interval '1 month')::date LOOP
    FOR g IN SELECT gf.*, l.nombre AS lnombre FROM gastos_fijos gf LEFT JOIN locales l ON l.id = gf.local_id
             WHERE gf.notas = 'DEMO' ORDER BY gf.id LOOP
      v_fecha := v_m + (g.dia_cobro - 1);
      CONTINUE WHEN v_fecha > v_hoy;
      -- un cobro por pagar (mes actual) y uno anulado, para probar esos estados
      v_est := CASE
        WHEN g.nombre = 'Internet San Martín' AND v_m = date_trunc('month', v_hoy)::date THEN 'PENDIENTE'
        WHEN g.nombre = 'Electricidad Paseo Aranjuez' AND v_m = (v_ini + 31) THEN 'ANULADO'
        ELSE 'PAGADO' END;
      v_mc := v_mc + 1;
      INSERT INTO movimientos_financieros (codigo, naturaleza, clase, categoria_id, gasto_fijo_id, periodo, local_id, local_nombre, fecha,
                                           monto, descripcion, proveedor, metodo_pago, estado, motivo_anulacion, id_usuario, nombre_usuario,
                                           fecha_creacion, fecha_modificacion)
      VALUES ('DEMO-M-' || lpad(v_mc::text, 5, '0'), 'EGRESO', 'FIJO', g.categoria_id, g.id, v_m, g.local_id, g.lnombre, v_fecha, g.monto,
              g.nombre, g.proveedor, g.metodo_pago, v_est,
              CASE WHEN v_est = 'ANULADO' THEN 'Cobro duplicado por la distribuidora' END,
              CASE WHEN v_est = 'PAGADO' THEN v_admin END, CASE WHEN v_est = 'PAGADO' THEN 'admin' ELSE 'sistema' END,
              (v_fecha::timestamp + interval '9 hours') AT TIME ZONE tz, (v_fecha::timestamp + interval '9 hours') AT TIME ZONE tz);
    END LOOP;
  END LOOP;

  -- ── Egresos variables e ingresos varios (registrados a mano), por local ────
  FOREACH v_lid IN ARRAY v_locs LOOP
    SELECT nombre INTO v_lname FROM locales WHERE id = v_lid;
    FOR v_m IN SELECT generate_series(v_ini, date_trunc('month', v_hoy)::date, interval '1 month')::date LOOP
      FOR v_i IN 1..8 LOOP
        v_fecha := v_m + floor(random() * 28)::int;
        CONTINUE WHEN v_fecha > v_hoy;
        v_arr := (ARRAY[
          'Mantenimiento|Mantenimiento de aire acondicionado|Frío Cochabamba|250|600|efectivo',
          'Mantenimiento|Reparación de camilla eléctrica|Muebles Médicos|150|450|efectivo',
          'Mantenimiento|Servicio técnico de equipo de cavitación|EstetiTec|300|900|transferencia',
          'Marketing|Impresión de flyers y banners|Imprenta Gráfica Andina|120|400|qr',
          'Marketing|Sesión de fotos para redes|Estudio Foto Lab|300|700|transferencia',
          'Servicios básicos|Agua potable|SEMAPA|90|180|efectivo',
          'Servicios básicos|Gas licuado|YPFB|40|90|efectivo',
          'Otros egresos|Artículos de oficina|Librería Universal|30|120|efectivo',
          'Otros egresos|Refrigerio para el personal|Supermercado Fidalga|50|150|efectivo',
          'Otros egresos|Lavandería de toallas y sábanas|Lavandería Express|80|200|qr'
        ])[1 + floor(random() * 10)::int];
        v_monto := round(split_part(v_arr, '|', 4)::numeric + random()::numeric * (split_part(v_arr, '|', 5)::numeric - split_part(v_arr, '|', 4)::numeric), 0);
        v_est := CASE WHEN random() < 0.04 THEN 'ANULADO'
                      WHEN v_m = date_trunc('month', v_hoy)::date AND random() < 0.15 THEN 'PENDIENTE'
                      ELSE 'PAGADO' END;
        v_mc := v_mc + 1;
        INSERT INTO movimientos_financieros (codigo, naturaleza, clase, categoria_id, local_id, local_nombre, fecha, monto, descripcion,
                                             proveedor, metodo_pago, estado, motivo_anulacion, id_usuario, nombre_usuario, fecha_creacion, fecha_modificacion)
        VALUES ('DEMO-M-' || lpad(v_mc::text, 5, '0'), 'EGRESO', 'VARIABLE',
                (SELECT id FROM categorias_financieras WHERE nombre = split_part(v_arr, '|', 1) AND naturaleza = 'EGRESO' LIMIT 1),
                v_lid, v_lname, v_fecha, v_monto, split_part(v_arr, '|', 2), split_part(v_arr, '|', 3), split_part(v_arr, '|', 6), v_est,
                CASE WHEN v_est = 'ANULADO' THEN 'Registrado por error' END, v_admin, 'admin',
                (v_fecha::timestamp + interval '15 hours') AT TIME ZONE tz, (v_fecha::timestamp + interval '15 hours') AT TIME ZONE tz);
      END LOOP;

      IF random() < 0.75 THEN
        v_fecha := v_m + floor(random() * 28)::int;
        IF v_fecha <= v_hoy THEN
          v_arr := (ARRAY[
            'Alquiler de cabina a terapeuta externa|Ana Torrez|400|900|transferencia',
            'Venta de equipo usado|Particular|500|1500|efectivo',
            'Reembolso de proveedor|Distribuidora Belleza SRL|60|250|transferencia'
          ])[1 + floor(random() * 3)::int];
          v_mc := v_mc + 1;
          INSERT INTO movimientos_financieros (codigo, naturaleza, clase, categoria_id, local_id, local_nombre, fecha, monto, descripcion,
                                               proveedor, metodo_pago, estado, id_usuario, nombre_usuario, fecha_creacion, fecha_modificacion)
          VALUES ('DEMO-M-' || lpad(v_mc::text, 5, '0'), 'INGRESO', NULL,
                  (SELECT id FROM categorias_financieras WHERE nombre = 'Otros ingresos' AND naturaleza = 'INGRESO' LIMIT 1),
                  v_lid, v_lname, v_fecha,
                  round(split_part(v_arr, '|', 3)::numeric + random()::numeric * (split_part(v_arr, '|', 4)::numeric - split_part(v_arr, '|', 3)::numeric), 0),
                  split_part(v_arr, '|', 1), split_part(v_arr, '|', 2), split_part(v_arr, '|', 5), 'PAGADO', v_admin, 'admin',
                  (v_fecha::timestamp + interval '16 hours') AT TIME ZONE tz, (v_fecha::timestamp + interval '16 hours') AT TIME ZONE tz);
        END IF;
      END IF;
    END LOOP;
  END LOOP;

  -- ── Compras, consumo de insumos y ventas, dia por dia ──────────────────────
  FOR v_d IN SELECT generate_series(v_ini, v_hoy, interval '1 day')::date LOOP
    v_mf := 1 + 0.07 * ((extract(year FROM v_d) * 12 + extract(month FROM v_d)) - (extract(year FROM v_ini) * 12 + extract(month FROM v_ini)));

    FOREACH v_lid IN ARRAY v_locs LOOP
      SELECT nombre INTO v_lname FROM locales WHERE id = v_lid;

      -- Compra de insumos (cada ~3 semanas; la primera abastece todo el catalogo)
      IF (v_d - v_ini) = ANY (ARRAY[1, 22, 43, 64, 85]) OR ((v_d - v_ini) = ANY (ARRAY[2, 23, 44, 65, 86]) AND v_lid <> v_locs[1]) THEN
        v_cc := v_cc + 1;
        v_codigo := 'DEMO-C-' || lpad(v_cc::text, 4, '0');
        v_prov := (ARRAY['Distribuidora Belleza SRL', 'Cosmetics Bolivia', 'Farmacia Chávez'])[1 + floor(random() * 3)::int];
        v_metodo := (ARRAY['transferencia', 'efectivo', 'qr'])[1 + floor(random() * 3)::int];
        v_ts := (v_d::timestamp + interval '10 hours') AT TIME ZONE tz;
        INSERT INTO compras_inventario (codigo, local_id, proveedor, fecha, total, notas, id_usuario, nombre_usuario, fecha_creacion)
        VALUES (v_codigo, v_lid, v_prov, v_d, 0, 'DEMO', v_admin, 'admin', v_ts) RETURNING id INTO v_compra;
        v_total := 0;
        FOR v_k IN 1..array_length(v_items, 1) LOOP
          -- La primera compra lleva todo; las siguientes, la mayoria. Guantes (local 2) y protector solar (local 1) no se
          -- reponen para que queden bajo el minimo y prueben las alertas de inventario.
          CONTINUE WHEN (v_d - v_ini) > 3 AND random() < 0.25;
          CONTINUE WHEN (v_d - v_ini) > 3 AND ((v_lid = v_locs[2] AND v_k = 6) OR (v_lid = v_locs[1] AND v_k = 9));
          v_cant := CASE WHEN (v_d - v_ini) <= 3 THEN
                      CASE WHEN (v_lid = v_locs[2] AND v_k = 6) OR (v_lid = v_locs[1] AND v_k = 9) THEN 6 ELSE 10 END
                    ELSE 3 + floor(random() * 7) END;
          SELECT round(costo_referencia * (0.92 + random()::numeric * 0.16), 2) INTO v_costo FROM inventario_items WHERE id = v_items[v_k];
          INSERT INTO detalle_compras (compra_id, item_id, cantidad, costo_unitario, subtotal)
          VALUES (v_compra, v_items[v_k], v_cant, v_costo, round(v_cant * v_costo, 2));
          v_total := v_total + round(v_cant * v_costo, 2);
          PERFORM pg_temp.demo_mov_stock(v_items[v_k], v_lid, 'ENTRADA_COMPRA', v_cant, v_costo, v_compra, NULL, v_ts, 'Compra ' || v_codigo, v_admin, 'admin');
          UPDATE inventario_items SET costo_referencia = v_costo, fecha_modificacion = v_ts WHERE id = v_items[v_k];
        END LOOP;
        UPDATE compras_inventario SET total = v_total WHERE id = v_compra;
        v_mc := v_mc + 1;
        INSERT INTO movimientos_financieros (codigo, naturaleza, clase, categoria_id, compra_id, local_id, local_nombre, fecha, monto, descripcion,
                                             proveedor, metodo_pago, estado, id_usuario, nombre_usuario, fecha_creacion, fecha_modificacion)
        VALUES ('DEMO-M-' || lpad(v_mc::text, 5, '0'), 'EGRESO', 'VARIABLE', c_compras, v_compra, v_lid, v_lname, v_d, v_total,
                'Compra ' || v_codigo, v_prov, v_metodo, 'PAGADO', v_admin, 'admin', v_ts, v_ts);
      END IF;

      -- Consumo semanal en cabina (insumos que no se venden)
      IF (v_d - v_ini) % 7 = (v_lid + 2) % 7 THEN
        FOREACH v_k IN ARRAY ARRAY[1, 2, 5, 6, 7, 10] LOOP
          v_qty := 1 + floor(random() * 2)::int;
          IF pg_temp.demo_stock(v_items[v_k], v_lid) >= v_qty THEN
            PERFORM pg_temp.demo_mov_stock(v_items[v_k], v_lid, 'SALIDA_SERVICIO', v_qty, NULL, NULL, NULL,
                                           (v_d::timestamp + interval '8 hours 30 minutes') AT TIME ZONE tz, 'Consumo en cabina', v_admin, 'admin');
          END IF;
        END LOOP;
      END IF;

      -- Ventas del dia
      v_base := CASE extract(dow FROM v_d) WHEN 0 THEN 0.6 WHEN 6 THEN 4.5 ELSE 7 END
                * CASE WHEN v_lid = v_locs[1] THEN 1.0 ELSE 0.85 END * v_mf;
      v_n := round(v_base * (0.6 + random()::numeric * 0.8))::int;
      v_hmax := CASE WHEN v_d = v_hoy THEN extract(hour FROM (now() AT TIME ZONE tz))::int - 1 ELSE 19 END;
      CONTINUE WHEN v_n <= 0 OR v_hmax < 9;

      FOR v_i IN 1..v_n LOOP
        v_pc := v_pc + 1;
        v_h := 9 + floor(random() * (v_hmax - 9 + 1))::int;
        v_ts := make_timestamptz(extract(year FROM v_d)::int, extract(month FROM v_d)::int, extract(day FROM v_d)::int, v_h,
                                 floor(random() * 60)::int, 0, tz);
        v_tipo := CASE WHEN random() < 0.45 THEN 'efectivo' WHEN random() < 0.64 THEN 'qr' WHEN random() < 0.6 THEN 'transferencia' ELSE 'tarjeta' END;
        v_activo := random() >= 0.03;
        v_estado := CASE WHEN v_activo AND random() < 0.01 THEN 'PENDIENTE' ELSE 'PAGADO' END;
        IF random() < 0.7 THEN
          v_k := 1 + floor(random() * array_length(v_clientes, 1))::int;
          v_cli := v_clientes[v_k]; v_cli_nombre := v_cnames[v_k];
        ELSE
          v_cli := NULL; v_cli_nombre := 'Consumidor final';
        END IF;
        IF v_lid = v_ger_local AND v_ger IS NOT NULL AND random() < 0.6 THEN
          v_cajero := v_ger; v_cajero_nom := 'gerente';
        ELSE
          v_cajero := v_admin; v_cajero_nom := 'admin';
        END IF;

        INSERT INTO pagos (codigo_pago, local_id, local_nombre, cliente_id, cliente_nombre, subtotal, descuento, total_final, estado, activo,
                           fecha_creacion, fecha_modificacion, tipo_pago, id_cajero, nombre_cajero, username_cajero, registrado_en, es_abono)
        VALUES ('DEMO-P-' || lpad(v_pc::text, 6, '0'), v_lid, v_lname, v_cli, v_cli_nombre, 0, 0, 0, v_estado, v_activo,
                v_ts, v_ts, v_tipo, v_cajero, v_cajero_nom, v_cajero_nom, v_ts, false)
        RETURNING id INTO v_pago;

        v_sub := 0;
        v_grupo := CASE WHEN random() < 0.94 THEN 'barato' WHEN random() < 0.95 THEN 'medio' ELSE 'caro' END;
        v_nlin := CASE WHEN v_grupo = 'barato' THEN 1 + (random() < 0.30)::int + (random() < 0.08)::int ELSE 1 END;
        FOR v_k IN 1..v_nlin LOOP
          SELECT * INTO r FROM demo_svc WHERE local_id = v_lid AND grupo = v_grupo ORDER BY random() LIMIT 1;
          IF r IS NULL THEN
            SELECT * INTO r FROM demo_svc WHERE local_id = v_lid AND grupo = 'barato' ORDER BY random() LIMIT 1;
          END IF;
          v_qty := CASE WHEN r.grupo = 'barato' AND random() < 0.15 THEN 2 ELSE 1 END;
          INSERT INTO detalle_pagos (pago_id, servicio_id, servicio, precio_unitario, cantidad, subtotal)
          VALUES (v_pago, r.servicio_id, r.nombre, r.costo, v_qty, r.costo * v_qty);
          v_sub := v_sub + r.costo * v_qty;
        END LOOP;

        -- Venta de un producto (cremas, fajas, protector) junto al servicio
        IF v_activo AND v_estado = 'PAGADO' AND random() < 0.12 THEN
          v_item_idx := (ARRAY[3, 4, 8, 9])[1 + floor(random() * 4)::int];
          v_item := v_items[v_item_idx];
          v_qty := 1 + (random() < 0.2)::int;
          IF pg_temp.demo_stock(v_item, v_lid) >= v_qty THEN
            INSERT INTO detalle_pagos (pago_id, servicio, precio_unitario, cantidad, subtotal, item_id)
            SELECT v_pago, nombre, precio_venta, v_qty, precio_venta * v_qty, id FROM inventario_items WHERE id = v_item;
            SELECT v_sub + precio_venta * v_qty INTO v_sub FROM inventario_items WHERE id = v_item;
            PERFORM pg_temp.demo_mov_stock(v_item, v_lid, 'SALIDA_VENTA', v_qty,
                                           (SELECT costo_referencia FROM inventario_items WHERE id = v_item), NULL, v_pago, v_ts,
                                           'Venta en caja DEMO-P-' || lpad(v_pc::text, 6, '0'), v_cajero, v_cajero_nom);
          END IF;
        END IF;

        v_desc := CASE WHEN random() < 0.15 THEN floor(v_sub * (0.05 + random() * 0.10)) ELSE 0 END;
        UPDATE pagos SET subtotal = v_sub, descuento = v_desc, total_final = v_sub - v_desc WHERE id = v_pago;
      END LOOP;
    END LOOP;
  END LOOP;

  -- Si el azar no dejo ningun egreso anulado, anula uno para probar ese estado.
  IF NOT EXISTS (SELECT 1 FROM movimientos_financieros WHERE codigo LIKE 'DEMO-M-%' AND estado = 'ANULADO' AND gasto_fijo_id IS NULL) THEN
    UPDATE movimientos_financieros SET estado = 'ANULADO', motivo_anulacion = 'Registrado por error'
    WHERE id = (SELECT id FROM movimientos_financieros WHERE codigo LIKE 'DEMO-M-%' AND clase = 'VARIABLE' AND compra_id IS NULL
                AND gasto_fijo_id IS NULL AND planilla_id IS NULL AND naturaleza = 'EGRESO' ORDER BY id LIMIT 1);
  END IF;

  RAISE NOTICE 'Demo cargado: % pagos, % movimientos, % compras', v_pc, v_mc, v_cc;
END
$seed$;
