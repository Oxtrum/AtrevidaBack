package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

type clientesOrdenRepo struct {
	rows   []models.ClientePG
	filter repository.FiltroClientes
	calls  int
}

func (r *clientesOrdenRepo) GetClientes(f repository.FiltroClientes) ([]models.ClientePG, error) {
	r.filter = f
	r.calls++
	return r.rows, nil
}
func (r *clientesOrdenRepo) CountClientes(repository.FiltroClientes) (int, error) {
	return len(r.rows), nil
}
func (r *clientesOrdenRepo) GetClienteByID(int) (*models.ClientePG, error)       { return nil, nil }
func (r *clientesOrdenRepo) CreateCliente(repository.CrearClienteInput) (int, error) { return 0, nil }
func (r *clientesOrdenRepo) UpdateCliente(repository.ActualizarClienteInput) error { return nil }
func (r *clientesOrdenRepo) DeleteCliente(int) error                             { return nil }

func clientesFilas(n int) []models.ClientePG {
	rows := make([]models.ClientePG, 0, n)
	for i := 1; i <= n; i++ {
		rows = append(rows, models.ClientePG{ID: i, Nombre: "N", Apellido: "A"})
	}
	return rows
}

func getClientesOrden(h *Container, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/clientes?"+query, nil)
	h.GetClientes(c)
	return w
}

func clientesSiguienteCursor(t *testing.T, w *httptest.ResponseRecorder) string {
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

func TestGetClientesPasaElOrdenPedidoAlRepositorio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &clientesOrdenRepo{rows: clientesFilas(3)}
	h := &Container{ClientesPG: services.NewClientesService(repo)}

	w := getClientesOrden(h, "ordenar_por=apellido&sentido=desc&limit=2")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if repo.filter.OrdenarPor != "apellido" || !repo.filter.OrdenDesc || repo.filter.Offset != 0 || repo.filter.CursorSet || repo.filter.PageLimit != 3 {
		t.Fatalf("filtro inesperado: %#v", repo.filter)
	}
}

func TestGetClientesSinOrdenNoCambiaElComportamiento(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &clientesOrdenRepo{rows: clientesFilas(3)}
	h := &Container{ClientesPG: services.NewClientesService(repo)}

	w := getClientesOrden(h, "limit=2")
	if w.Code != http.StatusOK || repo.filter.OrdenarPor != "" {
		t.Fatalf("status=%d filtro=%#v", w.Code, repo.filter)
	}
	cursor := clientesSiguienteCursor(t, w)

	w = getClientesOrden(h, "limit=2&cursor="+url.QueryEscape(cursor))
	if w.Code != http.StatusOK || !repo.filter.CursorSet || repo.filter.CursorID != 2 || repo.filter.Offset != 0 {
		t.Fatalf("la segunda pagina sin orden sigue siendo por valor: status=%d filtro=%#v", w.Code, repo.filter)
	}
}

func TestGetClientesOrdenadoAvanzaPorDesplazamiento(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &clientesOrdenRepo{rows: clientesFilas(3)}
	h := &Container{ClientesPG: services.NewClientesService(repo)}

	cursor := clientesSiguienteCursor(t, getClientesOrden(h, "ordenar_por=nombre&limit=2"))

	w := getClientesOrden(h, "ordenar_por=nombre&limit=2&cursor="+url.QueryEscape(cursor))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if repo.filter.Offset != 2 || repo.filter.CursorSet || repo.filter.OrdenarPor != "nombre" {
		t.Fatalf("la segunda pagina ordenada avanza por desplazamiento: %#v", repo.filter)
	}
}

func TestGetClientesRechazaOrdenInvalidoYCursorDeOtroOrden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &clientesOrdenRepo{rows: clientesFilas(3)}
	h := &Container{ClientesPG: services.NewClientesService(repo)}

	for _, query := range []string{"ordenar_por=password", "ordenar_por=nombre&sentido=arriba", "sentido=desc"} {
		if w := getClientesOrden(h, query); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d", query, w.Code)
		}
	}
	if repo.calls != 0 {
		t.Fatalf("un orden invalido no debe llegar al repositorio, llamadas = %d", repo.calls)
	}

	cursor := clientesSiguienteCursor(t, getClientesOrden(h, "ordenar_por=nombre&limit=2"))
	for _, query := range []string{"ordenar_por=apellido&limit=2", "ordenar_por=nombre&sentido=desc&limit=2", "limit=2"} {
		if w := getClientesOrden(h, query+"&cursor="+url.QueryEscape(cursor)); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: un cursor de otro orden debe rechazarse, status = %d", query, w.Code)
		}
	}
}
