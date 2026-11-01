package cuenta

import (
	"strings"
	"testing"
	"time"
)

func TestEstadoCargo(t *testing.T) {
	casos := []struct {
		monto, pagado float64
		want          string
	}{
		{1000, 0, CargoPendiente},
		{1000, 400, CargoParcial},
		{1000, 999.99, CargoParcial},
		{1000, 1000, CargoPagado},
		{30, 30, CargoPagado},
		// Un cargo en 0 (consumo sin cobro) no deja nada pendiente.
		{0, 0, CargoPagado},
		// Los decimales de punto flotante no deben dejar un centavo fantasma.
		{0.3, 0.1 + 0.2, CargoPagado},
	}
	for _, c := range casos {
		if got := EstadoCargo(c.monto, c.pagado); got != c.want {
			t.Fatalf("EstadoCargo(%v, %v) = %s, esperado %s", c.monto, c.pagado, got, c.want)
		}
	}
}

// La cuenta muestra dos numeros: saldo disponible (dinero recargado y no usado) y por pagar (cargos sin pagar).
// La deuda real es lo que falta despues de usar el saldo disponible y nunca es negativa.
func TestPosicionClienteSeparaSaldoDisponibleYPorPagar(t *testing.T) {
	casos := []struct {
		disponible, porPagar float64
		deuda                float64
	}{
		{0, 1030, 1030},
		{150, 250, 100},
		{500, 430, 0},
		{70, 0, 0},
		{0, 0, 0},
		{0.3, 0.1 + 0.2, 0},
	}
	for _, c := range casos {
		got := PosicionCliente(c.disponible, c.porPagar)
		if got.Deuda != c.deuda || got.Disponible != desdeCentavos(centavos(c.disponible)) || got.PorPagar != desdeCentavos(centavos(c.porPagar)) {
			t.Fatalf("PosicionCliente(%v, %v) = %+v, deuda esperada %v", c.disponible, c.porPagar, got, c.deuda)
		}
	}
}

func fecha(d int) time.Time { return time.Date(2026, 10, d, 12, 0, 0, 0, time.UTC) }

func TestSugerirRepartoVaDelCargoMasAntiguoAlMasNuevo(t *testing.T) {
	cargos := []CargoPorCobrar{
		{ID: 3, Pendiente: 30, Fecha: fecha(20)},
		{ID: 1, Pendiente: 1000, Fecha: fecha(5)},
		{ID: 2, Pendiente: 50, Fecha: fecha(10)},
	}
	reparto, sobrante := SugerirReparto(cargos, 1040)
	if len(reparto) != 2 || reparto[0] != (Reparto{CargoID: 1, Monto: 1000}) || reparto[1] != (Reparto{CargoID: 2, Monto: 40}) {
		t.Fatalf("reparto = %+v", reparto)
	}
	if sobrante != 0 {
		t.Fatalf("sobrante = %v", sobrante)
	}
}

func TestSugerirRepartoEmpataPorIdYDejaElSobrante(t *testing.T) {
	cargos := []CargoPorCobrar{
		{ID: 8, Pendiente: 20, Fecha: fecha(5)},
		{ID: 7, Pendiente: 20, Fecha: fecha(5)},
	}
	reparto, sobrante := SugerirReparto(cargos, 100)
	if len(reparto) != 2 || reparto[0].CargoID != 7 || reparto[1].CargoID != 8 {
		t.Fatalf("a igual fecha desempata por id: %+v", reparto)
	}
	if sobrante != 60 {
		t.Fatalf("sobrante = %v, esperado 60 (saldo a favor)", sobrante)
	}
}

func TestSugerirRepartoSinCargosOSinMontoNoRepartePorNada(t *testing.T) {
	if reparto, sobrante := SugerirReparto(nil, 100); len(reparto) != 0 || sobrante != 100 {
		t.Fatalf("sin cargos todo es sobrante: %+v %v", reparto, sobrante)
	}
	if reparto, sobrante := SugerirReparto([]CargoPorCobrar{{ID: 1, Pendiente: 10, Fecha: fecha(1)}}, 0); len(reparto) != 0 || sobrante != 0 {
		t.Fatalf("sin monto no hay reparto: %+v %v", reparto, sobrante)
	}
}

func TestSugerirRepartoIgnoraCargosSinPendiente(t *testing.T) {
	cargos := []CargoPorCobrar{
		{ID: 1, Pendiente: 0, Fecha: fecha(1)},
		{ID: 2, Pendiente: 25.5, Fecha: fecha(2)},
	}
	reparto, sobrante := SugerirReparto(cargos, 25.5)
	if len(reparto) != 1 || reparto[0] != (Reparto{CargoID: 2, Monto: 25.5}) || sobrante != 0 {
		t.Fatalf("reparto = %+v, sobrante = %v", reparto, sobrante)
	}
}

func TestSugerirRepartoNoMutaLaEntrada(t *testing.T) {
	cargos := []CargoPorCobrar{{ID: 2, Pendiente: 5, Fecha: fecha(9)}, {ID: 1, Pendiente: 5, Fecha: fecha(1)}}
	SugerirReparto(cargos, 3)
	if cargos[0].ID != 2 || cargos[1].ID != 1 {
		t.Fatalf("la entrada no debe reordenarse: %+v", cargos)
	}
}

func TestValidarAplicacionesAceptaUnRepartoCorrecto(t *testing.T) {
	pendientes := map[int]float64{1: 1000, 2: 30}
	err := ValidarAplicaciones(500, 0, []Reparto{{1, 400}, {2, 30}}, pendientes)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestValidarAplicacionesPuedeAplicarMenosQueElPagoYDejarSaldo(t *testing.T) {
	if err := ValidarAplicaciones(500, 0, []Reparto{{1, 100}}, map[int]float64{1: 1000}); err != nil {
		t.Fatalf("el resto queda como saldo a favor: %v", err)
	}
	if err := ValidarAplicaciones(500, 0, nil, map[int]float64{1: 1000}); err != nil {
		t.Fatalf("un pago sin aplicaciones es valido: %v", err)
	}
}

func TestValidarAplicacionesRechazaExcederElPago(t *testing.T) {
	err := ValidarAplicaciones(500, 100, []Reparto{{1, 450}}, map[int]float64{1: 1000})
	if err == nil || !strings.Contains(err.Error(), "no puede superar el monto disponible del pago") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidarAplicacionesRechazaExcederElPendienteDelCargo(t *testing.T) {
	err := ValidarAplicaciones(5000, 0, []Reparto{{2, 31}}, map[int]float64{2: 30})
	if err == nil || !strings.Contains(err.Error(), "no puede superar lo pendiente del cargo 2") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidarAplicacionesRechazaCargoInexistenteMontoNoPositivoYDuplicados(t *testing.T) {
	pendientes := map[int]float64{1: 100}
	casos := []struct {
		nombre string
		reparto []Reparto
		texto  string
	}{
		{"cargo inexistente", []Reparto{{9, 10}}, "cargo 9 no encontrado"},
		{"monto cero", []Reparto{{1, 0}}, "monto debe ser mayor a 0"},
		{"monto negativo", []Reparto{{1, -5}}, "monto debe ser mayor a 0"},
		{"cargo repetido", []Reparto{{1, 10}, {1, 20}}, "cargo 1 repetido"},
	}
	for _, c := range casos {
		err := ValidarAplicaciones(500, 0, c.reparto, pendientes)
		if err == nil || !strings.Contains(err.Error(), c.texto) {
			t.Fatalf("%s: error = %v", c.nombre, err)
		}
	}
}

func TestPendienteDe(t *testing.T) {
	casos := []struct{ monto, pagado, want float64 }{
		{1000, 400, 600},
		{1000, 1000, 0},
		{0, 0, 0},
		// Un pago de mas no deja un pendiente negativo.
		{100, 150, 0},
		{0.3, 0.1 + 0.2, 0},
	}
	for _, c := range casos {
		if got := PendienteDe(c.monto, c.pagado); got != c.want {
			t.Fatalf("PendienteDe(%v, %v) = %v, esperado %v", c.monto, c.pagado, got, c.want)
		}
	}
}
