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
