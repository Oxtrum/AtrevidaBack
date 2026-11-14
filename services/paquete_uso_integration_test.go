package services

// Desde cuando se puede usar un paquete (parametro paquetes.usar_desde): apenas se vende y queda a cuenta, o solo
// cuando ya esta pagado. Requiere la base descartable (ver trastienda_integration_test.go).

import (
	"strings"
	"testing"

	"atrevida-agenda-api/cuenta"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

func TestTrastiendaCuentaElPaqueteSeUsaSegunElParametro(t *testing.T) {
	e := setupCuenta(t)
	e.reservas.SetPlanes(pgsqlrepo.NewPlanesRepo(e.db))
	if _, err := e.cfgSvc.Update(map[string]interface{}{CfgUsarPaqueteDesde: "CUENTA"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = e.cfgSvc.Update(map[string]interface{}{CfgUsarPaqueteDesde: "CUENTA"}) })

	srv := e.nuevoServicio("Paquete uso", nil)
	planID, _ := e.nuevoPlanConSesion(srv)
	mustExec(t, e.db, `UPDATE planes SET cliente_id = $1, precio_total = 1000, local_nombre_texto = $3, combo_nombre_texto = 'Paquete uso' WHERE id = $2`, e.clienteID, planID, e.localName)
	cargoID, err := e.svc.CrearCargo(CrearCargoInput{ClienteID: e.clienteID, Tipo: "PAQUETE", Concepto: "Paquete uso", Monto: 1000, PlanID: &planID})
	if err != nil {
		t.Fatal(err)
	}
	usar := func() error {
		r := e.nuevaReserva(srv, "PENDIENTE", 2)
		_, err := e.reservas.ActualizarReservaConAvisos(ActualizarReservaPGInput{Id: r, Local: e.localName, NuevoPlanID: &planID})
		return err
	}

	// A cuenta: se puede usar aunque nada este pagado.
	if err := usar(); err != nil {
		t.Fatalf("con CUENTA el paquete sin pagar se puede usar: %v", err)
	}

	// Solo pagado: sin pagar no, con pago parcial tampoco, pagado del todo si.
	if _, err := e.cfgSvc.Update(map[string]interface{}{CfgUsarPaqueteDesde: "PAGADO"}); err != nil {
		t.Fatal(err)
	}
	if err := usar(); err == nil || !strings.Contains(err.Error(), "pagado") {
		t.Fatalf("con PAGADO y sin pagar debe rechazarse, err = %v", err)
	}
	if _, err := e.abono(400, cuenta.Reparto{CargoID: cargoID, Monto: 400}); err != nil {
		t.Fatal(err)
	}
	if err := usar(); err == nil {
		t.Fatal("con PAGADO y pago parcial debe rechazarse")
	}
	if _, err := e.abono(600, cuenta.Reparto{CargoID: cargoID, Monto: 600}); err != nil {
		t.Fatal(err)
	}
	if err := usar(); err != nil {
		t.Fatalf("con PAGADO y pagado del todo se puede usar: %v", err)
	}
}
