package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

type vinculoStoreHandlerFake struct {
	reservaID, clienteID int
	mismoTelefono        bool
	vinculadas           int
	err                  error
	limit, offset        int
}

func (f *vinculoStoreHandlerFake) PorVincular(limit, offset int) ([]models.ReservaPorVincularPG, int, error) {
	f.limit, f.offset = limit, offset
	return []models.ReservaPorVincularPG{{ID: 41, Cliente: "Lucia Paredes"}}, 9, f.err
}

func (f *vinculoStoreHandlerFake) Sugerencias(reservaID int) ([]models.ClienteSugeridoPG, error) {
	f.reservaID = reservaID
	return []models.ClienteSugeridoPG{{ClientePG: models.ClientePG{ID: 12, Nombre: "Lucia"}, PorTelefono: true}}, f.err
}

func (f *vinculoStoreHandlerFake) Vincular(reservaID, clienteID int, mismoTelefono bool) (int, error) {
	f.reservaID, f.clienteID, f.mismoTelefono = reservaID, clienteID, mismoTelefono
	return f.vinculadas, f.err
}

func routerVinculo(store *vinculoStoreHandlerFake) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &Container{ReservasVinculo: services.NewReservasVinculoService(store)}
	r := gin.New()
	r.GET("/reservas/por-vincular", h.GetReservasPorVincular)
	r.GET("/reservas/:id/sugerencias-cliente", h.GetSugerenciasClienteReserva)
	r.PATCH("/reservas/:id/cliente", h.PatchReservaCliente)
	return r
}

func llamarVinculo(r *gin.Engine, metodo, url, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPatchReservaClienteVinculaYDevuelveCuantas(t *testing.T) {
	store := &vinculoStoreHandlerFake{vinculadas: 3}

	w := llamarVinculo(routerVinculo(store), http.MethodPatch, "/reservas/41/cliente", `{"cliente_id":12,"vincular_mismo_telefono":true}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if store.reservaID != 41 || store.clienteID != 12 || !store.mismoTelefono {
		t.Fatalf("store = %+v", store)
	}
	var body struct {
		Data struct {
			Vinculadas int `json:"vinculadas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Data.Vinculadas != 3 {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestPatchReservaClienteSinClienteIDEs400(t *testing.T) {
	w := llamarVinculo(routerVinculo(&vinculoStoreHandlerFake{}), http.MethodPatch, "/reservas/41/cliente", `{}`)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "cliente_id") {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestPatchReservaClienteMapeaLosErroresDelRepositorio(t *testing.T) {
	casos := map[string]int{
		"reserva no encontrada":                    http.StatusNotFound,
		"cliente no encontrado":                    http.StatusNotFound,
		"la reserva ya tiene un cliente vinculado": http.StatusConflict,
		"conexion perdida":                         http.StatusInternalServerError,
	}
	for mensaje, esperado := range casos {
		store := &vinculoStoreHandlerFake{err: errors.New(mensaje)}
		w := llamarVinculo(routerVinculo(store), http.MethodPatch, "/reservas/41/cliente", `{"cliente_id":12}`)
		if w.Code != esperado {
			t.Errorf("%q: status = %d, want %d", mensaje, w.Code, esperado)
		}
	}
}

func TestGetReservasPorVincularPasaLaPaginaYDevuelveElTotal(t *testing.T) {
	store := &vinculoStoreHandlerFake{}

	w := llamarVinculo(routerVinculo(store), http.MethodGet, "/reservas/por-vincular?limit=20&offset=40", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if store.limit != 20 || store.offset != 40 {
		t.Fatalf("pagina = %d/%d, want 20/40", store.limit, store.offset)
	}
	var body struct {
		Data struct {
			Total    int           `json:"total"`
			Reservas []interface{} `json:"reservas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Data.Total != 9 || len(body.Data.Reservas) != 1 {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestGetSugerenciasClienteReservaDevuelveLasPistas(t *testing.T) {
	store := &vinculoStoreHandlerFake{}

	w := llamarVinculo(routerVinculo(store), http.MethodGet, "/reservas/41/sugerencias-cliente", "")

	if w.Code != http.StatusOK || store.reservaID != 41 {
		t.Fatalf("status = %d, reserva = %d, body = %s", w.Code, store.reservaID, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"por_telefono":true`) {
		t.Fatalf("body = %s", w.Body.String())
	}
}
