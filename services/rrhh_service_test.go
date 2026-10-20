package services

import (
	"testing"
	"time"
)

func cfgTest() ConfigTrastienda {
	c := configPorDefecto()
	c.AporteLaboralPct, c.AportePatronalPct, c.DiasMesFaltas = 10, 20, 30
	return c
}

func TestCalcularSueldoDesglose(t *testing.T) {
	v, usados, err := calcularSueldo(3000, true, novedadesPlanilla{Bonos: 300, DiasFalta: 1, OtrosDescuentos: 50}, cfgTest(), []float64{200}, false)
	if err != nil {
		t.Fatal(err)
	}
	// falta = 3000/30*1 = 100; ganado = 3000+300-100 = 3200; laboral 10% = 320; patronal 20% = 640
	if v.DescuentoFaltas != 100 || v.TotalGanado != 3200 || v.AporteLaboral != 320 || v.AportePatronal != 640 {
		t.Fatalf("desglose inesperado: %+v", v)
	}
	if usados != 1 || v.Adelantos != 200 || v.NetoPagar != 2630 || v.CostoEmpresa != 3840 {
		t.Fatalf("neto/costo inesperado: usados=%d %+v", usados, v)
	}
}

func TestCalcularSueldoSinAportes(t *testing.T) {
	v, _, err := calcularSueldo(1000, false, novedadesPlanilla{}, cfgTest(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if v.AporteLaboral != 0 || v.AportePatronal != 0 || v.NetoPagar != 1000 || v.CostoEmpresa != 1000 {
		t.Fatalf("sin aportes deberia pagar y costar el sueldo completo: %+v", v)
	}
}

func TestCalcularSueldoAdelantoNoCabeSeQuedaPendiente(t *testing.T) {
	// capacidad = 1000 - 100 (laboral) = 900: entra el de 500, el de 600 ya no cabe y se corta ahi.
	v, usados, err := calcularSueldo(1000, true, novedadesPlanilla{}, cfgTest(), []float64{500, 600, 10}, false)
	if err != nil {
		t.Fatal(err)
	}
	if usados != 1 || v.Adelantos != 500 || v.NetoPagar != 400 {
		t.Fatalf("usados=%d %+v", usados, v)
	}
}

func TestCalcularSueldoEditarExigeTodosLosAdelantos(t *testing.T) {
	if _, _, err := calcularSueldo(1000, true, novedadesPlanilla{}, cfgTest(), []float64{950}, true); err == nil {
		t.Fatal("el neto negativo debe rechazarse al editar")
	}
}

func TestCalcularSueldoValidaNovedades(t *testing.T) {
	casos := map[string]novedadesPlanilla{
		"bono negativo":    {Bonos: -1},
		"faltas > mes":     {DiasFalta: 31},
		"faltas anulan":    {DiasFalta: 30},
		"descuento > pago": {OtrosDescuentos: 5000},
	}
	for nombre, nov := range casos {
		if _, _, err := calcularSueldo(3000, true, nov, cfgTest(), nil, false); err == nil {
			t.Errorf("%s: esperaba error", nombre)
		}
	}
}

func TestFechaPagoUltimoDiaDelMes(t *testing.T) {
	feb := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	if got := fechaCobro(feb, 31); got.Day() != 28 {
		t.Fatalf("dia 31 en febrero debe caer el 28, fue %d", got.Day())
	}
}
