package services

// Integracion de frecuencias, adelanto de varios cobros y pago de cobros de gastos fijos contra una BD real
// (ver trastienda_integration_test.go).
//
//	TRASTIENDA_TEST_DB=atrevida_rrhhtest go test ./services -run GastoFijo -v

import (
	"reflect"
	"testing"
	"time"

	"atrevida-agenda-api/models"
)

func (e *trastiendaEnv) gastoConFrecuencia(frec string, dia int, inicio string, fin *string) int {
	e.t.Helper()
	g, err := e.gastos.Crear(GastoFijoInput{
		Nombre: "Gasto " + frec + e.sufijo, CategoriaID: e.categoriaE, Monto: 100, DiaCobro: dia, LocalID: &e.localID,
		FechaInicio: inicio, FechaFin: fin, Frecuencia: frec,
	})
	if err != nil {
		e.t.Fatal(err)
	}
	return g.ID
}

func TestTrastiendaGastoFijoTrimestralCobraCadaTresMeses(t *testing.T) {
	e := setupTrastienda(t)
	id := e.gastoConFrecuencia(FrecTrimestral, 5, "2027-01-01", nil)
	if _, err := e.gastos.GenerarHasta(fecha("2027-10-20")); err != nil {
		t.Fatal(err)
	}
	want := []string{"2027-01-05", "2027-04-05", "2027-07-05", "2027-10-05"}
	if got := e.periodos(id); !reflect.DeepEqual(got, want) {
		t.Fatalf("trimestral: esperaba %v, hay %v", want, got)
	}
}

func TestTrastiendaGastoFijoSemanalCobraCadaSieteDias(t *testing.T) {
	e := setupTrastienda(t)
	id := e.gastoConFrecuencia(FrecSemanal, 1, "2027-03-01", nil)
	if _, err := e.gastos.GenerarHasta(fecha("2027-03-20")); err != nil {
		t.Fatal(err)
	}
	want := []string{"2027-03-01", "2027-03-08", "2027-03-15"}
	if got := e.periodos(id); !reflect.DeepEqual(got, want) {
		t.Fatalf("semanal: esperaba %v, hay %v", want, got)
	}
	// Generar de nuevo no duplica.
	if n, err := e.gastos.GenerarHasta(fecha("2027-03-20")); err != nil || n != 0 {
		t.Fatalf("segunda generacion debe crear 0: n=%d err=%v", n, err)
	}
}

func TestTrastiendaGastoFijoContratoDeTresMesesTerminaSolo(t *testing.T) {
	e := setupTrastienda(t)
	fin := "2027-03-31"
	id := e.gastoConFrecuencia(FrecMensual, 5, "2027-01-01", &fin)
	if _, err := e.gastos.GenerarHasta(fecha("2027-09-01")); err != nil {
		t.Fatal(err)
	}
	if got := e.periodos(id); len(got) != 3 || got[2] != "2027-03-05" {
		t.Fatalf("un contrato de 3 meses debe generar solo ene-mar, hay %v", got)
	}
}

func TestTrastiendaGastoFijoAdelantarVariosCobros(t *testing.T) {
	e := setupTrastienda(t)
	id := e.gastoConFrecuencia(FrecMensual, 5, "2027-01-01", nil)
	if _, err := e.gastos.GenerarHasta(fecha("2027-01-20")); err != nil { // enero ya generado
		t.Fatal(err)
	}
	var fechas []string
	var adelantados []bool
	hoy := fecha("2027-01-20")
	err := e.gastos.recorrerVencidos(hoy, FiltroAdelanto{Cobros: 3}, func(g models.GastoFijo, _, cobro time.Time, adel bool) error {
		if g.ID == id {
			fechas = append(fechas, cobro.Format("2006-01-02"))
			adelantados = append(adelantados, adel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fechas, []string{"2027-02-05", "2027-03-05", "2027-04-05"}) {
		t.Fatalf("adelantar 3 debe ofrecer feb, mar y abr: %v", fechas)
	}
	for _, a := range adelantados {
		if !a {
			t.Fatalf("todos deben venir marcados como adelantados: %v", adelantados)
		}
	}
}

func TestTrastiendaGastoFijoPagarCobrosCreaYNoDuplica(t *testing.T) {
	e := setupTrastienda(t)
	cfg, _ := e.cfgSvc.Snapshot()
	hoy := cfg.Hoy()
	inicio := primerDiaMes(hoy).AddDate(0, -1, 0) // hace un mes: un cobro vencido
	id := e.gastoConFrecuencia(FrecMensual, 5, inicio.Format("2006-01-02"), nil)

	// Con un pago atrasado sin pagar no se puede adelantar.
	lejano := fechaCobro(primerDiaMes(hoy).AddDate(0, 2, 0), 5).Format("2006-01-02")
	if _, err := e.gastos.PagarCobros([]CobroPago{{GastoFijoID: id, FechaCobro: lejano}}, nil, "test"); err == nil {
		t.Fatal("adelantar con un pago atrasado pendiente debe rechazarse")
	}

	// Pagando el atrasado en la misma solicitud, y solo el cobro de dentro de 2 meses sin tocar los intermedios.
	vencidos, err := e.gastos.PendientesDeGenerar(FiltroAdelanto{GastoFijoID: &id})
	if err != nil || len(vencidos) == 0 {
		t.Fatalf("debe haber pagos vencidos: %v %v", vencidos, err)
	}
	pagar := []CobroPago{{GastoFijoID: id, FechaCobro: lejano}}
	for _, v := range vencidos {
		pagar = append(pagar, CobroPago{GastoFijoID: id, FechaCobro: v.FechaCobro})
	}
	if n, err := e.gastos.PagarCobros(pagar, nil, "test"); err != nil || n != len(pagar) {
		t.Fatalf("pagar atrasado y adelantado: n=%d err=%v", n, err)
	}
	if _, err := e.gastos.PagarCobros(pagar, nil, "test"); err != nil {
		t.Fatalf("repetir el pago no debe fallar: %v", err)
	}
	var cuantos int
	var estado string
	if err := e.db.Get(&cuantos, `SELECT COUNT(*) FROM movimientos_financieros WHERE gasto_fijo_id = $1 AND fecha = $2`, id, lejano); err != nil {
		t.Fatal(err)
	}
	if cuantos != 1 {
		t.Fatalf("pagar dos veces debe dejar 1 egreso, hay %d", cuantos)
	}
	if err := e.db.Get(&estado, `SELECT estado FROM movimientos_financieros WHERE gasto_fijo_id = $1 AND fecha = $2`, id, lejano); err != nil || estado != "PAGADO" {
		t.Fatalf("el egreso adelantado debe quedar PAGADO: %q %v", estado, err)
	}

	// Saltarse los meses del medio no los pierde: siguen ofreciendose.
	var intermedios []string
	if err := e.gastos.recorrerVencidos(hoy, FiltroAdelanto{Cobros: 3}, func(g models.GastoFijo, _, cobro time.Time, _ bool) error {
		if g.ID == id {
			intermedios = append(intermedios, cobro.Format("2006-01-02"))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, f := range intermedios {
		if f == lejano {
			t.Fatalf("el cobro ya pagado no debe ofrecerse de nuevo: %v", intermedios)
		}
	}
	if len(intermedios) < 2 {
		t.Fatalf("los cobros intermedios deben seguir pendientes de ofrecerse: %v", intermedios)
	}

	// Una fecha que no es cobro de la plantilla se rechaza.
	if _, err := e.gastos.PagarCobros([]CobroPago{{GastoFijoID: id, FechaCobro: fechaCobro(primerDiaMes(hoy).AddDate(0, 1, 0), 9).Format("2006-01-02")}}, nil, "test"); err == nil {
		t.Fatal("una fecha que no es cobro de la plantilla debe rechazarse")
	}
}

func TestTrastiendaGastoFijoSiguienteCobroTrasPagoAdelantado(t *testing.T) {
	g := models.GastoFijo{
		Frecuencia: FrecMensual, DiaCobro: 5, FechaInicio: "2027-01-01", Activo: true,
	}
	hoy := fecha("2027-01-20")
	if p := siguienteCobro(g, hoy); p == nil || p.Format("2006-01-02") != "2027-02-05" {
		t.Fatalf("sin pagos adelantados el siguiente es feb: %v", p)
	}
	pagado := "2027-03-05"
	g.PagadoHasta = &pagado
	if p := siguienteCobro(g, hoy); p == nil || p.Format("2006-01-02") != "2027-04-05" {
		t.Fatalf("pagado hasta mar, el siguiente es abr: %v", p)
	}
	fin := "2027-03-31"
	g.FechaFin = &fin
	if p := siguienteCobro(g, hoy); p != nil {
		t.Fatalf("con la vigencia terminada no hay siguiente cobro: %v", p)
	}
}

func TestTrastiendaGastoFijoAdelantarHastaUnaFechaYSoloUnGasto(t *testing.T) {
	e := setupTrastienda(t)
	id := e.gastoConFrecuencia(FrecMensual, 5, "2027-01-01", nil)
	otro := e.gastoConFrecuencia(FrecMensual, 5, "2027-01-01", nil)
	hoy := fecha("2027-01-20")
	hasta := fecha("2027-04-30")
	var fechas []string
	err := e.gastos.recorrerVencidos(hoy, FiltroAdelanto{Hasta: &hasta, GastoFijoID: &id}, func(g models.GastoFijo, _, cobro time.Time, adel bool) error {
		if g.ID == otro {
			t.Fatalf("el filtro por gasto no debe incluir otros gastos")
		}
		if g.ID == id && adel {
			fechas = append(fechas, cobro.Format("2006-01-02"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fechas, []string{"2027-02-05", "2027-03-05", "2027-04-05"}) {
		t.Fatalf("hasta el 30/04 debe ofrecer feb, mar y abr: %v", fechas)
	}
}
