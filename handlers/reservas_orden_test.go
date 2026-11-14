package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

func getReservasOrden(h *Container, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/reservas?"+query, nil)
	h.GetReservasSimplePG(c)
	return w
}

func reservasSiguienteCursor(t *testing.T, w *httptest.ResponseRecorder) string {
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

func TestGetReservasSimplePasaElOrdenPedidoAlRepositorio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &reservasPaginationRepo{rows: reservaRows(3), total: 3}
	h := &Container{ReservasPG: services.NewReservasPGService(repo, nil)}

	w := getReservasOrden(h, "orden=cronologico&ordenar_por=cliente&sentido=desc&limit=2")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if repo.filter.OrdenarPor != "cliente" || !repo.filter.OrdenDesc || repo.filter.Offset != 0 || repo.filter.CursorSet || repo.filter.PageLimit != 3 {
		t.Fatalf("filtro inesperado: %#v", repo.filter)
	}
}

func TestGetReservasSimpleSinOrdenPedidoNoCambiaElComportamiento(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &reservasPaginationRepo{rows: reservaRows(3), total: 3}
	h := &Container{ReservasPG: services.NewReservasPGService(repo, nil)}

	w := getReservasOrden(h, "orden=cronologico&limit=2")
	if w.Code != http.StatusOK || repo.filter.OrdenarPor != "" || !repo.filter.OrdenCronologico {
		t.Fatalf("status=%d filtro=%#v", w.Code, repo.filter)
	}
	cursor := reservasSiguienteCursor(t, w)

	w = getReservasOrden(h, "orden=cronologico&limit=2&cursor="+url.QueryEscape(cursor))
	if w.Code != http.StatusOK || !repo.filter.CursorSet || repo.filter.CursorID != 2 || repo.filter.Offset != 0 {
		t.Fatalf("la segunda pagina sin orden sigue siendo por valor: status=%d filtro=%#v", w.Code, repo.filter)
	}
}

func TestGetReservasSimpleOrdenadoAvanzaPorDesplazamiento(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &reservasPaginationRepo{rows: reservaRows(3), total: 3}
	h := &Container{ReservasPG: services.NewReservasPGService(repo, nil)}

	cursor := reservasSiguienteCursor(t, getReservasOrden(h, "ordenar_por=local&limit=2"))

	w := getReservasOrden(h, "ordenar_por=local&limit=2&cursor="+url.QueryEscape(cursor))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if repo.filter.Offset != 2 || repo.filter.CursorSet || repo.filter.OrdenarPor != "local" {
		t.Fatalf("la segunda pagina ordenada avanza por desplazamiento: %#v", repo.filter)
	}
}

func TestGetReservasSimpleRechazaOrdenInvalidoYCursorDeOtroOrden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &reservasPaginationRepo{rows: reservaRows(3), total: 3}
	h := &Container{ReservasPG: services.NewReservasPGService(repo, nil)}

	for _, query := range []string{"ordenar_por=password", "ordenar_por=cliente&sentido=arriba", "sentido=desc"} {
		if w := getReservasOrden(h, query); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d", query, w.Code)
		}
	}
	if repo.calls != 0 {
		t.Fatalf("un orden invalido no debe llegar al repositorio, llamadas = %d", repo.calls)
	}

	cursor := reservasSiguienteCursor(t, getReservasOrden(h, "ordenar_por=local&limit=2"))
	for _, query := range []string{"ordenar_por=cliente&limit=2", "ordenar_por=local&sentido=desc&limit=2", "limit=2"} {
		if w := getReservasOrden(h, query+"&cursor="+url.QueryEscape(cursor)); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: un cursor de otro orden debe rechazarse, status = %d", query, w.Code)
		}
	}
}
