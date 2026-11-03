package services

// Venta de insumos desde la caja contra una BD PostgreSQL real (ver la cabecera de
// trastienda_integration_test.go). Requiere la migracion 000076.
//
//	TRASTIENDA_TEST_DB=atrevida_migtest go test ./services -run TrastiendaVentaInsumo -v

import (
	"errors"
	"strings"
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

func (e *trastiendaEnv) pagosSvc() *PagosService {
	svc := NewPagosService(pgsqlrepo.NewPagosRepo(e.db))
	svc.Config = e.cfgSvc
	return svc
}

// ventaConInsumo registra un pago de un servicio suelto mas una linea de insumo (item, cantidad).
func (e *trastiendaEnv) ventaConInsumo(item, cantidad int, precio float64) (string, error) {
	e.t.Helper()
	descuento := 0.0
	return e.pagosSvc().CreatePago(CrearPagoInput{
		LocalID: e.localID, LocalNombre: e.localName, ClienteNombre: "Cliente casual",
		Descuento: &descuento, TipoPago: "efectivo", Estado: "PAGADO", Activo: true,
		Cajero: CajeroAuditoriaInput{Nombre: "Cajera Prueba"},
		Detalle: []CrearDetallePagoInput{
			{Servicio: "Limpieza facial", PrecioUnitario: 100, Cantidad: 1, Subtotal: 100},
			{ItemID: &item, Servicio: "Crema", PrecioUnitario: precio, Cantidad: cantidad, Subtotal: precio * float64(cantidad)},
		},
	})
}

type movVenta struct {
	Tipo            string   `db:"tipo"`
	Cantidad        float64  `db:"cantidad"`
	StockResultante float64  `db:"stock_resultante"`
	CostoUnitario   *float64 `db:"costo_unitario"`
	UsuarioNombre   *string  `db:"usuario_nombre"`
	Motivo          *string  `db:"motivo"`
}

func (e *trastiendaEnv) movimientosDePago(codigo string) []movVenta {
	e.t.Helper()
	out := []movVenta{}
	if err := e.db.Select(&out, `
		SELECT m.tipo, m.cantidad, m.stock_resultante, m.costo_unitario, m.usuario_nombre, m.motivo
		FROM inventario_movimientos m JOIN pagos p ON p.id = m.pago_id
		WHERE p.codigo_pago = $1 ORDER BY m.id`, codigo); err != nil {
		e.t.Fatal(err)
	}
	return out
}

func (e *trastiendaEnv) cantidadDePagos(cliente string) int {
	e.t.Helper()
	var n int
	if err := e.db.Get(&n, `SELECT COUNT(*) FROM pagos WHERE local_id = $1 AND cliente_nombre = $2`, e.localID, cliente); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func TestTrastiendaVentaInsumoDescuentaStockYRegistraKardex(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Crema", 10)
	mustExec(t, e.db, `UPDATE inventario_items SET costo_referencia = 12.50 WHERE id = $1`, item)

	codigo, err := e.ventaConInsumo(item, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got := e.stock(item); got != 8 {
		t.Fatalf("stock tras vender 2 = %v, quiero 8", got)
	}
	movs := e.movimientosDePago(codigo)
	if len(movs) != 1 {
		t.Fatalf("movimientos del pago = %d, quiero 1", len(movs))
	}
	m := movs[0]
	if m.Tipo != repository.MovSalidaVenta || m.Cantidad != 2 || m.StockResultante != 8 {
		t.Fatalf("movimiento = %+v", m)
	}
	if m.CostoUnitario == nil || *m.CostoUnitario != 12.50 {
		t.Fatalf("costo_unitario = %v, quiero 12.50", m.CostoUnitario)
	}
	if m.UsuarioNombre == nil || *m.UsuarioNombre != "Cajera Prueba" {
		t.Fatalf("usuario_nombre = %v", m.UsuarioNombre)
	}
	if m.Motivo == nil || !strings.Contains(*m.Motivo, codigo) {
		t.Fatalf("el motivo debe nombrar el pago %s, quedo %v", codigo, m.Motivo)
	}

	// El pago guarda el item en su detalle y el total incluye servicio e insumo.
	pago, err := pgsqlrepo.NewPagosRepo(e.db).GetPagoByCodigo(codigo)
	if err != nil {
		t.Fatal(err)
	}
	if pago.TotalFinal != 140 {
		t.Fatalf("total_final = %v, quiero 140", pago.TotalFinal)
	}
	if len(pago.Detalle) != 2 || pago.Detalle[0].ItemID != nil || pago.Detalle[1].ItemID == nil || *pago.Detalle[1].ItemID != item {
		t.Fatalf("detalle = %+v", pago.Detalle)
	}
}

func TestTrastiendaVentaInsumoMismoItemEnDosLineasSumaUnaSalida(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Mascarilla", 10)
	descuento := 0.0
	codigo, err := e.pagosSvc().CreatePago(CrearPagoInput{
		LocalID: e.localID, LocalNombre: e.localName, ClienteNombre: "Cliente casual",
		Descuento: &descuento, TipoPago: "qr", Estado: "PAGADO", Activo: true,
		Cajero: CajeroAuditoriaInput{Nombre: "Cajera Prueba"},
		Detalle: []CrearDetallePagoInput{
			{ItemID: &item, Servicio: "Mascarilla", PrecioUnitario: 5, Cantidad: 2, Subtotal: 10},
			{ItemID: &item, Servicio: "Mascarilla (regalo)", PrecioUnitario: 0, Cantidad: 1, Subtotal: 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	movs := e.movimientosDePago(codigo)
	if len(movs) != 1 || movs[0].Cantidad != 3 {
		t.Fatalf("movimientos = %+v, quiero una sola salida de 3", movs)
	}
	if got := e.stock(item); got != 7 {
		t.Fatalf("stock = %v, quiero 7", got)
	}
}

func TestTrastiendaVentaInsumoSinStockNoPermitidoNoCobraNiDescuenta(t *testing.T) {
	e := setupTrastienda(t)
	if _, err := e.cfgSvc.Update(map[string]interface{}{CfgPermitirStockNegativo: false}); err != nil {
		t.Fatal(err)
	}
	item := e.nuevoItem("Serum", 1)

	_, err := e.ventaConInsumo(item, 2, 30)
	if !errors.Is(err, repository.ErrStockInsuficiente) {
		t.Fatalf("error = %v, quiero stock insuficiente", err)
	}
	if got := e.stock(item); got != 1 {
		t.Fatalf("stock = %v, no debe cambiar", got)
	}
	if n := e.cantidadDePagos("Cliente casual"); n != 0 {
		t.Fatalf("hay %d pagos: el cobro y el descuento van en una sola transaccion", n)
	}

	// Con el parametro en true se vende aunque deje el stock en negativo.
	if _, err := e.cfgSvc.Update(map[string]interface{}{CfgPermitirStockNegativo: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ventaConInsumo(item, 2, 30); err != nil {
		t.Fatal(err)
	}
	if got := e.stock(item); got != -1 {
		t.Fatalf("stock = %v, quiero -1", got)
	}
}

func TestTrastiendaVentaInsumoItemInexistenteFalla(t *testing.T) {
	e := setupTrastienda(t)
	if _, err := e.ventaConInsumo(999999999, 1, 10); err == nil || !strings.Contains(err.Error(), "referencia no encontrada") {
		t.Fatalf("error = %v, quiero referencia no encontrada", err)
	}
	if n := e.cantidadDePagos("Cliente casual"); n != 0 {
		t.Fatalf("hay %d pagos, quiero 0", n)
	}
}

func TestTrastiendaAnularVentaDevuelveElStockUnaVez(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Tonico", 10)
	codigo, err := e.ventaConInsumo(item, 4, 15)
	if err != nil {
		t.Fatal(err)
	}
	if got := e.stock(item); got != 6 {
		t.Fatalf("stock tras vender = %v, quiero 6", got)
	}

	actor := CajeroAuditoriaInput{Nombre: "Gerencia Prueba"}
	if err := e.pagosSvc().DeletePago(codigo, actor); err != nil {
		t.Fatal(err)
	}
	if got := e.stock(item); got != 10 {
		t.Fatalf("stock tras anular = %v, quiero 10", got)
	}
	movs := e.movimientosDePago(codigo)
	if len(movs) != 2 || movs[1].Tipo != repository.MovReversionVenta || movs[1].Cantidad != 4 {
		t.Fatalf("movimientos = %+v, quiero salida y reversion de 4", movs)
	}
	if movs[1].UsuarioNombre == nil || *movs[1].UsuarioNombre != "Gerencia Prueba" {
		t.Fatalf("la reversion debe llevar a quien anula, quedo %v", movs[1].UsuarioNombre)
	}

	// Anular de nuevo no devuelve stock otra vez.
	if err := e.pagosSvc().DeletePago(codigo, actor); err == nil {
		t.Fatal("anular un pago ya anulado debe fallar")
	}
	if got := e.stock(item); got != 10 {
		t.Fatalf("stock tras el segundo intento = %v, quiero 10", got)
	}
}

func TestTrastiendaPagoConInsumosNoEditaSuDetalle(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Aceite", 10)
	descuento := 0.0
	// Un pago PENDIENTE sí se puede editar; con insumos, el detalle no, porque el stock ya salió.
	pendiente, err := e.pagosSvc().CreatePago(CrearPagoInput{
		LocalID: e.localID, LocalNombre: e.localName, ClienteNombre: "Cliente casual",
		Descuento: &descuento, TipoPago: "efectivo", Estado: "PENDIENTE", Activo: true,
		Cajero:  CajeroAuditoriaInput{Nombre: "Cajera Prueba"},
		Detalle: []CrearDetallePagoInput{{ItemID: &item, Servicio: "Aceite", PrecioUnitario: 25, Cantidad: 1, Subtotal: 25}},
	})
	if err != nil {
		t.Fatal(err)
	}
	nuevo := []ActualizarDetallePagoInput{{Servicio: "Otro", PrecioUnitario: 1, Cantidad: 1, Subtotal: 1}}
	err = e.pagosSvc().UpdatePago(ActualizarPagoInput{
		CodigoPago: pendiente, Detalle: &nuevo, Cajero: CajeroAuditoriaInput{Nombre: "Prueba"},
	})
	if err == nil || !strings.Contains(err.Error(), "insumos") {
		t.Fatalf("error = %v, quiero que rechace editar el detalle de un pago con insumos", err)
	}
	// Otros campos de la cabecera siguen editables.
	tipo := "qr"
	if err := e.pagosSvc().UpdatePago(ActualizarPagoInput{
		CodigoPago: pendiente, TipoPago: &tipo, Cajero: CajeroAuditoriaInput{Nombre: "Prueba"},
	}); err != nil {
		t.Fatalf("cambiar el tipo de pago debe seguir permitido: %v", err)
	}
}

// Los reportes de "servicios" cuentan solo servicios del catalogo; el insumo, el paquete y la linea libre se
// distinguen por tipo_linea y no inflan el ranking de servicios.
func TestTrastiendaReporteDeServiciosNoMezclaInsumosPaquetesNiLineasLibres(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Crema", 50)
	var servicioID int
	if err := e.db.Get(&servicioID, `INSERT INTO servicios (nombre, sesiones) VALUES ($1, 1) RETURNING id`, "Limpieza"+e.sufijo); err != nil {
		t.Fatal(err)
	}
	descuento := 0.0
	if _, err := e.pagosSvc().CreatePago(CrearPagoInput{
		LocalID: e.localID, LocalNombre: e.localName, ClienteNombre: "Cliente casual",
		Descuento: &descuento, TipoPago: "efectivo", Estado: "PAGADO", Activo: true,
		Cajero: CajeroAuditoriaInput{Nombre: "Cajera Prueba"},
		Detalle: []CrearDetallePagoInput{
			{ServicioID: &servicioID, Servicio: "Limpieza", PrecioUnitario: 100, Cantidad: 2},
			{ItemID: &item, Servicio: "Crema", PrecioUnitario: 20, Cantidad: 3},
			{Servicio: "Mario", PrecioUnitario: 5, Cantidad: 13},
			{Servicio: "Paquete: Relax", PrecioUnitario: 900, Cantidad: 1},
		},
	}); err != nil {
		t.Fatal(err)
	}

	ayer, manana := time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1)
	filtro := repository.FiltroResumenPagos{FechaDesde: ayer, FechaHasta: manana, Local: e.localName}

	lineas, err := pgsqlrepo.NewPagosRepo(e.db).ListVentasDetalle(filtro, 100)
	if err != nil {
		t.Fatal(err)
	}
	tipos := map[string]string{}
	for _, l := range lineas {
		tipos[l.Servicio] = l.TipoLinea
	}
	esperado := map[string]string{"Limpieza": "SERVICIO", "Crema": "INSUMO", "Mario": "OTRO", "Paquete: Relax": "PAQUETE"}
	for nombre, tipo := range esperado {
		if tipos[nombre] != tipo {
			t.Errorf("tipo_linea de %q = %q, quiero %q", nombre, tipos[nombre], tipo)
		}
	}

	resumen, err := pgsqlrepo.NewPagosRepo(e.db).GetResumenPagos(filtro)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range resumen.Totales {
		if row.EsGeneral && row.CantidadServiciosVendidos != 2 {
			t.Errorf("cantidad_servicios_vendidos = %d, quiero 2 (solo el servicio del catalogo)", row.CantidadServiciosVendidos)
		}
	}
	if len(resumen.Servicios) == 0 {
		t.Fatal("el resumen no trae servicios")
	}
	for _, s := range resumen.Servicios {
		if s.Servicio != "Limpieza" {
			t.Errorf("el ranking de servicios incluye %q", s.Servicio)
		}
	}
}

// El costo de los insumos vendidos entra al consumo de insumos del reporte financiero, y la anulacion lo descuenta.
func TestTrastiendaConsumoDeInsumosIncluyeLoVendidoEnCaja(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Crema", 10)
	mustExec(t, e.db, `UPDATE inventario_items SET costo_referencia = 12.50 WHERE id = $1`, item)
	desde := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	hasta := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	consumo := func() float64 {
		t.Helper()
		v, err := e.invRepo.ConsumoServicios(desde, hasta, &e.localID)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	codigo, err := e.ventaConInsumo(item, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got := consumo(); got != 25 {
		t.Fatalf("consumo tras vender 2 a costo 12.50 = %v, quiero 25", got)
	}
	if err := e.pagosSvc().DeletePago(codigo, CajeroAuditoriaInput{Nombre: "Gerencia"}); err != nil {
		t.Fatal(err)
	}
	if got := consumo(); got != 0 {
		t.Fatalf("consumo tras anular = %v, quiero 0", got)
	}
}
