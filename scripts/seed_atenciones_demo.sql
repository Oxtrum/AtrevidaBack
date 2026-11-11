-- Datos de prueba para Atenciones y Reportes de RRHH: reservas completadas con su atencion y reservas agendadas
-- por atender, en cada local. Cubre desde el inicio del mes anterior hasta hoy, y es reproducible (setseed).
-- Usa los empleados y clientes que ya existan (por ejemplo los de scripts/seed_finanzas_demo.sql); si no hay
-- empleados en un local, las atenciones de ese local salen con nombres escritos a mano.
-- Todo queda marcado para poder borrarlo con scripts/clean_atenciones_demo.sql:
--   reservas.notas = 'DEMO-ATENCION' (las atenciones cuelgan de esas reservas)
-- No descuenta inventario ni consume sesiones de paquetes.
-- Uso: go run ./cmd/_sql -f scripts/seed_atenciones_demo.sql   (o ejecutarlo con cualquier cliente de Postgres)

DO $seed$
DECLARE
  v_hoy       date := CURRENT_DATE;
  v_ini       date := (date_trunc('month', CURRENT_DATE) - interval '1 month')::date;
  v_dia       date;
  v_loc       record;
  v_emps      int[];
  v_emp       int;
  v_emp_nom   text;
  v_servs     text[];
  v_serv      text;
  v_clis      int[];
  v_cli       int;
  v_cli_nom   text;
  v_cli_tel   text;
  v_users     int[];
  v_uid       int;
  v_unom      text;
  v_n         int;
  v_i         int;
  v_hora      time;
  v_dur       int;
  v_res       int;
  v_idx       int;
  v_externos  constant text[] := ARRAY['Paola Mendoza', 'Valeria Ortiz'];
  v_nombres   constant text[] := ARRAY['Carla Gutierrez', 'Monica Salinas', 'Patricia Arce', 'Gabriela Rios', 'Silvia Camacho',
                                       'Natalia Vega', 'Fernanda Lopez', 'Claudia Montano', 'Veronica Suarez', 'Jimena Torrez'];
BEGIN
  IF EXISTS (SELECT 1 FROM reservas WHERE notas = 'DEMO-ATENCION') THEN
    RAISE NOTICE 'Ya hay datos DEMO-ATENCION; ejecuta primero scripts/clean_atenciones_demo.sql';
    RETURN;
  END IF;
  PERFORM setseed(0.42);

  SELECT array_agg(nombre ORDER BY id) INTO v_servs FROM (SELECT id, nombre FROM servicios WHERE activo ORDER BY id LIMIT 8) s;
  IF v_servs IS NULL THEN
    v_servs := ARRAY['Masaje reductor', 'Limpieza facial', 'Drenaje linfatico'];
  END IF;
  SELECT array_agg(id ORDER BY id) INTO v_clis FROM clientes;
  SELECT array_agg(id ORDER BY id) INTO v_users FROM (SELECT id FROM usuarios ORDER BY id LIMIT 3) u;

  FOR v_loc IN SELECT id, nombre FROM locales WHERE activo ORDER BY id LOOP
    -- Quienes atienden: el personal del local, sin recepcion ni limpieza.
    SELECT array_agg(id ORDER BY id) INTO v_emps FROM empleados
    WHERE activo AND local_id = v_loc.id AND COALESCE(cargo, '') NOT ILIKE ALL (ARRAY['%recep%', '%limpieza%']);

    v_dia := v_ini;
    WHILE v_dia <= v_hoy + 2 LOOP
      -- Domingo cerrado. Hasta hoy son atenciones hechas; hoy y los dos dias siguientes dejan ademas reservas agendadas.
      IF EXTRACT(DOW FROM v_dia) <> 0 THEN
        v_n := CASE WHEN v_dia > v_hoy THEN 3 ELSE 2 + floor(random() * 5)::int END;
        IF v_dia = v_hoy THEN v_n := v_n + 3; END IF;
        FOR v_i IN 1..v_n LOOP
          v_hora := make_time(8 + ((v_i * 2 + floor(random() * 2)::int) % 11), (floor(random() * 2)::int) * 30, 0);
          v_dur  := (ARRAY[30, 45, 60, 60, 90])[1 + floor(random() * 5)::int];
          v_serv := v_servs[1 + floor(random() * array_length(v_servs, 1))::int];

          v_cli := NULL;
          v_cli_tel := NULL;
          IF v_clis IS NOT NULL AND random() < 0.8 THEN
            v_cli := v_clis[1 + floor(random() * array_length(v_clis, 1))::int];
            SELECT BTRIM(CONCAT_WS(' ', nombre, apellido)), numero_telefono INTO v_cli_nom, v_cli_tel FROM clientes WHERE id = v_cli;
          ELSE
            v_cli_nom := v_nombres[1 + floor(random() * array_length(v_nombres, 1))::int];
          END IF;

          -- Las ultimas 3 de hoy y todas las de los dias siguientes quedan AGENDADAS, sin atender.
          IF v_dia > v_hoy OR (v_dia = v_hoy AND v_i > v_n - 3) THEN
            INSERT INTO reservas (local_id, local_nombre, tipo_espacio, fecha, hora_desde, hora_hasta, cliente, cliente_id, numero_telefono,
                                  servicio_nombre, servicio_solicitado, servicio_confirmado, estado, notas)
            VALUES (v_loc.id, v_loc.nombre, 'M', v_dia, v_hora, v_hora + make_interval(mins => v_dur), v_cli_nom, v_cli, v_cli_tel,
                    v_serv, v_serv, v_serv, 'AGENDADO', 'DEMO-ATENCION');
            CONTINUE;
          END IF;

          INSERT INTO reservas (local_id, local_nombre, tipo_espacio, fecha, hora_desde, hora_hasta, cliente, cliente_id, numero_telefono,
                                servicio_nombre, servicio_solicitado, servicio_confirmado, estado, notas)
          VALUES (v_loc.id, v_loc.nombre, 'M', v_dia, v_hora, v_hora + make_interval(mins => v_dur), v_cli_nom, v_cli, v_cli_tel,
                  v_serv, v_serv, v_serv, 'COMPLETADO', 'DEMO-ATENCION')
          RETURNING id INTO v_res;

          -- Reparto desigual a proposito (el primero atiende mas) y 1 de cada 10 con un nombre escrito a mano.
          v_emp := NULL;
          IF v_emps IS NULL OR random() < 0.1 THEN
            v_emp_nom := v_externos[1 + floor(random() * array_length(v_externos, 1))::int];
          ELSE
            v_idx := 1 + floor(power(random(), 1.7) * array_length(v_emps, 1))::int;
            v_emp := v_emps[v_idx];
            SELECT BTRIM(CONCAT_WS(' ', nombre, apellido)) INTO v_emp_nom FROM empleados WHERE id = v_emp;
          END IF;

          v_uid := NULL;
          v_unom := 'demo';
          IF v_users IS NOT NULL THEN
            v_uid := v_users[1 + floor(power(random(), 1.5) * array_length(v_users, 1))::int];
            SELECT username INTO v_unom FROM usuarios WHERE id = v_uid;
          END IF;

          -- 1 de cada 12 se registra sin horario, como pasa cuando solo se anota quien atendio.
          INSERT INTO atenciones (reserva_id, empleado_id, empleado_nombre, fecha_atencion, hora_inicio, hora_fin, local_id, lugar,
                                  observaciones, registrado_por_id, registrado_por, creado_en)
          VALUES (v_res, v_emp, v_emp_nom, v_dia,
                  CASE WHEN random() < 0.08 THEN NULL ELSE v_hora END,
                  NULL, v_loc.id, v_loc.nombre, NULL, v_uid, v_unom,
                  (v_dia + v_hora + make_interval(mins => v_dur + 5)) AT TIME ZONE 'America/La_Paz');
          UPDATE atenciones SET hora_fin = hora_inicio + make_interval(mins => v_dur) WHERE reserva_id = v_res AND hora_inicio IS NOT NULL;
        END LOOP;
      END IF;
      v_dia := v_dia + 1;
    END LOOP;
  END LOOP;

  RAISE NOTICE 'Atenciones demo: %, reservas agendadas demo: %',
    (SELECT COUNT(*) FROM atenciones a JOIN reservas r ON r.id = a.reserva_id WHERE r.notas = 'DEMO-ATENCION'),
    (SELECT COUNT(*) FROM reservas WHERE notas = 'DEMO-ATENCION' AND estado = 'AGENDADO');
END
$seed$;
