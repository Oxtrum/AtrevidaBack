package services

// Cargos de insumo de la cuenta del cliente contra una BD PostgreSQL real (ver la cabecera de
// trastienda_integration_test.go). Requiere la migracion 000077.
//
//	TRASTIENDA_TEST_DB=atrevida_migtest go test ./services -run TrastiendaCargoInsumo -v

import (
	"errors"
	"strings"
	"testing"

	repository "atrevida-agenda-api/repositories"
)

func (e *cuentaEnv) cargoInsumo(item int, cantidad, monto float64) (int, error) {
	e.t.Helper()
	return e.svc.CrearCargo(CrearCargoInput{
		ClienteID: e.clienteID, LocalID: &e.localID, Tipo: "INSUMO", Concepto: "Crema",
		Monto: monto, ItemID: &item, Cantidad: cantidad, CreadoPor: "Cajera Prueba",
	})
}

func (e *cuentaEnv) movimientosDeCargo(cargoID int) []movVenta {
	e.t.Helper()
	out := []movVenta{}
	if err := e.db.Select(&out, `
		SELECT tipo, cantidad, stock_resultante, costo_unitario, usuario_nombre, motivo
		FROM inventario_movimientos WHERE cargo_id = $1 ORDER BY id`, cargoID); err != nil {
		e.t.Fatal(err)
	}
	return out
}

func TestTrastiendaCargoInsumoBajaStockYQuedaEnElKardex(t *testing.T) {
	e := setupCuenta(t)
	item := e.nuevoItem("Crema", 10)
	mustExec(t, e.db, `UPDATE inventario_items SET costo_referencia = 12.50 WHERE id = $1`, item)

	id, err := e.cargoInsumo(item, 2, 40)
	if err != nil {
		t.Fatal(err)
	}
	if got := e.stock(item); got != 8 {
		t.Fatalf("stock = %v, quiero 8", got)
	}
	movs := e.movimientosDeCargo(id)
	if len(movs) != 1 || movs[0].Tipo != repository.MovSalidaVenta || movs[0].Cantidad != 2 {
		t.Fatalf("movimientos = %+v", movs)
	}
	if movs[0].CostoUnitario == nil || *movs[0].CostoUnitario != 12.50 {
		t.Fatalf("costo_unitario = %v", movs[0].CostoUnitario)
	}
	if movs[0].UsuarioNombre == nil || *movs[0].UsuarioNombre != "Cajera Prueba" {
		t.Fatalf("usuario = %v", movs[0].UsuarioNombre)
	}
	// El cargo guarda el insumo y la cantidad, y suma a lo que el cliente debe.
	c, err := e.svc.Obtener(e.clienteID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Cargos) != 1 || c.Cargos[0].ItemID == nil || *c.Cargos[0].ItemID != item || c.Cargos[0].Cantidad == nil || *c.Cargos[0].Cantidad != 2 {
		t.Fatalf("cargos = %+v", c.Cargos)
	}
	if c.Saldo.PorPagar != 40 {
		t.Fatalf("por pagar = %v, quiero 40", c.Saldo.PorPagar)
	}
}

func TestTrastiendaCargoInsumoSinStockNoPermitidoNoCreaElCargo(t *testing.T) {
	e := setupCuenta(t)
	if _, err := e.cfgSvc.Update(map[string]interface{}{CfgPermitirStockNegativo: false}); err != nil {
		t.Fatal(err)
	}
	item := e.nuevoItem("Serum", 1)
	if _, err := e.cargoInsumo(item, 2, 30); !errors.Is(err, repository.ErrStockInsuficiente) {
		t.Fatalf("error = %v, quiero stock insuficiente", err)
	}
	if got := e.stock(item); got != 1 {
		t.Fatalf("stock = %v, no debe cambiar", got)
	}
	c, _ := e.svc.Obtener(e.clienteID, true)
	if len(c.Cargos) != 0 {
		t.Fatalf("hay %d cargos: el cargo y el descuento van en una sola transaccion", len(c.Cargos))
	}
}

func TestTrastiendaCargoInsumoAnularDevuelveElStockUnaVez(t *testing.T) {
	e := setupCuenta(t)
	item := e.nuevoItem("Tonico", 10)
	id, err := e.cargoInsumo(item, 4, 60)
	if err != nil {
		t.Fatal(err)
	}
	actor := CajeroAuditoriaInput{Nombre: "Gerencia Prueba"}
	if err := e.svc.AnularCargo(id, "cargado por error", actor); err != nil {
		t.Fatal(err)
	}
	if got := e.stock(item); got != 10 {
		t.Fatalf("stock tras anular = %v, quiero 10", got)
	}
	movs := e.movimientosDeCargo(id)
	if len(movs) != 2 || movs[1].Tipo != repository.MovReversionVenta || movs[1].Cantidad != 4 {
		t.Fatalf("movimientos = %+v", movs)
	}
	if movs[1].UsuarioNombre == nil || *movs[1].UsuarioNombre != "Gerencia Prueba" {
		t.Fatalf("la reversion debe llevar a quien anula: %v", movs[1].UsuarioNombre)
	}
	if err := e.svc.AnularCargo(id, "otra vez", actor); err == nil {
		t.Fatal("anular un cargo ya anulado debe fallar")
	}
	if got := e.stock(item); got != 10 {
		t.Fatalf("stock tras el segundo intento = %v, quiero 10", got)
	}
}

func TestTrastiendaCargoInsumoValidaLosDatos(t *testing.T) {
	e := setupCuenta(t)
	item := e.nuevoItem("Gel", 10)
	casos := []struct {
		nombre string
		in     CrearCargoInput
		quiero string
	}{
		{"item en un cargo que no es de insumo", CrearCargoInput{ClienteID: e.clienteID, LocalID: &e.localID, Tipo: "SERVICIO", Concepto: "x", ItemID: &item, Cantidad: 1}, "solo aplica a cargos de insumo"},
		{"insumo sin local", CrearCargoInput{ClienteID: e.clienteID, Tipo: "INSUMO", Concepto: "x", ItemID: &item, Cantidad: 1}, "local_id es requerido"},
		{"insumo sin cantidad", CrearCargoInput{ClienteID: e.clienteID, LocalID: &e.localID, Tipo: "INSUMO", Concepto: "x", ItemID: &item}, "cantidad debe ser mayor a 0"},
		{"item inexistente", CrearCargoInput{ClienteID: e.clienteID, LocalID: &e.localID, Tipo: "INSUMO", Concepto: "x", ItemID: intPtr(999999999), Cantidad: 1}, "no encontrado"},
	}
	for _, c := range casos {
		if _, err := e.svc.CrearCargo(c.in); err == nil || !strings.Contains(err.Error(), c.quiero) {
			t.Errorf("%s: error = %v, quiero %q", c.nombre, err, c.quiero)
		}
	}
	if got := e.stock(item); got != 10 {
		t.Fatalf("stock = %v: un cargo rechazado no mueve stock", got)
	}
}


func (e *cuentaEnv) estadoCobro(reservaID int) string {
	e.t.Helper()
	r, err := e.reservas.GetReservaByID(reservaID, nil)
	if err != nil {
		e.t.Fatal(err)
	}
	if r.EstadoCobro == nil {
		return ""
	}
	return *r.EstadoCobro
}

// Una reserva completada con un cargo vigente ya no es "por cobrar": queda a cuenta del cliente.
func TestTrastiendaReservaConCargoQuedaACuenta(t *testing.T) {
	e := setupCuenta(t)
	srv := e.nuevoServicio("Masaje", nil)
	reserva := e.nuevaReserva(srv, "COMPLETADO", 0)
	if got := e.estadoCobro(reserva); got != "POR_COBRAR" {
		t.Fatalf("estado_cobro sin cargo = %q, quiero POR_COBRAR", got)
	}
	id, err := e.svc.CrearCargo(CrearCargoInput{ClienteID: e.clienteID, Tipo: "SERVICIO", Concepto: "Masaje", Monto: 100, ReservaID: &reserva})
	if err != nil {
		t.Fatal(err)
	}
	if got := e.estadoCobro(reserva); got != "A_CUENTA" {
		t.Fatalf("estado_cobro con cargo = %q, quiero A_CUENTA", got)
	}
	if err := e.svc.AnularCargo(id, "error", CajeroAuditoriaInput{}); err != nil {
		t.Fatal(err)
	}
	if got := e.estadoCobro(reserva); got != "POR_COBRAR" {
		t.Fatalf("estado_cobro con el cargo anulado = %q, quiero POR_COBRAR", got)
	}
}

func TestTrastiendaCargosEnLoteSeCreanJuntosOYNingunoSiUnoFalla(t *testing.T) {
	e := setupCuenta(t)
	crema := e.nuevoItem("Crema", 10)
	serum := e.nuevoItem("Serum", 1)
	servicio := CrearCargoInput{ClienteID: e.clienteID, Tipo: "SERVICIO", Concepto: "Limpieza", Monto: 100}
	insumo := func(item int, cantidad float64) CrearCargoInput {
		return CrearCargoInput{ClienteID: e.clienteID, LocalID: &e.localID, Tipo: "INSUMO", Concepto: "Insumo", Monto: 20, ItemID: &item, Cantidad: cantidad}
	}

	ids, err := e.svc.CrearCargos(e.clienteID, []CrearCargoInput{servicio, insumo(crema, 2)})
	if err != nil || len(ids) != 2 {
		t.Fatalf("lote valido: ids=%v err=%v", ids, err)
	}
	if got := e.stock(crema); got != 8 {
		t.Fatalf("stock = %v, quiero 8", got)
	}

	// El segundo cargo no tiene stock: el lote entero se revierte, tambien el servicio y el insumo que si alcanzaba.
	if _, err := e.cfgSvc.Update(map[string]interface{}{CfgPermitirStockNegativo: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.CrearCargos(e.clienteID, []CrearCargoInput{servicio, insumo(crema, 1), insumo(serum, 5)}); !errors.Is(err, repository.ErrStockInsuficiente) {
		t.Fatalf("error = %v, quiero stock insuficiente", err)
	}
	if got := e.stock(crema); got != 8 {
		t.Fatalf("stock de crema = %v: el lote fallido no debe descontar nada", got)
	}
	c, _ := e.svc.Obtener(e.clienteID, true)
	if len(c.Cargos) != 2 {
		t.Fatalf("hay %d cargos, quiero los 2 del primer lote", len(c.Cargos))
	}
}

func TestTrastiendaCargosEnLoteExigeAlMenosUno(t *testing.T) {
	e := setupCuenta(t)
	if _, err := e.svc.CrearCargos(e.clienteID, nil); err == nil || !strings.Contains(err.Error(), "requerido") {
		t.Fatalf("error = %v, quiero que pida al menos un cargo", err)
	}
}
