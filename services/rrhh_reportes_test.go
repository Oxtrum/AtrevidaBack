package services

import (
	"testing"
	"time"

	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

func dia(s string) time.Time {
	t, err := time.Parse(fechaISO, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestRangoPeriodoReporte(t *testing.T) {
	casos := []struct {
		periodo, fecha                   string
		desde, hasta, antDesde, antHasta string
	}{
		// 2026-11-11 es miercoles: la semana va de lunes 9 a domingo 15.
		{PeriodoSemanal, "2026-11-11", "2026-11-09", "2026-11-15", "2026-11-02", "2026-11-08"},
		{PeriodoSemanal, "2026-11-15", "2026-11-09", "2026-11-15", "2026-11-02", "2026-11-08"},
		{PeriodoSemanal, "2026-11-09", "2026-11-09", "2026-11-15", "2026-11-02", "2026-11-08"},
		{PeriodoQuincenal, "2026-11-15", "2026-11-01", "2026-11-15", "2026-10-16", "2026-10-31"},
		{PeriodoQuincenal, "2026-11-16", "2026-11-16", "2026-11-30", "2026-11-01", "2026-11-15"},
		{PeriodoQuincenal, "2026-03-05", "2026-03-01", "2026-03-15", "2026-02-16", "2026-02-28"},
		{PeriodoMensual, "2026-11-11", "2026-11-01", "2026-11-30", "2026-10-01", "2026-10-31"},
		{PeriodoMensual, "2026-01-31", "2026-01-01", "2026-01-31", "2025-12-01", "2025-12-31"},
	}
	for _, c := range casos {
		d, h, ad, ah, err := rangoPeriodoReporte(c.periodo, dia(c.fecha))
		if err != nil {
			t.Fatalf("%s %s: %v", c.periodo, c.fecha, err)
		}
		got := [4]string{d.Format(fechaISO), h.Format(fechaISO), ad.Format(fechaISO), ah.Format(fechaISO)}
		want := [4]string{c.desde, c.hasta, c.antDesde, c.antHasta}
		if got != want {
			t.Errorf("%s %s: got %v, want %v", c.periodo, c.fecha, got, want)
		}
	}
	if _, _, _, _, err := rangoPeriodoReporte("anual", dia("2026-11-11")); err == nil {
		t.Error("un periodo desconocido deberia fallar")
	}
}

func TestConstruirReporteAtenciones(t *testing.T) {
	id := func(v int) *int { return &v }
	min := func(v float64) *float64 { return &v }
	fila := func(fecha string, local int, localNombre string, emp *int, nombre string, minutos *float64, cliente, servicio, usuario string) pgsqlrepo.AtencionReporteFila {
		return pgsqlrepo.AtencionReporteFila{
			Fecha: dia(fecha), LocalID: id(local), LocalNombre: localNombre, EmpleadoID: emp, Nombre: nombre,
			Minutos: minutos, Cliente: cliente, Servicio: servicio, RegistradoPor: usuario,
		}
	}
	actual := []pgsqlrepo.AtencionReporteFila{
		fila("2026-11-02", 1, "SAN MARTIN", id(4), "Maria Rojas", min(60), "c1", "Masaje", "recepcion"),
		fila("2026-11-02", 1, "SAN MARTIN", id(4), "Maria Rojas", min(30), "c2", "Masaje", "recepcion"),
		fila("2026-11-03", 2, "PASEO ARANJUEZ", id(4), "Maria Rojas", nil, "c1", "Facial", "admin"),
		fila("2026-11-03", 2, "PASEO ARANJUEZ", nil, "Ana  Perez", min(45), "c3", "Facial", "admin"),
		// Mismo nombre escrito con otras mayusculas: es la misma persona sin ficha.
		fila("2026-11-04", 2, "PASEO ARANJUEZ", nil, "ana perez", min(45), "c4", "Facial", "admin"),
	}
	anterior := []pgsqlrepo.AtencionReporteFila{
		fila("2026-10-05", 1, "SAN MARTIN", id(4), "Maria Rojas", min(60), "c1", "Masaje", "recepcion"),
		fila("2026-10-06", 1, "SAN MARTIN", id(4), "Maria Rojas", min(60), "c1", "Masaje", "recepcion"),
	}

	general, locales := construirReporteAtenciones(actual, anterior)

	r := general.Resumen
	if r.Atenciones != 5 || r.AtencionesAnterior != 2 || r.Personal != 2 || r.DiasConAtencion != 3 || r.ClientesDistintos != 4 {
		t.Fatalf("resumen general inesperado: %+v", r)
	}
	if r.VariacionPct == nil || *r.VariacionPct != 150 {
		t.Fatalf("variacion general: %v", r.VariacionPct)
	}
	if r.MinutosAtendidos != 180 || r.SinHorario != 1 || r.DuracionPromedioMin != 45 || r.PromedioPorPersona != 2.5 {
		t.Fatalf("tiempos generales inesperados: %+v", r)
	}

	if len(general.Ranking) != 2 {
		t.Fatalf("ranking general: %+v", general.Ranking)
	}
	maria, ana := general.Ranking[0], general.Ranking[1]
	if maria.Posicion != 1 || maria.Nombre != "Maria Rojas" || maria.Atenciones != 3 || maria.ParticipacionPct != 60 {
		t.Fatalf("primer puesto inesperado: %+v", maria)
	}
	if maria.DiasTrabajados != 2 || maria.PromedioPorDia != 1.5 || maria.ClientesDistintos != 2 || maria.MinutosAtendidos != 90 || maria.DuracionPromedioMin != 45 {
		t.Fatalf("metricas de Maria inesperadas: %+v", maria)
	}
	if maria.ServicioTop != "Masaje" || maria.ServicioTopCantidad != 2 || maria.UltimaAtencion != "2026-11-03" || maria.Locales != "PASEO ARANJUEZ, SAN MARTIN" {
		t.Fatalf("detalle de Maria inesperado: %+v", maria)
	}
	if maria.VariacionPct == nil || *maria.VariacionPct != 50 {
		t.Fatalf("variacion de Maria: %v", maria.VariacionPct)
	}
	if ana.Posicion != 2 || ana.Atenciones != 2 || ana.EmpleadoID != nil || ana.VariacionPct != nil {
		t.Fatalf("segundo puesto inesperado: %+v", ana)
	}

	if len(general.Registradores) != 2 || general.Registradores[0].Usuario != "admin" || general.Registradores[0].Registros != 3 {
		t.Fatalf("registradores inesperados: %+v", general.Registradores)
	}
	if len(general.Serie) != 3 || general.Serie[0].Fecha != "2026-11-02" || general.Serie[0].Atenciones != 2 {
		t.Fatalf("serie inesperada: %+v", general.Serie)
	}

	if len(locales) != 2 || locales[0].LocalNombre != "PASEO ARANJUEZ" || locales[0].Resumen.Atenciones != 3 {
		t.Fatalf("locales inesperados: %+v", locales)
	}
	// En Paseo Aranjuez gana Ana (2) sobre Maria (1); en San Martin solo atendio Maria.
	if locales[0].Ranking[0].Nombre != "Ana  Perez" || locales[0].Ranking[0].Atenciones != 2 || locales[0].Resumen.AtencionesAnterior != 0 {
		t.Fatalf("ranking de Paseo Aranjuez inesperado: %+v", locales[0])
	}
	sm := locales[1]
	if sm.Resumen.Atenciones != 2 || sm.Resumen.AtencionesAnterior != 2 || *sm.Resumen.VariacionPct != 0 || sm.Ranking[0].Nombre != "Maria Rojas" {
		t.Fatalf("bloque de San Martin inesperado: %+v", sm)
	}
}

func TestConstruirReporteAtencionesVacio(t *testing.T) {
	general, locales := construirReporteAtenciones(nil, nil)
	if general.Resumen.Atenciones != 0 || general.Resumen.VariacionPct != nil || len(locales) != 0 {
		t.Fatalf("reporte vacio inesperado: %+v", general)
	}
	if general.Ranking == nil || general.Serie == nil || general.Registradores == nil {
		t.Fatal("las listas vacias deben salir como [] y no null")
	}
}
