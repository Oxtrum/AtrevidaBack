package services

import (
	"testing"
	"time"

	"atrevida-agenda-api/models"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

func TestPeriodosHaciaAtras(t *testing.T) {
	periodos, err := periodosHaciaAtras(PeriodoMensual, dia("2026-11-12"), 3)
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]string{{"2026-11-01", "2026-11-30"}, {"2026-10-01", "2026-10-31"}, {"2026-09-01", "2026-09-30"}}
	for i, p := range periodos {
		if got := [2]string{p.desde.Format(fechaISO), p.hasta.Format(fechaISO)}; got != want[i] {
			t.Errorf("periodo %d: got %v, want %v", i, got, want[i])
		}
	}
	if _, err := periodosHaciaAtras("anual", dia("2026-11-12"), 3); err == nil {
		t.Error("un periodo desconocido deberia fallar")
	}
}

func TestConstruirReporteClientes(t *testing.T) {
	id := func(v int) *int { return &v }
	fecha := func(s string) *time.Time { d := dia(s); return &d }
	pago := func(f string, local int, nombre string, cliente *int, monto float64) pgsqlrepo.PagoClienteFila {
		return pgsqlrepo.PagoClienteFila{Fecha: dia(f), LocalID: id(local), LocalNombre: nombre, ClienteID: cliente, Monto: monto}
	}
	reserva := func(f string, local int, nombre string, cliente *int, estado, motivo, servicio string) pgsqlrepo.ReservaClienteFila {
		return pgsqlrepo.ReservaClienteFila{
			Fecha: dia(f), LocalID: id(local), LocalNombre: nombre, ClienteID: cliente, Estado: estado,
			MotivoCancelacion: motivo, Servicio: servicio,
		}
	}
	periodos, err := periodosHaciaAtras(PeriodoMensual, dia("2026-11-12"), 4)
	if err != nil {
		t.Fatal(err)
	}

	d := datosReporteClientes{
		periodos: periodos,
		hoy:      dia("2026-11-12"),
		clientes: map[int]pgsqlrepo.ClienteBasico{
			1: {ID: 1, Nombre: "Ana Perez", Telefono: "70000001", PrimeraActividad: fecha("2026-08-03")},
			2: {ID: 2, Nombre: "Beto Rios", PrimeraActividad: fecha("2026-11-05")},
			3: {ID: 3, Nombre: "Carla Soto", PrimeraActividad: fecha("2026-10-02")},
			4: {ID: 4, Nombre: "Dora Vaca"},
		},
		pagos: []pgsqlrepo.PagoClienteFila{
			// Ana: paga en noviembre (dos locales), octubre y septiembre; en agosto no.
			pago("2026-11-03", 1, "SAN MARTIN", id(1), 300),
			pago("2026-11-10", 2, "PASEO ARANJUEZ", id(1), 100),
			pago("2026-10-08", 1, "SAN MARTIN", id(1), 200),
			pago("2026-09-08", 1, "SAN MARTIN", id(1), 50),
			// Beto: cliente nuevo.
			pago("2026-11-05", 1, "SAN MARTIN", id(2), 150),
			// Carla: solo pago en octubre, no volvio.
			pago("2026-10-02", 1, "SAN MARTIN", id(3), 80),
			// Venta de mostrador sin cliente.
			pago("2026-11-06", 1, "SAN MARTIN", nil, 50),
		},
		reservas: []pgsqlrepo.ReservaClienteFila{
			reserva("2026-11-03", 1, "SAN MARTIN", id(1), "COMPLETADO", "", "Masaje"),
			reserva("2026-11-10", 2, "PASEO ARANJUEZ", id(1), "COMPLETADO", "", "Masaje"),
			reserva("2026-11-11", 1, "SAN MARTIN", id(1), "COMPLETADO", "", "Sauna"),
			reserva("2026-11-20", 1, "SAN MARTIN", id(1), "AGENDADO", "", "Masaje"),
			reserva("2026-11-05", 1, "SAN MARTIN", id(2), "COMPLETADO", "", "Sauna"),
			// Dora solo falto: aparece en la lista pero no cuenta como activa.
			reserva("2026-11-04", 1, "SAN MARTIN", id(4), "CANCELADO", "INASISTENCIA", "Masaje"),
			reserva("2026-11-07", 1, "SAN MARTIN", nil, "COMPLETADO", "", "Masaje"),
			reserva("2026-11-08", 1, "SAN MARTIN", id(2), "RECHAZADO", "", "Masaje"),
		},
		deudas: []pgsqlrepo.DeudaClienteFila{
			{ClienteID: 3, LocalID: id(1), LocalNombre: "SAN MARTIN", Pendiente: 120, Cargos: 2, Desde: dia("2026-10-02")},
			{ClienteID: 1, LocalID: id(2), LocalNombre: "PASEO ARANJUEZ", Pendiente: 40, Cargos: 1, Desde: dia("2026-11-10")},
			// Dora debe 30 pero tiene 30 a favor sin aplicar: no es deudora.
			{ClienteID: 4, LocalID: id(1), LocalNombre: "SAN MARTIN", Pendiente: 30, Cargos: 1, Desde: dia("2026-11-04")},
		},
		saldosAFavor: map[int]float64{3: 20, 4: 30},
	}

	general, locales := construirReporteClientes(d)
	_, clientes := bloqueClientes(nil, "General", d)
	r := general.Resumen

	if r.ClientesActivos != 2 || r.ClientesActivosAnterior != 2 || r.ClientesNuevos != 1 || r.ClientesRecurrentes != 1 {
		t.Errorf("activos/nuevos: %+v", r)
	}
	if r.Ingresos != 550 || r.IngresosAnterior != 280 || r.IngresosSinCliente != 50 {
		t.Errorf("ingresos: %+v", r)
	}
	if r.IngresosSinClientePct != 8.3 {
		t.Errorf("sin cliente pct = %v, want 8.3", r.IngresosSinClientePct)
	}
	if r.ClientesQuePagaron != 2 || r.TicketPromedioCliente != 275 || r.TicketPromedioClienteAnterior != 140 {
		t.Errorf("ticket: %+v", r)
	}
	if r.Citas != 4 || r.CitasSinCliente != 1 || r.Canceladas != 1 || r.Inasistencias != 1 {
		t.Errorf("citas: %+v", r)
	}
	if r.ClientesQueVolvieron != 1 || r.TasaRetornoPct == nil || *r.TasaRetornoPct != 50 {
		t.Errorf("retorno: %+v", r)
	}
	if r.PorCobrar != 140 || r.Deudores != 2 || r.ClientesCompartidos != 1 {
		t.Errorf("deuda/compartidos: %+v", r)
	}

	if len(clientes) != 3 {
		t.Fatalf("clientes = %d, want 3 (Carla no tuvo movimiento en el periodo)", len(clientes))
	}
	ana := clientes[0]
	if ana.ClienteID != 1 || ana.Posicion != 1 || ana.Ingresos != 400 || ana.Pagos != 2 || ana.TicketPromedio != 200 {
		t.Errorf("ana dinero: %+v", ana)
	}
	if ana.Citas != 3 || ana.CitasAgendadas != 1 || ana.ServicioTop != "Masaje" || ana.ServicioTopCantidad != 2 {
		t.Errorf("ana citas: %+v", ana)
	}
	if ana.RachaPeriodos != 3 || ana.EsNuevo || ana.SaldoPendiente != 40 || ana.UltimaActividad != "2026-11-11" {
		t.Errorf("ana racha/saldo: %+v", ana)
	}
	if ana.VariacionPct == nil || *ana.VariacionPct != 100 || ana.Locales != "PASEO ARANJUEZ, SAN MARTIN" {
		t.Errorf("ana variacion/locales: %+v", ana)
	}
	beto := clientes[1]
	if beto.ClienteID != 2 || !beto.EsNuevo || beto.RachaPeriodos != 1 || beto.VariacionPct != nil {
		t.Errorf("beto: %+v", beto)
	}
	dora := clientes[2]
	if dora.ClienteID != 4 || dora.Canceladas != 1 || dora.Inasistencias != 1 || dora.RachaPeriodos != 0 {
		t.Errorf("dora: %+v", dora)
	}

	if len(general.Deudores) != 2 || general.Deudores[0].ClienteID != 3 || general.Deudores[0].DiasDeuda != 41 {
		t.Errorf("deudores: %+v", general.Deudores)
	}
	if len(general.Serie) != 4 || general.Serie[0].Fecha != "2026-11-03" || general.Serie[0].Ingresos != 300 || general.Serie[0].Clientes != 1 {
		t.Errorf("serie: %+v", general.Serie)
	}

	if len(locales) != 2 || locales[0].LocalNombre != "SAN MARTIN" || locales[1].LocalNombre != "PASEO ARANJUEZ" {
		t.Fatalf("locales: %+v", locales)
	}
	sm := locales[0].Resumen
	if sm.Ingresos != 450 || sm.ClientesActivos != 2 || sm.PorCobrar != 100 || sm.ClientesCompartidos != 0 || sm.IngresosSinCliente != 50 {
		t.Errorf("san martin: %+v", sm)
	}
	pa := locales[1]
	if pa.Resumen.Ingresos != 100 || pa.Resumen.ClientesActivos != 1 || pa.Resumen.TasaRetornoPct != nil || pa.TopIngresos[0].RachaPeriodos != 1 {
		t.Errorf("paseo aranjuez: %+v", pa)
	}
}

func TestRankingsYDetalleClientes(t *testing.T) {
	id := func(v int) *int { return &v }
	periodos, err := periodosHaciaAtras(PeriodoMensual, dia("2026-11-12"), 3)
	if err != nil {
		t.Fatal(err)
	}
	d := datosReporteClientes{periodos: periodos, hoy: dia("2026-11-12"), clientes: map[int]pgsqlrepo.ClienteBasico{}}
	// 15 clientes: el cliente n paga n*10 y completa (16-n) citas; los impares ademas cancelan una reserva.
	for n := 1; n <= 15; n++ {
		d.clientes[n] = pgsqlrepo.ClienteBasico{ID: n, Nombre: "Cliente " + string(rune('A'+n-1)), Telefono: "7000000" + string(rune('0'+n%10))}
		d.pagos = append(d.pagos, pgsqlrepo.PagoClienteFila{Fecha: dia("2026-11-03"), LocalID: id(1), LocalNombre: "SAN MARTIN", ClienteID: id(n), Monto: float64(n * 10)})
		for c := 0; c < 16-n; c++ {
			d.reservas = append(d.reservas, pgsqlrepo.ReservaClienteFila{Fecha: dia("2026-11-04"), LocalID: id(1), LocalNombre: "SAN MARTIN", ClienteID: id(n), Estado: "COMPLETADO", Servicio: "Masaje"})
		}
		if n%2 == 1 {
			d.reservas = append(d.reservas, pgsqlrepo.ReservaClienteFila{Fecha: dia("2026-11-05"), LocalID: id(1), LocalNombre: "SAN MARTIN", ClienteID: id(n), Estado: "CANCELADO"})
		}
		d.deudas = append(d.deudas, pgsqlrepo.DeudaClienteFila{ClienteID: n, LocalID: id(1), LocalNombre: "SAN MARTIN", Pendiente: float64(n), Cargos: 1, Desde: dia("2026-11-01")})
	}

	b, clientes := bloqueClientes(nil, "General", d)
	if len(clientes) != 15 {
		t.Fatalf("clientes = %d, want 15", len(clientes))
	}
	if len(b.TopIngresos) != 10 || b.TopIngresos[0].ClienteID != 15 || b.TopIngresos[9].ClienteID != 6 {
		t.Errorf("top ingresos: %d filas, primero %d", len(b.TopIngresos), b.TopIngresos[0].ClienteID)
	}
	if len(b.TopCitas) != 10 || b.TopCitas[0].ClienteID != 1 || b.TopCitas[0].Citas != 15 {
		t.Errorf("top citas: %+v", b.TopCitas[0])
	}
	if len(b.TopCanceladas) != 8 || len(b.TopFieles) != 0 {
		t.Errorf("canceladas = %d (want 8), fieles = %d (want 0)", len(b.TopCanceladas), len(b.TopFieles))
	}
	if len(b.Deudores) != 10 || b.Resumen.Deudores != 15 || b.Resumen.PorCobrar != 120 || b.Deudores[0].ClienteID != 15 {
		t.Errorf("deudores: lista %d, total %d, por cobrar %v", len(b.Deudores), b.Resumen.Deudores, b.Resumen.PorCobrar)
	}

	ids := func(filas []models.ClienteReporte) []int {
		out := make([]int, len(filas))
		for i, f := range filas {
			out[i] = f.ClienteID
		}
		return out
	}
	copia := func() []models.ClienteReporte { return append([]models.ClienteReporte{}, clientes...) }

	if got := ids(filtrarOrdenarClientes(copia(), "", "", false))[:3]; got[0] != 15 || got[1] != 14 || got[2] != 13 {
		t.Errorf("orden por defecto: %v", got)
	}
	if got := ids(filtrarOrdenarClientes(copia(), "", "citas", true))[:3]; got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("citas desc: %v", got)
	}
	if got := ids(filtrarOrdenarClientes(copia(), "", "nombre", false))[:2]; got[0] != 1 || got[1] != 2 {
		t.Errorf("nombre asc: %v", got)
	}
	// Empate en canceladas: desempata la posicion, asi el orden es estable entre paginas.
	if got := ids(filtrarOrdenarClientes(copia(), "", "canceladas", true))[:3]; got[0] != 15 || got[1] != 13 || got[2] != 11 {
		t.Errorf("canceladas desc: %v", got)
	}
	if got := ids(filtrarOrdenarClientes(copia(), "cliente c", "", false)); len(got) != 1 || got[0] != 3 {
		t.Errorf("busqueda: %v", got)
	}
	// Sin dato (nadie pago el mes anterior): variacion_pct queda en el orden por posicion en ambos sentidos.
	if got := ids(filtrarOrdenarClientes(copia(), "", "variacion_pct", true))[:2]; got[0] != 15 || got[1] != 14 {
		t.Errorf("variacion desc sin datos: %v", got)
	}
}
