package services

// El flujo de caja distingue un cobro de cuenta (abono) de una venta, y trae la hora del movimiento para las ventas.
//
//	TRASTIENDA_TEST_DB=atrevida_migtest go test ./services -run TrastiendaFlujoCobro -v

import (
	"strings"
	"testing"
	"time"

	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

func TestTrastiendaFlujoCobroDeCuentaSeNombraComoCobro(t *testing.T) {
	e := setupCuenta(t)
	var venta string
	if err := e.db.Get(&venta, `INSERT INTO pagos (
		local_id, local_nombre, cliente_nombre, subtotal, descuento, total_final, estado, tipo_pago, nombre_cajero, activo
	) VALUES ($1,$2,'Cliente de prueba',100,0,100,'PAGADO','efectivo','Cajera Prueba',TRUE) RETURNING codigo_pago`, e.localID, e.localName); err != nil {
		t.Fatal(err)
	}
	var pagoID int
	if err := e.db.Get(&pagoID, `SELECT id FROM pagos WHERE codigo_pago=$1`, venta); err != nil {
		t.Fatal(err)
	}
	mustExec(t, e.db, `INSERT INTO detalle_pagos (pago_id, servicio, precio_unitario, cantidad, subtotal) VALUES ($1,'Servicio de prueba',100,1,100)`, pagoID)
	abono, err := e.abono(50)
	if err != nil {
		t.Fatal(err)
	}

	filas, _, err := e.finRepo.ListFlujo(pgsqlrepo.FiltroFlujo{
		Desde: time.Now().AddDate(0, 0, -1), Hasta: time.Now().AddDate(0, 0, 1), LocalID: &e.localID,
	})
	if err != nil {
		t.Fatal(err)
	}
	porCodigo := map[string]struct {
		categoria, detalle string
		ts                 time.Time
	}{}
	for _, f := range filas {
		porCodigo[f.Codigo] = struct {
			categoria, detalle string
			ts                 time.Time
		}{f.Categoria, f.Detalle, f.Ts}
	}
	if f := porCodigo[venta]; f.categoria != "Venta" || !strings.HasPrefix(f.detalle, "Venta a ") || f.ts.IsZero() {
		t.Fatalf("la venta = %+v", f)
	}
	if f := porCodigo[abono]; f.categoria != "Cobro de cuenta" || !strings.HasPrefix(f.detalle, "Cobro a ") {
		t.Fatalf("el cobro de cuenta = %+v, quiero categoria Cobro de cuenta y detalle Cobro a ...", f)
	}
}

// Cada fila del libro dice con que metodo se pago: el tipo de pago de la venta o el metodo del movimiento (nulo si no se indico).
func TestTrastiendaFlujoTraeMetodoDePago(t *testing.T) {
	e := setupCuenta(t)
	var venta string
	if err := e.db.Get(&venta, `INSERT INTO pagos (
		local_id, local_nombre, cliente_nombre, subtotal, descuento, total_final, estado, tipo_pago, nombre_cajero, activo
	) VALUES ($1,$2,'Cliente de prueba',100,0,100,'PAGADO','qr','Cajera Prueba',TRUE) RETURNING codigo_pago`, e.localID, e.localName); err != nil {
		t.Fatal(err)
	}
	var categoriaID int
	if err := e.db.Get(&categoriaID, `SELECT id FROM categorias_financieras ORDER BY id LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	insertarMov := func(metodo interface{}) string {
		var codigo string
		if err := e.db.Get(&codigo, `INSERT INTO movimientos_financieros (naturaleza, clase, categoria_id, local_id, local_nombre, fecha, monto, metodo_pago, estado)
			SELECT 'EGRESO', 'VARIABLE', $1, $2, $3, CURRENT_DATE, 10, $4, 'PAGADO' RETURNING codigo`, categoriaID, e.localID, e.localName, metodo); err != nil {
			t.Fatal(err)
		}
		return codigo
	}
	conMetodo := insertarMov("transferencia")
	sinMetodo := insertarMov(nil)

	filas, _, err := e.finRepo.ListFlujo(pgsqlrepo.FiltroFlujo{
		Desde: time.Now().AddDate(0, 0, -1), Hasta: time.Now().AddDate(0, 0, 1), LocalID: &e.localID,
	})
	if err != nil {
		t.Fatal(err)
	}
	metodos := map[string]*string{}
	for _, f := range filas {
		metodos[f.Codigo] = f.MetodoPago
	}
	if m := metodos[venta]; m == nil || *m != "qr" {
		t.Fatalf("metodo de la venta = %v, quiero qr", m)
	}
	if m := metodos[conMetodo]; m == nil || *m != "transferencia" {
		t.Fatalf("metodo del movimiento = %v, quiero transferencia", m)
	}
	if m, hay := metodos[sinMetodo]; !hay || m != nil {
		t.Fatalf("movimiento sin metodo = %v, quiero nulo", m)
	}
}
