package cuenta

import "testing"

// Cobro de la pantalla de caja: el cajero marca cargos que se pagan completos. El saldo disponible se usa primero
// (si el cliente quiere) y lo que falta se paga con un metodo; una recarga opcional queda como saldo disponible.

func TestPlanearCobro(t *testing.T) {
	casos := []struct {
		nombre                     string
		total, disponible, recarga float64
		usarSaldo                  bool
		usaSaldo, falta, nuevoPago float64
	}{
		{"saldo parcial y complemento", 250, 150, 0, true, 150, 100, 100},
		{"saldo que alcanza", 250, 400, 0, true, 250, 0, 0},
		{"saldo que alcanza mas recarga", 250, 400, 50, true, 250, 0, 50},
		{"sin saldo", 250, 0, 0, true, 0, 250, 250},
		{"el cliente guarda su saldo", 250, 150, 0, false, 0, 250, 250},
		{"guarda su saldo y recarga", 250, 150, 100, false, 0, 250, 350},
		{"complemento mas recarga", 250, 150, 40, true, 150, 100, 140},
		{"centavos", 0.1 + 0.2, 0.1, 0, true, 0.1, 0.2, 0.2},
	}
	for _, c := range casos {
		got := PlanearCobro(c.total, c.disponible, c.usarSaldo, c.recarga)
		if got.Total != desdeCentavos(centavos(c.total)) || got.UsaSaldo != c.usaSaldo || got.Falta != c.falta || got.NuevoPago != c.nuevoPago {
			t.Errorf("%s: PlanearCobro(%v, %v, %v, %v) = %+v", c.nombre, c.total, c.disponible, c.usarSaldo, c.recarga, got)
		}
	}
}

func TestCubrirCargosUsaPrimeroElSaldoYLuegoElPagoNuevo(t *testing.T) {
	cargos := []CargoACobrar{{ID: 10, Pendiente: 100}, {ID: 11, Pendiente: 150}}
	fuentes := []FuenteDinero{{PagoID: 1, Disponible: 150}, {PagoID: 2, Disponible: 100}}
	got, err := CubrirCargos(cargos, fuentes)
	if err != nil {
		t.Fatal(err)
	}
	want := []Cobro{{PagoID: 1, CargoID: 10, Monto: 100}, {PagoID: 1, CargoID: 11, Monto: 50}, {PagoID: 2, CargoID: 11, Monto: 100}}
	if len(got) != len(want) {
		t.Fatalf("cobros = %+v, esperados %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cobro %d = %+v, esperado %+v", i, got[i], want[i])
		}
	}
}

func TestCubrirCargosCadaCargoQuedaCompleto(t *testing.T) {
	cargos := []CargoACobrar{{ID: 1, Pendiente: 30}, {ID: 2, Pendiente: 70.5}, {ID: 3, Pendiente: 0.25}}
	fuentes := []FuenteDinero{{PagoID: 7, Disponible: 40}, {PagoID: 8, Disponible: 60.75}}
	cobros, err := CubrirCargos(cargos, fuentes)
	if err != nil {
		t.Fatal(err)
	}
	porCargo := map[int]int64{}
	porPago := map[int]int64{}
	for _, c := range cobros {
		porCargo[c.CargoID] += centavos(c.Monto)
		porPago[c.PagoID] += centavos(c.Monto)
	}
	for _, c := range cargos {
		if porCargo[c.ID] != centavos(c.Pendiente) {
			t.Fatalf("el cargo %d debe quedar completo: %v", c.ID, porCargo)
		}
	}
	if porPago[7] != 4000 || porPago[8] != 6075 {
		t.Fatalf("cada pago aporta lo que tiene: %v", porPago)
	}
}

func TestCubrirCargosSaltaLasFuentesVaciasYFallaSiNoAlcanza(t *testing.T) {
	cobros, err := CubrirCargos([]CargoACobrar{{ID: 1, Pendiente: 50}}, []FuenteDinero{{PagoID: 1, Disponible: 0}, {PagoID: 2, Disponible: 50}})
	if err != nil || len(cobros) != 1 || cobros[0].PagoID != 2 {
		t.Fatalf("cobros = %+v, err = %v", cobros, err)
	}
	if _, err := CubrirCargos([]CargoACobrar{{ID: 1, Pendiente: 50}}, []FuenteDinero{{PagoID: 1, Disponible: 49.99}}); err == nil {
		t.Fatal("si el dinero no alcanza no se cubre nada")
	}
	if got, err := CubrirCargos(nil, nil); err != nil || len(got) != 0 {
		t.Fatalf("sin cargos no hay cobros: %+v %v", got, err)
	}
}
