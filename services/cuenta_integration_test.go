package services

// Pruebas de integracion de la cuenta del cliente contra una BD PostgreSQL real (ver la cabecera de
// trastienda_integration_test.go): comprueban el SQL de cargos, abonos y aplicaciones, que las pruebas con
// repositorio falso no ejecutan. Requieren las migraciones 000071 a 000075.
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

func (e *cuentaEnv) cuenta() (disponible, porPagar, deuda float64) {
	e.t.Helper()
	c, err := e.svc.Obtener(e.clienteID, false)
	if err != nil {
		e.t.Fatal(err)
	}
	return c.Saldo.Disponible, c.Saldo.PorPagar, c.Saldo.Deuda
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
	if disp, por, deuda := e.cuenta(); disp != 0 || por != 1030 || deuda != 1030 {
		t.Fatalf("saldo inicial = disponible %v por pagar %v deuda %v", disp, por, deuda)
	}

	// Pago parcial aplicado al paquete.
	if _, err := e.abono(600, cuenta.Reparto{CargoID: paquete, Monto: 600}); err != nil {
		t.Fatal(err)
	}
	if estado, pend := e.estadoCargo(paquete); estado != cuenta.CargoParcial || pend != 400 {
		t.Fatalf("paquete = %s pendiente %v", estado, pend)
	}
	if disp, por, deuda := e.cuenta(); disp != 0 || por != 430 || deuda != 430 {
		t.Fatalf("saldo tras el primer abono = disponible %v por pagar %v deuda %v", disp, por, deuda)
	}

	// Reglas de integridad: no se aplica mas que el pago ni mas que lo pendiente, y no queda nada a medias.
	if _, err := e.abono(100, cuenta.Reparto{CargoID: paquete, Monto: 101}); err == nil {
		t.Fatal("aplicar mas que el monto del abono debe fallar")
	}
	if _, err := e.abono(5000, cuenta.Reparto{CargoID: anestesia, Monto: 31}); err == nil {
		t.Fatal("aplicar mas que lo pendiente del cargo debe fallar")
	}
	if disp, por, deuda := e.cuenta(); disp != 0 || por != 430 || deuda != 430 {
		t.Fatalf("un abono rechazado no debe cambiar la cuenta: %v %v %v", disp, por, deuda)
	}

	// Adelanto sin aplicar: queda como saldo a favor.
	adelanto, err := e.abono(500)
	if err != nil {
		t.Fatal(err)
	}
	if disp, por, deuda := e.cuenta(); disp != 500 || por != 430 || deuda != 0 {
		t.Fatalf("tras el adelanto = disponible %v por pagar %v deuda %v", disp, por, deuda)
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
	if disp, por, deuda := e.cuenta(); disp != 70 || por != 0 || deuda != 0 {
		t.Fatalf("tras cubrir con el adelanto = disponible %v por pagar %v deuda %v", disp, por, deuda)
	}

	// Anular el abono libera lo que habia aplicado.
	if err := pgsqlrepo.NewPagosRepo(e.db).DeletePago(adelanto); err != nil {
		t.Fatal(err)
	}
	if estado, pend := e.estadoCargo(paquete); estado != cuenta.CargoParcial || pend != 400 {
		t.Fatalf("tras anular el abono, paquete = %s pendiente %v", estado, pend)
	}
	if disp, por, deuda := e.cuenta(); disp != 0 || por != 430 || deuda != 430 {
		t.Fatalf("tras anular el abono = disponible %v por pagar %v deuda %v", disp, por, deuda)
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
	if disp, por, deuda := e.cuenta(); disp != 0 || por != 400 || deuda != 400 {
		t.Fatalf("tras anular la anestesia = disponible %v por pagar %v deuda %v", disp, por, deuda)
	}
	if err := e.svc.AnularCargo(anestesia, "otra vez"); err == nil {
		t.Fatal("anular dos veces el mismo cargo debe fallar")
	}
}

func TestTrastiendaCuentaCargoEnCeroNoMueveElSaldo(t *testing.T) {
	e := setupCuenta(t)
	id := e.cargo("INSUMO", "Consumo sin cobro", 0)
	if disp, por, deuda := e.cuenta(); disp != 0 || por != 0 || deuda != 0 {
		t.Fatalf("saldo = disponible %v por pagar %v deuda %v", disp, por, deuda)
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

func (e *cuentaEnv) cobrar(cargos []int, usarSaldo bool, recarga float64, tipoPago string) (string, error) {
	res, err := e.svc.Cobrar(CobrarInput{
		ClienteID: e.clienteID, LocalID: e.localID, LocalNombre: e.localName, CargoIDs: cargos, UsarSaldo: usarSaldo,
		Recarga: recarga, TipoPago: tipoPago, Cajero: CajeroAuditoriaInput{Nombre: "Prueba"},
	})
	return res.CodigoPago, err
}

// El cajero marca dos servicios de 100 y 150, el cliente tiene 150 de saldo disponible y paga el resto con QR.
func TestTrastiendaCuentaCobroUsaElSaldoYSeComplementaConUnMetodo(t *testing.T) {
	e := setupCuenta(t)
	a := e.cargo("SERVICIO", "Masaje", 100)
	b := e.cargo("SERVICIO", "Limpieza", 150)
	c := e.cargo("SERVICIO", "Paquete", 400)
	if _, err := e.abono(150); err != nil {
		t.Fatal(err)
	}

	codigo, err := e.cobrar([]int{a, b}, true, 0, "qr")
	if err != nil || codigo == "" {
		t.Fatalf("codigo = %q, err = %v", codigo, err)
	}
	for _, id := range []int{a, b} {
		if estado, _ := e.estadoCargo(id); estado != cuenta.CargoPagado {
			t.Fatalf("el cargo %d debe quedar pagado completo: %s", id, estado)
		}
	}
	if estado, pend := e.estadoCargo(c); estado != cuenta.CargoPendiente || pend != 400 {
		t.Fatalf("el cargo no marcado no se toca: %s %v", estado, pend)
	}
	if disp, por, deuda := e.cuenta(); disp != 0 || por != 400 || deuda != 400 {
		t.Fatalf("tras cobrar = disponible %v por pagar %v deuda %v", disp, por, deuda)
	}
	// 150 del saldo (100 + 50) y 100 del pago nuevo: tres registros de cobro, ninguno editado.
	if total, _ := e.cobrosDelCliente(); total != 3 {
		t.Fatalf("filas de cobro = %d, esperadas 3", total)
	}
}

func TestTrastiendaCuentaCobroPuedeGuardarElSaldoYRecargar(t *testing.T) {
	e := setupCuenta(t)
	servicio := e.cargo("SERVICIO", "Masaje", 400)
	if _, err := e.abono(300); err != nil {
		t.Fatal(err)
	}

	// El cliente prefiere guardar su saldo: todo se paga con tarjeta.
	if _, err := e.cobrar([]int{servicio}, false, 0, "tarjeta"); err != nil {
		t.Fatal(err)
	}
	if disp, por, _ := e.cuenta(); disp != 300 || por != 0 {
		t.Fatalf("el saldo guardado no se toca: disponible %v por pagar %v", disp, por)
	}

	// Solo recarga, sin cargos.
	if _, err := e.cobrar(nil, true, 50, "efectivo"); err != nil {
		t.Fatal(err)
	}
	if disp, _, _ := e.cuenta(); disp != 350 {
		t.Fatalf("disponible tras recargar = %v", disp)
	}
}

func TestTrastiendaCuentaCobroConSaldoSuficienteNoPideMetodo(t *testing.T) {
	e := setupCuenta(t)
	if _, err := e.abono(350); err != nil {
		t.Fatal(err)
	}
	chico := e.cargo("SERVICIO", "Masaje", 100)
	grande := e.cargo("SERVICIO", "Paquete", 500)

	if codigo, err := e.cobrar([]int{chico}, true, 0, ""); err != nil || codigo != "" {
		t.Fatalf("el saldo cubre todo: codigo = %q, err = %v", codigo, err)
	}
	// Quedan 250 de saldo y el cargo es de 500: sin metodo de pago no se puede cobrar y no queda nada a medias.
	if _, err := e.cobrar([]int{grande}, true, 0, ""); err == nil {
		t.Fatal("si falta dinero nuevo hace falta un metodo de pago")
	}
	if disp, por, _ := e.cuenta(); disp != 250 || por != 500 {
		t.Fatalf("un cobro rechazado no cambia la cuenta: disponible %v por pagar %v", disp, por)
	}
}

func TestTrastiendaCuentaCobroRechazaCargosAjenosOYaPagados(t *testing.T) {
	e := setupCuenta(t)
	pagado := e.cargo("SERVICIO", "Masaje", 100)
	if _, err := e.cobrar([]int{pagado}, false, 0, "efectivo"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.cobrar([]int{pagado}, false, 0, "efectivo"); err == nil {
		t.Fatal("un cargo ya pagado no se cobra dos veces")
	}
	if _, err := e.cobrar([]int{999999999}, false, 0, "efectivo"); err == nil {
		t.Fatal("un cargo que no es del cliente no se cobra")
	}
}

// Tres personas comparten un telefono: vincular una reserva "con las demas del telefono" solo debe alcanzar a las
// que estan escritas con el mismo nombre; las de otra persona se quedan por vincular.
func TestTrastiendaCuentaVincularMismoTelefonoRespetaElNombre(t *testing.T) {
	e := setupCuenta(t)
	telefono := "+5917" + e.sufijo[len(e.sufijo)-7:]
	nueva := func(nombre string) int {
		var id int
		if err := e.db.Get(&id, `
			INSERT INTO reservas (local_id, local_nombre, tipo_espacio, fecha, hora_desde, hora_hasta, cliente, estado,
				servicio_nombre, telefono_e164, numero_telefono)
			VALUES ($1, $2, 'M', CURRENT_DATE + 1, '10:00', '11:00', $3, 'PENDIENTE', 'masaje', $4, $4) RETURNING id`,
			e.localID, e.localName, nombre, telefono); err != nil {
			t.Fatal(err)
		}
		return id
	}
	maria1 := nueva("Maria Lopez")
	maria2 := nueva("  maria lopez ")
	pedro := nueva("Pedro Pascal")

	repo := pgsqlrepo.NewReservasVinculoRepo(e.db)
	n, err := repo.Vincular(maria1, e.clienteID, true)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("vinculadas = %d, quiero 2 (las dos de Maria)", n)
	}
	cliente := func(id int) *int {
		var c *int
		if err := e.db.Get(&c, `SELECT cliente_id FROM reservas WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
		return c
	}
	if c := cliente(maria2); c == nil || *c != e.clienteID {
		t.Fatalf("la otra reserva de Maria debe quedar vinculada, cliente_id=%v", c)
	}
	if c := cliente(pedro); c != nil {
		t.Fatalf("la reserva de Pedro comparte telefono pero no debe vincularse, cliente_id=%v", *c)
	}
}
