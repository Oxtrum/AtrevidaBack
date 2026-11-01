package services

// Pruebas de integracion de la cuenta del cliente contra una BD PostgreSQL real (ver la cabecera de
// trastienda_integration_test.go): comprueban el SQL de cargos, abonos y aplicaciones, que las pruebas con
// repositorio falso no ejecutan. Requieren la migracion 000070 y la 000071.
//
//	TRASTIENDA_TEST_DB=atrevida_migtest go test ./services -run TrastiendaCuenta -v

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"atrevida-agenda-api/cuenta"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

type cuentaEnv struct {
	*trastiendaEnv
	svc       *CuentaService
	clienteID int
}

func setupCuenta(t *testing.T) *cuentaEnv {
	t.Helper()
	env := setupTrastienda(t)
	svc := NewCuentaService(pgsqlrepo.NewCuentaRepo(env.db))
	svc.Config = env.cfgSvc
	var clienteID int
	if err := env.db.Get(&clienteID, `
		INSERT INTO clientes (nombre, apellido, numero_telefono)
		VALUES ($1, 'Prueba', $2) RETURNING id`, "Cuenta"+env.sufijo, env.sufijo[len(env.sufijo)-9:]); err != nil {
		t.Fatalf("no se pudo crear el cliente de prueba: %v", err)
	}
	return &cuentaEnv{trastiendaEnv: env, svc: svc, clienteID: clienteID}
}

func (e *cuentaEnv) cargo(tipo, concepto string, monto float64) int {
	e.t.Helper()
	id, err := e.svc.CrearCargo(CrearCargoInput{ClienteID: e.clienteID, Tipo: tipo, Concepto: concepto, Monto: monto})
	if err != nil {
		e.t.Fatalf("CrearCargo(%s): %v", concepto, err)
	}
	return id
}

func (e *cuentaEnv) abono(monto float64, aplicaciones ...cuenta.Reparto) (string, error) {
	return e.svc.CrearAbono(CrearAbonoInput{
		ClienteID: e.clienteID, LocalID: e.localID, LocalNombre: e.localName, Monto: monto, TipoPago: "efectivo",
		Aplicaciones: aplicaciones, Cajero: CajeroAuditoriaInput{Nombre: "Prueba"},
	})
}

func (e *cuentaEnv) cuenta() (saldo string, absoluto float64, sinAplicar float64) {
	e.t.Helper()
	c, err := e.svc.Obtener(e.clienteID, false)
	if err != nil {
		e.t.Fatal(err)
	}
	return c.Saldo.Etiqueta, c.Saldo.Absoluto, c.SinAplicar
}

func (e *cuentaEnv) estadoCargo(id int) (estado string, pendiente float64) {
	e.t.Helper()
	c, err := e.svc.Obtener(e.clienteID, true)
	if err != nil {
		e.t.Fatal(err)
	}
	for _, cg := range c.Cargos {
		if cg.ID == id {
			return cg.Estado, cg.Pendiente
		}
	}
	e.t.Fatalf("cargo %d no aparece en la cuenta", id)
	return "", 0
}

// cobrosDelCliente cuenta las filas de pago_aplicaciones de los cargos del cliente, vigentes o anuladas.
func (e *cuentaEnv) cobrosDelCliente() (total, anuladas int) {
	e.t.Helper()
	if err := e.db.Get(&total, `
		SELECT COUNT(*) FROM pago_aplicaciones a JOIN cargos_cliente c ON c.id = a.cargo_id WHERE c.cliente_id = $1`, e.clienteID); err != nil {
		e.t.Fatal(err)
	}
	if err := e.db.Get(&anuladas, `
		SELECT COUNT(*) FROM pago_aplicaciones a JOIN cargos_cliente c ON c.id = a.cargo_id
		WHERE c.cliente_id = $1 AND a.anulado = TRUE AND a.motivo_anulacion IS NOT NULL`, e.clienteID); err != nil {
		e.t.Fatal(err)
	}
	return total, anuladas
}

func TestTrastiendaCuentaFlujoCompleto(t *testing.T) {
	e := setupCuenta(t)

	paquete := e.cargo("PAQUETE", "Paquete Relax", 1000)
	anestesia := e.cargo("INSUMO", "Anestesia", 30)
	if s, abs, _ := e.cuenta(); s != cuenta.SaldoDeuda || abs != 1030 {
		t.Fatalf("saldo inicial = %s %v", s, abs)
	}

	// Pago parcial aplicado al paquete.
	if _, err := e.abono(600, cuenta.Reparto{CargoID: paquete, Monto: 600}); err != nil {
		t.Fatal(err)
	}
	if estado, pend := e.estadoCargo(paquete); estado != cuenta.CargoParcial || pend != 400 {
		t.Fatalf("paquete = %s pendiente %v", estado, pend)
	}
	if s, abs, _ := e.cuenta(); s != cuenta.SaldoDeuda || abs != 430 {
		t.Fatalf("saldo tras el primer abono = %s %v", s, abs)
	}

	// Reglas de integridad: no se aplica mas que el pago ni mas que lo pendiente, y no queda nada a medias.
	if _, err := e.abono(100, cuenta.Reparto{CargoID: paquete, Monto: 101}); err == nil {
		t.Fatal("aplicar mas que el monto del abono debe fallar")
	}
	if _, err := e.abono(5000, cuenta.Reparto{CargoID: anestesia, Monto: 31}); err == nil {
		t.Fatal("aplicar mas que lo pendiente del cargo debe fallar")
	}
	if s, abs, _ := e.cuenta(); s != cuenta.SaldoDeuda || abs != 430 {
		t.Fatalf("un abono rechazado no debe cambiar la cuenta: %s %v", s, abs)
	}

	// Adelanto sin aplicar: queda como saldo a favor.
	adelanto, err := e.abono(500)
	if err != nil {
		t.Fatal(err)
	}
	if s, abs, sin := e.cuenta(); s != cuenta.SaldoAFavor || abs != 70 || sin != 500 {
		t.Fatalf("tras el adelanto = %s %v sin aplicar %v", s, abs, sin)
	}

	// Se aplica despues, ya con los cargos creados.
	if err := e.svc.AplicarAbono(adelanto, e.clienteID, []cuenta.Reparto{{CargoID: paquete, Monto: 400}, {CargoID: anestesia, Monto: 30}}); err != nil {
		t.Fatal(err)
	}
	if estado, _ := e.estadoCargo(paquete); estado != cuenta.CargoPagado {
		t.Fatalf("paquete = %s", estado)
	}
	if estado, _ := e.estadoCargo(anestesia); estado != cuenta.CargoPagado {
		t.Fatalf("anestesia = %s", estado)
	}
	if s, abs, sin := e.cuenta(); s != cuenta.SaldoAFavor || abs != 70 || sin != 70 {
		t.Fatalf("tras aplicar el adelanto = %s %v sin aplicar %v", s, abs, sin)
	}

	// Anular el abono libera lo que habia aplicado.
	if err := pgsqlrepo.NewPagosRepo(e.db).DeletePago(adelanto); err != nil {
		t.Fatal(err)
	}
	if estado, pend := e.estadoCargo(paquete); estado != cuenta.CargoParcial || pend != 400 {
		t.Fatalf("tras anular el abono, paquete = %s pendiente %v", estado, pend)
	}
	if s, abs, _ := e.cuenta(); s != cuenta.SaldoDeuda || abs != 430 {
		t.Fatalf("tras anular el abono = %s %v", s, abs)
	}
	// Las dos filas que cubria el abono se conservan como anuladas, con motivo; no se borran.
	if total, anuladas := e.cobrosDelCliente(); total != 3 || anuladas != 2 {
		t.Fatalf("tras anular el abono hay %d filas de cobro y %d anuladas, esperadas 3 y 2", total, anuladas)
	}

	// Anular un cargo: sale del saldo.
	if err := e.svc.AnularCargo(anestesia, "cargada por error"); err != nil {
		t.Fatal(err)
	}
	var motivo string
	if err := e.db.Get(&motivo, `
		SELECT a.motivo_anulacion FROM pago_aplicaciones a WHERE a.cargo_id = $1 AND a.pago_id = (
			SELECT id FROM pagos WHERE codigo_pago = $2) LIMIT 1`, anestesia, adelanto); err != nil {
		t.Fatal(err)
	}
	if motivo != "Pago anulado" {
		t.Fatalf("el cobro ya anulado conserva su primer motivo, quedo %q", motivo)
	}
	if s, abs, _ := e.cuenta(); s != cuenta.SaldoDeuda || abs != 400 {
		t.Fatalf("tras anular la anestesia = %s %v", s, abs)
	}
	if err := e.svc.AnularCargo(anestesia, "otra vez"); err == nil {
		t.Fatal("anular dos veces el mismo cargo debe fallar")
	}
}

func TestTrastiendaCuentaCargoEnCeroNoMueveElSaldo(t *testing.T) {
	e := setupCuenta(t)
	id := e.cargo("INSUMO", "Consumo sin cobro", 0)
	if s, abs, _ := e.cuenta(); s != cuenta.SaldoAlDia || abs != 0 {
		t.Fatalf("saldo = %s %v", s, abs)
	}
	if estado, _ := e.estadoCargo(id); estado != cuenta.CargoPagado {
		t.Fatalf("un cargo en 0 no deja nada pendiente: %s", estado)
	}
}

func TestTrastiendaCuentaFechaDelHechoYRegistro(t *testing.T) {
	e := setupCuenta(t)
	codigo, err := e.svc.CrearAbono(CrearAbonoInput{
		ClienteID: e.clienteID, LocalID: e.localID, LocalNombre: e.localName, Monto: 50, TipoPago: "qr",
		FechaHecho: time.Now().AddDate(0, 0, -3).Format("2006-01-02"), Cajero: CajeroAuditoriaInput{Nombre: "Prueba"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dias float64
	if err := e.db.Get(&dias, `SELECT EXTRACT(EPOCH FROM (registrado_en - fecha_creacion)) / 86400 FROM pagos WHERE codigo_pago = $1`, codigo); err != nil {
		t.Fatal(err)
	}
	if dias < 2 || dias > 4 {
		t.Fatalf("registrado_en debe ser unos 3 dias posterior a la fecha del hecho, fue %.2f", dias)
	}
}

// Dos cobros simultaneos sobre el mismo cargo: solo uno puede aplicarse completo.
func TestTrastiendaCuentaNoSobreAplicaConCobrosSimultaneos(t *testing.T) {
	e := setupCuenta(t)
	cargo := e.cargo("SERVICIO", "Sesion", 150)

	var wg sync.WaitGroup
	resultados := make([]error, 2)
	for i := range resultados {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, resultados[i] = e.abono(100, cuenta.Reparto{CargoID: cargo, Monto: 100})
		}(i)
	}
	wg.Wait()

	exitos := 0
	for _, err := range resultados {
		if err == nil {
			exitos++
			continue
		}
		var regla pgsqlrepo.ErrReglaCuenta
		if !errors.As(err, &regla) {
			t.Fatalf("el rechazo debe ser una regla de la cuenta: %v", err)
		}
	}
	if exitos != 1 {
		t.Fatalf("con 150 pendientes solo un cobro de 100 cabe, hubo %d exitos (%v)", exitos, fmt.Sprint(resultados))
	}
	if _, pend := e.estadoCargo(cargo); pend != 50 {
		t.Fatalf("pendiente = %v, esperado 50", pend)
	}
}

func TestTrastiendaCuentaListaSoloAQuienesDeben(t *testing.T) {
	e := setupCuenta(t)
	cargo := e.cargo("PAQUETE", "Paquete", 200)
	if _, err := e.abono(50, cuenta.Reparto{CargoID: cargo, Monto: 50}); err != nil {
		t.Fatal(err)
	}
	busqueda := FiltroDeudoresInput{Busqueda: "Cuenta" + e.sufijo}

	lista, err := e.svc.ListarDeudores(busqueda)
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 1 || lista[0].ClienteID != e.clienteID || lista[0].Deuda != 150 || lista[0].TotalCargos != 200 || lista[0].TotalAbonos != 50 {
		t.Fatalf("deudores = %+v", lista)
	}
	if n, err := e.svc.ContarDeudores(busqueda); err != nil || n != 1 {
		t.Fatalf("conteo = %d, %v", n, err)
	}

	// Al saldar la deuda deja de aparecer.
	if _, err := e.abono(150, cuenta.Reparto{CargoID: cargo, Monto: 150}); err != nil {
		t.Fatal(err)
	}
	lista, err = e.svc.ListarDeudores(busqueda)
	if err != nil || len(lista) != 0 {
		t.Fatalf("un cliente al dia no es deudor: %+v, %v", lista, err)
	}
}
