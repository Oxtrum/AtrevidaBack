package services

import (
	"strings"
	"testing"
	"time"

	"atrevida-agenda-api/cuenta"
	"atrevida-agenda-api/models"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

type cuentaFake struct {
	existe       bool
	cargos       []models.CargoCliente
	abonos       []models.AbonoCliente
	totalAbonos  float64
	totalCargos  float64
	cargoCreado  *pgsqlrepo.CargoInput
	abonoCreado  *pgsqlrepo.AbonoInput
	anuladoID    int
	anuladoMotivo string
	aplicado     []cuenta.Reparto
	deudores     []models.DeudorCliente
	filtroDeudor pgsqlrepo.FiltroDeudores
}

func (f *cuentaFake) ClienteExiste(int) (bool, error) { return f.existe, nil }
func (f *cuentaFake) CrearCargo(in pgsqlrepo.CargoInput) (int, error) {
	f.cargoCreado = &in
	return 12, nil
}
func (f *cuentaFake) AnularCargo(id int, motivo string) error {
	f.anuladoID, f.anuladoMotivo = id, motivo
	return nil
}
func (f *cuentaFake) ListarCargos(int, bool) ([]models.CargoCliente, error) { return f.cargos, nil }
func (f *cuentaFake) ListarAbonos(int) ([]models.AbonoCliente, error)      { return f.abonos, nil }
func (f *cuentaFake) TotalesCuenta(int) (float64, float64, error)          { return f.totalAbonos, f.totalCargos, nil }
func (f *cuentaFake) CrearAbono(in pgsqlrepo.AbonoInput) (string, error) {
	f.abonoCreado = &in
	return "PAGO-000007", nil
}
func (f *cuentaFake) AplicarAbono(_ string, _ int, ap []cuenta.Reparto) error {
	f.aplicado = ap
	return nil
}

func (f *cuentaFake) ListarDeudores(fl pgsqlrepo.FiltroDeudores) ([]models.DeudorCliente, error) {
	f.filtroDeudor = fl
	return f.deudores, nil
}
func (f *cuentaFake) ContarDeudores(pgsqlrepo.FiltroDeudores) (int, error) { return len(f.deudores), nil }

func servicioCuenta(f *cuentaFake) *CuentaService {
	s := NewCuentaService(f)
	s.Config = configFija{configPorDefecto()}
	s.Ahora = func() time.Time { return ahoraPrueba }
	return s
}

func TestObtenerCuentaCalculaSaldoYSinAplicar(t *testing.T) {
	f := &cuentaFake{
		existe: true, totalAbonos: 1500, totalCargos: 1030,
		cargos: []models.CargoCliente{{ID: 1, Monto: 1000, Pagado: 1000}, {ID: 2, Monto: 30, Pagado: 0}},
		abonos: []models.AbonoCliente{{CodigoPago: "PAGO-000001", Monto: 1500, Aplicado: 1000, SinAplicar: 500}},
	}
	got, err := servicioCuenta(f).Obtener(7, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Saldo.Etiqueta != cuenta.SaldoAFavor || got.Saldo.Absoluto != 470 {
		t.Fatalf("saldo = %+v", got.Saldo)
	}
	if got.SinAplicar != 500 || got.TotalCargos != 1030 || got.TotalAbonos != 1500 {
		t.Fatalf("totales = %+v", got)
	}
}

func TestObtenerCuentaDeUnClienteInexistente(t *testing.T) {
	_, err := servicioCuenta(&cuentaFake{existe: false}).Obtener(99, false)
	if err == nil || !strings.Contains(err.Error(), "cliente no encontrado") {
		t.Fatalf("error = %v", err)
	}
}

func TestCrearCargoValidaYNormaliza(t *testing.T) {
	f := &cuentaFake{existe: true}
	id, err := servicioCuenta(f).CrearCargo(CrearCargoInput{
		ClienteID: 7, Tipo: "paquete", Concepto: "  Paquete Relax  ", Monto: 1000, FechaHecho: "2026-10-25", CreadoPor: "Juan",
	})
	if err != nil || id != 12 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	c := f.cargoCreado
	if c == nil || c.Tipo != "PAQUETE" || c.Concepto != "Paquete Relax" || c.Monto != 1000 {
		t.Fatalf("cargo = %#v", c)
	}
	want := time.Date(2026, 10, 25, 12, 0, 0, 0, laPaz)
	if c.FechaHecho == nil || !c.FechaHecho.Equal(want) {
		t.Fatalf("fecha = %v", c.FechaHecho)
	}
}

func TestCrearCargoAceptaMontoCeroParaUnConsumoSinCobro(t *testing.T) {
	f := &cuentaFake{existe: true}
	if _, err := servicioCuenta(f).CrearCargo(CrearCargoInput{ClienteID: 7, Tipo: "INSUMO", Concepto: "Anestesia", Monto: 0}); err != nil {
		t.Fatalf("un cargo en 0 es valido: %v", err)
	}
}

func TestCrearCargoRechazaDatosInvalidos(t *testing.T) {
	casos := []struct {
		nombre string
		in     CrearCargoInput
		texto  string
	}{
		{"tipo invalido", CrearCargoInput{ClienteID: 7, Tipo: "REGALO", Concepto: "x", Monto: 1}, "tipo invalido"},
		{"sin concepto", CrearCargoInput{ClienteID: 7, Tipo: "OTRO", Concepto: "  ", Monto: 1}, "concepto es requerido"},
		{"concepto largo", CrearCargoInput{ClienteID: 7, Tipo: "OTRO", Concepto: strings.Repeat("a", 301), Monto: 1}, "no puede superar 300"},
		{"monto negativo", CrearCargoInput{ClienteID: 7, Tipo: "OTRO", Concepto: "x", Monto: -1}, "no puede ser negativo"},
		{"sin cliente", CrearCargoInput{Tipo: "OTRO", Concepto: "x", Monto: 1}, "cliente_id es requerido"},
		{"fecha futura", CrearCargoInput{ClienteID: 7, Tipo: "OTRO", Concepto: "x", Monto: 1, FechaHecho: "2026-10-29"}, "no puede ser futura"},
	}
	for _, c := range casos {
		f := &cuentaFake{existe: true}
		_, err := servicioCuenta(f).CrearCargo(c.in)
		if err == nil || !strings.Contains(err.Error(), c.texto) {
			t.Fatalf("%s: error = %v", c.nombre, err)
		}
		if f.cargoCreado != nil {
			t.Fatalf("%s: un cargo invalido no debe llegar al repositorio", c.nombre)
		}
	}
}

func TestAnularCargoExigeMotivo(t *testing.T) {
	f := &cuentaFake{}
	if err := servicioCuenta(f).AnularCargo(12, "   "); err == nil || !strings.Contains(err.Error(), "motivo es requerido") {
		t.Fatalf("error = %v", err)
	}
	if err := servicioCuenta(f).AnularCargo(12, " duplicado "); err != nil {
		t.Fatal(err)
	}
	if f.anuladoID != 12 || f.anuladoMotivo != "duplicado" {
		t.Fatalf("anulado = %d %q", f.anuladoID, f.anuladoMotivo)
	}
}

func abonoValido() CrearAbonoInput {
	return CrearAbonoInput{
		ClienteID: 7, LocalID: 1, LocalNombre: "SAN MARTIN", Monto: 500, TipoPago: "QR",
		Aplicaciones: []cuenta.Reparto{{CargoID: 1, Monto: 400}},
		Cajero:       CajeroAuditoriaInput{Nombre: "Juan"},
	}
}

func TestCrearAbonoNormalizaYEnviaAlRepositorio(t *testing.T) {
	f := &cuentaFake{existe: true}
	codigo, err := servicioCuenta(f).CrearAbono(abonoValido())
	if err != nil || codigo != "PAGO-000007" {
		t.Fatalf("codigo=%q err=%v", codigo, err)
	}
	a := f.abonoCreado
	if a == nil || a.TipoPago != "qr" || a.Monto != 500 || len(a.Aplicaciones) != 1 || a.CajeroNombre != "Juan" {
		t.Fatalf("abono = %#v", a)
	}
}

func TestCrearAbonoSinAplicacionesQuedaComoSaldoAFavor(t *testing.T) {
	f := &cuentaFake{existe: true}
	in := abonoValido()
	in.Aplicaciones = nil
	if _, err := servicioCuenta(f).CrearAbono(in); err != nil {
		t.Fatalf("un abono sin aplicaciones es valido: %v", err)
	}
}

func TestCrearAbonoRechazaDatosInvalidos(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*CrearAbonoInput)
		texto  string
	}{
		{"monto cero", func(i *CrearAbonoInput) { i.Monto = 0 }, "monto debe ser mayor a 0"},
		{"monto negativo", func(i *CrearAbonoInput) { i.Monto = -5 }, "monto debe ser mayor a 0"},
		{"tipo de pago", func(i *CrearAbonoInput) { i.TipoPago = "cheque" }, "tipo"},
		{"sin local", func(i *CrearAbonoInput) { i.LocalID = 0 }, "local_id es requerido"},
		{"sin cliente", func(i *CrearAbonoInput) { i.ClienteID = 0 }, "cliente_id es requerido"},
		{"fecha futura", func(i *CrearAbonoInput) { i.FechaHecho = "2026-10-29" }, "no puede ser futura"},
	}
	for _, c := range casos {
		f := &cuentaFake{existe: true}
		in := abonoValido()
		c.mutar(&in)
		_, err := servicioCuenta(f).CrearAbono(in)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), c.texto) {
			t.Fatalf("%s: error = %v", c.nombre, err)
		}
		if f.abonoCreado != nil {
			t.Fatalf("%s: un abono invalido no debe llegar al repositorio", c.nombre)
		}
	}
}

func TestAplicarAbonoExigeAlgoQueAplicar(t *testing.T) {
	f := &cuentaFake{}
	if err := servicioCuenta(f).AplicarAbono("PAGO-000007", 7, nil); err == nil || !strings.Contains(err.Error(), "aplicaciones es requerido") {
		t.Fatalf("error = %v", err)
	}
	if err := servicioCuenta(f).AplicarAbono("PAGO-000007", 7, []cuenta.Reparto{{CargoID: 2, Monto: 30}}); err != nil {
		t.Fatal(err)
	}
	if len(f.aplicado) != 1 || f.aplicado[0].CargoID != 2 {
		t.Fatalf("aplicado = %+v", f.aplicado)
	}
}

func TestListarDeudoresPasaElOrdenYLaPagina(t *testing.T) {
	f := &cuentaFake{deudores: []models.DeudorCliente{{ClienteID: 1, Deuda: 50}}}
	_, err := servicioCuenta(f).ListarDeudores(FiltroDeudoresInput{Busqueda: "  Maria ", OrdenarPor: "apellido", OrdenDesc: true, PageLimit: 51, Offset: 50})
	if err != nil {
		t.Fatal(err)
	}
	got := f.filtroDeudor
	if got.Busqueda != "Maria" || got.OrdenarPor != "apellido" || !got.OrdenDesc || got.Limit != 51 || got.Offset != 50 {
		t.Fatalf("filtro = %+v", got)
	}
}

func TestListarDeudoresRechazaOrdenFueraDeLaListaBlanca(t *testing.T) {
	f := &cuentaFake{}
	for _, clave := range []string{"password", "c.id", "deuda; DROP TABLE clientes"} {
		_, err := servicioCuenta(f).ListarDeudores(FiltroDeudoresInput{OrdenarPor: clave})
		if err == nil || !strings.Contains(err.Error(), "ordenar_por invalido") {
			t.Fatalf("%q: error = %v", clave, err)
		}
	}
	if _, err := servicioCuenta(f).ListarDeudores(FiltroDeudoresInput{Offset: -1}); err == nil || !strings.Contains(err.Error(), "offset invalido") {
		t.Fatalf("offset negativo: %v", err)
	}
}
