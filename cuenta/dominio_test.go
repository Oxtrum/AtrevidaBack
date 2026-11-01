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
		nombre  string
		reparto []Reparto
		texto   string
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
