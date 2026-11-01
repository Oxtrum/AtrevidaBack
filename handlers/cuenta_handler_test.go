package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"atrevida-agenda-api/cuenta"
	"atrevida-agenda-api/models"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

type cuentaStoreFake struct {
	existe      bool
	cargo       *pgsqlrepo.CargoInput
	abono       *pgsqlrepo.AbonoInput
	cobro       *pgsqlrepo.CobroInput
	errCobro    error
	anuladoID   int
	anuladoMot  string
	aplicadoA   string
	aplicado    []cuenta.Reparto
	errAbono    error
	totalAbonos float64
	totalCargos float64
	deudores    []models.DeudorCliente
	filtroDeud  pgsqlrepo.FiltroDeudores
}

func (f *cuentaStoreFake) ClienteExiste(int) (bool, error) { return f.existe, nil }
func (f *cuentaStoreFake) CrearCargo(in pgsqlrepo.CargoInput) (int, error) {
	f.cargo = &in
	return 12, nil
}
func (f *cuentaStoreFake) AnularCargo(id int, motivo string) error {
	f.anuladoID, f.anuladoMot = id, motivo
	return nil
}
func (f *cuentaStoreFake) ListarCargos(int, bool) ([]models.CargoCliente, error) {
	return []models.CargoCliente{{ID: 1, Monto: 100, Pagado: 40}}, nil
}
func (f *cuentaStoreFake) ListarAbonos(int) ([]models.AbonoCliente, error) {
	return []models.AbonoCliente{}, nil
}
func (f *cuentaStoreFake) TotalesCuenta(int) (float64, float64, error) {
	return f.totalAbonos, f.totalCargos, nil
}
func (f *cuentaStoreFake) CrearAbono(in pgsqlrepo.AbonoInput) (string, error) {
	if f.errAbono != nil {
		return "", f.errAbono
	}
	f.abono = &in
	return "PAGO-000009", nil
}
func (f *cuentaStoreFake) Cobrar(in pgsqlrepo.CobroInput) (pgsqlrepo.CobroResultado, error) {
	if f.errCobro != nil {
		return pgsqlrepo.CobroResultado{}, f.errCobro
	}
	f.cobro = &in
	return pgsqlrepo.CobroResultado{CodigoPago: "PAGO-000012", Plan: cuenta.PlanCobro{Total: 250, UsaSaldo: 150, Falta: 100, NuevoPago: 100}}, nil
}
func (f *cuentaStoreFake) AplicarAbono(codigo string, _ int, ap []cuenta.Reparto) error {
	f.aplicadoA, f.aplicado = codigo, ap
	return nil
}

func (f *cuentaStoreFake) ListarDeudores(fl pgsqlrepo.FiltroDeudores) ([]models.DeudorCliente, error) {
	f.filtroDeud = fl
	return f.deudores, nil
}
func (f *cuentaStoreFake) ContarDeudores(pgsqlrepo.FiltroDeudores) (int, error) { return len(f.deudores), nil }

func hCuenta(f *cuentaStoreFake) *Container {
	return &Container{Cuenta: services.NewCuentaService(f)}
}

// llamarCuenta ejecuta un handler con params de ruta, body JSON y un usuario autenticado.
func llamarCuenta(handler gin.HandlerFunc, params gin.Params, body string, usuario map[string]interface{}) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	for k, v := range usuario {
		c.Set(k, v)
	}
	handler(c)
	return w
}

var (
	adminCuenta = map[string]interface{}{"auth_rol_codigo": "admin_sys", "auth_username": "admin", "auth_user_id": 1}
	cajeroLocal1 = map[string]interface{}{
		"auth_rol_codigo": "cajero", "auth_username": "ana", "auth_user_id": 5,
		"auth_local_id": 1, "auth_nombre_local": "SAN MARTIN",
	}
	clienteSiete = gin.Params{{Key: "id", Value: "7"}}
)

func TestGetCuentaClienteDevuelveSaldoYCargosConEstado(t *testing.T) {
	f := &cuentaStoreFake{existe: true, totalAbonos: 40, totalCargos: 100}
	h := hCuenta(f)
	w := llamarCuenta(h.GetCuentaCliente, clienteSiete, "", adminCuenta)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			Cuenta models.CuentaCliente `json:"cuenta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	c := body.Data.Cuenta
	if c.Saldo.Deuda != 60 || c.Saldo.PorPagar != 60 || c.Saldo.Disponible != 0 || len(c.Cargos) != 1 || c.Cargos[0].Estado != cuenta.CargoParcial {
		t.Fatalf("cuenta = %+v", c)
	}
}

func TestGetCuentaClienteInexistenteEs404(t *testing.T) {
	w := llamarCuenta(hCuenta(&cuentaStoreFake{existe: false}).GetCuentaCliente, clienteSiete, "", adminCuenta)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestGetCuentaClienteConIdInvalidoEs400(t *testing.T) {
	w := llamarCuenta(hCuenta(&cuentaStoreFake{}).GetCuentaCliente, gin.Params{{Key: "id", Value: "abc"}}, "", adminCuenta)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestPostCargoClienteCreaYResponde201(t *testing.T) {
	f := &cuentaStoreFake{existe: true}
	w := llamarCuenta(hCuenta(f).PostCargoCliente, clienteSiete,
		`{"tipo":"paquete","concepto":"Paquete Relax","monto":1000,"fecha_hecho":"2026-10-25"}`, adminCuenta)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if f.cargo == nil || f.cargo.Tipo != "PAQUETE" || f.cargo.Monto != 1000 || f.cargo.ClienteID != 7 || f.cargo.CreadoPor != "admin" {
		t.Fatalf("cargo = %#v", f.cargo)
	}
}

func TestPostCargoClienteConMontoCeroEsValido(t *testing.T) {
	f := &cuentaStoreFake{existe: true}
	w := llamarCuenta(hCuenta(f).PostCargoCliente, clienteSiete, `{"tipo":"INSUMO","concepto":"Anestesia","monto":0}`, adminCuenta)
	if w.Code != http.StatusCreated {
		t.Fatalf("un consumo sin cobro es un cargo en 0: status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestPostCargoClienteRechazaDatosInvalidosComo400(t *testing.T) {
	for _, body := range []string{
		`{"tipo":"REGALO","concepto":"x","monto":1}`,
		`{"tipo":"OTRO","concepto":"","monto":1}`,
		`{"tipo":"OTRO","concepto":"x","monto":-1}`,
		`{"tipo":"OTRO","concepto":"x"}`,
		`no es json`,
	} {
		f := &cuentaStoreFake{existe: true}
		w := llamarCuenta(hCuenta(f).PostCargoCliente, clienteSiete, body, adminCuenta)
		if w.Code != http.StatusBadRequest || f.cargo != nil {
			t.Fatalf("%s: status = %d, cargo = %#v", body, w.Code, f.cargo)
		}
	}
}

func TestPostAnularCargoExigeMotivo(t *testing.T) {
	f := &cuentaStoreFake{}
	params := gin.Params{{Key: "id", Value: "12"}}
	if w := llamarCuenta(hCuenta(f).PostAnularCargo, params, `{"motivo":"  "}`, adminCuenta); w.Code != http.StatusBadRequest {
		t.Fatalf("sin motivo status = %d", w.Code)
	}
	if w := llamarCuenta(hCuenta(f).PostAnularCargo, params, `{"motivo":"duplicado"}`, adminCuenta); w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if f.anuladoID != 12 || f.anuladoMot != "duplicado" {
		t.Fatalf("anulado = %d %q", f.anuladoID, f.anuladoMot)
	}
}

func TestPostAbonoClienteCreaConAplicacionesYCajeroDelToken(t *testing.T) {
	f := &cuentaStoreFake{existe: true}
	body := `{"local_id":1,"local_nombre":"SAN MARTIN","monto":500,"tipo_pago":"qr","fecha_hecho":"2026-10-25",
		"aplicaciones":[{"cargo_id":1,"monto":400}]}`
	w := llamarCuenta(hCuenta(f).PostAbonoCliente, clienteSiete, body, cajeroLocal1)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	a := f.abono
	if a == nil || a.ClienteID != 7 || a.Monto != 500 || a.TipoPago != "qr" || len(a.Aplicaciones) != 1 || a.CajeroNombre != "ana" {
		t.Fatalf("abono = %#v", a)
	}
	if !strings.Contains(w.Body.String(), "PAGO-000009") {
		t.Fatalf("la respuesta debe traer el codigo: %s", w.Body.String())
	}
}

func TestPostAbonoClienteRespetaElLocalDelUsuario(t *testing.T) {
	f := &cuentaStoreFake{existe: true}
	body := `{"local_id":2,"local_nombre":"PASEO ARANJUEZ","monto":500,"tipo_pago":"efectivo"}`
	w := llamarCuenta(hCuenta(f).PostAbonoCliente, clienteSiete, body, cajeroLocal1)
	if w.Code != http.StatusForbidden || f.abono != nil {
		t.Fatalf("un usuario de otro local no puede cobrar aqui: status = %d", w.Code)
	}
}

func TestPostAbonoClienteRechazaMontoInvalidoComo400(t *testing.T) {
	for _, body := range []string{
		`{"local_id":1,"local_nombre":"SAN MARTIN","monto":0,"tipo_pago":"efectivo"}`,
		`{"local_id":1,"local_nombre":"SAN MARTIN","monto":10,"tipo_pago":"cheque"}`,
		`{"local_id":1,"local_nombre":"SAN MARTIN","monto":10,"tipo_pago":"qr","fecha_hecho":"2999-01-01"}`,
		`{"monto":10,"tipo_pago":"qr"}`,
	} {
		f := &cuentaStoreFake{existe: true}
		w := llamarCuenta(hCuenta(f).PostAbonoCliente, clienteSiete, body, adminCuenta)
		if w.Code != http.StatusBadRequest || f.abono != nil {
			t.Fatalf("%s: status = %d, abono = %#v", body, w.Code, f.abono)
		}
	}
}

func TestPostAbonoClienteMapeaLasReglasDeLaCuentaA4xxYLaInfraA500(t *testing.T) {
	body := `{"local_id":1,"local_nombre":"SAN MARTIN","monto":10,"tipo_pago":"qr"}`
	regla := &cuentaStoreFake{existe: true, errAbono: pgsqlrepo.ErrReglaCuenta{Msg: "la aplicacion no puede superar lo pendiente del cargo 3 (30.00)"}}
	if w := llamarCuenta(hCuenta(regla).PostAbonoCliente, clienteSiete, body, adminCuenta); w.Code != http.StatusBadRequest {
		t.Fatalf("regla: status = %d", w.Code)
	}
	noEncontrado := &cuentaStoreFake{existe: true, errAbono: pgsqlrepo.ErrReglaCuenta{Msg: "cliente no encontrado"}}
	if w := llamarCuenta(hCuenta(noEncontrado).PostAbonoCliente, clienteSiete, body, adminCuenta); w.Code != http.StatusNotFound {
		t.Fatalf("no encontrado: status = %d", w.Code)
	}
	infra := &cuentaStoreFake{existe: true, errAbono: assertErr("no se pudo registrar el abono")}
	if w := llamarCuenta(hCuenta(infra).PostAbonoCliente, clienteSiete, body, adminCuenta); w.Code != http.StatusInternalServerError {
		t.Fatalf("infraestructura: status = %d", w.Code)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

func TestPostAplicacionesAbonoAplicaYExigeAplicaciones(t *testing.T) {
	f := &cuentaStoreFake{existe: true}
	params := gin.Params{{Key: "id", Value: "7"}, {Key: "codigo_pago", Value: "PAGO-000009"}}
	if w := llamarCuenta(hCuenta(f).PostAplicacionesAbono, params, `{"aplicaciones":[]}`, adminCuenta); w.Code != http.StatusBadRequest {
		t.Fatalf("sin aplicaciones status = %d", w.Code)
	}
	w := llamarCuenta(hCuenta(f).PostAplicacionesAbono, params, `{"aplicaciones":[{"cargo_id":2,"monto":30}]}`, adminCuenta)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if f.aplicadoA != "PAGO-000009" || len(f.aplicado) != 1 || f.aplicado[0].CargoID != 2 {
		t.Fatalf("aplicado = %q %+v", f.aplicadoA, f.aplicado)
	}
}

func deudoresFilas(n int) []models.DeudorCliente {
	filas := make([]models.DeudorCliente, 0, n)
	for i := 1; i <= n; i++ {
		filas = append(filas, models.DeudorCliente{ClienteID: i, Nombre: "N", Apellido: "A", Deuda: float64(100 - i)})
	}
	return filas
}

func getDeudores(h *Container, query string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/cuentas/deudores?"+query, nil)
	h.GetDeudores(c)
	return w
}

func deudoresCursor(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Data struct {
			Paginacion struct {
				NextCursor *string `json:"next_cursor"`
			} `json:"paginacion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Data.Paginacion.NextCursor == nil {
		t.Fatalf("cursor ausente: err=%v body=%s", err, w.Body.String())
	}
	return *body.Data.Paginacion.NextCursor
}

func TestGetDeudoresPaginaPorDesplazamientoAunSinOrdenPedido(t *testing.T) {
	f := &cuentaStoreFake{deudores: deudoresFilas(3)}
	h := hCuenta(f)

	w := getDeudores(h, "limit=2")
	if w.Code != http.StatusOK || f.filtroDeud.Limit != 3 || f.filtroDeud.Offset != 0 || f.filtroDeud.OrdenarPor != "" {
		t.Fatalf("primera pagina: status=%d filtro=%+v", w.Code, f.filtroDeud)
	}
	cursor := deudoresCursor(t, w)

	w = getDeudores(h, "limit=2&cursor="+url.QueryEscape(cursor))
	if w.Code != http.StatusOK || f.filtroDeud.Offset != 2 {
		t.Fatalf("segunda pagina: status=%d filtro=%+v", w.Code, f.filtroDeud)
	}
}

func TestGetDeudoresPasaElOrdenPedido(t *testing.T) {
	f := &cuentaStoreFake{deudores: deudoresFilas(1)}
	w := getDeudores(hCuenta(f), "ordenar_por=apellido&sentido=desc&busqueda=Maria&limit=10")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	g := f.filtroDeud
	if g.OrdenarPor != "apellido" || !g.OrdenDesc || g.Busqueda != "Maria" {
		t.Fatalf("filtro = %+v", g)
	}
}

func TestGetDeudoresRechazaOrdenInvalidoYCursorDeOtroOrden(t *testing.T) {
	f := &cuentaStoreFake{deudores: deudoresFilas(3)}
	h := hCuenta(f)
	for _, q := range []string{"ordenar_por=password", "ordenar_por=deuda&sentido=arriba", "sentido=desc"} {
		if w := getDeudores(h, q); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d", q, w.Code)
		}
	}
	cursor := deudoresCursor(t, getDeudores(h, "ordenar_por=nombre&limit=2"))
	for _, q := range []string{"ordenar_por=apellido&limit=2", "ordenar_por=nombre&sentido=desc&limit=2", "limit=2", "limit=2&busqueda=x"} {
		if w := getDeudores(h, q+"&cursor="+url.QueryEscape(cursor)); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: un cursor de otro orden o busqueda debe rechazarse, status = %d", q, w.Code)
		}
	}
}

func TestPostCobroClienteCobraLosCargosMarcadosYDevuelveElPlan(t *testing.T) {
	f := &cuentaStoreFake{existe: true}
	body := `{"local_id":1,"local_nombre":"SAN MARTIN","cargo_ids":[10,11],"usar_saldo":true,"tipo_pago":"qr","recarga":20,"fecha_hecho":"2026-10-25"}`
	w := llamarCuenta(hCuenta(f).PostCobroCliente, clienteSiete, body, cajeroLocal1)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	c := f.cobro
	if c == nil || c.ClienteID != 7 || len(c.CargoIDs) != 2 || !c.UsarSaldo || c.TipoPago != "qr" || c.Recarga != 20 || c.CajeroNombre != "ana" {
		t.Fatalf("cobro = %#v", c)
	}
	for _, esperado := range []string{"PAGO-000012", `"usa_saldo":150`, `"falta":100`} {
		if !strings.Contains(w.Body.String(), esperado) {
			t.Fatalf("la respuesta debe traer %s: %s", esperado, w.Body.String())
		}
	}
}

func TestPostCobroClienteRespetaElLocalDelUsuario(t *testing.T) {
	f := &cuentaStoreFake{existe: true}
	body := `{"local_id":2,"local_nombre":"PASEO ARANJUEZ","cargo_ids":[10],"usar_saldo":true}`
	w := llamarCuenta(hCuenta(f).PostCobroCliente, clienteSiete, body, cajeroLocal1)
	if w.Code != http.StatusForbidden || f.cobro != nil {
		t.Fatalf("un usuario de otro local no puede cobrar aqui: status = %d", w.Code)
	}
}

func TestPostCobroClienteRechazaPedidosInvalidosComo400(t *testing.T) {
	for _, body := range []string{
		`{"local_id":1,"local_nombre":"SAN MARTIN","cargo_ids":[],"recarga":0}`,
		`{"local_id":1,"local_nombre":"SAN MARTIN","cargo_ids":[1,1],"usar_saldo":true}`,
		`{"local_id":1,"local_nombre":"SAN MARTIN","cargo_ids":[1],"tipo_pago":"cheque"}`,
		`{"local_id":1,"local_nombre":"SAN MARTIN","cargo_ids":[1],"tipo_pago":"qr","fecha_hecho":"2999-01-01"}`,
		`{"cargo_ids":[1],"tipo_pago":"qr"}`,
		`no es json`,
	} {
		f := &cuentaStoreFake{existe: true}
		w := llamarCuenta(hCuenta(f).PostCobroCliente, clienteSiete, body, adminCuenta)
		if w.Code != http.StatusBadRequest || f.cobro != nil {
			t.Fatalf("%s: status = %d, cobro = %#v", body, w.Code, f.cobro)
		}
	}
}

func TestPostCobroClienteMapeaLasReglasA4xxYLaInfraA500(t *testing.T) {
	body := `{"local_id":1,"local_nombre":"SAN MARTIN","cargo_ids":[3],"usar_saldo":true,"tipo_pago":"qr"}`
	regla := &cuentaStoreFake{existe: true, errCobro: pgsqlrepo.ErrReglaCuenta{Msg: "el dinero disponible no alcanza para cubrir los cargos"}}
	if w := llamarCuenta(hCuenta(regla).PostCobroCliente, clienteSiete, body, adminCuenta); w.Code != http.StatusBadRequest {
		t.Fatalf("regla: status = %d", w.Code)
	}
	noEncontrado := &cuentaStoreFake{existe: true, errCobro: pgsqlrepo.ErrReglaCuenta{Msg: "cargo 3 no encontrado entre los pendientes del cliente"}}
	if w := llamarCuenta(hCuenta(noEncontrado).PostCobroCliente, clienteSiete, body, adminCuenta); w.Code != http.StatusNotFound {
		t.Fatalf("no encontrado: status = %d", w.Code)
	}
	infra := &cuentaStoreFake{existe: true, errCobro: assertErr("no se pudo registrar el cobro")}
	if w := llamarCuenta(hCuenta(infra).PostCobroCliente, clienteSiete, body, adminCuenta); w.Code != http.StatusInternalServerError {
		t.Fatalf("infraestructura: status = %d", w.Code)
	}
}
