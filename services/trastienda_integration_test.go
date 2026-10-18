package services

// Pruebas de integracion del modulo de trastienda contra una BD PostgreSQL real.
//
// Se omiten salvo que se defina TRASTIENDA_TEST_DB (nombre de una BD DESCARTABLE, p. ej.
// atrevida_migtest, con todas las migraciones aplicadas). Usan las variables DB_HOST, DB_PORT, DB_USER,
// DB_PASSWORD y DB_SSLMODE para conectar. Nunca apuntar a atrevida_db: las pruebas insertan datos.
//
//	TRASTIENDA_TEST_DB=atrevida_migtest go test ./services -run Trastienda -v

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"atrevida-agenda-api/config"
	"atrevida-agenda-api/db"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"

	"github.com/jmoiron/sqlx"
)

type trastiendaEnv struct {
	t          *testing.T
	db         *sqlx.DB
	cfgSvc     *ConfiguracionService
	inv        *InventarioService
	consumo    *InventarioConsumo
	fin        *FinanzasService
	gastos     *GastosFijosService
	reporte    *ReporteFinancieroService
	reservas   *ReservasPGService
	invRepo    *pgsqlrepo.InventarioRepo
	finRepo    *pgsqlrepo.FinanzasRepo
	localID    int
	localName  string
	sufijo     string
	categoriaE int
}

func setupTrastienda(t *testing.T) *trastiendaEnv {
	t.Helper()
	name := os.Getenv("TRASTIENDA_TEST_DB")
	if name == "" {
		t.Skip("TRASTIENDA_TEST_DB no definido: se omiten las pruebas de integracion de trastienda")
	}
	if name == "atrevida_db" {
		t.Fatal("no uses atrevida_db: estas pruebas insertan datos")
	}
	conn, err := db.Connect(&config.Config{DB: config.DBConfig{
		Host: os.Getenv("DB_HOST"), Port: os.Getenv("DB_PORT"), User: os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"), Name: name, SSLMode: getenvDef("DB_SSLMODE", "disable"),
		MaxOpenConns: 20, MaxIdleConns: 5,
	}})
	if err != nil {
		t.Fatalf("no se pudo conectar a %s: %v", name, err)
	}
	t.Cleanup(func() { conn.Close() })

	invRepo := pgsqlrepo.NewInventarioRepo(conn)
	finRepo := pgsqlrepo.NewFinanzasRepo(conn)
	cfgSvc := NewConfiguracionService(pgsqlrepo.NewConfiguracionRepo(conn))
	finSvc := NewFinanzasService(finRepo, cfgSvc)
	gastos := NewGastosFijosService(finRepo, cfgSvc, finSvc)
	env := &trastiendaEnv{
		t: t, db: conn, cfgSvc: cfgSvc, invRepo: invRepo, finRepo: finRepo,
		inv:     NewInventarioService(invRepo, cfgSvc),
		consumo: NewInventarioConsumo(invRepo, cfgSvc),
		fin:     finSvc, gastos: gastos,
		reporte:  NewReporteFinancieroService(finRepo, invRepo, gastos, cfgSvc),
		reservas: NewReservasPGService(pgsqlrepo.NewReservasRepo(conn), pgsqlrepo.NewServiciosRepo(conn)),
		sufijo:   fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	env.reservas.SetInventarioConsumo(env.consumo)

	// Local propio de la prueba: aisla stock, reservas y gastos de otras ejecuciones.
	env.localName = "LOCAL-T" + env.sufijo
	mustExec(t, conn, `INSERT INTO locales (nombre) VALUES ($1)`, env.localName)
	if err := conn.Get(&env.localID, `SELECT id FROM locales WHERE nombre = $1`, env.localName); err != nil {
		t.Fatal(err)
	}
	if err := conn.Get(&env.categoriaE, `SELECT id FROM categorias_financieras WHERE naturaleza = 'EGRESO' ORDER BY id LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	// Config por defecto para cada prueba.
	if _, err := cfgSvc.Update(map[string]interface{}{
		CfgPoliticaSinStock: "ADVERTIR", CfgDescontarEnEstado: "COMPLETADO", CfgPermitirStockNegativo: true,
		CfgConsiderarComprometido: true, CfgEstadoGastoFijoGen: "PENDIENTE",
	}); err != nil {
		t.Fatal(err)
	}
	return env
}

func getenvDef(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustExec(t *testing.T, conn *sqlx.DB, q string, args ...interface{}) {
	t.Helper()
	if _, err := conn.Exec(q, args...); err != nil {
		t.Fatalf("%s: %v", strings.Fields(q)[0], err)
	}
}

// nuevoItem crea un item activo con stock inicial en el local de la prueba.
func (e *trastiendaEnv) nuevoItem(nombre string, stock float64) int {
	e.t.Helper()
	item, err := e.inv.CrearItem(pgsqlrepo.ItemInput{Nombre: nombre + e.sufijo, UnidadMedida: "ML", StockMinimo: 5})
	if err != nil {
		e.t.Fatal(err)
	}
	if stock > 0 {
		if _, err := e.inv.Ajustar(AjusteInput{
			ItemID: item.ID, LocalID: e.localID, Tipo: repository.MovAjustePositivo, Cantidad: stock, Motivo: "carga inicial",
		}); err != nil {
			e.t.Fatal(err)
		}
	}
	return item.ID
}

// nuevoServicio crea un servicio con los insumos dados (item -> cantidad por sesion).
func (e *trastiendaEnv) nuevoServicio(nombre string, insumos map[int]float64) string {
	e.t.Helper()
	full := nombre + e.sufijo
	var id int
	if err := e.db.Get(&id, `INSERT INTO servicios (nombre, sesiones) VALUES ($1, 1) RETURNING id`, full); err != nil {
		e.t.Fatal(err)
	}
	list := make([]struct {
		item int
		qty  float64
	}, 0)
	for item, qty := range insumos {
		list = append(list, struct {
			item int
			qty  float64
		}{item, qty})
	}
	for _, in := range list {
		mustExec(e.t, e.db, `INSERT INTO servicio_insumos (servicio_id, item_id, cantidad_por_sesion) VALUES ($1,$2,$3)`, id, in.item, in.qty)
	}
	return full
}

// nuevaReserva inserta una reserva activa del local de la prueba para el servicio dado.
func (e *trastiendaEnv) nuevaReserva(servicio, estado string, diasDesdeHoy int) int {
	e.t.Helper()
	var id int
	if err := e.db.Get(&id, `
		INSERT INTO reservas (local_id, local_nombre, tipo_espacio, fecha, hora_desde, hora_hasta, cliente, estado,
			servicio_nombre, servicio_confirmado)
		VALUES ($1, $2, 'M', CURRENT_DATE + $3::int, '10:00', '11:00', 'Cliente Prueba', $4, $5, $5) RETURNING id`,
		e.localID, e.localName, diasDesdeHoy, estado, servicio); err != nil {
		e.t.Fatal(err)
	}
	return id
}

func (e *trastiendaEnv) stock(itemID int) float64 {
	e.t.Helper()
	var s float64
	err := e.db.Get(&s, `SELECT COALESCE((SELECT cantidad FROM inventario_stock WHERE item_id = $1 AND local_id = $2), 0)`, itemID, e.localID)
	if err != nil {
		e.t.Fatal(err)
	}
	return s
}

func (e *trastiendaEnv) contarMov(reservaID int, tipo string) int {
	e.t.Helper()
	var n int
	if err := e.db.Get(&n, `SELECT COUNT(*) FROM inventario_movimientos WHERE reserva_id = $1 AND tipo = $2`, reservaID, tipo); err != nil {
		e.t.Fatal(err)
	}
	return n
}

// ─── Inventario ↔ reservas ────────────────────────────────────────────────────

func TestTrastiendaCompletarDescuentaExactoYEsIdempotente(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Gel", 100)
	srv := e.nuevoServicio("Masaje", map[int]float64{item: 30})
	reserva := e.nuevaReserva(srv, "AGENDADO", 1)

	if _, err := e.reservas.ActualizarEstadoReservaConAvisos(ActualizarEstadoReservaInput{Id: reserva, Estado: "COMPLETADO"}); err != nil {
		t.Fatalf("completar: %v", err)
	}
	if got := e.stock(item); got != 70 {
		t.Fatalf("stock tras completar = %v, quiero 70", got)
	}

	// Reintento directo del descuento (doble clic / reintento): no descuenta dos veces.
	tx, _ := e.db.Beginx()
	if err := pgsqlrepo.ConsumirPorReservaTx(tx, reserva, repository.ConsumoActor{}, true); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if got := e.stock(item); got != 70 {
		t.Fatalf("el reintento descontó de nuevo: stock = %v", got)
	}
	if n := e.contarMov(reserva, repository.MovSalidaServicio); n != 1 {
		t.Fatalf("salidas = %d, quiero 1", n)
	}
}

func TestTrastiendaEliminarReservaCompletadaRevierteStock(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Aceite", 100)
	srv := e.nuevoServicio("Drenaje", map[int]float64{item: 25})
	reserva := e.nuevaReserva(srv, "AGENDADO", 1)
	if _, err := e.reservas.ActualizarEstadoReservaConAvisos(ActualizarEstadoReservaInput{Id: reserva, Estado: "COMPLETADO"}); err != nil {
		t.Fatal(err)
	}
	if err := e.reservas.EliminarReserva(reserva, repository.ConsumoActor{}); err != nil {
		t.Fatalf("eliminar: %v", err)
	}
	if got := e.stock(item); got != 100 {
		t.Fatalf("stock tras eliminar = %v, quiero 100 (restituido)", got)
	}
	if n := e.contarMov(reserva, repository.MovReversionServicio); n != 1 {
		t.Fatalf("reversiones = %d, quiero 1", n)
	}
}

func TestTrastiendaSinStockNegativoNoPermitidoNoCompleta(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Crema", 10)
	srv := e.nuevoServicio("Facial", map[int]float64{item: 30})
	reserva := e.nuevaReserva(srv, "AGENDADO", 1)
	if _, err := e.cfgSvc.Update(map[string]interface{}{CfgPermitirStockNegativo: false}); err != nil {
		t.Fatal(err)
	}

	_, err := e.reservas.ActualizarEstadoReservaConAvisos(ActualizarEstadoReservaInput{Id: reserva, Estado: "COMPLETADO"})
	if err == nil || !strings.Contains(err.Error(), "stock insuficiente") {
		t.Fatalf("quiero error 'stock insuficiente', obtuve %v", err)
	}
	var estado string
	_ = e.db.Get(&estado, `SELECT estado FROM reservas WHERE id = $1`, reserva)
	if estado != "AGENDADO" {
		t.Fatalf("la transición se aplicó pese al error: estado = %s", estado)
	}
	if got := e.stock(item); got != 10 {
		t.Fatalf("el stock cambió pese al error: %v", got)
	}
}

func TestTrastiendaSinStockPermitidoCompletaYDejaNegativo(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Banda", 10)
	srv := e.nuevoServicio("Presoterapia", map[int]float64{item: 30})
	reserva := e.nuevaReserva(srv, "AGENDADO", 1)
	if _, err := e.reservas.ActualizarEstadoReservaConAvisos(ActualizarEstadoReservaInput{Id: reserva, Estado: "COMPLETADO"}); err != nil {
		t.Fatalf("con permitir_stock_negativo=true debe completar: %v", err)
	}
	if got := e.stock(item); got != -20 {
		t.Fatalf("stock = %v, quiero -20", got)
	}
}

func TestTrastiendaDescuentosConcurrentesNoLeenElMismoStock(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Gel conc", 30)
	srv := e.nuevoServicio("Conc", map[int]float64{item: 30})
	r1 := e.nuevaReserva(srv, "AGENDADO", 1)
	r2 := e.nuevaReserva(srv, "AGENDADO", 1)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, id := range []int{r1, r2} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx, err := e.db.Beginx()
			if err != nil {
				errs[i] = err
				return
			}
			defer tx.Rollback()
			if err := pgsqlrepo.ConsumirPorReservaTx(tx, id, repository.ConsumoActor{}, false); err != nil {
				errs[i] = err
				return
			}
			errs[i] = tx.Commit()
		}()
	}
	wg.Wait()

	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		}
	}
	if ok != 1 {
		t.Fatalf("exactamente 1 descuento debe aplicarse con stock para uno solo; aplicados=%d errs=%v", ok, errs)
	}
	if got := e.stock(item); got != 0 {
		t.Fatalf("stock = %v, quiero 0", got)
	}
}

func TestTrastiendaAdvertenciaConStockCeroYComprometido(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Gel adv", 50)
	srv := e.nuevoServicio("Adv", map[int]float64{item: 30})

	// Stock 50 alcanza para una; la primera reserva agendada no recibe aviso.
	r1 := e.nuevaReserva(srv, "AGENDADO", 2)
	warns, err := e.consumo.Recalcular(r1)
	if err != nil || len(warns) != 0 {
		t.Fatalf("r1: warns=%v err=%v, quiero sin advertencias", warns, err)
	}
	// Una segunda reserva compite por el mismo stock: 50 - 30 comprometido = 20 < 30 -> advertencia.
	r2 := e.nuevaReserva(srv, "AGENDADO", 3)
	warns, err = e.consumo.Recalcular(r2)
	if err != nil || len(warns) != 1 || warns[0].Disponible != 20 || warns[0].Requerido != 30 {
		t.Fatalf("r2: warns=%+v err=%v, quiero 1 advertencia (requerido 30, disponible 20)", warns, err)
	}
	var alerta bool
	_ = e.db.Get(&alerta, `SELECT alerta_inventario FROM reservas WHERE id = $1`, r2)
	if !alerta {
		t.Fatal("alerta_inventario debe quedar en true")
	}

	// Sin considerar comprometido ya no hay advertencia.
	if _, err := e.cfgSvc.Update(map[string]interface{}{CfgConsiderarComprometido: false}); err != nil {
		t.Fatal(err)
	}
	if warns, _ = e.consumo.Recalcular(r2); len(warns) != 0 {
		t.Fatalf("sin considerar comprometido no debe advertir: %+v", warns)
	}

	// Servicio sin stock: advertencia aunque no haya otras reservas.
	vacio := e.nuevoItem("Gel vacio", 0)
	srv0 := e.nuevoServicio("SinStock", map[int]float64{vacio: 5})
	r3 := e.nuevaReserva(srv0, "PENDIENTE", 4)
	if warns, _ = e.consumo.Recalcular(r3); len(warns) != 1 {
		t.Fatalf("stock 0 debe advertir: %+v", warns)
	}
}

func TestTrastiendaPoliticaBloquearSoloParaAutenticados(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Gel blk", 0)
	srv := e.nuevoServicio("Blk", map[int]float64{item: 5})
	if _, err := e.cfgSvc.Update(map[string]interface{}{CfgPoliticaSinStock: "BLOQUEAR"}); err != nil {
		t.Fatal(err)
	}
	if err := e.consumo.ValidarPoliticaAlCrear(true, e.localID, []string{srv}); err == nil ||
		!strings.Contains(err.Error(), "stock insuficiente") {
		t.Fatalf("autenticado + BLOQUEAR debe rechazar con 'stock insuficiente': %v", err)
	}
	if err := e.consumo.ValidarPoliticaAlCrear(false, e.localID, []string{srv}); err != nil {
		t.Fatalf("el formulario publico siempre registra: %v", err)
	}
}

// ─── Compras ──────────────────────────────────────────────────────────────────

func TestTrastiendaCompraAtomicaCreaUnEgresoYSumaStock(t *testing.T) {
	e := setupTrastienda(t)
	a := e.nuevoItem("Compra A", 0)
	b := e.nuevoItem("Compra B", 0)
	c := e.nuevoItem("Compra C", 0)

	compra, err := e.fin.RegistrarCompra(CompraInput{
		LocalID: e.localID, Fecha: "2026-10-18",
		Detalle: []CompraLineaInput{{a, 10, 2.5}, {b, 4, 10}, {c, 1, 3.333}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if compra.Total != 68.33 {
		t.Fatalf("total = %v, quiero 68.33", compra.Total)
	}
	if e.stock(a) != 10 || e.stock(b) != 4 || e.stock(c) != 1 {
		t.Fatalf("stock = %v/%v/%v, quiero 10/4/1", e.stock(a), e.stock(b), e.stock(c))
	}
	var egresos int
	_ = e.db.Get(&egresos, `SELECT COUNT(*) FROM movimientos_financieros WHERE compra_id = $1 AND clase = 'VARIABLE' AND monto = 68.33`, compra.ID)
	if egresos != 1 {
		t.Fatalf("egresos de la compra = %d, quiero 1 por el total", egresos)
	}
}

func TestTrastiendaCompraConLineaInvalidaNoDejaNadaAMedias(t *testing.T) {
	e := setupTrastienda(t)
	a := e.nuevoItem("Atom A", 0)
	inactivo := e.nuevoItem("Atom B", 0)
	if err := e.inv.DesactivarItem(inactivo); err != nil {
		t.Fatal(err)
	}
	_, err := e.fin.RegistrarCompra(CompraInput{
		LocalID: e.localID, Fecha: "2026-10-18",
		Detalle: []CompraLineaInput{{a, 10, 1}, {inactivo, 5, 1}},
	})
	if err == nil {
		t.Fatal("debe fallar por item inactivo")
	}
	if e.stock(a) != 0 {
		t.Fatalf("stock de la primera línea = %v: quedó algo a medias", e.stock(a))
	}
	var n int
	_ = e.db.Get(&n, `SELECT COUNT(*) FROM compras_inventario WHERE local_id = $1`, e.localID)
	if n != 0 {
		t.Fatalf("quedó %d compra(s) a medias", n)
	}
}

func TestTrastiendaAnularCompraConsumidaDa409YNoCambiaNada(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Anul", 0)
	compra, err := e.fin.RegistrarCompra(CompraInput{LocalID: e.localID, Fecha: "2026-10-18", Detalle: []CompraLineaInput{{item, 10, 1}}})
	if err != nil {
		t.Fatal(err)
	}
	// Se consume parte del stock comprado.
	if _, err := e.inv.Ajustar(AjusteInput{ItemID: item, LocalID: e.localID, Tipo: repository.MovAjusteNegativo, Cantidad: 4, Motivo: "uso"}); err != nil {
		t.Fatal(err)
	}
	err = e.fin.AnularCompra(compra.ID, "error", repository.ConsumoActor{})
	if err == nil || statusMsg(err) != "stock insuficiente" {
		t.Fatalf("quiero 'stock insuficiente' (409), obtuve %v", err)
	}
	if e.stock(item) != 6 {
		t.Fatalf("stock = %v, quiero 6 sin cambios", e.stock(item))
	}
	actual, _ := e.fin.ObtenerCompra(compra.ID)
	if actual.Estado != "REGISTRADA" {
		t.Fatalf("estado = %s, debe seguir REGISTRADA", actual.Estado)
	}
}

func statusMsg(err error) string {
	if strings.Contains(err.Error(), "stock insuficiente") {
		return "stock insuficiente"
	}
	return err.Error()
}

func TestTrastiendaAnularCompraRevierteStockYEgreso(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Anul ok", 0)
	compra, err := e.fin.RegistrarCompra(CompraInput{LocalID: e.localID, Fecha: "2026-10-18", Detalle: []CompraLineaInput{{item, 10, 2}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.fin.AnularCompra(compra.ID, "duplicada", repository.ConsumoActor{}); err != nil {
		t.Fatal(err)
	}
	if e.stock(item) != 0 {
		t.Fatalf("stock = %v, quiero 0", e.stock(item))
	}
	var estado string
	_ = e.db.Get(&estado, `SELECT estado FROM movimientos_financieros WHERE compra_id = $1`, compra.ID)
	if estado != "ANULADO" {
		t.Fatalf("egreso = %s, quiero ANULADO", estado)
	}
}

func TestTrastiendaAjusteNegativoSinMotivoOSinStock(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Ajuste", 5)
	if _, err := e.inv.Ajustar(AjusteInput{ItemID: item, LocalID: e.localID, Tipo: repository.MovAjusteNegativo, Cantidad: 1}); err == nil ||
		!strings.Contains(err.Error(), "motivo") {
		t.Fatalf("ajuste sin motivo debe fallar: %v", err)
	}
	if _, err := e.inv.Ajustar(AjusteInput{ItemID: item, LocalID: e.localID, Tipo: repository.MovAjusteNegativo, Cantidad: 99, Motivo: "x"}); err == nil ||
		!strings.Contains(err.Error(), "stock insuficiente") {
		t.Fatalf("ajuste negativo mayor al stock debe fallar con stock insuficiente: %v", err)
	}
}

// ─── Gastos fijos ─────────────────────────────────────────────────────────────

func (e *trastiendaEnv) nuevoGastoFijo(dia int, inicio string, fin *string) int {
	e.t.Helper()
	g, err := e.gastos.Crear(GastoFijoInput{
		Nombre: "Gasto " + e.sufijo, CategoriaID: e.categoriaE, Monto: 100, DiaCobro: dia, LocalID: &e.localID,
		FechaInicio: inicio, FechaFin: fin,
	})
	if err != nil {
		e.t.Fatal(err)
	}
	return g.ID
}

func (e *trastiendaEnv) periodos(gastoID int) []string {
	e.t.Helper()
	var out []string
	if err := e.db.Select(&out, `SELECT fecha::text FROM movimientos_financieros WHERE gasto_fijo_id = $1 ORDER BY fecha`, gastoID); err != nil {
		e.t.Fatal(err)
	}
	return out
}

func fecha(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestTrastiendaGastoFijoDia31EnFebreroUsaUltimoDia(t *testing.T) {
	e := setupTrastienda(t)
	id := e.nuevoGastoFijo(31, "2027-02-01", nil)
	if _, err := e.gastos.GenerarHasta(fecha("2027-03-15")); err != nil {
		t.Fatal(err)
	}
	got := e.periodos(id)
	// Febrero 2027 no tiene 29/30/31: se cobra el 28. Marzo cobra el 31, que aun no vencio el 15.
	if len(got) != 1 || got[0] != "2027-02-28" {
		t.Fatalf("fechas = %v, quiero [2027-02-28]", got)
	}
	// Año bisiesto: febrero 2028 -> 29.
	id2 := e.nuevoGastoFijo(31, "2028-02-01", nil)
	if _, err := e.gastos.GenerarHasta(fecha("2028-02-29")); err != nil {
		t.Fatal(err)
	}
	if got := e.periodos(id2); len(got) != 1 || got[0] != "2028-02-29" {
		t.Fatalf("bisiesto: fechas = %v, quiero [2028-02-29]", got)
	}
}

func TestTrastiendaGenerarHastaEsIdempotenteYHaceCatchUp(t *testing.T) {
	e := setupTrastienda(t)
	id := e.nuevoGastoFijo(5, "2029-01-01", nil)
	hoy := fecha("2029-04-10")
	for i := 0; i < 2; i++ {
		if _, err := e.gastos.GenerarHasta(hoy); err != nil {
			t.Fatal(err)
		}
	}
	got := e.periodos(id)
	want := []string{"2029-01-05", "2029-02-05", "2029-03-05", "2029-04-05"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("fechas = %v, quiero %v (4 periodos, sin duplicar tras correr dos veces)", got, want)
	}
}

func TestTrastiendaEditarPlantillaNoCambiaEgresosGeneradosYFechaFinCorta(t *testing.T) {
	e := setupTrastienda(t)
	fin := "2030-02-28"
	id := e.nuevoGastoFijo(10, "2030-01-01", &fin)
	if _, err := e.gastos.GenerarHasta(fecha("2030-01-20")); err != nil {
		t.Fatal(err)
	}
	nuevo := 999.0
	if err := e.gastos.Actualizar(id, GastoFijoPatch{Monto: &nuevo}); err != nil {
		t.Fatal(err)
	}
	var monto float64
	_ = e.db.Get(&monto, `SELECT monto FROM movimientos_financieros WHERE gasto_fijo_id = $1`, id)
	if monto != 100 {
		t.Fatalf("el egreso ya generado cambió a %v; debe conservar 100", monto)
	}
	// Los periodos futuros usan el monto nuevo y fecha_fin corta la generacion.
	if _, err := e.gastos.GenerarHasta(fecha("2030-06-30")); err != nil {
		t.Fatal(err)
	}
	got := e.periodos(id)
	if len(got) != 2 || got[1] != "2030-02-10" {
		t.Fatalf("fechas = %v, quiero enero y febrero solamente (fecha_fin)", got)
	}
	_ = e.db.Get(&monto, `SELECT monto FROM movimientos_financieros WHERE gasto_fijo_id = $1 AND fecha = '2030-02-10'`, id)
	if monto != 999 {
		t.Fatalf("el periodo nuevo debe usar el monto editado: %v", monto)
	}
}

// ─── Reporte ──────────────────────────────────────────────────────────────────

func TestTrastiendaReporteUtilidadYAnuladosNoSuman(t *testing.T) {
	e := setupTrastienda(t)
	cat := e.categoriaE
	dia := "2031-05-10"
	var catIngreso int
	_ = e.db.Get(&catIngreso, `SELECT id FROM categorias_financieras WHERE naturaleza = 'INGRESO' LIMIT 1`)

	mk := func(nat, clase, estado string, monto float64, categoria int) {
		mustExec(e.t, e.db, `
			INSERT INTO movimientos_financieros (naturaleza, clase, categoria_id, local_id, local_nombre, fecha, monto, estado)
			VALUES ($1, NULLIF($2,''), $3, $4, $5, $6, $7, $8)`, nat, clase, categoria, e.localID, e.localName, dia, monto, estado)
	}
	mk("EGRESO", "FIJO", "PAGADO", 1000, cat)
	mk("EGRESO", "VARIABLE", "PAGADO", 500, cat)
	mk("EGRESO", "VARIABLE", "ANULADO", 7777, cat)  // nunca suma
	mk("EGRESO", "VARIABLE", "PENDIENTE", 300, cat) // aparte, fuera de utilidad
	mk("INGRESO", "", "PAGADO", 200, catIngreso)

	rep, err := e.reporte.Generar(dia, dia, e.localName, "mes")
	if err != nil {
		t.Fatal(err)
	}
	r := rep.Resumen
	if r.EgresosFijos != 1000 || r.EgresosVariables != 500 || r.EgresosTotal != 1500 {
		t.Fatalf("egresos = %+v", r)
	}
	if r.EgresosPendientes != 300 {
		t.Fatalf("pendientes = %v, quiero 300 aparte", r.EgresosPendientes)
	}
	if r.IngresosOtros != 200 || r.UtilidadNeta != r.IngresosTotal-r.EgresosTotal {
		t.Fatalf("utilidad = %v, ingresos_total %v, egresos_total %v", r.UtilidadNeta, r.IngresosTotal, r.EgresosTotal)
	}
	if r.UtilidadNeta != -1300 {
		t.Fatalf("utilidad = %v, quiero -1300 (200 - 1500)", r.UtilidadNeta)
	}
}

func TestTrastiendaReporteVentasCoincideConPagos(t *testing.T) {
	e := setupTrastienda(t)
	dia := "2032-07-04"
	mustExec(t, e.db, `
		INSERT INTO pagos (local_id, local_nombre, cliente_nombre, subtotal, descuento, total_final, estado, tipo_pago, nombre_cajero, activo, fecha_creacion)
		VALUES ($1,$2,'Cliente',150,0,150,'PAGADO','efectivo','test',TRUE,$3::timestamptz),
		       ($1,$2,'Cliente',999,0,999,'PENDIENTE','efectivo','test',TRUE,$3::timestamptz)`, e.localID, e.localName, dia+" 12:00:00-04")
	rep, err := e.reporte.Generar(dia, dia, e.localName, "dia")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Resumen.IngresosVentas != 150 {
		t.Fatalf("ventas = %v, quiero 150 (solo PAGADO)", rep.Resumen.IngresosVentas)
	}
}

// ─── Sesiones de plan (RN-18) ─────────────────────────────────────────────────

func (e *trastiendaEnv) nuevoPlanConSesion(servicioNombre string) (planID, planServicioID int) {
	e.t.Helper()
	var servicioID int
	if err := e.db.Get(&servicioID, `SELECT id FROM servicios WHERE nombre = $1`, servicioNombre); err != nil {
		e.t.Fatal(err)
	}
	if err := e.db.Get(&planID, `
		INSERT INTO planes (cliente, cliente_nombre_texto, subtotal, precio_total, local_id)
		VALUES ('Cliente Plan', 'Cliente Plan', 100, 100, $1) RETURNING id`, e.localID); err != nil {
		e.t.Fatal(err)
	}
	if err := e.db.Get(&planServicioID, `
		INSERT INTO plan_servicios (plan_id, servicio_id_origen, nombre_texto, sesiones_contratadas, sesion_numero)
		VALUES ($1, $2, $3, 1, 1) RETURNING id`, planID, servicioID, servicioNombre); err != nil {
		e.t.Fatal(err)
	}
	return planID, planServicioID
}

func (e *trastiendaEnv) planesRepoConConsumo() *pgsqlrepo.PlanesRepo {
	repo := pgsqlrepo.NewPlanesRepo(e.db)
	repo.ConsumoSesion = func(tx *sqlx.Tx, servicioID, localID, planMovimientoID int) error {
		return e.consumo.ConsumirSesionPlan(tx, servicioID, localID, planMovimientoID, repository.ConsumoActor{})
	}
	return repo
}

func TestTrastiendaSesionDePlanSinReservaDescuentaUnaSolaVez(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Plan gel", 100)
	srv := e.nuevoServicio("PlanSrv", map[int]float64{item: 20})
	planID, _ := e.nuevoPlanConSesion(srv)
	repo := e.planesRepoConConsumo()

	for i := 0; i < 2; i++ { // marcar, desmarcar y volver a marcar no debe duplicar el descuento
		if _, err := repo.MarcarSesion(planID, 1, true); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.MarcarSesion(planID, 1, false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.MarcarSesion(planID, 1, true); err != nil {
		t.Fatal(err)
	}
	if got := e.stock(item); got != 80 {
		t.Fatalf("stock = %v, quiero 80 (un solo descuento de 20)", got)
	}
}

func TestTrastiendaSesionDePlanConReservaLigadaNoDescuentaDoble(t *testing.T) {
	e := setupTrastienda(t)
	item := e.nuevoItem("Plan gel 2", 100)
	srv := e.nuevoServicio("PlanSrv2", map[int]float64{item: 20})
	planID, planServicioID := e.nuevoPlanConSesion(srv)
	reserva := e.nuevaReserva(srv, "AGENDADO", 1)
	mustExec(t, e.db, `INSERT INTO plan_movimientos (plan_servicio_id, reserva_id, tipo, cantidad) VALUES ($1, $2, 'RESERVA', 1)`, planServicioID, reserva)

	// La reserva se completa (descuenta 20) y luego se marca la sesion: no debe descontar otros 20.
	if _, err := e.reservas.ActualizarEstadoReservaConAvisos(ActualizarEstadoReservaInput{Id: reserva, Estado: "COMPLETADO"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.planesRepoConConsumo().MarcarSesion(planID, 1, true); err != nil {
		t.Fatal(err)
	}
	if got := e.stock(item); got != 80 {
		t.Fatalf("stock = %v, quiero 80 (descuento doble evitado)", got)
	}
}
