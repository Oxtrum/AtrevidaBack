package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

type pagosOrdenRepo struct {
	rows   []models.PagoPG
	filter repository.FiltroPagos
	calls  int
}

func (r *pagosOrdenRepo) GetPagos(f repository.FiltroPagos) ([]models.PagoPG, error) {
	r.filter = f
	r.calls++
	return r.rows, nil
}
func (r *pagosOrdenRepo) CountPagos(repository.FiltroPagos) (int, error) { return len(r.rows), nil }
func (r *pagosOrdenRepo) GetPagoByCodigo(string) (*models.PagoCompletoPG, error) {
	return nil, nil
}
func (r *pagosOrdenRepo) CreatePago(repository.CrearPagoInput) (string, error) { return "", nil }
func (r *pagosOrdenRepo) UpdatePago(repository.ActualizarPagoInput) error     { return nil }
func (r *pagosOrdenRepo) DeletePago(string, repository.ConsumoActor) error     { return nil }
func (r *pagosOrdenRepo) GetResumenPagos(repository.FiltroResumenPagos) (repository.PagoResumenAgregado, error) {
	return repository.PagoResumenAgregado{}, nil
}
func (r *pagosOrdenRepo) ListVentasDetalle(repository.FiltroResumenPagos, int) ([]repository.VentaDetalleRow, error) {
	return nil, nil
}

func pagosFilas(n int) []models.PagoPG {
	rows := make([]models.PagoPG, 0, n)
	for i := 1; i <= n; i++ {
		rows = append(rows, models.PagoPG{ID: i, FechaCreacion: time.Date(2026, 10, 28, 10, i, 0, 0, time.UTC)})
	}
	return rows
}

func getPagosOrden(h *Container, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/pagos?"+query, nil)
	h.GetPagos(c)
	return w
}

func siguienteCursor(t *testing.T, w *httptest.ResponseRecorder) string {
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

func TestGetPagosPasaElOrdenPedidoAlRepositorio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &pagosOrdenRepo{rows: pagosFilas(3)}
	h := &Container{PagosPG: services.NewPagosService(repo)}

	w := getPagosOrden(h, "ordenar_por=total_final&sentido=desc&limit=2")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if repo.filter.OrdenarPor != "total_final" || !repo.filter.OrdenDesc || repo.filter.Offset != 0 || repo.filter.CursorSet || repo.filter.PageLimit != 3 {
		t.Fatalf("filtro inesperado: %#v", repo.filter)
	}
}

func TestGetPagosSinOrdenNoCambiaElComportamiento(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &pagosOrdenRepo{rows: pagosFilas(3)}
	h := &Container{PagosPG: services.NewPagosService(repo)}

	w := getPagosOrden(h, "limit=2")
	if w.Code != http.StatusOK || repo.filter.OrdenarPor != "" || repo.filter.OrdenDesc {
		t.Fatalf("status=%d filtro=%#v", w.Code, repo.filter)
	}
	cursor := siguienteCursor(t, w)

	w = getPagosOrden(h, "limit=2&cursor="+url.QueryEscape(cursor))
	if w.Code != http.StatusOK || !repo.filter.CursorSet || repo.filter.CursorID != 2 || repo.filter.Offset != 0 {
		t.Fatalf("la segunda pagina sin orden sigue siendo por valor: status=%d filtro=%#v", w.Code, repo.filter)
	}
}

func TestGetPagosOrdenadoAvanzaPorDesplazamiento(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &pagosOrdenRepo{rows: pagosFilas(3)}
	h := &Container{PagosPG: services.NewPagosService(repo)}

	cursor := siguienteCursor(t, getPagosOrden(h, "ordenar_por=cliente_nombre&limit=2"))

	w := getPagosOrden(h, "ordenar_por=cliente_nombre&limit=2&cursor="+url.QueryEscape(cursor))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if repo.filter.Offset != 2 || repo.filter.CursorSet || repo.filter.OrdenarPor != "cliente_nombre" {
		t.Fatalf("la segunda pagina ordenada avanza por desplazamiento: %#v", repo.filter)
	}
}

func TestGetPagosRechazaOrdenInvalido(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &pagosOrdenRepo{rows: pagosFilas(1)}
	h := &Container{PagosPG: services.NewPagosService(repo)}

	for _, query := range []string{
		"ordenar_por=password",
		"ordenar_por=total_final&sentido=arriba",
		"sentido=desc",
		"ordenar_por=" + url.QueryEscape("p.id; DROP TABLE pagos"),
	} {
		if w := getPagosOrden(h, query); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, body = %s", query, w.Code, w.Body.String())
		}
	}
	if repo.calls != 0 {
		t.Fatalf("un orden invalido no debe llegar al repositorio, llamadas = %d", repo.calls)
	}
}

func TestGetPagosCursorDeOtroOrdenSeRechaza(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &pagosOrdenRepo{rows: pagosFilas(3)}
	h := &Container{PagosPG: services.NewPagosService(repo)}

	cursor := siguienteCursor(t, getPagosOrden(h, "ordenar_por=cliente_nombre&limit=2"))

	for _, query := range []string{
		"ordenar_por=total_final&limit=2",
		"ordenar_por=cliente_nombre&sentido=desc&limit=2",
		"limit=2",
	} {
		if w := getPagosOrden(h, query+"&cursor="+url.QueryEscape(cursor)); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: un cursor de otro orden debe rechazarse, status = %d", query, w.Code)
		}
	}
}

func TestStatusPagoErrorTratarLosErroresDeFechaHechoComoValidacion(t *testing.T) {
	for _, msg := range []string{
		"fecha_hecho con formato invalido, use YYYY-MM-DD",
		"fecha_hecho no puede ser futura",
		"fecha_hecho no puede ser anterior a 3 dias",
	} {
		if got := statusPagoError(errors.New(msg)); got != http.StatusBadRequest {
			t.Fatalf("%q: status = %d, esperado 400", msg, got)
		}
	}
}
