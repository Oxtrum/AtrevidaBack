package handlers

import (
	"encoding/json"
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

type planesOrdenRepo struct {
	rows   []models.PlanPG
	filter repository.FiltroPlanes
	calls  int
}

func (r *planesOrdenRepo) ListPlanes(f repository.FiltroPlanes) ([]models.PlanPG, error) {
	r.filter = f
	r.calls++
	return r.rows, nil
}
func (r *planesOrdenRepo) CountPlanes(repository.FiltroPlanes) (int, error) { return len(r.rows), nil }
func (r *planesOrdenRepo) GetPlanByID(int) (*models.PlanCompletoPG, error)  { return nil, nil }
func (r *planesOrdenRepo) CreatePlan(repository.CrearPlanInput) (int, error) { return 0, nil }
func (r *planesOrdenRepo) UpdatePlan(repository.ActualizarPlanInput) error   { return nil }
func (r *planesOrdenRepo) UpdatePlanEstado(int, string, *int) error          { return nil }
func (r *planesOrdenRepo) MarcarSesion(int, int, bool) (int, error)          { return 0, nil }
func (r *planesOrdenRepo) CobrarPlan(int, string) error                      { return nil }

func planesFilas(n int) []models.PlanPG {
	rows := make([]models.PlanPG, 0, n)
	for i := 1; i <= n; i++ {
		rows = append(rows, models.PlanPG{ID: i, Estado: "ACTIVO", CreadoEn: time.Date(2026, 10, 28, 10, i, 0, 0, time.UTC)})
	}
	return rows
}

func getPlanesOrden(h *Container, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("auth_rol_codigo", "admin_sys")
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/planes?"+query, nil)
	h.GetPlanes(c)
	return w
}

func planesSiguienteCursor(t *testing.T, w *httptest.ResponseRecorder) string {
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

func TestGetPlanesPasaElOrdenPedidoAlRepositorio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &planesOrdenRepo{rows: planesFilas(3)}
	h := &Container{PlanesPG: services.NewPlanesService(repo, nil)}

	w := getPlanesOrden(h, "orden=prioridad_estado&ordenar_por=precio_total&sentido=desc&limit=2")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if repo.filter.OrdenarPor != "precio_total" || !repo.filter.OrdenDesc || repo.filter.Offset != 0 || repo.filter.CursorSet || repo.filter.PageLimit != 3 {
		t.Fatalf("filtro inesperado: %#v", repo.filter)
	}
}

// La ficha del cliente pide sus paquetes por id: el nombre no sirve, hay clientes homonimos.
func TestGetPlanesFiltraPorClienteID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &planesOrdenRepo{rows: planesFilas(1)}
	h := &Container{PlanesPG: services.NewPlanesService(repo, nil)}

	w := getPlanesOrden(h, "cliente_id=12")

	if w.Code != http.StatusOK || repo.filter.ClienteID == nil || *repo.filter.ClienteID != 12 {
		t.Fatalf("status=%d filtro=%#v", w.Code, repo.filter)
	}
	if w := getPlanesOrden(h, "cliente_id=abc"); w.Code != http.StatusBadRequest {
		t.Fatalf("cliente_id invalido: status = %d, want 400", w.Code)
	}
}

func TestGetPlanesSinOrdenPedidoConservaLaPrioridadDeEstado(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &planesOrdenRepo{rows: planesFilas(3)}
	h := &Container{PlanesPG: services.NewPlanesService(repo, nil)}

	w := getPlanesOrden(h, "orden=prioridad_estado&limit=2")
	if w.Code != http.StatusOK || repo.filter.OrdenarPor != "" || !repo.filter.OrdenPrioridadEstado {
		t.Fatalf("status=%d filtro=%#v", w.Code, repo.filter)
	}
	cursor := planesSiguienteCursor(t, w)

	w = getPlanesOrden(h, "orden=prioridad_estado&limit=2&cursor="+url.QueryEscape(cursor))
	if w.Code != http.StatusOK || !repo.filter.CursorSet || repo.filter.CursorID != 2 || repo.filter.Offset != 0 {
		t.Fatalf("la segunda pagina sin orden sigue siendo por valor: status=%d filtro=%#v", w.Code, repo.filter)
	}
}

func TestGetPlanesOrdenadoAvanzaPorDesplazamiento(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &planesOrdenRepo{rows: planesFilas(3)}
	h := &Container{PlanesPG: services.NewPlanesService(repo, nil)}

	cursor := planesSiguienteCursor(t, getPlanesOrden(h, "ordenar_por=cliente&limit=2"))

	w := getPlanesOrden(h, "ordenar_por=cliente&limit=2&cursor="+url.QueryEscape(cursor))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if repo.filter.Offset != 2 || repo.filter.CursorSet || repo.filter.OrdenarPor != "cliente" {
		t.Fatalf("la segunda pagina ordenada avanza por desplazamiento: %#v", repo.filter)
	}
}

func TestGetPlanesRechazaOrdenInvalidoYCursorDeOtroOrden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &planesOrdenRepo{rows: planesFilas(3)}
	h := &Container{PlanesPG: services.NewPlanesService(repo, nil)}

	for _, query := range []string{"ordenar_por=password", "ordenar_por=cliente&sentido=arriba", "sentido=desc"} {
		if w := getPlanesOrden(h, query); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d", query, w.Code)
		}
	}
	if repo.calls != 0 {
		t.Fatalf("un orden invalido no debe llegar al repositorio, llamadas = %d", repo.calls)
	}

	cursor := planesSiguienteCursor(t, getPlanesOrden(h, "ordenar_por=cliente&limit=2"))
	for _, query := range []string{"ordenar_por=estado&limit=2", "ordenar_por=cliente&sentido=desc&limit=2", "limit=2"} {
		if w := getPlanesOrden(h, query+"&cursor="+url.QueryEscape(cursor)); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: un cursor de otro orden debe rechazarse, status = %d", query, w.Code)
		}
	}
}
