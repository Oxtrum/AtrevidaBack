package services

// Integracion de RRHH contra una BD real (ver trastienda_integration_test.go: TRASTIENDA_TEST_DB).
//
//	TRASTIENDA_TEST_DB=atrevida_rrhhtest go test ./services -run RRHH -v

import (
	"testing"

	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

func TestRRHHPlanillaGeneraEgresoYEsIdempotente(t *testing.T) {
	env := setupTrastienda(t)
	svc := NewRRHHService(pgsqlrepo.NewRRHHRepo(env.db), env.cfgSvc)
	cfg, _ := env.cfgSvc.Snapshot()
	hoy := cfg.Hoy()

	// Ingreso hace 2 meses, pago el dia 1: deben existir 3 sueldos (hace 2 meses, el pasado y este mes).
	ingreso := primerDiaMes(hoy).AddDate(0, -2, 0)
	dia := 1
	emp, err := svc.CrearEmpleado(EmpleadoInput{
		Nombre: "Ana", Apellido: "Test" + env.sufijo, SueldoBase: 3000, DiaPago: &dia, LocalID: &env.localID,
		FechaIngreso: ingreso.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RegistrarAdelanto(emp.ID, ingreso.Format("2006-01-02"), 200, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.GenerarHoy(); err != nil {
		t.Fatal(err)
	}
	if n, err := svc.GenerarHoy(); err != nil || n != 0 {
		t.Fatalf("segunda ejecucion debe generar 0 (idempotente): n=%d err=%v", n, err)
	}
	empID := emp.ID
	planillas, err := svc.ListarPlanillas(FiltroPlanillas{EmpleadoID: &empID})
	if err != nil || len(planillas) != 3 {
		t.Fatalf("esperaba 3 planillas, hay %d (err=%v)", len(planillas), err)
	}
	primera := planillas[len(planillas)-1] // orden periodo DESC
	if primera.Adelantos != 200 {
		t.Fatalf("el adelanto debe descontarse en la primera planilla: %+v", primera)
	}
	if primera.MovimientoCodigo == nil {
		t.Fatal("la planilla debe tener egreso asociado")
	}

	// El egreso nace PENDIENTE por el costo para la empresa y no se puede tocar desde Finanzas.
	var estado string
	var monto float64
	if err := env.db.QueryRow(`SELECT estado, monto FROM movimientos_financieros WHERE planilla_id = $1`, primera.ID).Scan(&estado, &monto); err != nil {
		t.Fatal(err)
	}
	if estado != "PENDIENTE" || monto != primera.CostoEmpresa {
		t.Fatalf("egreso inesperado: %s %.2f vs costo %.2f", estado, monto, primera.CostoEmpresa)
	}
	var movID int
	if err := env.db.Get(&movID, `SELECT id FROM movimientos_financieros WHERE planilla_id = $1`, primera.ID); err != nil {
		t.Fatal(err)
	}
	m := 1.0
	if err := env.fin.EditarMovimiento(movID, MovimientoPatch{Monto: &m}); err == nil {
		t.Fatal("editar el egreso de una planilla desde Finanzas debe rechazarse")
	}

	// Editar bonos y faltas recalcula la planilla y sincroniza el egreso.
	bonos, faltas := 300.0, 1.0
	edit, err := svc.ActualizarPlanilla(primera.ID, PlanillaPatch{Bonos: &bonos, DiasFalta: &faltas})
	if err != nil {
		t.Fatal(err)
	}
	if edit.TotalGanado != 3200 || edit.Adelantos != 200 {
		t.Fatalf("recalculo inesperado: %+v", edit)
	}
	if err := env.db.QueryRow(`SELECT monto FROM movimientos_financieros WHERE planilla_id = $1`, primera.ID).Scan(&monto); err != nil || monto != edit.CostoEmpresa {
		t.Fatalf("el egreso debe seguir al costo: %.2f vs %.2f (%v)", monto, edit.CostoEmpresa, err)
	}

	// Pagar pasa planilla y egreso a PAGADO y bloquea la edicion.
	if _, err := svc.Pagar(primera.ID, nil); err != nil {
		t.Fatal(err)
	}
	if err := env.db.QueryRow(`SELECT estado FROM movimientos_financieros WHERE planilla_id = $1`, primera.ID).Scan(&estado); err != nil || estado != "PAGADO" {
		t.Fatalf("el egreso debe quedar PAGADO: %s %v", estado, err)
	}
	if _, err := svc.ActualizarPlanilla(primera.ID, PlanillaPatch{Bonos: &bonos}); err == nil {
		t.Fatal("una planilla pagada no se edita")
	}

	// Anular libera el adelanto y anula el egreso; regenerar crea una planilla nueva que lo vuelve a descontar.
	if _, err := svc.Anular(primera.ID, "prueba"); err != nil {
		t.Fatal(err)
	}
	pend, _ := svc.ListarAdelantos(&empID, true)
	if len(pend) != 1 {
		t.Fatalf("el adelanto debe volver a pendiente tras anular: %d", len(pend))
	}
	if _, err := svc.GenerarHoy(); err != nil {
		t.Fatal(err)
	}
	if n, _ := svc.PendientesDeGenerar(); len(n) != 0 {
		t.Fatalf("el catch-up no debe regenerar periodos anulados: %v", n)
	}
	regenerada, err := svc.GenerarPlanillaManual(empID, ingreso.Format("2006-01"))
	if err != nil || regenerada.Adelantos != 200 {
		t.Fatalf("regenerar debe volver a descontar el adelanto: %+v %v", regenerada, err)
	}
	if _, err := svc.GenerarPlanillaManual(empID, ingreso.Format("2006-01")); err == nil {
		t.Fatal("no puede haber dos planillas vigentes del mismo periodo")
	}

	// Aguinaldo: ingreso hace ~2 meses (<90 dias) no genera.
	if _, err := svc.GenerarAguinaldos(hoy.Year()); err != nil {
		t.Fatal(err)
	}
}
