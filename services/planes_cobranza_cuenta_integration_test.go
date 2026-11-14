package services

// El pago de un paquete vendido a cuenta (un cargo ligado al plan) debe verse en el propio plan: la ficha del cliente
// y la lista de paquetes leen estado_cobranza. Requiere la base descartable (ver trastienda_integration_test.go).

import (
	"testing"

	"atrevida-agenda-api/cuenta"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

func TestTrastiendaCuentaElPagoDelPaqueteSeVeEnSuCobranza(t *testing.T) {
	e := setupCuenta(t)
	srv := e.nuevoServicio("Paquete cobranza", nil)
	planID, _ := e.nuevoPlanConSesion(srv)
	mustExec(t, e.db, `UPDATE planes SET cliente_id = $1, precio_total = 1000, local_nombre_texto = $3, combo_nombre_texto = 'Paquete cobranza' WHERE id = $2`, e.clienteID, planID, e.localName)
	cargoID, err := e.svc.CrearCargo(CrearCargoInput{ClienteID: e.clienteID, Tipo: "PAQUETE", Concepto: "Paquete cobranza", Monto: 1000, PlanID: &planID})
	if err != nil {
		t.Fatal(err)
	}

	repo := pgsqlrepo.NewPlanesRepo(e.db)
	cobranza := func() string {
		planes, err := repo.ListPlanes(repository.FiltroPlanes{ClienteID: &e.clienteID})
		if err != nil || len(planes) != 1 {
			t.Fatalf("planes = %d, err = %v", len(planes), err)
		}
		detalle, err := repo.GetPlanByID(planID)
		if err != nil || detalle.EstadoCobranza != planes[0].EstadoCobranza {
			t.Fatalf("el detalle y el listado deben coincidir: %v / %v (%v)", detalle, planes[0].EstadoCobranza, err)
		}
		return planes[0].EstadoCobranza
	}

	if got := cobranza(); got != "PENDIENTE" {
		t.Fatalf("recien vendido a cuenta = %s, quiero PENDIENTE", got)
	}
	if _, err := e.abono(400, cuenta.Reparto{CargoID: cargoID, Monto: 400}); err != nil {
		t.Fatal(err)
	}
	if got := cobranza(); got != "PARCIAL" {
		t.Fatalf("con 400 de 1000 = %s, quiero PARCIAL", got)
	}
	if _, err := e.abono(600, cuenta.Reparto{CargoID: cargoID, Monto: 600}); err != nil {
		t.Fatal(err)
	}
	if got := cobranza(); got != "PAGADO" {
		t.Fatalf("con 1000 de 1000 = %s, quiero PAGADO", got)
	}

	pagados, err := repo.ListPlanes(repository.FiltroPlanes{ClienteID: &e.clienteID, EstadoCobranza: "PAGADO"})
	if err != nil || len(pagados) != 1 {
		t.Fatalf("el filtro por cobranza debe usar el mismo valor: %d, %v", len(pagados), err)
	}
}
